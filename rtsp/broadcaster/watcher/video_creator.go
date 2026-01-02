package watcher

import "time"

type VideoCreator struct {
	Converter            *Converter
	SharedMemoryReceiver *SharedMemoryReceiver
}

func NewVideoCreator(
	sharedMemoryReceiver *SharedMemoryReceiver,
	converter *Converter,
) (*VideoCreator, error) {
	return &VideoCreator{
		Converter:            converter,
		SharedMemoryReceiver: sharedMemoryReceiver,
	}, nil
}
func (v *VideoCreator) StartWatchingFrames() {
	v.SharedMemoryReceiver.WatchSharedMemory()
}
func (v *VideoCreator) SaveFramesForLater() {
	v.SharedMemoryReceiver.SaveFrameForLater()
}
func (v *VideoCreator) StartConversionWorkflow(actualFps *float64, width *uint32, height *uint32) {
	v.Converter.Framerate = actualFps
	v.Converter.Width = width
	v.Converter.Height = height
	v.Converter.RunUntilComplete()
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if !v.Converter.hasJob {
				v.Converter.RunUntilComplete()
			}
		}
	}()
	v.Converter.Watch()
}

func (v *VideoCreator) Close() {
	v.Converter.Close()
	v.SharedMemoryReceiver.Close()
}
