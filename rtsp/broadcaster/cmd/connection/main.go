package main

import (
	"context"
	"fmt"

	golog "github.com/ipfs/go-log/v2"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/watcher"
)

func main() {
	memory, _ := watcher.NewSharedMemoryReceiver("video_frame")
	defer memory.Close()
	go memory.WatchSharedMemory()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	golog.SetAllLoggers(golog.LevelError)

	host, kademliaDHT, _ := connection.MakeEnhancedHost(ctx, 10000, false, 0)
	//host, _ := connection.MakeBasicHost(10000, false, 0)
	defer host.Close()
	defer kademliaDHT.Close()

	controller := connection.NewController(host)
	controller.StartListening(ctx)
	controller.HandleConnectedPeers()
	go func() {
		for frame := range memory.Frames {
			controller.BroadcastFrame(frame)
		}
	}()
	peerChan := connection.InitMDNS(host, "tstrz-voting-p2p-app-v1.0.0")

	for {
		peer := <-peerChan // will block until we discover a peer
		fmt.Println("Found peer:", peer, ", connecting")
		<-ctx.Done()
	}
}
