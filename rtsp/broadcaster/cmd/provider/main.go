package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	golog "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multihash"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/watcher"
)

func main() {

	rendezvous := "tstrz-b-p2p-app-v1.0.0"
	// Create a key from the rendezvous string
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

	Provider := connection.NewProvider(host)
	Provider.StartListening(ctx)
	Provider.HandleConnectedPeers()
	// ADD THIS: Announce on DHT for browser clients to find us
	go func() {
		// Create a proper CID from the rendezvous string
		hash := sha256.Sum256([]byte(rendezvous))
		mh, _ := multihash.EncodeName(hash[:], "sha2-256")
		rendezvousBytes := cid.NewCidV1(cid.Raw, mh)
		// Wait for DHT to be ready
		time.Sleep(5 * time.Second)
		fmt.Printf("Announcing on DHT with rendezvous: %s\n", rendezvous)

		// Announce ourselves as a provider for this rendezvous
		err := kademliaDHT.Provide(ctx, rendezvousBytes, true)
		if err != nil {
			fmt.Printf("Failed to announce on DHT: %v\n", err)
		} else {
			fmt.Println("Successfully announced on DHT!")
		}

		// Keep announcing periodically
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := kademliaDHT.Provide(ctx, rendezvousBytes, true)
				if err != nil {
					fmt.Printf("Failed to re-announce on DHT: %v\n", err)
				} else {
					fmt.Println("Re-announced on DHT")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		for frame := range memory.Frames {
			Provider.BroadcastFrame(frame)
		}
	}()
	peerChan := connection.InitMDNS(host, "tstrz-voting-p2p-app-v1.0.0")

	for {
		peer := <-peerChan // will block until we discover a peer
		fmt.Println("Found peer:", peer, ", connecting")
		<-ctx.Done()
	}
}
