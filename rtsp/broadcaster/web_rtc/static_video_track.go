package web_rtc

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
)

type StaticVideoTrack struct {
	track      *webrtc.TrackLocalStaticSample
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	playCtx    context.Context
	playCancel context.CancelFunc
	reader     *h264reader.H264Reader
	file       *os.File
	frameDur   time.Duration
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
		frameDur:   time.Second / 30,
		totalDur:   -1 * time.Second,
		isLoop:     false,
	}, nil
}

func (vt *StaticVideoTrack) LoadVideo(filePath string) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	if vt.playing {
		vt.playing = false
	}
	if vt.file != nil {
		vt.file.Close()
		vt.file = nil
		vt.reader = nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open video file: %w", err)
	}

	reader, err := h264reader.NewReader(file)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to create H264 reader: %w", err)
	}

	vt.file = file
	vt.reader = reader
	vt.currentPos = 0
	vt.frameCount = 0
	frameDur, err := GetFrameDuration(filePath)
	if err != nil {
		log.Printf("Warning: could not detect framerate, using default: %v", err)
		vt.frameDur = time.Second / 30 // fallback
	} else {
		vt.frameDur = frameDur
		log.Printf("Detected frame duration: %v (%.2f fps)", frameDur, float64(time.Second)/float64(frameDur))
	}
	vt.frameDur = frameDur
	if err := vt.ReadDuration(filePath); err != nil {
		return fmt.Errorf("Can not read file duration %w", err)
	}

	return nil
}

func (vt *StaticVideoTrack) ReadDuration(filePath string) error {
	// Count total frames to calculate duration
	fileDuration, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileDuration.Close()
	readerDuration, err := h264reader.NewReader(fileDuration)
	if err != nil {
		return err
	}
	totalFrames := int64(0)
	for {
		nal, err := readerDuration.NextNAL()
		if err == io.EOF {
			break
		}
		nalUnitType := nal.Data[0] & 0x1F
		if nalUnitType == 1 || nalUnitType == 5 {
			totalFrames++
		}
	}
	vt.totalDur = time.Duration(totalFrames) * vt.frameDur
	return nil
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

	if vt.file == nil || vt.reader == nil {
		return fmt.Errorf("no video loaded")
	}

	if _, err := vt.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	reader, err := h264reader.NewReader(vt.file)
	if err != nil {
		return fmt.Errorf("failed to recreate reader: %w", err)
	}
	vt.reader = reader

	targetFrame := int64(position / vt.frameDur)
	currentFrame := int64(0)
	lastKeyframe := int64(0)

	for currentFrame < targetFrame {
		nal, err := vt.reader.NextNAL()
		if err != nil {
			if err == io.EOF {
				vt.currentPos = time.Duration(currentFrame) * vt.frameDur
				vt.frameCount = currentFrame
				return nil
			}
			return fmt.Errorf("failed to parse NAL while seeking: %w", err)
		}

		nalUnitType := nal.Data[0] & 0x1F
		if nalUnitType == 5 { // IDR (keyframe)
			lastKeyframe = currentFrame
		}

		if nalUnitType == 1 || nalUnitType == 5 {
			currentFrame++
		}
	}

	vt.currentPos = time.Duration(currentFrame) * vt.frameDur
	vt.frameCount = currentFrame

	log.Printf("Seeked to frame %d (%.2fs), last keyframe at %d",
		currentFrame, vt.currentPos.Seconds(), lastKeyframe)

	return nil
}

func (vt *StaticVideoTrack) GetPosition() time.Duration {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.currentPos
}

func (vt *StaticVideoTrack) playInBackground() {
	defer vt.playWait.Done()
	ticker := time.NewTicker(vt.frameDur)
	defer ticker.Stop()

	var accessUnit [][]byte // Accumulate NALs for one access unit

	for {
		select {
		case <-vt.ctx.Done():
			return
		case <-vt.playCtx.Done():
			vt.playing = false
			return
		case <-ticker.C:
			vt.mu.Lock()

			if vt.reader == nil {
				vt.mu.Unlock()
				continue
			}

			nal, err := vt.reader.NextNAL()
			if err != nil {
				if err == io.EOF {
					// Flush any remaining access unit
					if len(accessUnit) > 0 {
						frameData := aggregateNALs(accessUnit)
						if err := vt.track.WriteSample(media.Sample{
							Data:     frameData,
							Duration: vt.frameDur,
						}); err != nil {
							fmt.Printf("Error writing final sample: %v\n", err)
						}
						vt.currentPos += vt.frameDur
						vt.frameCount++
						accessUnit = nil
					}

					log.Printf("Video playback finished (EOF) %s loop: %t", vt.currentPos.String(), vt.isLoop)
					if vt.isLoop {
						vt.file.Seek(0, io.SeekStart)
						reader, _ := h264reader.NewReader(vt.file)
						vt.reader = reader
						vt.currentPos = 0
						vt.frameCount = 0
						accessUnit = nil
						vt.mu.Unlock()
						continue
					} else {
						vt.playing = false
						vt.mu.Unlock()
						return
					}
				}
				log.Printf("Error reading NAL: %v\n", err)
				vt.mu.Unlock()
				continue
			}

			nalUnitType := nal.Data[0] & 0x1F

			// Check if this NAL starts a new access unit (VCL NAL)
			// NAL types 1 (coded slice), 5 (IDR slice) are VCL NALs
			isVCL := nalUnitType == 1 || nalUnitType == 5

			// If we have an access unit and encounter a new VCL NAL, write the previous one
			if len(accessUnit) > 0 && isVCL {
				frameData := aggregateNALs(accessUnit)
				if err := vt.track.WriteSample(media.Sample{
					Data:     frameData,
					Duration: vt.frameDur,
				}); err != nil {
					fmt.Printf("Error writing sample: %v\n", err)
				}
				vt.currentPos += vt.frameDur
				vt.frameCount++
				accessUnit = nil
			}

			// Accumulate NAL in current access unit
			accessUnit = append(accessUnit, nal.Data)

			vt.mu.Unlock()
		}
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

	if vt.file != nil {
		return vt.file.Close()
	}
	return nil
}
func (vt *StaticVideoTrack) PlayFrame() {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	if vt.reader == nil {
		return
	}

	nal, err := vt.reader.NextNAL()
	if err != nil {
		if err == io.EOF {
			return
		}
		fmt.Printf("Error reading NAL: %v\n", err)
		return
	}

	if err := vt.track.WriteSample(media.Sample{
		Data:     nal.Data,
		Duration: vt.frameDur,
	}); err != nil {
		fmt.Printf("Error writing sample: %v\n", err)
	}

	nalUnitType := nal.Data[0] & 0x1F
	if nalUnitType == 1 || nalUnitType == 5 {
		vt.currentPos += vt.frameDur
	}
}

func (vt *StaticVideoTrack) PlayBackFrame() {
	previousPos := vt.currentPos - vt.frameDur*2
	if previousPos < 0 {
		previousPos = 0
	}
	if err := vt.Seek(previousPos); err != nil {
		fmt.Printf("Error seeking back: %v\n", err)
		return
	}
	vt.PlayFrame()
}
