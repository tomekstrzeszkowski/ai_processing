package web_rtc

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"log"
	"sync"
	"time"

	"github.com/pion/mediadevices/pkg/codec/vpx"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"strzcam.com/broadcaster/watcher"
)

type frameReader struct {
	frameChan chan image.Image
	width     int
	height    int
}

func newFrameReader(width, height int) *frameReader {
	return &frameReader{
		frameChan: make(chan image.Image, 1),
		width:     width,
		height:    height,
	}
}

func (r *frameReader) Read() (image.Image, func(), error) {
	frame, ok := <-r.frameChan
	if !ok {
		return nil, func() {}, fmt.Errorf("frame channel closed")
	}
	return frame, func() {}, nil
}

type VideoTrack struct {
	track   *webrtc.TrackLocalStaticSample
	reader  *frameReader
	mu      sync.Mutex
	encoder interface {
		Read() ([]byte, func(), error)
		Close() error
	}
	ctx    context.Context
	cancel context.CancelFunc
	frame  chan []byte
}

func NewVideoTrack() (*VideoTrack, error) {
	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
		"live",
		"video_frame_live",
	)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &VideoTrack{
		track:  track,
		ctx:    ctx,
		cancel: cancel,
		frame:  make(chan []byte, 1),
	}, nil
}

func (vt *VideoTrack) SendFrame(frame image.Image) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	// Initialize encoder on first frame
	if vt.encoder == nil {
		bounds := frame.Bounds()
		width, height := bounds.Dx(), bounds.Dy()

		vt.reader = newFrameReader(width, height)

		params, err := vpx.NewVP8Params()
		if err != nil {
			return err
		}
		params.BitRate = 2_000_000
		params.KeyFrameInterval = 60

		vt.encoder, err = params.BuildVideoEncoder(vt.reader, prop.Media{
			Video: prop.Video{
				Width:  width,
				Height: height,
			},
		})
		if err != nil {
			return err
		}
	}
	vt.reader.frameChan <- frame

	encodedFrame, release, err := vt.encoder.Read()
	defer release()
	if err != nil {
		log.Printf("Error reading from encoder: %v", err)
		return err
	}
	if err := vt.track.WriteSample(media.Sample{
		Data:     encodedFrame,
		Duration: time.Second / 30,
	}); err != nil {
		log.Printf("Error writing sample: %v", err)
	}

	return nil
}
func (vt *VideoTrack) Start(memory *watcher.SharedMemoryReceiver) {
	var currentFrame image.Image
	var cancel context.CancelFunc

	for frame := range memory.Frames {

		img, _, err := image.Decode(bytes.NewReader(frame))
		if err != nil {
			log.Printf("Error decoding image: %v", err)
			continue
		}
		if cancel != nil {
			cancel()
		}
		currentFrame = img
		vt.SendFrame(currentFrame)
		ctx, cancelFunc := context.WithCancel(vt.ctx)
		cancel = cancelFunc

		go func(frame image.Image) {
			ticker := time.NewTicker(time.Second / 30)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					vt.SendFrame(frame)
				case <-ctx.Done():
					return
				}
			}
		}(currentFrame)
	}
	if cancel != nil {
		cancel()
	}
}

func (vt *VideoTrack) Close() error {
	vt.cancel()
	if vt.encoder != nil {
		return vt.encoder.Close()
	}
	return nil
}
