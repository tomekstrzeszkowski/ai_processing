package web_rtc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/codec/vpx"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"strzcam.com/broadcaster/watcher"
)

var iceServers = []webrtc.ICEServer{
	{
		URLs: []string{
			"stun:stun.l.google.com:19302",
			"stun:stun2.l.google.com:19302",
			"stun:stun3.l.google.com:19302",
			"stun:stun.1und1.de:3478",
			"stun:stun.avigora.com:3478",
			"stun:stun.avigora.fr:3478",
		},
	},
	{
		URLs:       []string{"turn:global.turn.twilio.com:3478?transport=udp"},
		Username:   "dc2d2894d5a9023620c467b0e71cfa6a35457e6679785ed6ae9856fe5bdfa269",
		Credential: "tE2DajzSbc123",
	},
	{
		URLs:       []string{"turn:openrelay.metered.ca:80", "turn:openrelay.metered.ca:443"},
		Username:   "openrelayproject",
		Credential: "openrelayproject",
	},
	{
		URLs:       []string{"turn:openrelay.metered.ca:443?transport=tcp"},
		Username:   "openrelayproject",
		Credential: "openrelayproject",
	},
}

type Message struct {
	Type string                   `json:"type"`
	SDP  string                   `json:"sdp,omitempty"`
	Ice  []map[string]interface{} `json:"ice,omitempty"`
}

type VideoTrack struct {
	track      *webrtc.TrackLocalStaticSample
	encoder    codec.ReadCloser
	frameCount int
	mu         sync.Mutex
	width      int
	height     int
}

// frameReader implements codec.Reader interface for feeding frames to encoder
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

func NewVideoTrack(width, height int) (*VideoTrack, error) {
	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
		"video",
		"pion",
	)
	if err != nil {
		return nil, err
	}

	// Create frame reader
	reader := newFrameReader(width, height)

	// Create VP8 encoder params
	params, err := vpx.NewVP8Params()
	if err != nil {
		return nil, fmt.Errorf("failed to create VP8 params: %w", err)
	}
	params.BitRate = 2_000_000 // 2 Mbps
	params.KeyFrameInterval = 60

	// Build encoder
	encoder, err := params.BuildVideoEncoder(reader, prop.Media{
		Video: prop.Video{
			Width:  width,
			Height: height,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build encoder: %w", err)
	}

	return &VideoTrack{
		track:      track,
		encoder:    encoder,
		frameCount: 0,
		width:      width,
		height:     height,
	}, nil
}

func (vt *VideoTrack) Start() {
	memory, err := watcher.NewSharedMemoryReceiver("video_frame")
	if err != nil {
		log.Printf("Error creating shared memory receiver: %v", err)
		return
	}

	go func() {
		ticker := time.NewTicker(time.Second / 30) // 30 FPS
		defer ticker.Stop()

		for range ticker.C {
			frameData, _, err := memory.ReadFrameFromShm()
			if err != nil {
				log.Printf("Error reading frame: %v", err)
				continue
			}

			// Decode image
			img, _, err := image.Decode(bytes.NewReader(frameData))
			fmt.Printf("Decoded image size: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())
			if err != nil {
				log.Printf("Error decoding image: %v", err)
				continue
			}

			// Read encoded data from encoder
			encodedFrame, release, err := vt.encoder.Read()
			if err != nil {
				log.Printf("Error reading from encoder: %v", err)
				continue
			}

			vt.mu.Lock()
			vt.frameCount++
			vt.mu.Unlock()

			// Write sample to track
			if err := vt.track.WriteSample(media.Sample{
				Data:     encodedFrame,
				Duration: time.Second / 30,
			}); err != nil {
				log.Printf("Error writing sample: %v", err)
			}

			release()
		}
	}()
}

func (vt *VideoTrack) Close() error {
	if vt.encoder != nil {
		return vt.encoder.Close()
	}
	return nil
}

func listen(conn *websocket.Conn, pc *webrtc.PeerConnection) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		switch msg.Type {
		case "offer":
			// Handle offer if needed
		case "answer":
			answer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  msg.SDP,
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

func RunLive() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Create WebRTC configuration
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	// Create peer connection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	// Create video track (adjust width/height to match your frames)
	videoTrack, err := NewVideoTrack(1920, 1080)
	if err != nil {
		log.Fatal(err)
	}
	defer videoTrack.Close()

	// Add track to peer connection
	rtpSender, err := pc.AddTrack(videoTrack.track)
	if err != nil {
		log.Fatal(err)
	}

	// Read RTP packets (required for RTCP)
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, err := rtpSender.Read(rtcpBuf); err != nil {
				return
			}
		}
	}()

	// Set up connection state handler
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Connection state: %s\n", state.String())
	})

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws?userId=53", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Start listening for messages
	go listen(conn, pc)

	// Create offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		log.Fatal(err)
	}

	// Send offer
	offerMsg := Message{
		Type: pc.LocalDescription().Type.String(),
		SDP:  pc.LocalDescription().SDP,
	}
	offerData, err := json.Marshal(offerMsg)
	if err != nil {
		log.Fatal(err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, offerData); err != nil {
		log.Fatal(err)
	}

	// Start video track
	videoTrack.Start()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("Shutting down...")
}
