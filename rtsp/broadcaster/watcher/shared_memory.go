package watcher

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type PathProvider interface {
	GetSavePath() string
}

type DefaultPathProvider struct{}

func (d DefaultPathProvider) GetSavePath() string {
	return SavePath
}

type SignificantFrame struct {
	Data     *[]byte
	Detected int
	Before   *CircularBuffer
	After    *CircularBuffer
}

type SharedMemoryReceiver struct {
	shmPath           string
	watcher           *fsnotify.Watcher
	Frames            chan []byte
	SignificantFrames chan SignificantFrame
	pathProvider      PathProvider
}

func NewSharedMemoryReceiverWithPath(shmName string, pathProvider PathProvider) (*SharedMemoryReceiver, error) {
	if err := os.MkdirAll(pathProvider.GetSavePath(), 0755); err != nil {
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
		pathProvider:      pathProvider,
	}

	// Watch the shared memory directory
	err = watcher.Add("/dev/shm")
	if err != nil {
		return nil, err
	}

	return receiver, nil
}

func NewSharedMemoryReceiver(shmName string) (*SharedMemoryReceiver, error) {
	return NewSharedMemoryReceiverWithPath(shmName, DefaultPathProvider{})
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

	before := NewCircularBuffer(showWhatWasBefore)
	var after *CircularBuffer
	for {
		select {
		case event, ok := <-smr.watcher.Events:
			if !ok {
				return
			}

			// Check if it's our target file and it was written to
			if event.Name == smr.shmPath &&
				(event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				// Small delay to ensure write is complete
				time.Sleep(1 * time.Millisecond)

				frameData, detected, err := smr.readFrameFromShm()
				if err != nil {
					log.Printf("Error reading frame from shared memory: %v", err)
					continue
				}

				log.Printf("New frame received: %d bytes, that was %d", len(frameData), detected)
				smr.Frames <- frameData
				if detected != -1 {
					frameSignificant := make([]byte, len(frameData))
					copy(frameSignificant, frameData)
					sf := SignificantFrame{
						Data: &frameSignificant, Detected: detected, Before: before, After: after,
					}
					go smr.SendSignificantFrame(sf)
					after = NewCircularBuffer(showWhatWasAfter)
				} else if after != nil {
					after.Add(frameData)
				} else {
					before.Add(frameData)
				}
				if after != nil && after.IsFull() {
					sf := SignificantFrame{Data: nil, Detected: -1, Before: before, After: after}
					go smr.SendSignificantFrame(sf)
					after = nil
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
		path := fmt.Sprintf("%s/%d-%02d-%02d", smr.pathProvider.GetSavePath(), year, month, day)
		i, path := TouchDirAndGetIterator(path, saveChunkSize)
		if detectedFrame.Data != nil {
			if detectedFrame.After != nil && detectedFrame.After.Size() > 0 {
				for _, frameAfter := range detectedFrame.After.GetAll() {
					SaveFrame(i, frameAfter, path)
					i += 1
				}
				//after will be created again after this method
				log.Printf("Frames before from previous detection: %d", detectedFrame.After.Size())
			} else {
				for _, frameBefore := range detectedFrame.Before.GetAll() {
					SaveFrame(i, frameBefore, path)
					i += 1
				}
			}
			SaveFrame(i, *detectedFrame.Data, path)
			i += 1
			detectedFrame.Before.Clear()
		} else {
			for _, frameAfter := range detectedFrame.After.GetAll() {
				SaveFrame(i, frameAfter, path)
				i += 1
			}
		}
	}
}
