package main

import "strzcam.com/broadcaster/watcher"

func main() {
	memory, _ := watcher.NewSharedMemoryReceiver("video_frame")
	defer memory.Close()
	go memory.WatchSharedMemory()
	server, _ := watcher.NewServer()

	server.PrepareEndpoints()
	go func() {
		for frame := range memory.Frames {
			server.BroadcastFrame(frame)
		}
	}()
	server.Start(":8080")
}
