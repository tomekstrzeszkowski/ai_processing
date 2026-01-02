package watcher

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"strzcam.com/broadcaster/frame"
)

type HLSConverter struct {
	segmentDir    string
	playlistPath  string
	ffmpegCmd     *exec.Cmd
	frameWriter   io.WriteCloser
	mu            sync.Mutex
	segmentNumber int
	Frames        chan []frame.Frame
	width         int
	height        int
	fps           float64
}

func NewHLSConverter(outputDir string, frames chan []frame.Frame) (*HLSConverter, error) {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			panic(fmt.Sprintf("Cannot create directory: %v", err))
		}
	}
	return &HLSConverter{
		segmentDir:   outputDir,
		playlistPath: filepath.Join(outputDir, "stream.m3u8"),
		Frames:       frames,
		width:        0, // Will be set from first frame
		height:       0, // Will be set from first frame
		fps:          1, // Default FPS
	}, nil
}

func (h *HLSConverter) Start() error {
	go h.processFrames()
	log.Println("HLS converter started, waiting for first frame...")
	return nil
}

func (h *HLSConverter) processFrames() {
	// Wait for first frame to get dimensions
	firstFrameSet, ok := <-h.Frames
	if !ok {
		log.Println("Frame channel closed before receiving first frame")
		return
	}

	if len(firstFrameSet) == 0 {
		log.Println("Received empty frame set")
		return
	}

	// Get dimensions from first frame
	firstChunk := firstFrameSet[0]
	h.width = int(firstChunk.Width)
	h.height = int(firstChunk.Height)

	log.Printf("Received first frame: %dx%d, starting FFmpeg", h.width, h.height)

	// Now start FFmpeg with correct dimensions
	if err := h.startFFmpeg(); err != nil {
		log.Printf("Failed to start FFmpeg: %v", err)
		return
	}

	// Write the first frame
	for _, chunk := range firstFrameSet {
		if _, err := h.frameWriter.Write(chunk.Data); err != nil {
			log.Printf("Error writing first frame to ffmpeg: %v", err)
			return
		}
	}

	// Continue writing remaining frames
	h.writeFramesToFFmpeg()
}

func (h *HLSConverter) startFFmpeg() error {
	h.ffmpegCmd = exec.Command("ffmpeg",
		"-f", "rawvideo",
		"-pixel_format", "yuv420p",
		"-video_size", fmt.Sprintf("%dx%d", h.width, h.height),
		"-framerate", fmt.Sprintf("%f", h.fps),
		"-i", "pipe:0",
		"-vf", fmt.Sprintf("fps=%f", h.fps), // Force constant FPS
		"-vsync", "cfr",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-g", "30",
		"-sc_threshold", "0",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "6",
		"-hls_flags", "delete_segments+omit_endlist+independent_segments",
		"-hls_segment_type", "mpegts",
		"-hls_allow_cache", "0",
		"-hls_segment_filename", filepath.Join(h.segmentDir, "segment_%03d.ts"),
		h.playlistPath,
	)

	h.ffmpegCmd.Stderr = os.Stderr
	h.ffmpegCmd.Stdout = os.Stdout

	stdinPipe, err := h.ffmpegCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get ffmpeg stdin pipe: %w", err)
	}
	h.frameWriter = stdinPipe

	if err := h.ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	log.Printf("FFmpeg started with dimensions %dx%d @ %d fps", h.width, h.height, h.fps)
	return nil
}

func (h *HLSConverter) writeFramesToFFmpeg() {
	if h.frameWriter == nil {
		return
	}
	defer h.frameWriter.Close()

	for frameSet := range h.Frames {
		for _, frame := range frameSet {
			fmt.Printf("FPS set in frame: %f %dx%d\n", frame.Fps, frame.Width, frame.Height)
			// Verify dimensions haven't changed
			if int(frame.Width) != h.width || int(frame.Height) != h.height {
				log.Printf("WARNING: Frame dimensions changed from %dx%d to %dx%d - this may cause issues",
					h.width, h.height, frame.Width, frame.Height)
			}

			if _, err := h.frameWriter.Write(frame.Data); err != nil {
				log.Printf("Error writing to ffmpeg stdin: %v", err)
				return
			}
			if (frame.Fps > 0) && (math.Abs(frame.Fps-h.fps) > 1) {
				h.SetFpsAndSize(frame.Fps, int(frame.Width), int(frame.Height))
				log.Printf("Adaptive FPS adjustment: setting FPS to %.1f", h.fps)
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

func (h *HLSConverter) SetFpsAndSize(fps float64, width int, height int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fps = fps
	h.width = width
	h.height = height
}
