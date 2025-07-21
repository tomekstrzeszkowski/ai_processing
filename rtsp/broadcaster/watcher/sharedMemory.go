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
	Data     []byte
	Detected int
	Before   *CircularBuffer
}

type SharedMemoryReceiver struct {
	shmPath           string
	watcher           *fsnotify.Watcher
	Frames            chan []byte
	SignificantFrames chan SignificantFrame
}

func NewSharedMemoryReceiver(shmName string) (*SharedMemoryReceiver, error) {
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

	before := NewCircularBuffer(90000) // 30 FPS * 60 seconds * 5 minutes = 90000 frames
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
					sf := SignificantFrame{Data: frameSignificant, Detected: detected, Before: before}
					select {
					case smr.SignificantFrames <- sf:
					default:
						log.Printf("Significant frame channel is full, dropping frame and so sorry")
					}

				} else {
					before.Add(frameData)
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
		log.Printf("Saving significant frame: %d bytes, detected %d", len(detectedFrame.Data), detectedFrame.Detected)
		log.Printf("Frames before: %d", detectedFrame.Before.Size())
		detectedFrame.Before.Clear()
	}
}
