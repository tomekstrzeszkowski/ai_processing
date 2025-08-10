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
	Data     *[]byte
	Detected int
	Before   *CircularBuffer
}

type SharedMemoryReceiver struct {
	shmPath           string
	watcher           *fsnotify.Watcher
	Frames            chan []byte
	SignificantFrames chan SignificantFrame
	configProvider    ConfigProvider
}

func NewSharedMemoryReceiverWithConfig(shmName string, configProvider ConfigProvider) (*SharedMemoryReceiver, error) {
	if err := os.MkdirAll(configProvider.GetSavePath(), 0755); err != nil {
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

func (smr *SharedMemoryReceiver) readFrameFromShm() ([]byte, int, error) {
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
	case <-time.After(100 * time.Millisecond):
		log.Printf("Timeout sending significant frame")
	}
}
func (smr *SharedMemoryReceiver) WatchSharedMemory() {
	log.Println("Starting shared memory watcher...")
	showWhatWasAfter := smr.configProvider.GetShowWhatWasAfter()
	showWhatWasBefore := smr.configProvider.GetShowWhatWasBefore()
	before := NewCircularBuffer(showWhatWasBefore)
	after := 0
	var lastFrameData []byte
	for {
		select {
		case event, ok := <-smr.watcher.Events:
			if !ok {
				return
			}

			// Check if it's our target file and it was written to
			if event.Name == smr.shmPath &&
				(event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {

				frameData, detected, err := smr.readFrameFromShm()
				if err != nil {
					log.Printf("Error reading frame from shared memory: %v", err)
					continue
				}
				// skip the same event triggered twice
				if bytes.Equal(frameData, lastFrameData) {
					continue
				}
				lastFrameData = frameData

				log.Printf("New frame received: %d bytes, that was %d, before %d, after %d", len(frameData), detected, before.Size(), after)
				smr.Frames <- frameData
				if detected != -1 {
					frameSignificant := make([]byte, len(frameData))
					copy(frameSignificant, frameData)
					sf := SignificantFrame{
						Data: &frameSignificant, Detected: detected, Before: before,
					}
					go smr.SendSignificantFrame(sf)
					after = showWhatWasAfter + 1
				} else if after-1 <= 0 {
					before.Add(frameData)
				}
				if after != 0 {
					after--
					if detected == -1 {
						frameAfter := make([]byte, len(frameData))
						copy(frameAfter, frameData)
						sf := SignificantFrame{Data: &frameAfter, Detected: -1, Before: nil}
						go smr.SendSignificantFrame(sf)
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
		year, month, day := time.Now().Date()
		path := fmt.Sprintf("%s/%d-%02d-%02d", smr.configProvider.GetSavePath(), year, month, day)
		i, path := TouchDirAndGetIterator(path, saveChunkSize)
		if detectedFrame.Before != nil {
			for _, frameBefore := range detectedFrame.Before.GetAll() {
				SaveFrame(i, frameBefore, path)
				i += 1
			}
			detectedFrame.Before.Clear()
		}
		SaveFrame(i, *detectedFrame.Data, path)
		i += 1
	}
}
