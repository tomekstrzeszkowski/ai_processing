package webRTC

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

/*
package main

import (
	"log"
	"net/http"

	webrtc "strzcam.com/broadcaster/webRTC"
)

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

*/

// Message types for signaling
type MessageType string

const (
	MessageTypeOffer     MessageType = "offer"
	MessageTypeAnswer    MessageType = "answer"
	MessageTypeCandidate MessageType = "candidate"
	MessageTypeJoin      MessageType = "join"
	MessageTypeLeave     MessageType = "leave"
)

// SignalingMessage represents WebRTC signaling messages
type SignalingMessage struct {
	Type      MessageType                `json:"type"`
	RoomID    string                     `json:"roomId,omitempty"`
	ClientID  string                     `json:"clientId,omitempty"`
	Offer     *webrtc.SessionDescription `json:"offer,omitempty"`
	Answer    *webrtc.SessionDescription `json:"answer,omitempty"`
	Candidate *webrtc.ICECandidate       `json:"candidate,omitempty"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID     string
	Conn   *websocket.Conn
	RoomID string
	Send   chan []byte
}

// Room represents a signaling room
type Room struct {
	ID      string
	Clients map[string]*Client
	mu      sync.RWMutex
}

// Hub manages rooms and clients
type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// VideoEncoder handles video encoding
type VideoEncoder struct {
	ffmpegCmd *exec.Cmd
	stdin     *bytes.Buffer
	stdout    *bytes.Buffer
	mutex     sync.Mutex
}

func NewVideoEncoder() *VideoEncoder {
	return &VideoEncoder{
		stdin:  &bytes.Buffer{},
		stdout: &bytes.Buffer{},
	}
}

// Convert JPEG to H.264 using FFmpeg (requires FFmpeg installed)
func (ve *VideoEncoder) ConvertJPEGToH264(jpegData []byte) ([]byte, error) {
	ve.mutex.Lock()
	defer ve.mutex.Unlock()

	// FFmpeg command to convert JPEG to H.264
	cmd := exec.Command("ffmpeg",
		"-f", "image2pipe", // Input format
		"-vcodec", "mjpeg", // Input codec
		"-i", "pipe:0", // Input from stdin
		"-vcodec", "libx264", // Output codec
		"-preset", "ultrafast", // Fast encoding
		"-tune", "zerolatency", // Low latency
		"-crf", "23", // Quality
		"-pix_fmt", "yuv420p", // Pixel format
		"-f", "h264", // Output format
		"pipe:1", // Output to stdout
	)

	cmd.Stdin = bytes.NewReader(jpegData)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg error: %v", err)
	}

	return output, nil
}

// Simpler approach: Convert JPEG to raw YUV data for WebRTC
func (ve *VideoEncoder) ConvertJPEGToYUV(jpegData []byte) ([]byte, error) {
	// Decode JPEG
	img, err := jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %v", err)
	}

	// Get dimensions from the image
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Convert to YUV420P format (simplified)
	yuvData := make([]byte, width*height*3/2) // YUV420P size

	// This is a simplified conversion - for production use a proper library
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to YUV (simplified)
			yVal := uint8((299*r + 587*g + 114*b) / 1000 / 256)
			yuvData[(y-bounds.Min.Y)*width+(x-bounds.Min.X)] = yVal
		}
	}

	return yuvData, nil
}

// VideoFrameStreamer handles video frame streaming
type VideoFrameStreamer struct {
	frames  chan []byte
	peers   map[string]*webrtc.PeerConnection
	mu      sync.RWMutex
	encoder *VideoEncoder
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func NewVideoFrameStreamer() *VideoFrameStreamer {
	return &VideoFrameStreamer{
		frames:  make(chan []byte, 100),
		peers:   make(map[string]*webrtc.PeerConnection),
		encoder: NewVideoEncoder(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClientToRoom(client)
		case client := <-h.unregister:
			h.removeClientFromRoom(client)
		}
	}
}

func (h *Hub) addClientToRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[client.RoomID]
	if !exists {
		room = &Room{
			ID:      client.RoomID,
			Clients: make(map[string]*Client),
		}
		h.rooms[client.RoomID] = room
	}

	room.mu.Lock()
	room.Clients[client.ID] = client
	room.mu.Unlock()

	log.Printf("Client %s joined room %s", client.ID, client.RoomID)
}

func (h *Hub) removeClientFromRoom(client *Client) {
	h.mu.RLock()
	room, exists := h.rooms[client.RoomID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	room.mu.Lock()
	delete(room.Clients, client.ID)
	isEmpty := len(room.Clients) == 0
	room.mu.Unlock()

	if isEmpty {
		h.mu.Lock()
		delete(h.rooms, client.RoomID)
		h.mu.Unlock()
	}

	close(client.Send)
	log.Printf("Client %s left room %s", client.ID, client.RoomID)
}

func (h *Hub) broadcastJPEGFrame(frameData []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Create a message with the JPEG frame
	message := map[string]interface{}{
		"type": "jpeg_frame",
		"data": frameData,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling JPEG frame: %v", err)
		return
	}

	// Send to all clients in all rooms
	for _, room := range h.rooms {
		room.mu.RLock()
		for _, client := range room.Clients {
			select {
			case client.Send <- messageBytes:
			default:
				close(client.Send)
				delete(room.Clients, client.ID)
			}
		}
		room.mu.RUnlock()
	}
}
func (h *Hub) broadcastToRoom(roomID, senderID string, message []byte) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for clientID, client := range room.Clients {
		if clientID != senderID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(room.Clients, clientID)
			}
		}
	}
}

func (vfs *VideoFrameStreamer) processJPEGFrame(jpegData []byte) error {
	// Decode JPEG to get image dimensions
	img, err := jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return fmt.Errorf("failed to decode JPEG: %v", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	log.Printf("Processing JPEG frame: %dx%d", width, height)

	// Create a simple RTP packet with JPEG payload (Motion JPEG over RTP)
	// This is a simplified approach - proper implementation would need RTP packaging
	rtpPayload := make([]byte, len(jpegData)+12) // RTP header + JPEG data

	// Simple RTP header (12 bytes)
	rtpPayload[0] = 0x80 // Version 2, no padding, no extension, no CSRC
	rtpPayload[1] = 0x1A // Payload type for JPEG (26)
	// Sequence number (bytes 2-3) - could be incremented
	// Timestamp (bytes 4-7) - could be actual timestamp
	// SSRC (bytes 8-11) - source identifier

	// Copy JPEG data after RTP header
	copy(rtpPayload[12:], jpegData)

	// Add to frame buffer for WebRTC streaming
	vfs.AddFrame(rtpPayload)

	return nil
}

func (vfs *VideoFrameStreamer) AddFrame(frame []byte) {
	select {
	case vfs.frames <- frame:
	default:
		// Drop frame if buffer is full
		log.Println("Frame buffer full, dropping frame")
	}
}

func (vfs *VideoFrameStreamer) CreatePeerConnection(clientID string) (*webrtc.PeerConnection, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// Try to create H.264 track first, fallback to VP8
	var videoTrack *webrtc.TrackLocalStaticSample

	// For JPEG frames, we'll use Motion JPEG (MJPEG) codec
	videoTrack, err = webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:  "video/jpeg",
			ClockRate: 90000,
		},
		"video",
		"pion",
	)

	if err != nil {
		videoTrack, err = webrtc.NewTrackLocalStaticSample(
			webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
			"video",
			"pion",
		)
		if err != nil {
			// Final fallback to VP8
			videoTrack, err = webrtc.NewTrackLocalStaticSample(
				webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
				"video",
				"pion",
			)
			if err != nil {
				return nil, err
			}
		}
	}

	// Add track to peer connection
	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		return nil, err
	}

	vfs.mu.Lock()
	vfs.peers[clientID] = peerConnection
	vfs.mu.Unlock()

	// Start streaming frames to this peer
	go vfs.streamFramesToPeer(clientID, videoTrack)

	return peerConnection, nil
}

// func (vfs *VideoFrameStreamer) streamFramesToPeer(clientID string, track *webrtc.TrackLocalStaticSample) {
// 	ticker := time.NewTicker(33 * time.Millisecond) // ~30 FPS
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			select {
// 			case frame := <-vfs.frames:
// 				if err := track.WriteSample(media.Sample{
// 					Data:     frame,
// 					Duration: time.Second / 30,
// 				}); err != nil {
// 					log.Printf("Error writing sample for client %s: %v", clientID, err)
// 					return
// 				}
// 			default:
// 				// No frame available, continue
// 			}
// 		}
// 	}
// }

func (vfs *VideoFrameStreamer) streamFramesToPeer(clientID string, track *webrtc.TrackLocalStaticSample) {
	for jpegData := range vfs.frames {
		err := track.WriteSample(media.Sample{
			Data:     jpegData,         // Raw JPEG bytes
			Duration: time.Second / 30, // Assuming 30 FPS
		})
		if err != nil {
			log.Printf("Error writing sample: %v", err)
			return
		}
	}
}

func HandleWebSocket(hub *Hub, streamer *VideoFrameStreamer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	clientID := r.URL.Query().Get("clientId")
	roomID := r.URL.Query().Get("roomId")

	if clientID == "" || roomID == "" {
		conn.Close()
		return
	}

	client := &Client{
		ID:     clientID,
		Conn:   conn,
		RoomID: roomID,
		Send:   make(chan []byte, 256),
	}

	hub.register <- client

	go handleClientWrite(client)
	go handleClientRead(client, hub, streamer)
}

func handleClientWrite(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func handleClientRead(client *Client, hub *Hub, streamer *VideoFrameStreamer) {
	defer func() {
		log.Printf("Client %s read handler closing", client.ID)
		hub.unregister <- client
		client.Conn.Close()
	}()

	// Increase message size limit for WebRTC messages
	client.Conn.SetReadLimit(65536)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	log.Printf("Client %s read handler started", client.ID)

	for {
		_, messageBytes, err := client.Conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for client %s: %v", client.ID, err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected WebSocket close error: %v", err)
			}
			break
		}

		log.Printf("Client %s sent message: %s", client.ID, string(messageBytes))

		var message SignalingMessage
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			log.Printf("JSON unmarshal error from client %s: %v", client.ID, err)
			continue
		}

		log.Printf("Client %s message type: %s", client.ID, message.Type)

		switch message.Type {
		case MessageTypeOffer, MessageTypeAnswer, MessageTypeCandidate:
			// Forward signaling messages to other clients in the room
			log.Printf("Forwarding %s message from client %s", message.Type, client.ID)
			hub.broadcastToRoom(client.RoomID, client.ID, messageBytes)
		case MessageTypeJoin:
			log.Printf("Client %s joining, creating peer connection", client.ID)
			// Handle join - create peer connection here
			peerConnection, err := streamer.CreatePeerConnection(client.ID)
			if err != nil {
				log.Printf("Error creating peer connection for client %s: %v", client.ID, err)
				continue
			}

			// Set up ICE candidate handler
			peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
				if candidate == nil {
					return
				}

				log.Printf("Sending ICE candidate to client %s", client.ID)
				candidateMessage := SignalingMessage{
					Type:      MessageTypeCandidate,
					ClientID:  client.ID,
					Candidate: candidate,
				}

				messageBytes, _ := json.Marshal(candidateMessage)
				select {
				case client.Send <- messageBytes:
				default:
					log.Printf("Failed to send ICE candidate to client %s", client.ID)
					close(client.Send)
				}
			})

		case MessageTypeLeave:
			log.Printf("Client %s leaving, cleaning up peer connection", client.ID)
			// Clean up peer connection
			streamer.mu.Lock()
			if peer, exists := streamer.peers[client.ID]; exists {
				peer.Close()
				delete(streamer.peers, client.ID)
			}
			streamer.mu.Unlock()
		}
	}
}

// HTTP handler to receive video frames and broadcast to WebSocket clients
// Example usage:
// while true;
//
//	do
//	  if [ -f "/dev/shm/video_frame" ]; then
//	    curl -X POST -H "Content-Type: application/octet-stream" --data-binary @/dev/shm/video_frame http://localhost:8080/frame;
//	  fi;
//	  sleep 0.033  # ~30 FPS
//
// done
func HandleVideoFrame(streamer *VideoFrameStreamer, hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the frame data from request body
		frameData, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read frame data", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		frameLength := binary.BigEndian.Uint32(frameData[:4])
		if int(frameLength) != len(frameData)-4 {
			log.Printf("Length mismatch: header says %d, actual data is %d", frameLength, len(frameData)-4)
		}
		jpegData := frameData[4:]

		log.Printf("Received JPEG frame of %d bytes", len(frameData))

		// Option 1: Broadcast JPEG frame directly via WebSocket
		hub.broadcastJPEGFrame(jpegData)

		// Option 2: Convert for WebRTC (uncomment to enable)

		convertedData, err := streamer.encoder.ConvertJPEGToYUV(jpegData)
		if err != nil {
			log.Printf("Error converting JPEG: %v", err)
			// Fallback to JPEG broadcast
			hub.broadcastJPEGFrame(jpegData)
		} else {
			log.Printf("Converted: %d bytes", len(convertedData))
			streamer.AddFrame(convertedData)
		}

		// Option 3: Simple approach - encode JPEG as keyframe (experimental)
		// go func() {
		// 	if err := streamer.processJPEGFrame(jpegData); err != nil {
		// 		log.Printf("Error processing JPEG frame: %v", err)
		// 	}
		// }()

		w.WriteHeader(http.StatusOK)
	}
}

func ServeClient(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>WebRTC Video Stream</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        #video { width: 640px; height: 480px; border: 1px solid #ccc; }
        .controls { margin: 10px 0; }
        button { margin: 5px; padding: 10px 20px; }
        #status { margin: 10px 0; font-weight: bold; }
    </style>
</head>
<body>
    <h1>WebRTC Video Stream</h1>
    <div class="controls">
        <button onclick="startStream()">Start Stream</button>
        <button onclick="stopStream()">Stop Stream</button>
        <label>
            <input type="checkbox" id="useWebRTC" checked> Use WebRTC (uncheck for JPEG stream)
        </label>
    </div>
    <div id="status">Disconnected</div>
    <video id="video" autoplay muted style="display:block;"></video>
    <img id="jpeg-display" style="display:none; width: 640px; height: 480px; border: 1px solid #ccc;" />

    <script>
        let ws = null;
        let pc = null;
        let localStream = null;
        const clientId = Math.random().toString(36).substr(2, 9);
        const roomId = 'default';

        const video = document.getElementById('video');
        const jpegDisplay = document.getElementById('jpeg-display');
        const status = document.getElementById('status');
        const useWebRTC = document.getElementById('useWebRTC');

        function updateStatus(message) {
            status.textContent = message;
            console.log(message);
        }

        async function startStream() {
            try {
                updateStatus('Connecting...');
                
                // Connect to WebSocket
                const wsUrl = 'ws://localhost:8080/ws?clientId=' + clientId + '&roomId=' + roomId;
                ws = new WebSocket(wsUrl);
                
                ws.onopen = function() {
                    updateStatus('Connected to signaling server');
                    if (useWebRTC.checked) {
                        video.style.display = 'block';
                        jpegDisplay.style.display = 'none';
                        setTimeout(initWebRTC, 100);
                    } else {
                        video.style.display = 'none';
                        jpegDisplay.style.display = 'block';
                        updateStatus('Ready to receive JPEG frames');
                    }
                };
                
                ws.onmessage = function(event) {
                    const message = JSON.parse(event.data);
                    
                    if (message.type === 'jpeg_frame') {
                        // Handle JPEG frame
						const binaryString = atob(message.data);
						const uint8Array = new Uint8Array(binaryString.length);
						for (let i = 0; i < binaryString.length; i++) {
							uint8Array[i] = binaryString.charCodeAt(i);
						}
						
						// Create blob with proper MIME type
						const blob = new Blob([uint8Array], {type: 'image/jpeg'});;
                        const url = URL.createObjectURL(blob);
                        jpegDisplay.onload = function() {
                            URL.revokeObjectURL(url);
                        };
                        jpegDisplay.src = url;
                        updateStatus('Receiving JPEG frames');
                    } else {
                        // Handle WebRTC signaling
                        handleSignalingMessage(message);
                    }
                };
                
                ws.onclose = function(event) {
                    updateStatus('Disconnected (code: ' + event.code + ')');
                    console.log('WebSocket closed with code:', event.code, 'reason:', event.reason);
                };
                
                ws.onerror = function(error) {
                    updateStatus('WebSocket error');
                    console.error('WebSocket error:', error);
                };
                
            } catch (error) {
                updateStatus('Error: ' + error.message);
            }
        }

        async function initWebRTC() {
            try {
                // Create peer connection
                pc = new RTCPeerConnection({
                    iceServers: [
                        { urls: 'stun:stun.l.google.com:19302' }
                    ]
                });

                // Handle incoming tracks (video from server)
                pc.ontrack = function(event) {
                    updateStatus('Receiving video stream', event.streams);
                    video.srcObject = event.streams[0];
                };

                // Handle ICE candidates
                pc.onicecandidate = function(event) {
                    if (event.candidate) {
                        sendSignalingMessage({
                            type: 'candidate',
                            candidate: event.candidate
                        });
                    }
                };

                pc.onconnectionstatechange = function() {
                    updateStatus('Connection state: ' + pc.connectionState);
                };

                // Create offer to receive stream
                const offer = await pc.createOffer({ offerToReceiveVideo: true });
                await pc.setLocalDescription(offer);

                sendSignalingMessage({
                    type: 'offer',
                    offer: offer
                });

                // Tell server we're joining (this will trigger peer connection creation on server)
                sendSignalingMessage({
                    type: 'join'
                });

            } catch (error) {
                updateStatus('WebRTC error: ' + error.message);
            }
        }

        async function handleSignalingMessage(message) {
            try {
                switch (message.type) {
                    case 'answer':
                        if (message.answer) {
                            await pc.setRemoteDescription(message.answer);
                        }
                        break;
                    
                    case 'candidate':
                        if (message.candidate) {
                            await pc.addIceCandidate(message.candidate);
                        }
                        break;
                    
                    case 'offer':
                        if (message.offer) {
                            await pc.setRemoteDescription(message.offer);
                            const answer = await pc.createAnswer();
                            await pc.setLocalDescription(answer);
                            sendSignalingMessage({
                                type: 'answer',
                                answer: answer
                            });
                        }
                        break;
                }
            } catch (error) {
                updateStatus('Signaling error: ' + error.message);
            }
        }

        function sendSignalingMessage(message) {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify(message));
            }
        }

        function stopStream() {
            if (pc) {
                pc.close();
                pc = null;
            }
            if (ws) {
                sendSignalingMessage({ type: 'leave' });
                ws.close();
                ws = null;
            }
            if (video.srcObject) {
                video.srcObject = null;
            }
            updateStatus('Disconnected');
        }

        // Handle page unload
        window.addEventListener('beforeunload', function() {
            stopStream();
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	hub := NewHub()
	streamer := NewVideoFrameStreamer()

	go hub.Run()

	// WebSocket endpoint for signaling
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		HandleWebSocket(hub, streamer, w, r)
	})

	// HTTP endpoint to receive video frames
	http.HandleFunc("/frame", HandleVideoFrame(streamer, hub))

	// Serve the WebRTC client
	http.HandleFunc("/", ServeClient)

	log.Println("WebRTC Signaling Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
