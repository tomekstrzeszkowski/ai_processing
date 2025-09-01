package video

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"syscall"
	"time"

	"bufio"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
)

type Video struct {
	Name string
	Size int64
}

func GetVideoByPath(path string) ([]byte, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return data, nil
}

func GetVideoByDateRange(path string, start time.Time, end time.Time) ([]Video, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}
	var videoList []Video
	pattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(\d+)\.mp4$`)

	err := filepath.Walk(path, func(pathWalk string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing file %s: %v", pathWalk, err)
		}
		if info.IsDir() {
			return nil
		}
		fileName := info.Name()
		matches := pattern.FindStringSubmatch(fileName)
		if matches == nil {
			return nil // Skip files that don't match pattern
		}
		dateStr := matches[1] // YYYY-MM-DD part
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil // Skip files with invalid date format
		}
		if (fileDate.Equal(start) || fileDate.After(start)) &&
			(fileDate.Equal(end) || fileDate.Before(end)) {
			stat, _ := info.Sys().(*syscall.Stat_t)
			videoList = append(videoList, Video{
				Name: fileName,
				Size: int64(stat.Blocks) * 512,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by date (newest first), then by filename for same dates
	sort.Slice(videoList, func(i, j int) bool {
		matchA := pattern.FindStringSubmatch(videoList[i].Name)[1]
		matchB := pattern.FindStringSubmatch(videoList[j].Name)[1]
		dateA, _ := time.Parse("2006-01-02", matchA)
		dateB, _ := time.Parse("2006-01-02", matchB)
		if dateA.Equal(dateB) {
			// For same date, sort by filename (which includes the number suffix)
			return videoList[i].Name > videoList[j].Name
		}
		return dateA.After(dateB)
	})

	return videoList, nil
}

// Stream file directly from disk to network without loading into memory
func StreamFileToNetwork(stream network.Stream, filePath string) error {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	sizeHeader := fmt.Sprintf("%d\n", fileInfo.Size())
	if _, err := stream.Write([]byte(sizeHeader)); err != nil {
		return fmt.Errorf("failed to write size header: %w", err)
	}

	// Stream file in chunks
	const chunkSize = 64 * 1024 // 64KB chunks
	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read file chunk: %w", err)
		}

		if _, err := stream.Write(buffer[:n]); err != nil {
			return fmt.Errorf("failed to write chunk to stream: %w", err)
		}
	}

	return nil
}

// Client side: receiving large files
func ReceiveVideoFile(stream network.Stream, outputPath string) error {
	defer stream.Close()

	// Read file size header
	buf := bufio.NewReader(stream)
	sizeStr, err := buf.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read size header: %w", err)
	}

	fileSize, err := strconv.ParseInt(strings.TrimSpace(sizeStr), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid size header: %w", err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy data from stream to file
	written, err := io.CopyN(outFile, buf, fileSize)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	if written != fileSize {
		return fmt.Errorf("incomplete transfer: got %d bytes, expected %d", written, fileSize)
	}

	return nil
}

func ValidateFilename(filename string) error {
	cleaned := filepath.Clean(filename)
	if strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return fmt.Errorf("invalid filename: path traversal detected")
	}
	if cleaned == "" || cleaned == "." {
		return fmt.Errorf("invalid filename: empty or current directory")
	}

	return nil
}

// Optional: Progress tracking for large files
func StreamFileWithProgress(stream network.Stream, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := fileInfo.Size()

	// Send size header
	sizeHeader := fmt.Sprintf("%d\n", fileSize)
	if _, err := stream.Write([]byte(sizeHeader)); err != nil {
		return fmt.Errorf("failed to write size header: %w", err)
	}

	// Progress tracking
	var totalSent int64
	const chunkSize = 64 * 1024
	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read file chunk: %w", err)
		}

		if _, err := stream.Write(buffer[:n]); err != nil {
			return fmt.Errorf("failed to write chunk to stream: %w", err)
		}

		totalSent += int64(n)

		// Log progress (every 100MB for large files)
		if totalSent%(100*1024*1024) == 0 || totalSent == fileSize {
			progress := float64(totalSent) / float64(fileSize) * 100
			log.Printf("Progress: %.1f%% (%d/%d bytes)", progress, totalSent, fileSize)
		}
	}

	return nil
}

func ConvertAndGetVideoForWeb(filePath string) ([]byte, error) {
	tempFile, err := os.CreateTemp("", "temp-*.mp4")
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %v", err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	cmd := exec.Command("ffmpeg",
		"-y",         // Force overwrite without asking
		"-f", "h264", // Force input format to H264
		"-i", filePath, // Input file
		"-c:v", "copy", // Copy video stream without re-encoding
		"-f", "mp4", // Force MP4 container
		"-movflags", "+faststart", // Enable fast start for streaming
		tempFile.Name())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg output: %s", stderr.String())
		return nil, fmt.Errorf("error converting video: %v\nFFmpeg error: %s", err, stderr.String())
	}

	// Read the entire file into memory
	videoBytes, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("error reading converted file: %v", err)
	}

	return videoBytes, nil
}
