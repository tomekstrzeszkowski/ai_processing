package web_rtc

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/abema/go-mp4"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type VideoSample struct {
	Data     []byte
	Duration time.Duration
	IsSync   bool
}
type StaticVideoTrack struct {
	track      *webrtc.TrackLocalStaticSample
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	playCtx    context.Context
	playCancel context.CancelFunc
	samples    []VideoSample
	sps        []byte
	pps        []byte
	currentIdx int
	timescale  uint32
	totalDur   time.Duration
	playing    bool
	currentPos time.Duration
	frameCount int64
	rtpSender  *webrtc.RTPSender
	playWait   sync.WaitGroup
	isLoop     bool
}

func NewStaticVideoTrack() (*StaticVideoTrack, error) {
	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"static_file",
		"video_frame_static_file",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create video track: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	playCtx, playCancel := context.WithCancel(context.Background())

	return &StaticVideoTrack{
		track:      track,
		ctx:        ctx,
		cancel:     cancel,
		playCtx:    playCtx,
		playCancel: playCancel,
		totalDur:   -1 * time.Second,
		isLoop:     false,
	}, nil
}

func (vt *StaticVideoTrack) LoadVideo(filePath string) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	videoPath, err := ensureMP4Container(filePath)
	if err != nil {
		return fmt.Errorf("failed to probe MP4: %w", err)
	}
	file, err := os.Open(videoPath)
	if err != nil {
		return fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	// Parse MP4 structure
	info, err := mp4.Probe(file)
	if err != nil {
		return fmt.Errorf("failed to probe MP4: %w", err)
	}

	// Find video track
	var videoTrack *mp4.Track
	for _, track := range info.Tracks {
		if track.AVC != nil {
			videoTrack = track
			break
		}
	}

	if videoTrack == nil {
		return fmt.Errorf("no H.264 video track found in MP4")
	}

	vt.timescale = videoTrack.Timescale

	// Extract SPS/PPS from avcC box
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start: %w", err)
	}

	boxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{
		mp4.BoxTypeMoov(),
		mp4.BoxTypeTrak(),
		mp4.BoxTypeMdia(),
		mp4.BoxTypeMinf(),
		mp4.BoxTypeStbl(),
		mp4.BoxTypeStsd(),
		mp4.BoxTypeAvc1(),
		mp4.BoxTypeAvcC(),
	})
	if err != nil {
		return fmt.Errorf("failed to extract avcC box: %w", err)
	}

	if len(boxes) == 0 {
		return fmt.Errorf("avcC box not found")
	}

	avcC, ok := boxes[0].Payload.(*mp4.AVCDecoderConfiguration)
	if !ok {
		return fmt.Errorf("failed to cast to AVCDecoderConfiguration")
	}

	if len(avcC.SequenceParameterSets) > 0 {
		vt.sps = avcC.SequenceParameterSets[0].NALUnit
	}
	if len(avcC.PictureParameterSets) > 0 {
		vt.pps = avcC.PictureParameterSets[0].NALUnit
	}

	if vt.sps == nil || vt.pps == nil {
		return fmt.Errorf("failed to extract SPS/PPS from MP4")
	}

	log.Printf("Found video track: timescale=%d, samples=%d", vt.timescale, len(videoTrack.Samples))
	log.Printf("SPS length: %d, PPS length: %d", len(vt.sps), len(vt.pps))

	// Find IDR frames
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start: %w", err)
	}
	idrFrames, err := mp4.FindIDRFrames(file, videoTrack)
	if err != nil {
		log.Printf("Warning: failed to find IDR frames: %v", err)
	}
	idrSet := make(map[int]bool)
	for _, idx := range idrFrames {
		idrSet[idx] = true
	}

	// Calculate sample offsets from chunks
	sampleOffsets := calculateSampleOffsets(videoTrack)

	// Read all samples
	vt.samples = make([]VideoSample, 0, len(videoTrack.Samples))
	vt.totalDur = 0

	for i, sample := range videoTrack.Samples {
		if i >= len(sampleOffsets) {
			break
		}

		offset := sampleOffsets[i]
		isIDR := idrSet[i]

		// Read sample data from file
		if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek to sample: %w", err)
		}

		data := make([]byte, sample.Size)
		if _, err := io.ReadFull(file, data); err != nil {
			return fmt.Errorf("failed to read sample: %w", err)
		}

		// Convert AVCC to Annex B format
		annexBData := vt.avccToAnnexB(data, isIDR)

		// Calculate duration from TimeDelta
		duration := time.Duration(sample.TimeDelta) * time.Second / time.Duration(vt.timescale)

		vt.samples = append(vt.samples, VideoSample{
			Data:     annexBData,
			Duration: duration,
			IsSync:   isIDR,
		})

		vt.totalDur += duration
	}

	log.Printf("Video loaded: %d samples, duration %v", len(vt.samples), vt.totalDur)

	return nil
}

// avccToAnnexB converts AVCC format (length-prefixed) to Annex B format (start codes)
func (vt *StaticVideoTrack) avccToAnnexB(avccData []byte, isIDR bool) []byte {
	var result bytes.Buffer
	startCode := []byte{0x00, 0x00, 0x00, 0x01}

	// Prepend SPS and PPS for IDR frames (keyframes)
	if isIDR {
		result.Write(startCode)
		result.Write(vt.sps)
		result.Write(startCode)
		result.Write(vt.pps)
	}

	// Parse AVCC NAL units (4-byte length prefix)
	offset := 0
	for offset < len(avccData) {
		if offset+4 > len(avccData) {
			break
		}

		nalLength := int(binary.BigEndian.Uint32(avccData[offset:]))
		offset += 4

		if offset+nalLength > len(avccData) {
			log.Printf("Warning: NAL length exceeds data bounds")
			break
		}

		result.Write(startCode)
		result.Write(avccData[offset : offset+nalLength])
		offset += nalLength
	}

	return result.Bytes()
}

func (vt *StaticVideoTrack) Play() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.playing = true
	vt.playCtx, vt.playCancel = context.WithCancel(vt.ctx)
	vt.playWait.Add(1)
	go vt.playInBackground()
}

func (vt *StaticVideoTrack) Pause() {
	log.Printf("Play pause")
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.playing = false
	vt.playCancel()
	vt.playCancel = nil
	vt.playWait.Wait()
}

func (vt *StaticVideoTrack) Seek(position time.Duration) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	if len(vt.samples) == 0 {
		return fmt.Errorf("no video loaded")
	}

	// Find the sample at the target position (seek to nearest keyframe before)
	targetPos := time.Duration(0)
	targetIdx := 0
	lastKeyframeIdx := 0

	for i, sample := range vt.samples {
		if sample.IsSync {
			lastKeyframeIdx = i
		}
		if targetPos+sample.Duration > position {
			targetIdx = lastKeyframeIdx
			break
		}
		targetPos += sample.Duration
	}

	// Calculate position at keyframe
	vt.currentPos = 0
	for i := 0; i < targetIdx; i++ {
		vt.currentPos += vt.samples[i].Duration
	}
	vt.currentIdx = targetIdx

	log.Printf("Seeked to sample %d (%.2fs), keyframe: %v",
		targetIdx, vt.currentPos.Seconds(), vt.samples[targetIdx].IsSync)

	return nil
}

func (vt *StaticVideoTrack) GetPosition() time.Duration {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.currentPos
}

func (vt *StaticVideoTrack) playInBackground() {
	defer vt.playWait.Done()

	log.Printf("Starting playback, total samples: %d", len(vt.samples))

	for {
		select {
		case <-vt.ctx.Done():
			return
		case <-vt.playCtx.Done():
			return
		default:
		}

		vt.mu.Lock()

		if vt.currentIdx >= len(vt.samples) {
			if vt.isLoop {
				log.Printf("Video playback finished, looping")
				vt.currentIdx = 0
				vt.currentPos = 0
				vt.mu.Unlock()
				continue
			} else {
				vt.playing = false
				vt.mu.Unlock()
				return
			}
		}

		sample := vt.samples[vt.currentIdx]
		idx := vt.currentIdx

		if err := vt.track.WriteSample(media.Sample{
			Data:     sample.Data,
			Duration: sample.Duration,
		}); err != nil && err != io.ErrClosedPipe {
			log.Printf("Error writing sample: %v", err)
		}

		vt.currentPos += sample.Duration
		vt.currentIdx++

		if idx%100 == 0 {
			log.Printf("Sent sample %d/%d, pos: %v", idx, len(vt.samples), vt.currentPos)
		}

		vt.mu.Unlock()

		// Wait for frame duration
		time.Sleep(sample.Duration)
	}
}

// Helper function to concatenate NAL units with start codes
func aggregateNALs(nals [][]byte) []byte {
	var result []byte
	for _, nal := range nals {
		result = append(result, []byte{0, 0, 0, 1}...) // Annex B start code
		result = append(result, nal...)
	}
	return result
}

func (vt *StaticVideoTrack) Close() error {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.cancel()
	if vt.playCancel != nil {
		vt.playCancel()
		vt.playCancel = nil
	}
	vt.playing = false
	return nil
}
func (vt *StaticVideoTrack) PlayFrame() {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	// if vt.reader == nil {
	// 	return
	// }

	// nal, err := vt.reader.NextNAL()
	// if err != nil {
	// 	if err == io.EOF {
	// 		return
	// 	}
	// 	fmt.Printf("Error reading NAL: %v\n", err)
	// 	return
	// }

	// if err := vt.track.WriteSample(media.Sample{
	// 	Data:     nal.Data,
	// 	Duration: vt.frameDur,
	// }); err != nil {
	// 	fmt.Printf("Error writing sample: %v\n", err)
	// }

	// nalUnitType := nal.Data[0] & 0x1F
	// if nalUnitType == 1 || nalUnitType == 5 {
	// 	vt.currentPos += vt.frameDur
	// }
}

func (vt *StaticVideoTrack) PlayBackFrame() {
	// previousPos := vt.currentPos - vt.frameDur*2
	// if previousPos < 0 {
	// 	previousPos = 0
	// }
	// if err := vt.Seek(previousPos); err != nil {
	// 	fmt.Printf("Error seeking back: %v\n", err)
	// 	return
	// }
	// vt.PlayFrame()
}
