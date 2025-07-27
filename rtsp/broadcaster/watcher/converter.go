package watcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RemoveOldestDirs(savePath string, skipDirs []string) {
	for RemoveOldestDir(savePath, skipDirs) {
	}
}

func RemoveOldestVideoFiles(savePath string, skipDates []string) {
	for RemoveOldestVideo(savePath, []string{".mp4"}, skipDates) {
	}
}

func Convert(chunkPath string) error {
	patches := strings.Split(chunkPath, "/")
	inputPattern := filepath.Join(chunkPath, "frame%d.jpg")
	dateDirName, chunkDirName := patches[len(patches)-2], patches[len(patches)-1]
	fmt.Printf("Converting frames in %s %v to video...\n", dateDirName, patches)
	outputPath := filepath.Join(append(patches[:len(patches)-2], fmt.Sprintf("%s-%s.mp4", dateDirName, chunkDirName))...)
	fmt.Println("Output path:", outputPath)

	// FFmpeg command arguments
	args := []string{
		"-framerate", "30",
		"-i", inputPattern,
		"-vf", "scale=1900:1068",
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-profile:v", "baseline",
		"-level", "3.1",
		"-bf", "0",
		"-f", "h264",
		outputPath,
	}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	fmt.Printf("Successfully converted frames to %s\n", outputPath)
	return nil
}

func ConvertLastChunkToVideo(savePath string, skipDirs []string) {
	chunkPath := GetOldestChunkInDateDir(savePath, skipDirs)
	if chunkPath == "" {
		fmt.Println("No chunk found to convert.")
		return
	}
	Convert(chunkPath)
	os.RemoveAll(chunkPath)
}
