package web_rtc

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/pion/webrtc/v3"
)

func updateStatus(ctx context.Context, dataChannel *webrtc.DataChannel, staticVideoTrack *StaticVideoTrack) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if dataChannel == nil || staticVideoTrack == nil {
				return
			}
			log.Printf("updateSeek %s", staticVideoTrack.currentPos.Seconds())
			if err := SendStatus(dataChannel, staticVideoTrack.currentPos.Seconds()); err == nil {
				if !staticVideoTrack.playing {
					return
				}
			}
		}
	}
}

func SendStatus(dataChannel *webrtc.DataChannel, position float64) error {
	statusMessage, err := json.Marshal(StatusSeekMessage{
		Type: "status",
		Seek: position,
	})
	if err == nil {
		dataChannel.Send(statusMessage)
	}
	return err
}

func SendStatusLoadVideo(dataChannel *webrtc.DataChannel, isPlaying bool, position float64, isLoop bool, duration *float64) error {
	var durationValue float64
	if duration != nil {
		durationValue = *duration
	}
	statusMessage, err := json.Marshal(StatusLoadVideoMessage{
		Type:      "status",
		IsPlaying: isPlaying,
		IsLoop:    isLoop,
		Duration:  durationValue,
	})
	if err == nil {
		dataChannel.Send(statusMessage)
	}
	return err
}
func SendStatusLoop(dataChannel *webrtc.DataChannel, isLoop bool) error {
	statusMessage, err := json.Marshal(StatusIsLoopMessage{
		Type:   "status",
		IsLoop: isLoop,
	})
	if err == nil {
		dataChannel.Send(statusMessage)
	}
	return err
}
func SendStatusIsPlaying(dataChannel *webrtc.DataChannel, isPlaying bool) error {
	statusMessage, err := json.Marshal(StatusIsPlayingMessage{
		Type:      "status",
		IsPlaying: isPlaying,
	})
	if err == nil {
		dataChannel.Send(statusMessage)
	}
	return err
}
