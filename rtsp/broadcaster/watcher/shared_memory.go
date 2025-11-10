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
)

type ConfigProvider interface {
	GetSavePath() string
	GetShowWhatWasBefore() int
	GetShowWhatWasAfter() int
}

type DefaultConfigProvider struct{}

func (d DefaultConfigProvider) GetSavePath() string {
	return SavePath
}
func (d DefaultConfigProvider) GetShowWhatWasBefore() int {
	return showWhatWasBefore
}
func (d DefaultConfigProvider) GetShowWhatWasAfter() int {
	return showWhatWasAfter
}

type SignificantFrame struct {
	Data     []byte
	Detected int
	Before   *CircularBuffer
}

type SharedMemoryReceiver struct {
	shmPath           string
	watcher           *fsnotify.Watcher
	Frames            chan []byte
	SignificantFrames chan SignificantFrame
	configProvider    ConfigProvider
	savePath          string
	ActualFps         float64
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
		Frames:            make(chan []byte, 10),
		SignificantFrames: make(chan SignificantFrame, 100),
		configProvider:    configProvider,
		savePath:          saveFramePath,
		ActualFps:         30,
	}

	// Watch the shared memory directory
	err = watcher.Add("/dev/shm")
	if err != nil {
		return nil, err
	}

	return receiver, nil
}

func NewSharedMemoryReceiver(shmName string) (*SharedMemoryReceiver, error) {
	return NewSharedMemoryReceiverWithConfig(shmName, DefaultConfigProvider{})
}

func (smr *SharedMemoryReceiver) ReadFrameFromShm() ([]byte, int, error) {
	// Check if file exists
	detected := -1
	if _, err := os.Stat(smr.shmPath); os.IsNotExist(err) {
		return nil, detected, fmt.Errorf("no valid shared memory file found")
	}

	// Read the entire file
	data, err := os.ReadFile(smr.shmPath)
	if err != nil {
		return nil, detected, err
	}

	if len(data) < 5 {
		return nil, detected, fmt.Errorf("invalid frame data: too short")
	}
	detected = int(int8(data[0]))
	dataLength := binary.LittleEndian.Uint32(data[1:5])
	frameData := data[5 : 5+dataLength]
	return frameData, detected, nil
}
func (smr *SharedMemoryReceiver) SendSignificantFrame(sf SignificantFrame) {
	select {
	case smr.SignificantFrames <- sf:
	case <-time.After(500 * time.Millisecond):
		log.Printf("Timeout sending significant frame")
	}
}
func (smr *SharedMemoryReceiver) logStats(actualFps float64, frameLength int, detected int, beforeSize int, after int) {
	log.Printf(
		"[FPS %f] New frame received: %d bytes, that was %d, before %d, after %d",
		actualFps,
		frameLength,
		detected,
		beforeSize,
		after,
	)
}
func (smr *SharedMemoryReceiver) GetBaseDir() string {
	year, month, day := time.Now().Date()
	return fmt.Sprintf("%s/%d-%02d-%02d", smr.savePath, year, month, day)
}
func (smr *SharedMemoryReceiver) WatchSharedMemory() {
	log.Println("Starting shared memory watcher...")
	showWhatWasAfter := smr.configProvider.GetShowWhatWasAfter()
	showWhatWasBefore := smr.configProvider.GetShowWhatWasBefore()
	before := NewCircularBuffer(showWhatWasBefore)
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

			// Check if it's our target file and it was written to
			if event.Name == smr.shmPath &&
				(event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {

				frameData, detected, err := smr.ReadFrameFromShm()
				if err != nil {
					log.Printf("Error reading frame from shared memory: %v", err)
					continue
				}
				// skip the same event triggered twice
				if bytes.Equal(frameData, lastFrameData) {
					continue
				}
				lastFrameData = frameData
				elapsedTime := time.Since(startTime)
				frameCount++
				if elapsedTime > time.Second {
					smr.ActualFps = float64(frameCount) / elapsedTime.Seconds()
					frameCount = 0
					startTime = time.Now()
				}
				smr.logStats(smr.ActualFps, len(frameData), detected, before.Size(), after)
				smr.Frames <- frameData
				if detected != -1 {
					sf := SignificantFrame{
						Data: frameData, Detected: detected, Before: before,
					}
					go smr.SendSignificantFrame(sf)
					after = showWhatWasAfter + 1
				} else if after-1 <= 0 {
					before.Add(frameData)
				}
				if after != 0 {
					after--
					if detected == -1 {
						sf := SignificantFrame{Data: frameData, Detected: -1, Before: nil}
						go smr.SendSignificantFrame(sf)
					}
					if after == 0 {
						//create a new dir for next event
						CreateNewDirIndex(smr.GetBaseDir())
					}
				}
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
		i, path, err := TouchDirAndGetIndex(smr.GetBaseDir(), saveChunkSize)
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
		SaveFrame(i, detectedFrame.Data, path)
		i += 1
	}
}
