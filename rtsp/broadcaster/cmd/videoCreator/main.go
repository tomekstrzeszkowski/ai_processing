package main

import "strzcam.com/broadcaster/watcher"

func main() {
	memory, _ := watcher.NewSharedMemoryReceiver("video_frame")
	converter, _ := watcher.NewConverter(watcher.SavePath)
	creator, _ := watcher.NewVideoCreator(memory, converter)
	defer creator.Close()
	go creator.StartWatchingFrames()
	go creator.SaveFramesForLater()
	go creator.StartConversionWorkflow()

	server, _ := watcher.NewServer(8072)
	server.PrepareEndpoints()
	go func() {
		for frame := range creator.SharedMemoryReceiver.Frames {
			server.BroadcastFrame(frame)
		}
	}()
	server.Start()
}
