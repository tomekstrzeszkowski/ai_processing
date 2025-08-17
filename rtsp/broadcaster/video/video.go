package video

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"syscall"
	"time"
)

type Video struct {
	Name string
	Size int64
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
