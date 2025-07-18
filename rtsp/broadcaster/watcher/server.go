package watcher

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Server struct {
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
	upgrader   websocket.Upgrader
}

func NewServer() (*Server, error) {
	receiver := &Server{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
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
	for client := range s.clients {
		err := client.WriteJSON(message)
		if err != nil {
			log.Printf("Error sending frame to client: %v", err)
			client.Close()
			delete(s.clients, client)
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

	s.clientsMux.Lock()
	s.clients[conn] = true
	s.clientsMux.Unlock()

	log.Printf("New WebSocket client connected. Total clients: %d", len(s.clients))

	// Handle client disconnection
	defer func() {
		s.clientsMux.Lock()
		delete(s.clients, conn)
		s.clientsMux.Unlock()
		conn.Close()
		log.Printf("WebSocket client disconnected. Total clients: %d", len(s.clients))
	}()

	// Keep connection alive and handle ping/pong
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Handle preflight requests
	if r.Method == "OPTIONS" {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
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

	// Serve static files for testing
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Video Stream</title>
</head>
<body>
    <h1>Live Video Stream</h1>
    <img id="video" style="max-width: 100%;" />
    <div id="status"></div>
    
    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
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
func (s *Server) Start(addr string) {
	http.ListenAndServe(addr, nil)
}
