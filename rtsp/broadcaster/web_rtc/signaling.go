package web_rtc

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

var clients map[*connInfo]int
var clientsMux sync.RWMutex
var upgrader websocket.Upgrader

type connInfo struct {
	conn     *websocket.Conn
	writeMux sync.Mutex
}

func RunServer(port int) {
	clients = make(map[*connInfo]int)
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userId, _ := strconv.Atoi(r.URL.Query().Get("userId"))
		conn, _ := upgrader.Upgrade(w, r, nil)
		client := &connInfo{
			conn: conn,
		}

		clientsMux.Lock()
		clients[client] = userId
		clientsMux.Unlock()

		log.Printf("New WebSocket client connected %i", userId)

		defer func() {
			clientsMux.Lock()
			delete(clients, client)
			clientsMux.Unlock()
			conn.Close()
			log.Println("Client disconnected")
		}()
		for {
			var msg SignalingMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("Read error:", err)
				break
			}
			clientsMux.RLock()
			for otherClient, otherUserId := range clients {
				if otherClient != client {
					log.Println("SEND offer", userId, "to", otherUserId, msg.Type)
					otherClient.writeMux.Lock()
					err := otherClient.conn.WriteJSON(msg)
					otherClient.writeMux.Unlock()
					if err != nil {
						log.Println("Write error:", err)
					}
				}
			}
			clientsMux.RUnlock()
		}
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
