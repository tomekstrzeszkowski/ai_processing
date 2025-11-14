package web_rtc

import (
	"encoding/json"
	"fmt"
	"log"

	_ "image/jpeg"
	_ "image/png"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v3"
	"strzcam.com/broadcaster/watcher"
)

func listen(conn *websocket.Conn, pc *webrtc.PeerConnection) {
	for {
		_, message, err := conn.ReadMessage()
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
			// no need for now
		case "answer":
			answer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  msg.Sdp,
			}
			if err := pc.SetRemoteDescription(answer); err != nil {
				log.Printf("Error setting remote description: %v", err)
			}
		case "ice":
			for _, iceData := range msg.Ice {
				candidateStr, ok := iceData["candidate"].(string)
				if !ok {
					continue
				}

				candidate := webrtc.ICECandidateInit{
					Candidate: candidateStr,
				}

				if sdpMid, ok := iceData["sdpMid"].(string); ok {
					candidate.SDPMid = &sdpMid
				}

				if sdpMLineIndex, ok := iceData["sdpMLineIndex"].(float64); ok {
					idx := uint16(sdpMLineIndex)
					candidate.SDPMLineIndex = &idx
				}

				if err := pc.AddICECandidate(candidate); err != nil {
					log.Printf("Error adding ICE candidate: %v", err)
				}
			}
		}
	}
}

func RunLive(signalingUrl string) {
	memory, err := watcher.NewSharedMemoryReceiver("video_frame")
	if err != nil {
		panic(fmt.Sprintf("Error creating shared memory receiver: %v", err))
	}
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
	wsClient, _, err := websocket.DefaultDialer.Dial(signalingUrl, nil)
	if err != nil {
		panic(err)
	}
	defer wsClient.Close()
	videoTrack, err := NewVideoTrack()
	if err != nil {
		panic(err)
	}
	defer videoTrack.Close()

	offeror, _ := NewOfferor(wsClient)
	defer offeror.Close()
	pc, err := offeror.CreatePeerConnection(videoTrack)

	go listen(wsClient, pc)
	offeror.CreateAndSendOffer()
	videoTrack.Start(memory)
}
