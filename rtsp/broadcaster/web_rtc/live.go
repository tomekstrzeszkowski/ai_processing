package web_rtc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"sync"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
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
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: iceServers,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	videoTrack, err := NewVideoTrack()
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
	wsClient, _, err := websocket.DefaultDialer.Dial(signalingUrl, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer wsClient.Close()
	go listen(wsClient, pc)
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		log.Fatal(err)
	}
	offerData, err := json.Marshal(offer)
	if err != nil {
		log.Fatal(err)
	}

	if err := wsClient.WriteMessage(websocket.TextMessage, offerData); err != nil {
		log.Fatal(err)
	}
	videoTrack.Start(memory)
}
