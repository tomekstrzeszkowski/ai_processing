package web_rtc

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

var clients map[int]*connInfo
var clientsMux sync.RWMutex
var upgrader websocket.Upgrader

type connInfo struct {
	conn      *websocket.Conn
	writeMux  sync.Mutex
	ice       []SignalingMessage
	offer     *SignalingMessage
	offerFrom int
}

func RunServer(port int) {
	clients = make(map[int]*connInfo)
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
	go func() {
		for clientId, client := range clients {
			log.Printf("Client %d: offer from: %d", clientId, client.offerFrom)
		}
	}()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userId, _ := strconv.Atoi(r.URL.Query().Get("userId"))
		conn, _ := upgrader.Upgrade(w, r, nil)
		client := &connInfo{
			conn:      conn,
			ice:       []SignalingMessage{},
			offer:     &SignalingMessage{},
			offerFrom: 0,
		}

		clientsMux.Lock()
		clients[userId] = client
		clientsMux.Unlock()

		log.Printf("New WebSocket client connected %d", userId)
		if len(clients) > 1 {
			clientsMux.RLock()
			for otherUserId, otherClient := range clients {
				if otherUserId != userId {
					for ice := range otherClient.ice {
						log.Printf("Send saved ice %d to %d", otherUserId, userId)
						client.writeMux.Lock()
						err := client.conn.WriteJSON(ice)
						client.writeMux.Unlock()
						if err != nil {
							log.Println("Write error:", err)
						}
					}
					otherClient.ice = []SignalingMessage{}
					if otherClient.offer != nil {
						log.Printf("Send saved offer %d to %d", otherUserId, userId)
						client.writeMux.Lock()
						err := client.conn.WriteJSON(otherClient.offer)
						client.writeMux.Unlock()
						if err != nil {
							log.Println("Write error:", err)
						}
						otherClient.offer = nil
						client.offerFrom = otherUserId
					}
				}
			}
			clientsMux.RUnlock()
		}

		defer func() {
			clientsMux.Lock()
			if offeree, ok := clients[client.offerFrom]; ok {
				fmt.Printf("Flush offer for %d", client.offerFrom)
				offeree.ice = []SignalingMessage{}
				offeree.offer = nil
				offeree.offerFrom = 0
			}
			delete(clients, userId)
			clientsMux.Unlock()
			conn.Close()
			log.Printf("Client disconnected %d", userId)
		}()
		for {
			var msg SignalingMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("Read error:", err)
				break
			}
			log.Printf("Client %d has new message %s, client count: %d", userId, msg.Type, len(clients))
			clientsMux.RLock()
			messageSent := false
			for otherUserId, otherClient := range clients {
				if otherUserId != userId && (msg.Type == "ice" || msg.Type == "offer" || msg.Type == "answer") {
					log.Printf("Send %s %d to %d", msg.Type, userId, otherUserId)
					otherClient.writeMux.Lock()
					err := otherClient.conn.WriteJSON(msg)
					if msg.Type == "offer" {
						otherClient.offerFrom = userId
					}
					otherClient.writeMux.Unlock()
					if err != nil {
						log.Println("Write error:", err)
					}
					messageSent = true
				}
			}
			//save message if it's not send
			if !messageSent {
				switch msg.Type {
				case "ice":
					client.ice = append(client.ice, msg)
					log.Printf("Saved ice %d", userId)
				case "offer":
					client.offer = &msg
					log.Printf("Saved offer %d", userId)
				case "failed", "flush":
					client.ice = []SignalingMessage{}
					client.offer = nil
				}
			}
			clientsMux.RUnlock()
		}
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
