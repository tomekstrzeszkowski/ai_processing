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
		width:        0,  // Will be set as frame received
		height:       0,  // Will be set as frame received
		fps:          10, // Default FPS
	}, nil
}

func (h *HLSConverter) Start() error {
	go h.processFrames()
	log.Println("HLS converter started, waiting for first frame...")
	return nil
}

func (h *HLSConverter) startFFmpeg() error {
	h.ffmpegCmd = exec.Command("ffmpeg",
		"-f", "rawvideo",
		"-pixel_format", "yuv420p",
		"-video_size", fmt.Sprintf("%dx%d", h.width, h.height),
		"-framerate", fmt.Sprintf("%.2f", h.fps),
		"-thread_queue_size", "2048",
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-tune", "zerolatency",
		"-crf", "28",
		"-pix_fmt", "yuv420p",
		"-g", fmt.Sprintf("%d", int(h.fps)),
		"-keyint_min", fmt.Sprintf("%d", int(h.fps)),
		"-sc_threshold", "0",
		"-b:v", "2500k",
		"-maxrate", "5000k",
		"-bufsize", "10000k",
		"-bsf:v", "h264_mp4toannexb", // Ensure Annex B format with SPS/PPS
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "6",
		"-hls_flags", "delete_segments+omit_endlist",
		"-hls_segment_type", "mpegts",
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

	log.Printf("FFmpeg started with dimensions %dx%d @ %.2f fps", h.width, h.height, h.fps)
	return nil
}

func (h *HLSConverter) processFrames() {
	for frameSet := range h.Frames {
		var combinedData []byte
		for _, f := range frameSet {
			log.Printf("Processing frame: %dx%d @ %.2f fps", f.Width, f.Height, f.Fps)
			expectedSize := int(f.Width) * int(f.Height) * 3 / 2

			// Verify frame size
			if len(f.Data) != expectedSize {
				log.Printf("WARNING: Frame size mismatch: got %d, expected %d for %dx%d",
					len(f.Data), expectedSize, f.Width, f.Height)
				continue
			}
			if f.Width != uint32(h.width) || f.Height != uint32(h.height) || math.Abs(h.fps-f.Fps) > 1 {
				h.SetFpsAndSize(f.Fps+0.1, int(f.Width), int(f.Height))
			}
			if h.frameWriter == nil {
				if err := h.startFFmpeg(); err != nil {
					log.Printf("Failed to start FFmpeg: %v", err)
					return
				}
				defer h.frameWriter.Close()
			}
			combinedData = append(combinedData, f.Data...)
		}
		if h.frameWriter == nil {
			continue
		}
		if _, err := h.frameWriter.Write(combinedData); err != nil {
			log.Printf("Error writing to ffmpeg stdin: %v", err)
			return
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

func (h *HLSConverter) SetFpsAndSize(fps float64, width int, height int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fps = fps
	h.width = width
	h.height = height
}
