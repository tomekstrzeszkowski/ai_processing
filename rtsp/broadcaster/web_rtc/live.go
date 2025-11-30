package web_rtc

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "image/jpeg"
	_ "image/png"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v3"
	"strzcam.com/broadcaster/watcher"
)

func listen(wsClient *websocket.Conn, videoTrack *VideoTrack, savePath string) {
	offeror, _ := NewOfferor(wsClient, savePath)
	defer offeror.Close()
	offeror.CreatePeerConnection(videoTrack)
	offeror.CreateAndSendOffer()
	for {
		_, message, err := wsClient.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}
		var msg SignalingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		switch msg.Type {
		case "offer":
			log.Printf("Offeree not supported.")
		case "answer":
			if offeror.pc == nil {
				continue
			}
			answer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  msg.Sdp,
			}
			if err := offeror.pc.SetRemoteDescription(answer); err != nil {
				log.Printf("Error setting remote description: %v", err)
			}
		case "ice":
			if offeror.pc == nil {
				continue
			}
			for _, iceData := range msg.Ice {
				candidate := webrtc.ICECandidateInit{}
				if candidateStr, ok := iceData["candidate"].(string); ok {
					candidate.Candidate = candidateStr
				}
				if sdpMid, ok := iceData["sdpMid"].(string); ok {
					candidate.SDPMid = &sdpMid
				}
				if sdpMLineIndex, ok := iceData["sdpMLineIndex"].(float64); ok {
					idx := uint16(sdpMLineIndex)
					candidate.SDPMLineIndex = &idx
				}
				if err := offeror.pc.AddICECandidate(candidate); err != nil {
					log.Printf("Error adding ICE candidate: %v", err)
				}
			}
		case "start":
			log.Printf("Starting...")
			defer offeror.Close()
			offeror.CreatePeerConnection(videoTrack)
			offeror.CreateAndSendOffer()
		}
	}
}

func RunLive(signalingUrl string) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
	videoFrame := os.Getenv("VIDEO_FRAME")
	isLiveStream := os.Getenv("WEBRTC_STREAM_LIVE")
	wsClient, _, err := websocket.DefaultDialer.Dial(signalingUrl, nil)
	if err != nil {
		panic(err)
	}
	defer wsClient.Close()
	savePath := fmt.Sprintf("%s_%s", watcher.SavePath, videoFrame)
	var videoTrack *VideoTrack = nil
	if isLiveStream == "true" {
		memory, err := watcher.NewSharedMemoryReceiver(videoFrame)
		if err != nil {
			panic(fmt.Sprintf("Error creating shared memory receiver: %v", err))
		}
		go memory.WatchSharedMemoryReadOnly()
		videoTrack, err = NewVideoTrack()
		if err != nil {
			panic(err)
		}
		defer videoTrack.Close()
		go videoTrack.Start(memory)
	}

	go listen(wsClient, videoTrack, savePath)
	select {}
}
