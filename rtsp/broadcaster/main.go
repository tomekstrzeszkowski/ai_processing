package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

type SharedMemoryReceiver struct {
	shmPath    string
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
	watcher    *fsnotify.Watcher
	upgrader   websocket.Upgrader
}

func NewSharedMemoryReceiver(shmName string) (*SharedMemoryReceiver, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	receiver := &SharedMemoryReceiver{
		shmPath: filepath.Join("/dev/shm", shmName),
		clients: make(map[*websocket.Conn]bool),
		watcher: watcher,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	// Watch the shared memory directory
	err = watcher.Add("/dev/shm")
	if err != nil {
		return nil, err
	}

	return receiver, nil
}

func (smr *SharedMemoryReceiver) readFrameFromShm() ([]byte, error) {
	// Check if file exists
	if _, err := os.Stat(smr.shmPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("shared memory file does not exist")
	}

	// Read the entire file
	data, err := os.ReadFile(smr.shmPath)
	if err != nil {
		return nil, err
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("invalid frame data: too short")
	}

	// Read frame size from header
	frameSize := binary.BigEndian.Uint32(data[:4])

	if len(data) < int(4+frameSize) {
		return nil, fmt.Errorf("invalid frame data: incomplete")
	}

	// Extract frame data
	frameData := data[4 : 4+frameSize]
	return frameData, nil
}

func (smr *SharedMemoryReceiver) broadcastFrame(frameData []byte) {
	smr.clientsMux.RLock()
	defer smr.clientsMux.RUnlock()

	message := map[string]interface{}{
		"type":      "frame",
		"data":      frameData,
		"timestamp": time.Now().Unix(),
	}

	// Send to all connected WebSocket clients
	for client := range smr.clients {
		err := client.WriteJSON(message)
		if err != nil {
			log.Printf("Error sending frame to client: %v", err)
			client.Close()
			delete(smr.clients, client)
		}
	}
}

func (smr *SharedMemoryReceiver) watchSharedMemory() {
	log.Println("Starting shared memory watcher...")

	for {
		select {
		case event, ok := <-smr.watcher.Events:
			if !ok {
				return
			}

			// Check if it's our target file and it was written to
			if event.Name == smr.shmPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				// Small delay to ensure write is complete
				time.Sleep(1 * time.Millisecond)

				frameData, err := smr.readFrameFromShm()
				if err != nil {
					log.Printf("Error reading frame from shared memory: %v", err)
					continue
				}

				log.Printf("New frame received: %d bytes", len(frameData))
				smr.broadcastFrame(frameData)
			}

		case err, ok := <-smr.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (smr *SharedMemoryReceiver) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := smr.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	smr.clientsMux.Lock()
	smr.clients[conn] = true
	smr.clientsMux.Unlock()

	log.Printf("New WebSocket client connected. Total clients: %d", len(smr.clients))

	// Handle client disconnection
	defer func() {
		smr.clientsMux.Lock()
		delete(smr.clients, conn)
		smr.clientsMux.Unlock()
		conn.Close()
		log.Printf("WebSocket client disconnected. Total clients: %d", len(smr.clients))
	}()

	// Keep connection alive and handle ping/pong
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (smr *SharedMemoryReceiver) handleStatus(w http.ResponseWriter, r *http.Request) {
	smr.clientsMux.RLock()
	clientCount := len(smr.clients)
	smr.clientsMux.RUnlock()

	status := map[string]interface{}{
		"status":   "running",
		"clients":  clientCount,
		"shm_path": smr.shmPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (smr *SharedMemoryReceiver) Close() {
	if smr.watcher != nil {
		smr.watcher.Close()
	}
}

func main() {
	receiver, err := NewSharedMemoryReceiver("video_frame")
	if err != nil {
		log.Fatal("Failed to create shared memory receiver:", err)
	}
	defer receiver.Close()

	// Start watching shared memory in background
	go receiver.watchSharedMemory()

	// Setup HTTP handlers
	http.HandleFunc("/ws", receiver.handleWebSocket)
	http.HandleFunc("/status", receiver.handleStatus)

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

	log.Println("Starting server on :8080")
	log.Println("WebSocket endpoint: ws://localhost:8080/ws")
	log.Println("Status endpoint: http://localhost:8080/status")
	log.Println("Test page: http://localhost:8080/")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
