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

	peerChan := connection.InitMDNS(host, connection.RendezVous)
	for {
		peer := <-peerChan
		if peer.ID == host.ID() {
			// if other end peer id greater than us, don't connect to it, just wait for it to connect us
			fmt.Println("Found peer:", peer, " id is greater than us, wait for it to connect to us")
			continue
		}
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
					server.BroadcastFramesAdaptative(frames)
				}
			}
		}
	}
}
