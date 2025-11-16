package web_rtc

import (
	"context"
	"fmt"
	"io"
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
}

func NewStaticVideoTrack() (*StaticVideoTrack, error) {
	// H.264 track
	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video",
		"pion-video",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create video track: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &StaticVideoTrack{
		track:    track,
		ctx:      ctx,
		cancel:   cancel,
		frameDur: time.Millisecond * 33, // 30fps to match your ffmpeg config
	}, nil
}

// LoadVideo loads an H.264 video file (Annex-B format)
func (vt *StaticVideoTrack) LoadVideo(filePath string) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	// Close existing file if any
	if vt.file != nil {
		vt.file.Close()
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

// Play starts playing the video
func (vt *StaticVideoTrack) Play() {
	vt.mu.Lock()
	if vt.playing {
		vt.mu.Unlock()
		return
	}
	vt.playing = true
	vt.mu.Unlock()

	go vt.playLoop()
}

// Pause pauses the video playback
func (vt *StaticVideoTrack) Pause() {
	vt.mu.Lock()
	vt.playing = false
	vt.mu.Unlock()
}

// Seek seeks to a specific position in the video
// Note: Seeks to nearest keyframe (GOP boundary)
func (vt *StaticVideoTrack) Seek(position time.Duration) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	if vt.file == nil || vt.reader == nil {
		return fmt.Errorf("no video loaded")
	}

	// Seek to beginning of file
	if _, err := vt.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	// Recreate reader
	reader, err := h264reader.NewReader(vt.file)
	if err != nil {
		return fmt.Errorf("failed to recreate reader: %w", err)
	}
	vt.reader = reader

	// Calculate target frame number
	targetFrame := int64(position / vt.frameDur)

	// Skip NAL units until we reach a keyframe near the target
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

		// Check if this is a frame NAL (non-IDR or IDR)
		nalUnitType := nal.Data[0] & 0x1F
		if nalUnitType == 1 || nalUnitType == 5 { // Non-IDR or IDR slice
			currentFrame++

			// If we're close to target and this is a keyframe (IDR), stop here
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

// GetPosition returns current playback position
func (vt *StaticVideoTrack) GetPosition() time.Duration {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.currentPos
}

func (vt *StaticVideoTrack) playLoop() {
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
				continue
			}

			if vt.reader == nil {
				vt.mu.Unlock()
				continue
			}

			nal, err := vt.reader.NextNAL()
			if err != nil {
				if err == io.EOF {
					// Loop back to beginning
					vt.file.Seek(0, io.SeekStart)
					reader, _ := h264reader.NewReader(vt.file)
					vt.reader = reader
					vt.currentPos = 0
					vt.frameCount = 0
				}
				vt.mu.Unlock()
				continue
			}

			// Write NAL unit to track
			if err := vt.track.WriteSample(media.Sample{
				Data:     nal.Data,
				Duration: vt.frameDur,
			}); err != nil {
				fmt.Printf("Error writing sample: %v\n", err)
			}

			// Update position only for actual frame NALs
			nalUnitType := nal.Data[0] & 0x1F
			if nalUnitType == 1 || nalUnitType == 5 { // Non-IDR or IDR slice
				vt.currentPos += vt.frameDur
				vt.frameCount++
			}

			vt.mu.Unlock()
		}
	}
}

// Close cleans up resources
func (vt *StaticVideoTrack) Close() error {
	vt.cancel()
	vt.mu.Lock()
	defer vt.mu.Unlock()

	if vt.file != nil {
		return vt.file.Close()
	}
	return nil
}

// // VideoController manages video playback with signaling support
// type VideoController struct {
// 	videoTrack *StaticVideoTrack
// 	onCommand  func(command string, value interface{})
// }

// // HandleClientCommand processes commands from the client
// func (vc *VideoController) HandleClientCommand(command string, value interface{}) error {
// 	switch command {
// 	case "seek":
// 		position, ok := value.(float64) // seconds
// 		if !ok {
// 			return fmt.Errorf("invalid seek value")
// 		}
// 		return vc.videoTrack.Seek(time.Duration(position) * time.Second)

// 	case "play":
// 		vc.videoTrack.Play()
// 		return nil

// 	case "pause":
// 		vc.videoTrack.Pause()
// 		return nil

// 	case "getPosition":
// 		pos := vc.videoTrack.GetPosition()
// 		if vc.onCommand != nil {
// 			vc.onCommand("position", pos.Seconds())
// 		}
// 		return nil

// 	default:
// 		return fmt.Errorf("unknown command: %s", command)
// 	}
// }
