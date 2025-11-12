package web_rtc

import (
	"bytes"
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

type VideoTrack struct {
	track   *webrtc.TrackLocalStaticSample
	reader  *frameReader
	mu      sync.Mutex
	encoder interface {
		Read() ([]byte, func(), error)
		Close() error
	}
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

	// Send to webRTC peer
	if err := vt.track.WriteSample(media.Sample{
		Data:     encodedFrame,
		Duration: time.Second / 30,
	}); err != nil {
		log.Printf("Error writing sample: %v", err)
	}

	return nil
}

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

func NewVideoTrack() (*VideoTrack, error) {
	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
		"video",
		"pion",
	)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &VideoTrack{
		track: track,
	}, nil
}

func (vt *VideoTrack) Start(memory *watcher.SharedMemoryReceiver) {
	ticker := time.NewTicker(time.Second / 30) // 30 FPS
	defer ticker.Stop()

	for range ticker.C {
		frameData, _, err := memory.ReadFrameFromShm()
		if err != nil {
			log.Printf("Error reading frame: %v", err)
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(frameData))
		if err != nil {
			log.Printf("Error decoding image: %v", err)
			continue
		}
		vt.SendFrame(img)
	}
}

func (vt *VideoTrack) Close() error {
	if vt.encoder != nil {
		return vt.encoder.Close()
	}
	return nil
}
