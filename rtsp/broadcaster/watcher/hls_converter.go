package watcher

import (
	"fmt"
	"io"
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
	frameWriter   io.WriteCloser
	mu            sync.Mutex
	segmentNumber int
	Frames        chan [][]byte
}

func NewHLSConverter(outputDir string, frames chan [][]byte) (*HLSConverter, error) {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			panic(fmt.Sprintf("Cannot create directory: %v", err))
		}
	}
	return &HLSConverter{
		segmentDir:   outputDir,
		playlistPath: filepath.Join(outputDir, "stream.m3u8"),
		Frames:       frames,
	}, nil
}

func (h *HLSConverter) Start() error {
	// Use FFmpeg to convert MJPEG to HLS
	h.ffmpegCmd = exec.Command("ffmpeg",
		"-f", "mjpeg",
		"-re",
		"-i", "pipe:0",
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-g", "30", // Keyframe every 30 frames (more frequent)
		"-sc_threshold", "0",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "6", // Keep 6 segments (~12 seconds)
		"-hls_flags", "delete_segments+omit_endlist+independent_segments",
		"-hls_segment_type", "mpegts",
		"-hls_allow_cache", "0", // Disable caching
		"-hls_segment_filename", filepath.Join(h.segmentDir, "segment_%03d.ts"),
		h.playlistPath,
	)

	// Capture FFmpeg's stderr for debugging
	h.ffmpegCmd.Stderr = os.Stderr
	h.ffmpegCmd.Stdout = os.Stdout

	stdinPipe, err := h.ffmpegCmd.StdinPipe()
	if err != nil {
		log.Printf("failed to get ffmpeg stdin pipe: %v", err)
		return fmt.Errorf("failed to get ffmpeg stdin pipe: %w", err)
	}
	h.frameWriter = stdinPipe

	if err := h.ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	log.Printf("Starting to write frames to ffmpeg")
	go h.writeFramesToFFmpeg()

	log.Println("HLS converter started")
	return nil
}

func (h *HLSConverter) writeFramesToFFmpeg() {
	if h.frameWriter == nil {
		return
	}
	defer h.frameWriter.Close()
	log.Printf("Starting to write frames to ffmpeg")

	for frame := range h.Frames {
		for _, chunk := range frame {
			log.Print("Frame")
			if _, err := h.frameWriter.Write(chunk); err != nil {
				log.Printf("Error writing to ffmpeg stdin: %v", err)
				return
			}
		}
	}

	log.Printf("Frame channel closed, stopping write")
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
