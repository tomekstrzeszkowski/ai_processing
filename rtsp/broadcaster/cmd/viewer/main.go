package main

import (
	"context"
	"fmt"
	"log"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
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
	defer host.Close()
	defer kademliaDHT.Close()

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
		fmt.Println("Found peer:", peer, ", connecting")
		viewer, _ := connection.CreateAndConnectNewViewer(ctx, &host, peer)
		if viewer == nil {
			continue
		}
		server.AddViewer(viewer)
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
			}
		}
	}
}
