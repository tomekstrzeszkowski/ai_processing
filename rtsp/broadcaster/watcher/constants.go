package watcher

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const SavePath = "./saved"

type Config struct { // Sizes in GB
	ConvertFramesBeforeDays int
	SaveChunkSize           int
	ConvertedVideoSpace     int
	SaveDirMaxSize          int
	ShowWhatWasBefore       int
	ShowWhatWasAfter        int
}

func NewConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
	saveChunkSize := getEnvAsInt("SAVE_CHUNK_SIZE", 1024*1024*1024)
	return Config{
		ConvertFramesBeforeDays: getEnvAsInt("CONVERT_FRAMES_BEFORE_DAYS", 1),
		SaveChunkSize:           saveChunkSize,
		ConvertedVideoSpace:     getEnvAsInt("CONVERTED_VIDEO_SPACE", saveChunkSize*10),
		SaveDirMaxSize:          getEnvAsInt("SAVE_DIR_MAX_SIZE", saveChunkSize*100),
		ShowWhatWasBefore:       getEnvAsInt("CONVERT_FRAMES_BEFORE_DAYS", 30*60*1), // FPS * seconds * minutes
		ShowWhatWasAfter:        getEnvAsInt("CONVERT_FRAMES_BEFORE_DAYS", 30*60*1),
	}
}

func getEnvAsInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultValue
}
