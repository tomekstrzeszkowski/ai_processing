package connection

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"strzcam.com/broadcaster/frame"
	"strzcam.com/broadcaster/video"
)

type Viewer struct {
	ID              peer.ID
	Host            *host.Host
	Info            *peer.AddrInfo
	lastFramePacket *time.Time
}

func splitJPEGFrames(data []byte) ([][]byte, error) {
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

	return frames, nil
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

func (v *Viewer) GetData() (*time.Time, []byte) {
	stream, err := (*v.Host).NewStream(context.Background(), (*v.Info).ID, "/get-frame/1.0.0")
	if err != nil {
		log.Println(err)
		return nil, []byte{}
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("Error reading stream: %v", err)
		return nil, []byte{}
	}
	timestampBytes := data[0:8]
	unixTimestamp := binary.BigEndian.Uint64(timestampBytes)
	timestamp := time.UnixMicro(int64(unixTimestamp))

	return &timestamp, data[8:]
}

func (v *Viewer) isTimestampHealthy(ts *time.Time) bool {
	if ts == nil {
		return false
	}
	return !v.lastFramePacket.After(*ts)
}
func (v *Viewer) GetFrames() ([]frame.Frame, error) {
	ts, dataFrames := v.GetData()
	if v.lastFramePacket == nil {
		v.lastFramePacket = ts
	}
	if !v.isTimestampHealthy(ts) {
		return nil, fmt.Errorf("Received frames from the past!")
	}
	v.lastFramePacket = ts
	var frames []frame.Frame
	json.Unmarshal(dataFrames, &frames)
	return frames, nil
}
func (v *Viewer) GetVideoList(start time.Time, end time.Time) []video.Video {
	stream, err := (*v.Host).NewStream(context.Background(), (*v.Info).ID, "/get-video-list/1.0.0")
	if err != nil {
		log.Println(err)
		return []video.Video{}
	}
	defer stream.Close()
	dateRange := fmt.Sprintf("%s-%s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	stream.Write([]byte(dateRange + "\n"))
	data, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("Error reading stream: %v", err)
		return []video.Video{}
	}
	var videoList []video.Video
	err = json.Unmarshal(data, &videoList)
	return videoList
}

func (v *Viewer) GetVideo(name string) []byte {
	stream, err := (*v.Host).NewStream(context.Background(), (*v.Info).ID, "/get-video/1.0.0")
	if err != nil {
		log.Println(err)
		return []byte{}
	}
	defer stream.Close()
	stream.Write([]byte(name + "\n"))
	data, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("Error reading stream: %v", err)
		return []byte{}
	}
	return data
}
