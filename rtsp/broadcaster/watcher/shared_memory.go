package watcher

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var possibleDetections = []int{0, 1, 2, 3, 4}

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
}

func NewSharedMemoryReceiver(shmName string) (*SharedMemoryReceiver, error) {
	if err := os.MkdirAll("./saved", 0755); err != nil {
		panic(fmt.Sprintf("Cannot create directory: %v", err))
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	receiver := &SharedMemoryReceiver{
		shmPath:           filepath.Join("/dev/shm", shmName),
		watcher:           watcher,
		Frames:            make(chan []byte),
		SignificantFrames: make(chan SignificantFrame),
	}

	// Watch the shared memory directory
	err = watcher.Add("/dev/shm")
	if err != nil {
		return nil, err
	}
	go receiver.SaveFrameForLater()

	return receiver, nil
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
func (smr *SharedMemoryReceiver) WatchSharedMemory() {
	log.Println("Starting shared memory watcher...")

	before := NewCircularBuffer(2) // 30 FPS * 60 seconds * 5 minutes = 90000 frames
	var after *CircularBuffer
	for {
		select {
		case event, ok := <-smr.watcher.Events:
			if !ok {
				return
			}

			// Check if it's our target file and it was written to
			if strings.HasPrefix(event.Name, smr.shmPath) &&
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
					sf := SignificantFrame{Data: &frameSignificant, Detected: detected, Before: before, After: after}
					select {
					case smr.SignificantFrames <- sf:
					default:
						log.Printf("Significant frame channel is full, dropping frame and so sorry")
					}
					after = NewCircularBuffer(2)
				} else if after != nil {
					after.Add(frameData)
				} else {
					before.Add(frameData)
				}
				if after != nil && after.IsFull() {
					sf := SignificantFrame{Data: nil, Detected: -1, Before: before, After: after}
					select {
					case smr.SignificantFrames <- sf:
					default:
						log.Printf("After buffer is full, dropping frame and so sorry")
					}
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
		path := fmt.Sprintf("./saved/%d-%02d-%02d", year, month, day)
		i, path := TouchDirAndGetIterator(path, 10*1024*1024) // 10MB limit
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
