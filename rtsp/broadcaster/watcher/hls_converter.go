package watcher

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type HLSConverter struct {
	segmentDir    string
	playlistPath  string
	ffmpegCmd     *exec.Cmd
	frameWriter   *os.File
	mu            sync.Mutex
	segmentNumber int
	Frames        chan [][]byte
}

func NewHLSConverter(outputDir string, frames chan [][]byte) (*HLSConverter, error) {
	segmentDir := filepath.Join(outputDir, "segments")
	if err := os.MkdirAll(segmentDir, 0755); err != nil {
		return nil, err
	}

	return &HLSConverter{
		segmentDir:   segmentDir,
		playlistPath: filepath.Join(outputDir, "stream.m3u8"),
		Frames:       frames,
	}, nil
}

func (h *HLSConverter) Start() error {
	// Create a named pipe for JPEG frames
	pipePath := filepath.Join(h.segmentDir, "frames.mjpeg")

	// Use FFmpeg to convert MJPEG to HLS
	h.ffmpegCmd = exec.Command("ffmpeg",
		"-f", "mjpeg",
		"-i", "pipe:0", // Read from stdin
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-g", "60", // Keyframe interval
		"-sc_threshold", "0",
		"-f", "hls",
		"-hls_time", "2", // 2 second segments
		"-hls_list_size", "5", // Keep 5 segments
		"-hls_flags", "delete_segments+append_list",
		"-hls_segment_filename", filepath.Join(h.segmentDir, "segment_%03d.ts"),
		h.playlistPath,
	)
	if err := h.ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	fw, err := os.OpenFile(pipePath, os.O_WRONLY, 0600)
	if err != nil {
		_ = h.ffmpegCmd.Process.Kill()
		return fmt.Errorf("failed to open fifo for writing: %w", err)
	}
	h.frameWriter = fw
	go h.writeFramesToFFmpeg(h.frameWriter)

	log.Println("HLS converter started")
	return nil
}

func (h *HLSConverter) writeFramesToFFmpeg(stdin *os.File) {
	defer stdin.Close()

	for frame := range h.Frames {
		for _, chunk := range frame {
			if _, err := stdin.Write(chunk); err != nil {
				log.Printf("Error writing to ffmpeg pipe: %v", err)
				return
			}
		}
	}
}

func (h *HLSConverter) Stop() error {
	if h.frameWriter != nil {
		_ = h.frameWriter.Close()
		h.frameWriter = nil
	}
	if h.ffmpegCmd != nil && h.ffmpegCmd.Process != nil {
		return h.ffmpegCmd.Process.Kill()
	}
	return nil
}
