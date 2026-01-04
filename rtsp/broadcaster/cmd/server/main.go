package main

import (
	frameUtils "strzcam.com/broadcaster/frame"
	"strzcam.com/broadcaster/watcher"
)

func main() {
	memory, _ := watcher.NewSharedMemoryReceiver("video_frame")
	defer memory.Close()
	go memory.WatchSharedMemory(true)
	go memory.SaveFrameForLater()
	server, _ := watcher.NewServer(7071)

	server.PrepareEndpoints()
	go func() {
		frames := []frameUtils.Frame{}
		for frame := range memory.Frames {
			frames = append(frames, frame)
		}
		server.BroadcastFrame(frames)
	}()
	server.Start()
}
