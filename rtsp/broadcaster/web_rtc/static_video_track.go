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
	reader     *h264reader.H264Reader
	file       *os.File
	frameDur   time.Duration
	playing    bool
	currentPos time.Duration
	frameCount int64
	rtpSender  *webrtc.RTPSender
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

	return &StaticVideoTrack{
		track:    track,
		ctx:      ctx,
		cancel:   cancel,
		frameDur: time.Millisecond * 33,
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

	return nil
}

func (vt *StaticVideoTrack) Play(isLoop bool) {
	vt.mu.Lock()
	if vt.playing {
		vt.mu.Unlock()
		return
	}
	vt.playing = true
	vt.mu.Unlock()

	go vt.playLoop(isLoop)
}

func (vt *StaticVideoTrack) Pause() {
	vt.mu.Lock()
	vt.playing = false
	vt.mu.Unlock()
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
	log.Print("Seeking to frame ", targetFrame)
	currentFrame := int64(0)
	foundKeyframe := false

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
		if nalUnitType == 1 || nalUnitType == 5 { // Non-IDR or IDR slice
			currentFrame++
			if nalUnitType == 5 && currentFrame >= targetFrame-30 {
				foundKeyframe = true
				break
			}
		}
	}

	vt.currentPos = time.Duration(currentFrame) * vt.frameDur
	vt.frameCount = currentFrame

	if !foundKeyframe && targetFrame > 0 {
		fmt.Printf("Warning: Seeked to frame %d (requested %d), may not be at keyframe\n",
			currentFrame, targetFrame)
	}

	return nil
}

func (vt *StaticVideoTrack) GetPosition() time.Duration {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.currentPos
}

func (vt *StaticVideoTrack) playLoop(isLoop bool) {
	ticker := time.NewTicker(vt.frameDur)
	defer ticker.Stop()

	for {
		select {
		case <-vt.ctx.Done():
			return
		case <-ticker.C:
			vt.mu.Lock()

			if !vt.playing {
				vt.mu.Unlock()
				return
			}

			if vt.reader == nil {
				vt.mu.Unlock()
				continue
			}

			nal, err := vt.reader.NextNAL()
			if err != nil {
				if err == io.EOF {
					log.Printf("Video playback finished (EOF) %s", vt.currentPos.String())
					if isLoop {
						vt.file.Seek(0, io.SeekStart)
						reader, _ := h264reader.NewReader(vt.file)
						vt.reader = reader
						vt.currentPos = 0
						vt.frameCount = 0
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
			if err := vt.track.WriteSample(media.Sample{
				Data:     nal.Data,
				Duration: vt.frameDur,
			}); err != nil {
				fmt.Printf("Error writing sample: %v\n", err)
			}
			nalUnitType := nal.Data[0] & 0x1F
			if nalUnitType == 1 || nalUnitType == 5 {
				vt.currentPos += vt.frameDur
				vt.frameCount++
			}

			vt.mu.Unlock()
		}
	}
}

func (vt *StaticVideoTrack) Close() error {
	vt.cancel()
	vt.mu.Lock()
	defer vt.mu.Unlock()

	vt.playing = false

	if vt.file != nil {
		return vt.file.Close()
	}
	return nil
}
