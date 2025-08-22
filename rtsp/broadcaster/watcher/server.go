package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/video"
)

type connInfo struct {
	conn     *websocket.Conn
	writeMux sync.Mutex
}
type Command int8

const (
	Idle Command = iota
	GetVideoList
	GetVideo
)

type Server struct {
	clients    map[*connInfo]bool
	clientsMux sync.RWMutex
	upgrader   websocket.Upgrader
	port       uint16
	Viewers    []*connection.Viewer
}

func NewServer(port uint16) (*Server, error) {
	receiver := &Server{
		clients: make(map[*connInfo]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		port: port,
	}
	return receiver, nil
}

func (s *Server) BroadcastFrame(frameData []byte) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	message := map[string]interface{}{
		"type":      "frame",
		"data":      frameData,
		"timestamp": time.Now().Unix(),
	}

	// Send to all connected WebSocket clients
	for clientInfo := range s.clients {
		// Lock the write mutex for this specific connection
		clientInfo.writeMux.Lock()
		err := clientInfo.conn.WriteJSON(message)
		clientInfo.writeMux.Unlock()

		if err != nil {
			log.Printf("Error sending frame to client: %v", err)
			clientInfo.conn.Close()
			delete(s.clients, clientInfo)
		}
	}
}
func (s *Server) BroadcastFramesAdaptative(frames [][]byte) {
	frameCount := len(frames)

	// Adaptive interval: faster for fewer frames, slower for many frames
	var interval time.Duration
	if frameCount <= 5 {
		interval = 200 * time.Millisecond // 5 FPS for few frames
	} else if frameCount <= 15 {
		interval = 100 * time.Millisecond // 10 FPS for medium
	} else {
		interval = 50 * time.Millisecond // 20 FPS for many frames
	}

	go func(frames [][]byte, delay time.Duration) {
		for i, frame := range frames {
			if i > 0 {
				time.Sleep(delay)
			}
			s.BroadcastFrame(frame)
		}
	}(frames, interval)
}
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	clientInfo := &connInfo{
		conn: conn,
	}

	// Register new client
	s.clientsMux.Lock()
	s.clients[clientInfo] = true
	s.clientsMux.Unlock()

	log.Printf("New WebSocket client connected. Total clients: %d", len(s.clients))

	// Handle client disconnect
	go func() {
		for {
			// Read message (we don't process it, just detect disconnection)
			if _, _, err := conn.ReadMessage(); err != nil {
				s.clientsMux.Lock()
				delete(s.clients, clientInfo)
				s.clientsMux.Unlock()
				conn.Close()
				break
			}
		}
	}()
}
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Handle preflight requests
	s.setCORSHeaders(w)
	w.WriteHeader(http.StatusOK)

	s.clientsMux.RLock()
	clientCount := len(s.clients)
	s.clientsMux.RUnlock()

	status := map[string]interface{}{
		"status":  "running",
		"clients": clientCount,
		//"shm_path": s.shmPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
func (s *Server) AddViewer(v *connection.Viewer) {
	for _, existingViewer := range s.Viewers {
		if existingViewer.ID == v.ID {
			return
		}
	}
	s.Viewers = append(s.Viewers, v)
}
func (s *Server) GetViewer() *connection.Viewer {
	if len(s.Viewers) > 0 {
		return s.Viewers[0]
	}
	return nil
}
func (s *Server) getVideo(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	videoName := r.PathValue("name")
	viewer := s.GetViewer()
	videoData := viewer.GetVideo(videoName)
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	reader := bytes.NewReader(videoData)
	w.WriteHeader(http.StatusOK)
	_, err := io.Copy(w, reader)
	if err != nil {
		log.Printf("Error streaming video: %v", err)
		// Note: Can't write error header here as we've already written the success header
		return
	}
}
func (s *Server) getVideoList(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	startParam := r.URL.Query().Get("start")
	endParam := r.URL.Query().Get("end")
	if startParam == "" {
		startParam = "2011-11-10"
	}
	if endParam == "" {
		endParam = "2011-11-11"
	}
	start, _ := time.Parse("2006-01-02", startParam)
	end, _ := time.Parse("2006-01-02", endParam)
	var videoList []video.Video
	viewer := s.GetViewer()
	videoList = viewer.GetVideoList(start, end)
	json.NewEncoder(w).Encode(videoList)
}

// Add CORS headers to response
func (s *Server) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (s *Server) PrepareEndpoints() {
	// Setup HTTP handlers
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/status", s.handleStatus)
	http.HandleFunc("/video-list", s.getVideoList)
	http.HandleFunc("/video/{name}", s.getVideo)

	// Serve static files for testing
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Video Stream</title>
</head>
<body>
    <h1>Live Video Stream` + fmt.Sprintf("%d", s.port) + `</h1>
    <img id="video" style="max-width: 100%;" />
    <div id="status"></div>
    
    <script>
        const ws = new WebSocket('ws://localhost:` + fmt.Sprintf("%d", s.port) + `/ws');
        const videoImg = document.getElementById('video');
        const statusDiv = document.getElementById('status');
        
        ws.onmessage = function(event) {
            const message = JSON.parse(event.data);
            if (message.type === 'frame') {
                console.log(message.data)
                const binaryString = atob(message.data);
                const uint8Array = new Uint8Array(binaryString.length);
                for (let i = 0; i < binaryString.length; i++) {
                    uint8Array[i] = binaryString.charCodeAt(i);
                }
                
                // Create blob with proper MIME type
                const blob = new Blob([uint8Array], {type: 'image/jpeg'});
                
                // Create object URL and set as image source
                const url = URL.createObjectURL(blob);
                videoImg.src = url;
                
                statusDiv.innerHTML = 'Frame received at: ' + new Date().toLocaleTimeString();
            }
        };
        
        ws.onopen = function() {
            statusDiv.innerHTML = 'Connected';
        };
        
        ws.onclose = function() {
            statusDiv.innerHTML = 'Disconnected';
        };
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})
}
func (s *Server) Start() {
	http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}
