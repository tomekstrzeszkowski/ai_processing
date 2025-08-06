package watcher

const SavePath = "./saved"

// Persisting frames
const ConvertFramesBeforeDays = 1

const saveChunkSize = 5 * 1024 * 1024               //1024 * 1024 * 1024 // 1GB
const convertedVideoSpace = 10 * 1024 * 1024 * 1024 //10 *1024 * 1024 * 1024 // 10GB

const saveDirMaxSize = 100 * saveChunkSize

const showWhatWasBefore = 90000 // 30 FPS * 60 seconds * 5 minutes = 90000 frames
const showWhatWasAfter = 90000  // 30 FPS * 60 seconds * 5 minutes = 90000 frames
