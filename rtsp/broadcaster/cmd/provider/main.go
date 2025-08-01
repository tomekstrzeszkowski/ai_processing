package main

import (
	"context"
	"fmt"

	golog "github.com/ipfs/go-log/v2"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/watcher"
)

func main() {
	// Create a key from the rendezvous string
	memory, _ := watcher.NewSharedMemoryReceiver("video_frame")
	defer memory.Close()
	go memory.WatchSharedMemory()
	go memory.SaveFrameForLater()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	golog.SetAllLoggers(golog.LevelError)

	host, kademliaDHT, _ := connection.MakeEnhancedHost(ctx, 10000, false, 0)
	//host, _ := connection.MakeBasicHost(10000, false, 0)
	defer host.Close()
	defer kademliaDHT.Close()

	Provider := connection.NewProvider(host)
	Provider.StartListening(ctx)
	Provider.HandleConnectedPeers()
	connection.AnnounceDHT(ctx, kademliaDHT, connection.RendezVous)

	go func() {
		for frame := range memory.Frames {
			Provider.BroadcastFrame(frame)
		}
	}()
	peerChan := connection.InitMDNS(host, connection.RendezVous)

	for {
		peer := <-peerChan // will block until we discover a peer
		if peer.ID == host.ID() {
			// if other end peer id greater than us, don't connect to it, just wait for it to connect us
			fmt.Println("Found peer:", peer, " id is greater than us, wait for it to connect to us")
			continue
		}
		fmt.Println("Found peer:", peer, ", connecting")
		<-ctx.Done()
	}
}
