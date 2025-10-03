package main

import (
	"fmt"

	"strzcam.com/broadcaster/watcher"
)

func main() {
	memory, _ := watcher.NewSharedMemoryReceiver("video_frame")
	converter, _ := watcher.NewConverter(fmt.Sprintf("%s_video_frame", watcher.SavePath))
	creator, _ := watcher.NewVideoCreator(memory, converter)
	defer creator.Close()
	go creator.StartWatchingFrames()
	go creator.SaveFramesForLater()
	go creator.StartConversionWorkflow(&memory.ActualFps)

	server, _ := watcher.NewServer(7072)
	server.PrepareEndpoints()
	go func() {
		for frame := range creator.SharedMemoryReceiver.Frames {
			server.BroadcastFrame(frame)
		}
	}()
	server.Start()
}
