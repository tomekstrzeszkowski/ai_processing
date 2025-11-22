package watcher

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Converter struct {
	savePath     string
	watcher      *fsnotify.Watcher
	hasJob       bool
	watchingDirs []string
	mux          sync.RWMutex
	Framerate    *float64
}

func NewConverter(saveVideoPath string) (*Converter, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	frameRate := 30.0
	c := &Converter{
		savePath:     saveVideoPath,
		watcher:      watcher,
		hasJob:       false,
		watchingDirs: []string{saveVideoPath},
		Framerate:    &frameRate,
	}
	c.AddToWatch(saveVideoPath)
	dateDirs, _ := GetDateDirNames(saveVideoPath, []string{})
	fmt.Printf("Watching directories: %v\n", dateDirs)
	for _, dateDir := range dateDirs {
		c.AddToWatch(filepath.Join(saveVideoPath, dateDir))
	}
	return c, nil
}
func (c *Converter) Watch() {
	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				path := strings.Split(event.Name, "/")
				if len(path[len(path)-1]) == 10 {
					c.watcher.Add(event.Name)
				} else {
					if !c.hasJob {
						skipDates := c.GetSkipDates()
						for {
							RemoveOldestDirs(c.savePath, skipDates)
							RemoveOldestVideoFiles(c.savePath, skipDates)
							c.hasJob = c.convertLastChunkToVideo(c.savePath)
							if !c.hasJob {
								break
							}
						}
					}
				}
			}

		case _, ok := <-c.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
func (c *Converter) Close() {
	if c.watcher != nil {
		c.watcher.Close()
	}
}
func (c *Converter) AddToWatch(path string) {
	c.watchingDirs = append(c.watchingDirs, path)
	c.watcher.Add(path)
}
func (c *Converter) GetSkipDates() []string {
	now := time.Now()
	skipDates := []string{now.Format("2006-01-02")}
	for i := 1; i <= ConvertFramesBeforeDays; i++ {
		pastDate := now.AddDate(0, 0, -i) // Subtract i days
		skipDates = append(skipDates, pastDate.Format("2006-01-02"))
	}
	return skipDates
}

func RemoveOldestDirs(savePath string, skipDirs []string) {
	for RemoveOldestDir(savePath, skipDirs) {
	}
}

func RemoveOldestVideoFiles(savePath string, skipDates []string) {
	for RemoveOldestVideo(savePath, []string{".mp4"}, skipDates) {
	}
}

func (c *Converter) convert(chunkPath string) error {
	patches := strings.Split(chunkPath, "/")
	inputPattern := filepath.Join(chunkPath, "frame%d.jpg")
	dateDirName, chunkDirName := patches[len(patches)-2], patches[len(patches)-1]
	fmt.Printf("[FPS:%f] Converting frames in %s %v\n", *c.Framerate, dateDirName, patches)
	outputPath := filepath.Join(append(patches[:len(patches)-2], fmt.Sprintf("%s-%s.mp4", dateDirName, chunkDirName))...)

	args := []string{
		"-framerate", fmt.Sprintf("%d", int(math.Ceil(*c.Framerate))),
		"-i", inputPattern,
		"-vf", "scale=1900:1068,fps=fps=30:round=up",
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-profile:v", "baseline",
		"-level", "3.1",
		"-bf", "0",
		"-g", "30", // GOP size = framerate for better seeking
		"-keyint_min", "30", // Minimum GOP size
		"-sc_threshold", "0",
		"-f", "h264",
		outputPath,
	}
	var stderr bytes.Buffer
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %w", err)
	}
	duration := parseDurationFromFFmpegOutput(stderr.String())
	fmt.Printf("FFmpeg conversion succeeded: %s (%.2f seconds)\n", outputPath, duration)
	return nil
}
func parseDurationFromFFmpegOutput(output string) float64 {
	re := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2}\.\d{2})`)
	matches := re.FindStringSubmatch(output)
	if len(matches) == 4 {
		hours, _ := strconv.ParseFloat(matches[1], 64)
		minutes, _ := strconv.ParseFloat(matches[2], 64)
		seconds, _ := strconv.ParseFloat(matches[3], 64)
		return hours*3600 + minutes*60 + seconds
	}
	return 0
}
func (c *Converter) convertLastChunkToVideo(savePath string) bool {
	dirCount := CountChunksInDateDir(savePath, []string{})
	fmt.Printf("Number of chunks in date dir: %d\n", dirCount)
	chunkPath := GetOldestChunkInDateDir(savePath, []string{})
	if chunkPath == "" {
		fmt.Println("No chunk found to convert.")
		return false
	}
	fmt.Printf("Converting last chunk: %s\n", chunkPath)
	parts := strings.Split(chunkPath, "/")
	dateDir := parts[len(parts)-2]
	now := time.Now()
	fmt.Printf("dir count %d, date dir %s, now %s\n", dirCount, dateDir, now.Format("2006-01-02"))
	if dirCount < 2 && dateDir == now.Format("2006-01-02") {
		// For now just skip last chunk, idea for changing this is to save size and
		// convert all frames, then create a new video from it. Then concatenate
		// the videos together with adding size. When the size is close to the limit
		// just create a new chunk and remove the old one.
		// This will allow to convert the last chunk and not wait for the next one.
		fmt.Println("There is only one chunk that can be busy.")
		return false
	}
	c.convert(chunkPath)
	err := os.RemoveAll(chunkPath)
	if err != nil {
		// another gorouting is writing file to the channel
		fmt.Printf("Error removing chunk directory: %v\n", err)
		err := os.RemoveAll(chunkPath)
		fmt.Printf("Re-try removing chunk directory: %v\n", err)
	}
	return true
}

func (c *Converter) RunUntilComplete() {
	skipDates := c.GetSkipDates()
	c.mux.Lock()
	defer c.mux.Unlock()
	for {
		RemoveOldestDirs(c.savePath, skipDates)
		RemoveOldestVideoFiles(c.savePath, skipDates)
		c.hasJob = c.convertLastChunkToVideo(c.savePath)
		if !c.hasJob {
			break
		}
	}
}
