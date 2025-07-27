package watcher

const SavePath = "./saved"

const ConvertFramesBeforeDays = 2

const saveChunkSize = 1024 * 1024 * 1024 // 1GB
// const convertedVideoSpace = 1024 * 1024 * 1024 // 1GB
const convertedVideoSpace = 10 * 1024 * 1024

// const saveChunkSize = 9 * 1024 * 1024 // 9MB
const saveDirMaxSize = 100*saveChunkSize + convertedVideoSpace
