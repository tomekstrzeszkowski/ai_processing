package connection

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type Viewer struct {
	ID              peer.ID
	Host            *host.Host
	Info            *peer.AddrInfo
	lastFramePacket *time.Time
}

func splitTimestampedJPEGFrames(data []byte) ([]time.Time, [][]byte, error) {
	var timestamps []time.Time
	var frames [][]byte
	start := 0

	for i := 8; i < len(data)-1; i++ { // Start at byte 8 (after first timestamp)
		// Look for JPEG start marker (0xFF 0xD8)
		if data[i] == 0xFF && data[i+1] == 0xD8 && i > start+8 {
			// Extract timestamp from current frame (8 bytes before frame data)
			timestampBytes := data[start : start+8]
			unixTimestamp := binary.BigEndian.Uint64(timestampBytes)
			timestamp := time.UnixMicro(int64(unixTimestamp))
			timestamps = append(timestamps, timestamp)

			// Extract frame data (skip the 8-byte timestamp)
			frames = append(frames, data[start+8:i])
			start = i - 8 // Next timestamp starts 8 bytes before JPEG marker
		}
	}

	// Add the last frame
	if start < len(data) {
		if start+8 >= len(data) {
			return timestamps, frames, fmt.Errorf("incomplete frame: not enough data for timestamp")
		}

		// Extract timestamp for last frame
		timestampBytes := data[start : start+8]
		unixTimestamp := binary.BigEndian.Uint64(timestampBytes)
		timestamp := time.UnixMicro(int64(unixTimestamp))
		timestamps = append(timestamps, timestamp)

		// Extract last frame data
		frames = append(frames, data[start+8:])
	}

	return timestamps, frames, nil
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
func (v *Viewer) sortFramesByTimestamp(ts []time.Time, frames [][]byte) ([]time.Time, [][]byte) {
	indices := make([]int, len(ts))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return ts[indices[i]].Before(ts[indices[j]])
	})

	var sortedTs []time.Time
	var sortedFrames [][]byte

	for _, idx := range indices {
		if v.lastFramePacket == nil {
			v.lastFramePacket = &ts[idx]
		}
		log.Printf("sorting %s", ts[idx])
		if !v.lastFramePacket.After(ts[idx]) {
			sortedTs = append(sortedTs, ts[idx])
			sortedFrames = append(sortedFrames, frames[idx])
			v.lastFramePacket = &ts[idx]
		}

	}
	return sortedTs, sortedFrames
}
func (v *Viewer) GetFrames() ([]time.Time, [][]byte, error) {
	ts, frames, _ := splitTimestampedJPEGFrames(v.GetFrame())
	ts, frames = v.sortFramesByTimestamp(ts, frames)
	return ts, frames, nil
}
