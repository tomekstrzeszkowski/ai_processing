package main

import (
	"context"
	"fmt"
	"log"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/watcher"
)

func main() {
	server, _ := watcher.NewServer()
	server.PrepareEndpoints()
	go server.Start(":8080")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	golog.SetAllLoggers(golog.LevelError)

	host, kademliaDHT, _ := connection.MakeEnhancedHost(ctx, 10001, false, 0)
	//host, _ := connection.MakeBasicHost(10001, false, 0)
	defer host.Close()
	defer kademliaDHT.Close()

	peerChan := connection.InitMDNS(host, "tstrz-b-p2p-app-v1.0.0")

	for {
		peer := <-peerChan
		fmt.Println("Found peer:", peer, ", connecting")
		viewer, _ := connection.CreateAndConnectNewViewer(ctx, &host, peer)
		if viewer == nil {
			continue
		}
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("Exiting.")
				return
			case <-ticker.C:
				frames := viewer.GetFrames()
				frameCount := len(frames)
				if frameCount > 0 {
					log.Printf("Broadcasting frames: %d\n", frameCount)
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
							server.BroadcastFrame(frame)
						}
					}(frames, interval)
				}
			}
		}
	}
}
