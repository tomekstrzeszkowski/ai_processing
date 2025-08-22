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
	server, _ := watcher.NewServer(8080)
	server.PrepareEndpoints()
	go server.Start()

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
				frames, err := viewer.GetFrames()
				if err != nil {
					log.Fatal(err)
					return
				}
				frameCount := len(frames)
				if frameCount > 0 {
					log.Printf("Broadcasting frames: %d\n", frameCount)
					server.BroadcastFramesAdaptative(frames)
				}
				switch server.WaitingForCommand {
				case watcher.GetVideo:
					video := viewer.GetVideo(server.VideoName)
					if !watcher.IsVideoChannelClosed(server.VideoData) {
						server.VideoData <- video
						close(server.VideoData)
					}
					server.WaitingForCommand = watcher.Idle
				case watcher.GetVideoList:
					videoList := viewer.GetVideoList(time.Now(), time.Now())
					if !watcher.IsChannelClosed(server.VideoList) {
						for _, video := range videoList {
							server.VideoList <- video
						}
						close(server.VideoList)
					}
					server.WaitingForCommand = watcher.Idle
				}
			}
		}
	}
}
