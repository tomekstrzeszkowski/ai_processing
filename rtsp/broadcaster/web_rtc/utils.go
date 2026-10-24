package web_rtc

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/abema/go-mp4"
)

type FFProbeOutput struct {
	Streams []struct {
		RFrameRate string `json:"r_frame_rate"`
	} `json:"streams"`
}

func GetFrameDuration(filePath string) (time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "v:0",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe FFProbeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return 0, err
	}

	if len(probe.Streams) == 0 {
		return 0, fmt.Errorf("no video stream found")
	}

	// Parse "30/1" format
	var num, den int
	fmt.Sscanf(probe.Streams[0].RFrameRate, "%d/%d", &num, &den)
	fps := float64(num) / float64(den)

	return time.Duration(float64(time.Second) / fps), nil
}

// ensureMP4Container checks if the video file is a valid MP4 container.
// If it's a raw H.264 stream, it converts it using ffmpeg.
// Returns the path to a valid MP4 file.
func ensureMP4Container(inputPath string) (string, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}

	// Check if it's already a valid MP4 by trying to probe it
	_, probeErr := mp4.Probe(file)
	file.Close()

	if probeErr == nil {
		// Already a valid MP4
		log.Printf("File %s is already a valid MP4 container", inputPath)
		return inputPath, nil
	}

	log.Printf("File %s is not a valid MP4 container, converting with ffmpeg...", inputPath)

	// Create output path with _converted suffix
	ext := filepath.Ext(inputPath)
	base := inputPath[:len(inputPath)-len(ext)]
	outputPath := base + "_converted" + ext

	// Run ffmpeg to convert raw H.264 to MP4 container
	cmd := exec.Command("ffmpeg",
		"-y",         // Overwrite output
		"-f", "h264", // Input format is raw H.264
		"-i", inputPath, // Input file
		"-c", "copy", // Copy codec (no re-encoding)
		outputPath, // Output file
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg conversion failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("Successfully converted to %s", outputPath)
	return outputPath, nil
}

// calculateSampleOffsets computes the file offset for each sample based on chunk information
func calculateSampleOffsets(track *mp4.Track) []uint64 {
	offsets := make([]uint64, len(track.Samples))

	sampleIdx := 0
	for _, chunk := range track.Chunks {
		chunkOffset := chunk.DataOffset
		for i := uint32(0); i < chunk.SamplesPerChunk && sampleIdx < len(track.Samples); i++ {
			offsets[sampleIdx] = chunkOffset
			chunkOffset += uint64(track.Samples[sampleIdx].Size)
			sampleIdx++
		}
	}

	return offsets
}
