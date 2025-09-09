package watcher

const SavePath = "./saved"

// Persisting frames
const ConvertFramesBeforeDays = 1

const saveChunkSize = 1024 * 1024 * 1024            // 1GB
const convertedVideoSpace = 10 * 1024 * 1024 * 1024 // 10GB

const saveDirMaxSize = 100 * saveChunkSize

const showWhatWasBefore = 100 // ~4 FPS * 60 seconds * 1 minutes = 240 frames
const showWhatWasAfter = 100  // ~4 FPS * 60 seconds * 1 minutes = 240 frames
