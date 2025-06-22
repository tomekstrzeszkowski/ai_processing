package connection

import (
	"context"
	"io"
	"log"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type Viewer struct {
	ID   peer.ID
	Host *host.Host
	Info *peer.AddrInfo
}

func splitJPEGFrames(data []byte) [][]byte {
	var frames [][]byte
	start := 0

	for i := 0; i < len(data)-1; i++ {
		// Look for JPEG start marker (0xFF 0xD8)
		if data[i] == 0xFF && data[i+1] == 0xD8 && i > start {
			frames = append(frames, data[start:i])
			start = i
		}
	}

	// Add the last frame
	if start < len(data) {
		frames = append(frames, data[start:])
	}

	return frames
}
func CreateAndConnectNewViewer(ctx context.Context, host *host.Host, info peer.AddrInfo) (*Viewer, error) {
	fullAddr := GetHostAddress(*host)
	log.Printf("I'm %s\n", fullAddr)
	if err := (*host).Connect(ctx, info); err != nil {
		log.Println("Connection failed:", err)
		return nil, err
	}
	(*host).Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	log.Println("sender opening stream")
	(*host).Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	return &Viewer{
		ID:   info.ID,
		Host: host,
		Info: &info,
	}, nil
}

func (v *Viewer) GetFrame() []byte {
	stream, err := (*v.Host).NewStream(context.Background(), (*v.Info).ID, "/get-frame/1.0.0")
	if err != nil {
		log.Println(err)
		return []byte{}
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("Error reading stream: %v", err)
		return []byte{}
	}

	return data
}
func (v *Viewer) GetFrames() [][]byte {
	return splitJPEGFrames(v.GetFrame())
}
