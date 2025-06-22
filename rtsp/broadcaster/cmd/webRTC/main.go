package main

import (
	"log"
	"net/http"

	webrtc "strzcam.com/broadcaster/webRTC"
)

// ! WARNING frames are send via WS, WebRTC does not work yet
func main() {
	hub := webrtc.NewHub()
	streamer := webrtc.NewVideoFrameStreamer()

	go hub.Run()

	// WebSocket endpoint for signaling
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		webrtc.HandleWebSocket(hub, streamer, w, r)
	})

	// HTTP endpoint to receive video frames
	http.HandleFunc("/frame", webrtc.HandleVideoFrame(streamer, hub))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		webrtc.ServeClient(w, r)
	})

	log.Println("WebRTC Signaling Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// AI:

// package main

// import (
// 	"bytes"
// 	"fmt"
// 	"image"
// 	"image/jpeg"
// 	"log"
// 	"sync"
// 	"time"

// 	"github.com/pion/webrtc/v3"
// 	"github.com/pion/webrtc/v3/pkg/media"
// )

// // You'll need: go get github.com/gen2brain/x264-go
// // import "github.com/gen2brain/x264-go/x264"

// type VideoFrameStreamer struct {
// 	peers        map[string]*PeerInfo
// 	mu           sync.RWMutex
// 	frameChannel chan []byte // JPEG frames
// 	width        int
// 	height       int
// 	fps          int
// }

// type PeerInfo struct {
// 	connection *webrtc.PeerConnection
// 	track      *webrtc.TrackLocalStaticSample
// 	encoder    *H264Encoder
// }

// type H264Encoder struct {
// 	// encoder *x264.Encoder // Uncomment when you add x264-go
// 	width  int
// 	height int
// 	fps    int
// }

// func NewH264Encoder(width, height, fps int) (*H264Encoder, error) {
// 	encoder := &H264Encoder{
// 		width:  width,
// 		height: height,
// 		fps:    fps,
// 	}

// 	// TODO: Initialize x264 encoder here
// 	/*
// 	opts := &x264.Options{
// 		Width:     width,
// 		Height:    height,
// 		FrameRate: fps,
// 		Tune:      "zerolatency",
// 		Preset:    "ultrafast",
// 		Profile:   "baseline",
// 		LogLevel:  x264.LogNone,
// 	}

// 	enc, err := x264.NewEncoder(opts)
// 	if err != nil {
// 		return nil, err
// 	}
// 	encoder.encoder = enc
// 	*/

// 	return encoder, nil
// }

// func (e *H264Encoder) EncodeJPEG(jpegData []byte) ([]byte, error) {
// 	// Decode JPEG to get raw image data
// 	img, err := jpeg.Decode(bytes.NewReader(jpegData))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode JPEG: %v", err)
// 	}

// 	// Convert to YUV420P format
// 	yuvData, err := e.imageToYUV420P(img)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to convert to YUV: %v", err)
// 	}

// 	// TODO: Encode with x264
// 	/*
// 	yuvImg := &x264.YUVImage{
// 		Y:      yuvData.Y,
// 		U:      yuvData.U,
// 		V:      yuvData.V,
// 		Width:  e.width,
// 		Height: e.height,
// 	}

// 	return e.encoder.Encode(yuvImg)
// 	*/

// 	// For now, return mock H.264 data (replace with actual encoding)
// 	return e.generateMockH264Frame(), nil
// }

// type YUVData struct {
// 	Y, U, V []byte
// }

// func (e *H264Encoder) imageToYUV420P(img image.Image) (*YUVData, error) {
// 	bounds := img.Bounds()
// 	width := bounds.Dx()
// 	height := bounds.Dy()

// 	// Ensure dimensions match encoder settings
// 	if width != e.width || height != e.height {
// 		return nil, fmt.Errorf("image dimensions (%dx%d) don't match encoder (%dx%d)",
// 			width, height, e.width, e.height)
// 	}

// 	// Allocate YUV420P buffers
// 	ySize := width * height
// 	uvSize := ySize / 4

// 	yData := make([]byte, ySize)
// 	uData := make([]byte, uvSize)
// 	vData := make([]byte, uvSize)

// 	// Convert RGB to YUV420P
// 	for y := 0; y < height; y++ {
// 		for x := 0; x < width; x++ {
// 			r, g, b, _ := img.At(x, y).RGBA()

// 			// Convert to 8-bit
// 			r8 := uint8(r >> 8)
// 			g8 := uint8(g >> 8)
// 			b8 := uint8(b >> 8)

// 			// RGB to YUV conversion
// 			yVal := uint8((66*int(r8) + 129*int(g8) + 25*int(b8) + 128) >> 8 + 16)
// 			yData[y*width+x] = yVal

// 			// Sample U and V at 2x2 intervals for 4:2:0 subsampling
// 			if x%2 == 0 && y%2 == 0 {
// 				uVal := uint8((-38*int(r8) - 74*int(g8) + 112*int(b8) + 128) >> 8 + 128)
// 				vVal := uint8((112*int(r8) - 94*int(g8) - 18*int(b8) + 128) >> 8 + 128)

// 				uvIndex := (y/2)*(width/2) + (x/2)
// 				uData[uvIndex] = uVal
// 				vData[uvIndex] = vVal
// 			}
// 		}
// 	}

// 	return &YUVData{Y: yData, U: uData, V: vData}, nil
// }

// func (e *H264Encoder) generateMockH264Frame() []byte {
// 	// Mock H.264 NAL units - replace this with actual x264 encoding
// 	// This is just for testing - won't produce valid video
// 	nalHeader := []byte{0x00, 0x00, 0x00, 0x01} // NAL start code
// 	nalUnit := []byte{0x67, 0x42, 0x00, 0x1E} // Mock SPS

// 	frame := make([]byte, 0, len(nalHeader)+len(nalUnit)+100)
// 	frame = append(frame, nalHeader...)
// 	frame = append(frame, nalUnit...)

// 	// Add some mock frame data
// 	mockData := make([]byte, 50)
// 	for i := range mockData {
// 		mockData[i] = byte(i % 256)
// 	}
// 	frame = append(frame, mockData...)

// 	return frame
// }

// func (e *H264Encoder) Close() error {
// 	// TODO: Close x264 encoder
// 	/*
// 	if e.encoder != nil {
// 		e.encoder.Close()
// 	}
// 	*/
// 	return nil
// }

// func NewVideoFrameStreamer(width, height, fps int) *VideoFrameStreamer {
// 	return &VideoFrameStreamer{
// 		peers:        make(map[string]*PeerInfo),
// 		frameChannel: make(chan []byte, 10),
// 		width:        width,
// 		height:       height,
// 		fps:          fps,
// 	}
// }

// func (vfs *VideoFrameStreamer) CreatePeerConnection(clientID string) (*webrtc.PeerConnection, error) {
// 	config := webrtc.Configuration{
// 		ICEServers: []webrtc.ICEServer{
// 			{URLs: []string{"stun:stun.l.google.com:19302"}},
// 		},
// 	}

// 	peerConnection, err := webrtc.NewPeerConnection(config)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Create H.264 track
// 	videoTrack, err := webrtc.NewTrackLocalStaticSample(
// 		webrtc.RTPCodecCapability{
// 			MimeType:  webrtc.MimeTypeH264,
// 			ClockRate: 90000,
// 		},
// 		"video",
// 		"pion",
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create H.264 track: %v", err)
// 	}

// 	// Add track to peer connection
// 	rtpSender, err := peerConnection.AddTrack(videoTrack)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to add track: %v", err)
// 	}

// 	// Create H.264 encoder for this peer
// 	encoder, err := NewH264Encoder(vfs.width, vfs.height, vfs.fps)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create encoder: %v", err)
// 	}

// 	// Handle RTCP
// 	go func() {
// 		rtcpBuf := make([]byte, 1500)
// 		for {
// 			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
// 				return
// 			}
// 		}
// 	}()

// 	// Store peer info
// 	peerInfo := &PeerInfo{
// 		connection: peerConnection,
// 		track:      videoTrack,
// 		encoder:    encoder,
// 	}

// 	vfs.mu.Lock()
// 	vfs.peers[clientID] = peerInfo
// 	vfs.mu.Unlock()

// 	// Start streaming to this peer
// 	go vfs.streamToPeer(clientID)

// 	// Handle connection state changes
// 	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
// 		log.Printf("Peer %s state: %s", clientID, s.String())
// 		if s == webrtc.PeerConnectionStateFailed ||
// 			s == webrtc.PeerConnectionStateDisconnected ||
// 			s == webrtc.PeerConnectionStateClosed {
// 			vfs.removePeer(clientID)
// 		}
// 	})

// 	return peerConnection, nil
// }

// func (vfs *VideoFrameStreamer) streamToPeer(clientID string) {
// 	ticker := time.NewTicker(time.Second / time.Duration(vfs.fps))
// 	defer ticker.Stop()

// 	var lastFrame []byte

// 	for {
// 		select {
// 		case <-ticker.C:
// 			// Get latest frame
// 			select {
// 			case frame := <-vfs.frameChannel:
// 				lastFrame = frame
// 			default:
// 				// Use last frame if no new frame available
// 			}

// 			if lastFrame == nil {
// 				continue
// 			}

// 			vfs.mu.RLock()
// 			peerInfo, exists := vfs.peers[clientID]
// 			vfs.mu.RUnlock()

// 			if !exists {
// 				return
// 			}

// 			// Encode JPEG to H.264
// 			h264Data, err := peerInfo.encoder.EncodeJPEG(lastFrame)
// 			if err != nil {
// 				log.Printf("Encoding error for peer %s: %v", clientID, err)
// 				continue
// 			}

// 			// Send H.264 frame
// 			sample := media.Sample{
// 				Data:     h264Data,
// 				Duration: time.Second / time.Duration(vfs.fps),
// 			}

// 			if err := peerInfo.track.WriteSample(sample); err != nil {
// 				log.Printf("Error writing sample to peer %s: %v", clientID, err)
// 				return
// 			}

// 		case <-time.After(5 * time.Second):
// 			// Check if peer still exists
// 			vfs.mu.RLock()
// 			_, exists := vfs.peers[clientID]
// 			vfs.mu.RUnlock()
// 			if !exists {
// 				return
// 			}
// 		}
// 	}
// }

// // AddJPEGFrame adds a JPEG frame to be encoded and streamed
// func (vfs *VideoFrameStreamer) AddJPEGFrame(jpegData []byte) {
// 	select {
// 	case vfs.frameChannel <- jpegData:
// 		// Frame added successfully
// 	default:
// 		// Channel full, drop oldest frame
// 		select {
// 		case <-vfs.frameChannel:
// 			vfs.frameChannel <- jpegData
// 		default:
// 		}
// 	}
// }

// func (vfs *VideoFrameStreamer) removePeer(clientID string) {
// 	vfs.mu.Lock()
// 	defer vfs.mu.Unlock()

// 	if peerInfo, exists := vfs.peers[clientID]; exists {
// 		peerInfo.connection.Close()
// 		peerInfo.encoder.Close()
// 		delete(vfs.peers, clientID)
// 		log.Printf("Removed peer: %s", clientID)
// 	}
// }

// func (vfs *VideoFrameStreamer) Close() error {
// 	vfs.mu.Lock()
// 	defer vfs.mu.Unlock()

// 	for clientID, peerInfo := range vfs.peers {
// 		peerInfo.connection.Close()
// 		peerInfo.encoder.Close()
// 		delete(vfs.peers, clientID)
// 	}

// 	close(vfs.frameChannel)
// 	return nil
// }

// // Example usage:
// func main() {
// 	// Create streamer for 1920x1080 at 30fps
// 	streamer := NewVideoFrameStreamer(1920, 1080, 30)
// 	defer streamer.Close()

// 	// Example: Add JPEG frames (same as your WebSocket data)
// 	// jpegData := getJPEGFromCamera() // Your JPEG data
// 	// streamer.AddJPEGFrame(jpegData)

// 	// Create peer connection
// 	// peerConnection, err := streamer.CreatePeerConnection("client123")
// 	// Handle WebRTC signaling...
// }

// /*
// TO MAKE THIS WORK, YOU NEED TO:

// 1. Install x264-go:
//    go get github.com/gen2brain/x264-go

// 2. Uncomment the x264 imports and code sections marked with TODO

// 3. The flow will be:
//    JPEG → Decode → RGB → YUV420P → H.264 → WebRTC

// 4. On the client side, you'll receive H.264 video stream that browsers can decode natively

// 5. Client-side JavaScript will be much simpler:
//    ```javascript
//    const video = document.getElementById('video');
//    const pc = new RTCPeerConnection();

//    pc.ontrack = (event) => {
//        video.srcObject = event.streams[0];
//    };
//    ```

// This approach gives you proper video streaming with H.264 compression instead of sending individual JPEG frames.
// */
