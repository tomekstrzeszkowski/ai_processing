package connection

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"strzcam.com/broadcaster/video"
)

type Provider struct {
	host        host.Host
	frame       []byte
	frameBuffer [][]byte
	path        string
}

func NewProvider(host host.Host, path string) *Provider {
	return &Provider{host: host, path: path}
}

func (p *Provider) HandleConnectedPeers() {
	subscription, err := p.host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
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

func (p *Provider) StartListening(ctx context.Context) {
	fullAddr := GetHostAddress(p.host)
	log.Printf("I am %s\n", fullAddr)
	p.host.SetStreamHandler("/get-frame/1.0.0", func(stream network.Stream) {
		now := time.Now()
		timestamp := make([]byte, 8)
		binary.BigEndian.PutUint64(timestamp, uint64(now.UnixMicro()))
		stream.Write(timestamp)
		for _, frame := range p.frameBuffer {
			stream.Write(frame)
		}
		stream.Close()
		p.frameBuffer = [][]byte{}
	})
	p.host.SetStreamHandler("/get-video/1.0.0", func(stream network.Stream) {

	})

	p.host.SetStreamHandler("/get-video-list/1.0.0", func(stream network.Stream) {
		log.Printf("Get video list")
		buf := bufio.NewReader(stream)
		timeRangeData, _ := buf.ReadString('\n')
		timeRangeData = strings.ReplaceAll(timeRangeData, "\n", "")
		parts := strings.SplitN(timeRangeData, "-", 8)
		start, _ := time.Parse("2006-01-02", strings.Join(parts[0:3], "-"))
		end, _ := time.Parse("2006-01-02", strings.Join(parts[3:], "-"))
		videoList, _ := video.GetVideoByDateRange(p.path, start, end)
		jsonData, err := json.Marshal(videoList)
		if err != nil {
			log.Printf("Error marshaling JSON: %v", err)
			return
		}
		log.Printf("video list %v", videoList)
		stream.Write(jsonData)
		stream.Close()
	})
}

func (p *Provider) BroadcastFrame(frame []byte) {
	p.frameBuffer = append(p.frameBuffer, frame)
}
