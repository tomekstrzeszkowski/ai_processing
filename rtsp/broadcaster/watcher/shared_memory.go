package watcher

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"strzcam.com/broadcaster/frame"
)

type ConfigProvider interface {
	GetSavePath() string
	GetShowWhatWasBefore() int
	GetShowWhatWasAfter() int
	GetSaveChunkSize() int
}

type DefaultConfigProvider struct {
	config Config
}

func NewDefaultConfigProvider() DefaultConfigProvider {
	return DefaultConfigProvider{config: NewConfig()}
}

func (d DefaultConfigProvider) GetSavePath() string {
	return SavePath
}
func (d DefaultConfigProvider) GetShowWhatWasBefore() int {
	return d.config.ShowWhatWasBefore
}
func (d DefaultConfigProvider) GetShowWhatWasAfter() int {
	return d.config.ShowWhatWasAfter
}
func (d DefaultConfigProvider) GetSaveChunkSize() int {
	return d.config.SaveChunkSize
}

type SignificantFrame struct {
	Frame  frame.Frame
	Before *CircularBuffer
}
type SharedMemoryReceiver struct {
	shmPath           string
	watcher           *fsnotify.Watcher
	Frames            chan frame.Frame
	SignificantFrames chan SignificantFrame
	configProvider    ConfigProvider
	savePath          string
	ActualFps         float64
	FrameWidth        uint32
	FrameHeight       uint32
}

func NewSharedMemoryReceiverWithConfig(shmName string, configProvider ConfigProvider) (*SharedMemoryReceiver, error) {
	saveFramePath := fmt.Sprintf("%s_video_frame", configProvider.GetSavePath())
	if err := os.MkdirAll(saveFramePath, 0755); err != nil {
		panic(fmt.Sprintf("Cannot create directory: %v", err))
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	receiver := &SharedMemoryReceiver{
		shmPath:           filepath.Join("/dev/shm", shmName),
		watcher:           watcher,
		Frames:            make(chan frame.Frame, 10),
		SignificantFrames: make(chan SignificantFrame, 100),
		configProvider:    configProvider,
		savePath:          saveFramePath,
		ActualFps:         30,
		FrameWidth:        0,
		FrameHeight:       0,
	}
	err = watcher.Add("/dev/shm")
	if err != nil {
		return nil, err
	}

	return receiver, nil
}

func NewSharedMemoryReceiver(shmName string) (*SharedMemoryReceiver, error) {
	return NewSharedMemoryReceiverWithConfig(shmName, NewDefaultConfigProvider())
}

func (smr *SharedMemoryReceiver) ReadFrameFromShm() (frame.Frame, error) {
	// Check if file exists
	detected := -1
	if _, err := os.Stat(smr.shmPath); os.IsNotExist(err) {
		return frame.Frame{Detected: detected}, fmt.Errorf("no valid shared memory file found")
	}
	data, err := os.ReadFile(smr.shmPath)
	if err != nil {
		return frame.Frame{Detected: detected}, err
	}
	return frame.Frame{
		Data:     data[9:],
		Width:    binary.LittleEndian.Uint32(data[1:5]),
		Height:   binary.LittleEndian.Uint32(data[5:9]),
		Detected: int(int8(data[0])),
	}, nil
}
func (smr *SharedMemoryReceiver) SendSignificantFrame(sf SignificantFrame) {
	select {
	case smr.SignificantFrames <- sf:
	case <-time.After(500 * time.Millisecond):
		//log.Printf("Timeout sending significant frame")
	}
}
func (smr *SharedMemoryReceiver) logStats(frame frame.Frame, before *CircularBuffer, after int) {
	beforeSize := 0
	if before != nil {
		beforeSize = before.Size()
	}
	log.Printf(
		"[FPS %f] New frame %dx%d received: %d bytes, that was %d, before %d, after %d",
		frame.Fps,
		frame.Width,
		frame.Height,
		len(frame.Data),
		frame.Detected,
		beforeSize,
		after,
	)
}
func (smr *SharedMemoryReceiver) GetBaseDir() string {
	year, month, day := time.Now().Date()
	return fmt.Sprintf("%s/%d-%02d-%02d", smr.savePath, year, month, day)
}
func (smr *SharedMemoryReceiver) WatchSharedMemory(saveForLater bool) {
	log.Println("Starting shared memory watcher...")
	showWhatWasAfter := smr.configProvider.GetShowWhatWasAfter()
	var before *CircularBuffer
	if saveForLater {
		before = NewCircularBuffer(smr.configProvider.GetShowWhatWasBefore())
	}
	after := 0
	var lastFrameData []byte
	startTime := time.Now()
	frameCount := 0
	for {
		select {
		case event, ok := <-smr.watcher.Events:
			if !ok {
				return
			}
			if event.Name == smr.shmPath &&
				(event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {

				frame, err := smr.ReadFrameFromShm()
				if err != nil {
					log.Printf("Error reading frame from shared memory: %v", err)
					continue
				}
				// skip the same event triggered twice
				if bytes.Equal(frame.Data, lastFrameData) {
					continue
				}
				lastFrameData = frame.Data
				elapsedTime := time.Since(startTime)
				frameCount++
				if elapsedTime > time.Second {
					smr.ActualFps = float64(frameCount) / elapsedTime.Seconds()
					frameCount = 0
					startTime = time.Now()
				}
				frame.Fps = smr.ActualFps
				smr.FrameHeight = frame.Height
				smr.FrameWidth = frame.Width
				smr.Frames <- frame
				if saveForLater && frame.Detected != -1 {
					sf := SignificantFrame{
						Frame:  frame,
						Before: before,
					}
					go smr.SendSignificantFrame(sf)
					after = showWhatWasAfter + 1
				} else if saveForLater && after-1 <= 0 {
					before.Add(frame.Data)
				}
				if after != 0 {
					after--
					if frame.Detected == -1 {
						sf := SignificantFrame{Frame: frame, Before: nil}
						go smr.SendSignificantFrame(sf)
					}
					if after == 0 {
						CreateNewDirIndex(smr.GetBaseDir())
					}
				}
				smr.logStats(frame, before, after)
			}

		case err, ok := <-smr.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
func (smr *SharedMemoryReceiver) Close() {
	if smr.watcher != nil {
		smr.watcher.Close()
	}
}
func (smr *SharedMemoryReceiver) SaveFrameForLater() {
	for detectedFrame := range smr.SignificantFrames {
		i, path, err := TouchDirAndGetIndex(smr.GetBaseDir(), int64(smr.configProvider.GetSaveChunkSize()))
		if err != nil {
			log.Printf("Can not save frame for later! %v", err)
			return
		}
		if detectedFrame.Before != nil {
			for _, frameBefore := range detectedFrame.Before.GetAll() {
				SaveFrame(i, frameBefore, path)
				i += 1
			}
			detectedFrame.Before.Clear()
		}
		SaveFrame(i, detectedFrame.Frame.Data, path)
		i += 1
		if !IsMetadataExists(path) {
			SaveMetadata(detectedFrame.Frame.Width, detectedFrame.Frame.Height, path)
		}
	}
}
