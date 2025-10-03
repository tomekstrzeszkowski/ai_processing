package main

import (
	"context"
	"fmt"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/watcher"
	"log"
	"time"
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
	defer host.Close()
	defer kademliaDHT.Close()

	Provider := connection.NewProvider(host, savePath)
	Provider.StartListening(ctx)
	Provider.HandleConnectedPeers()
	rendezVous, _ := connection.GetRendezVousCid(connection.RendezVous)
	announced := false
	for i := range 10 {
	    if connection.AnnounceDHT(ctx, kademliaDHT, rendezVous) {
	        announced = true
	        break
	    }
	    if i < 9 {
	    	// Exponential-ish backoff
	    	log.Printf("Failed to make initial DHT announcement attempt %d", i)
	        time.Sleep(time.Second * time.Duration((i+1)*i))
	    }
	}

	if !announced {
	    log.Printf("Failed to make initial DHT announcement after 10 attempts")
	} else {
		log.Printf("DHT announced!")
	}
	connection.AnnounceDHTPeriodically(ctx, kademliaDHT, rendezVous)

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
			log.Println("Found peer via MDNS:", peer)
		case peer = <-dhtPeerChan:
			log.Println("Found peer via DHT:", peer)
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
