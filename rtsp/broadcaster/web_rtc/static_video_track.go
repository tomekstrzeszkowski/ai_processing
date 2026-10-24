package web_rtc

import (
	"bufio"
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
	totalDur   time.Duration
	playing    bool
	currentPos time.Duration
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
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	reader, err := h264reader.NewReader(bufio.NewReader(file))
	if err != nil {
		return fmt.Errorf("failed to create H264 reader: %w", err)
	}

	vt.samples = nil
	vt.sps = nil
	vt.pps = nil
	vt.currentIdx = 0
	vt.currentPos = 0

	var accessUnit [][]byte
	hasVCL, isSync, hasParams := false, false, false
	flush := func() {
		if !hasVCL {
			return
		}
		if isSync && !hasParams && vt.sps != nil && vt.pps != nil {
			accessUnit = append([][]byte{vt.sps, vt.pps}, accessUnit...)
		}
		vt.samples = append(vt.samples, VideoSample{
			Data:   aggregateNALs(accessUnit),
			IsSync: isSync,
		})
		accessUnit = nil
		hasVCL, isSync, hasParams = false, false, false
	}

	for {
		nal, err := reader.NextNAL()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read NAL: %w", err)
		}
		if len(nal.Data) < 2 {
			continue
		}

		switch nal.UnitType {
		case h264reader.NalUnitTypeCodedSliceNonIdr, h264reader.NalUnitTypeCodedSliceIdr:
			// first_mb_in_slice == 0 marks the first slice of a new picture
			if hasVCL && nal.Data[1]&0x80 != 0 {
				flush()
			}
			hasVCL = true
			isSync = isSync || nal.UnitType == h264reader.NalUnitTypeCodedSliceIdr
		case h264reader.NalUnitTypeSPS, h264reader.NalUnitTypePPS, h264reader.NalUnitTypeAUD:
			flush()
			if nal.UnitType == h264reader.NalUnitTypeSPS {
				vt.sps = nal.Data
				hasParams = true
			} else if nal.UnitType == h264reader.NalUnitTypePPS {
				vt.pps = nal.Data
				hasParams = true
			}
		}
		accessUnit = append(accessUnit, nal.Data)
	}
	flush()

	if len(vt.samples) == 0 {
		return fmt.Errorf("no video frames found in %s", filePath)
	}
	frameDur, ok := spsFrameDuration(vt.sps)
	if !ok {
		log.Printf("Warning: no timing info in SPS, using 30 fps")
		frameDur = time.Second / 30
	}
	for i := range vt.samples {
		vt.samples[i].Duration = frameDur
	}
	vt.totalDur = time.Duration(len(vt.samples)) * frameDur

	log.Printf("Video loaded: %d samples, duration %v", len(vt.samples), vt.totalDur)

	return nil
}

func (vt *StaticVideoTrack) Play() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	if vt.playing {
		return
	}
	if vt.currentIdx >= len(vt.samples) {
		vt.currentIdx = 0
		vt.currentPos = 0
	}
	vt.playing = true
	vt.playCtx, vt.playCancel = context.WithCancel(vt.ctx)
	vt.playWait.Add(1)
	go vt.playInBackground()
}

func (vt *StaticVideoTrack) Pause() {
	log.Printf("Play pause")
	vt.mu.Lock()
	vt.playing = false
	if vt.playCancel != nil {
		vt.playCancel()
		vt.playCancel = nil
	}
	vt.mu.Unlock()
	vt.playWait.Wait()
}

func (vt *StaticVideoTrack) Seek(position time.Duration) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	if len(vt.samples) == 0 {
		return fmt.Errorf("no video loaded")
	}

	target := vt.indexAt(position)
	keyframe := vt.keyframeBefore(target)
	if vt.playing {
		vt.setIdx(keyframe)
	} else {
		vt.writeRange(keyframe, target)
	}

	log.Printf("Seeked to sample %d (%.2fs)", vt.currentIdx, vt.currentPos.Seconds())

	return nil
}

func (vt *StaticVideoTrack) indexAt(position time.Duration) int {
	pos := time.Duration(0)
	for i, sample := range vt.samples {
		if pos+sample.Duration > position {
			return i
		}
		pos += sample.Duration
	}
	return len(vt.samples) - 1
}

func (vt *StaticVideoTrack) keyframeBefore(idx int) int {
	for i := idx; i > 0; i-- {
		if vt.samples[i].IsSync {
			return i
		}
	}
	return 0
}

func (vt *StaticVideoTrack) setIdx(idx int) {
	vt.currentIdx = idx
	vt.currentPos = 0
	for i := 0; i < idx; i++ {
		vt.currentPos += vt.samples[i].Duration
	}
}

// writeRange decodes samples from..to without delay so the client ends up showing frame "to"
func (vt *StaticVideoTrack) writeRange(from, to int) {
	for i := from; i <= to; i++ {
		vt.writeSample(vt.samples[i])
	}
	vt.setIdx(to + 1)
}

func (vt *StaticVideoTrack) writeSample(sample VideoSample) {
	if err := vt.track.WriteSample(media.Sample{
		Data:     sample.Data,
		Duration: sample.Duration,
	}); err != nil && err != io.ErrClosedPipe {
		log.Printf("Error writing sample: %v", err)
	}
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

		vt.writeSample(sample)

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

type bitReader struct {
	data []byte
	pos  int
}

func (r *bitReader) bits(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v <<= 1
		if r.pos/8 < len(r.data) {
			v |= uint32(r.data[r.pos/8]>>(7-r.pos%8)) & 1
		}
		r.pos++
	}
	return v
}

func (r *bitReader) ue() uint32 {
	zeros := 0
	for r.bits(1) == 0 && zeros < 32 {
		zeros++
	}
	return (1<<zeros - 1) + r.bits(zeros)
}

func (r *bitReader) se() {
	r.ue()
}

// spsFrameDuration reads the frame duration from SPS VUI timing info
func spsFrameDuration(sps []byte) (time.Duration, bool) {
	if len(sps) < 4 {
		return 0, false
	}
	rbsp := make([]byte, 0, len(sps))
	for i := 1; i < len(sps); i++ {
		if i >= 3 && sps[i] == 3 && sps[i-1] == 0 && sps[i-2] == 0 {
			continue
		}
		rbsp = append(rbsp, sps[i])
	}
	r := &bitReader{data: rbsp}
	profile := r.bits(8)
	r.bits(16)
	r.ue()
	switch profile {
	case 100, 110, 122, 244, 44, 83, 86, 118, 128, 138, 139, 134, 135:
		if r.ue() == 3 {
			r.bits(1)
		}
		r.ue()
		r.ue()
		r.bits(1)
		if r.bits(1) == 1 {
			return 0, false
		}
	}
	r.ue()
	switch r.ue() {
	case 0:
		r.ue()
	case 1:
		r.bits(1)
		r.se()
		r.se()
		for n := r.ue(); n > 0; n-- {
			r.se()
		}
	}
	r.ue()
	r.bits(1)
	r.ue()
	r.ue()
	if r.bits(1) == 0 {
		r.bits(1)
	}
	r.bits(1)
	if r.bits(1) == 1 {
		r.ue()
		r.ue()
		r.ue()
		r.ue()
	}
	if r.bits(1) == 0 {
		return 0, false
	}
	if r.bits(1) == 1 && r.bits(8) == 255 {
		r.bits(32)
	}
	if r.bits(1) == 1 {
		r.bits(1)
	}
	if r.bits(1) == 1 {
		r.bits(4)
		if r.bits(1) == 1 {
			r.bits(24)
		}
	}
	if r.bits(1) == 1 {
		r.ue()
		r.ue()
	}
	if r.bits(1) == 0 {
		return 0, false
	}
	numUnitsInTick := r.bits(32)
	timeScale := r.bits(32)
	if numUnitsInTick == 0 || timeScale == 0 {
		return 0, false
	}
	return time.Duration(2 * uint64(numUnitsInTick) * uint64(time.Second) / uint64(timeScale)), true
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
	if vt.currentIdx >= len(vt.samples) {
		return
	}
	vt.writeRange(vt.currentIdx, vt.currentIdx)
}

func (vt *StaticVideoTrack) PlayBackFrame() {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	target := vt.currentIdx - 2
	if target < 0 {
		return
	}
	vt.writeRange(vt.keyframeBefore(target), target)
}
