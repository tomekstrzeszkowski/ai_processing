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

type SharedMemoryReceiver struct {
	shmPath string
	watcher *fsnotify.Watcher
	Frames  chan []byte
}

func NewSharedMemoryReceiver(shmName string) (*SharedMemoryReceiver, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	receiver := &SharedMemoryReceiver{
		shmPath: filepath.Join("/dev/shm", shmName),
		watcher: watcher,
		Frames:  make(chan []byte),
	}

	// Watch the shared memory directory
	err = watcher.Add("/dev/shm")
	if err != nil {
		return nil, err
	}

	return receiver, nil
}

func (smr *SharedMemoryReceiver) readFrameFromShm() ([]byte, error) {
	// Check if file exists
	if _, err := os.Stat(smr.shmPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("shared memory file does not exist")
	}

	// Read the entire file
	data, err := os.ReadFile(smr.shmPath)
	if err != nil {
		return nil, err
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("invalid frame data: too short")
	}

	// Read frame size from header
	frameSize := binary.BigEndian.Uint32(data[:4])

	if len(data) < int(4+frameSize) {
		return nil, fmt.Errorf("invalid frame data: incomplete")
	}

	// Extract frame data
	frameData := data[4 : 4+frameSize]
	return frameData, nil
}
func (smr *SharedMemoryReceiver) WatchSharedMemory() {
	log.Println("Starting shared memory watcher...")

	for {
		select {
		case event, ok := <-smr.watcher.Events:
			if !ok {
				return
			}

			// Check if it's our target file and it was written to
			if event.Name == smr.shmPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				// Small delay to ensure write is complete
				time.Sleep(1 * time.Millisecond)

				frameData, err := smr.readFrameFromShm()
				if err != nil {
					log.Printf("Error reading frame from shared memory: %v", err)
					continue
				}

				log.Printf("New frame received: %d bytes", len(frameData))
				smr.Frames <- frameData
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
