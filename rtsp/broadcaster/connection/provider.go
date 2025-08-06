package connection

import (
	"context"
	"encoding/binary"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

type Provider struct {
	host        host.Host
	frame       []byte
	frameBuffer [][]byte
}

func NewProvider(host host.Host) *Provider {
	return &Provider{host: host}
}

func (c *Provider) HandleConnectedPeers() {
	subscription, err := c.host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer subscription.Close()
		for evt := range subscription.Out() {
			connectEvent := evt.(event.EvtPeerConnectednessChanged)
			switch connectEvent.Connectedness {
			case network.Connected:
				//fmt.Println("connected {}", connectEvent.Peer)
			case network.NotConnected:
				//fmt.Println("disconnected {}", connectEvent.Peer)
			}
		}
	}()
}

func (c *Provider) StartListening(ctx context.Context) {
	fullAddr := GetHostAddress(c.host)
	log.Printf("I am %s\n", fullAddr)
	c.host.SetStreamHandler("/get-frame/1.0.0", func(stream network.Stream) {
		now := time.Now()
		timestamp := make([]byte, 8)
		binary.BigEndian.PutUint64(timestamp, uint64(now.UnixMicro()))
		for _, frame := range c.frameBuffer {
			stream.Write(append(timestamp, frame...))
		}
		stream.Close()
		c.frameBuffer = [][]byte{}
	})
}

func (c *Provider) BroadcastFrame(frame []byte) {
	c.frameBuffer = append(c.frameBuffer, frame)
}
