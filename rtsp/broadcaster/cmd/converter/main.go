package main

import (
	"time"

	"strzcam.com/broadcaster/watcher"
)

func main() {
	now := time.Now()
	skipDates := []string{now.Format("2006-01-02")}
	for i := 1; i <= watcher.ConvertFramesBeforeDays; i++ {
		pastDate := now.AddDate(0, 0, -i) // Subtract i days
		skipDates = append(skipDates, pastDate.Format("2006-01-02"))
	}
	// TODO: change it to file watcher
	for {
		watcher.RemoveOldestDirs(watcher.SavePath, skipDates)
		watcher.RemoveOldestVideoFiles(watcher.SavePath, skipDates)
		watcher.ConvertLastChunkToVideo(watcher.SavePath, skipDates)
	}
}
