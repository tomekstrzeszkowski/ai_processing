package main

import (
	"context"
	"fmt"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/watcher"
)

func main() {
	// Create a key from the rendezvous string
	savePath := fmt.Sprintf("%s_video_frame", watcher.SavePath)
	memory, _ := watcher.NewSharedMemoryReceiver("video_frame")
	converter, _ := watcher.NewConverter(savePath)
	creator, _ := watcher.NewVideoCreator(memory, converter)
	defer creator.Close()
	go creator.StartWatchingFrames()
	go creator.SaveFramesForLater()
	go creator.StartConversionWorkflow(&memory.ActualFps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	golog.SetAllLoggers(golog.LevelError)

	host, kademliaDHT, _ := connection.MakeEnhancedHost(ctx, 10000, false, 0)
	//host, _ := connection.MakeBasicHost(10000, false, 0)
	defer host.Close()
	defer kademliaDHT.Close()

	Provider := connection.NewProvider(host, savePath)
	Provider.StartListening(ctx)
	Provider.HandleConnectedPeers()
	connection.AnnounceDHT(ctx, kademliaDHT, connection.RendezVous)

	go func() {
		for frame := range creator.SharedMemoryReceiver.Frames {
			Provider.BroadcastFrame(frame)
		}
	}()
	mdnsPeerChan := connection.InitMDNS(host, connection.RendezVous)
	dhtPeerChan := connection.InitDHTDiscovery(ctx, host, kademliaDHT, connection.RendezVous)
	for {
		var peer peer.AddrInfo
		select {
		case peer = <-mdnsPeerChan:
			fmt.Println("Found peer via MDNS:", peer)
		case peer = <-dhtPeerChan:
			fmt.Println("Found peer via DHT:", peer)
		case <-ctx.Done():
			return
		}
		if peer.ID == host.ID() {
			fmt.Println("Found peer:", peer, " id is greater than us, wait for it to connect to us")
			continue
		}
		<-ctx.Done()
	}
}
