package web_rtc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pion/webrtc/v3"
)

func updateSeek(ctx context.Context, dataChannel *webrtc.DataChannel, staticVideoTrack *StaticVideoTrack) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if dataChannel == nil || staticVideoTrack == nil {
				continue
			}
			if err := SendSeekPosition(staticVideoTrack.currentPos.Seconds(), dataChannel); err == nil {
				//log.Printf("Sending %s is playing %b", seekMessage, staticVideoTrack.playing)
				if !staticVideoTrack.playing {
					return
				}
			}
		}
	}
}

func SendSeekPosition(position float64, dataChannel *webrtc.DataChannel) error {
	seekMessage, err := json.Marshal(SeekMessage{Type: "seek", Seek: position})
	if err == nil {
		dataChannel.Send(seekMessage)
	}
	return err
}
