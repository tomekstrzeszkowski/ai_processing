package watcher

import (
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"time"
)

type TestPathProvider struct {
	path string
}

func (tp TestPathProvider) GetSavePath() string {
	return tp.path
}

func createFrame(buffer []byte, detected byte, shmName string) {
	header := make([]byte, 5)
	header[0] = detected
	binary.LittleEndian.PutUint32(header[1:], uint32(len(buffer)))
	file, err := os.Create("/dev/shm/" + shmName)
	if err != nil {
		panic("Failed to create shared memory file: " + err.Error())
	}
	defer file.Close()
	file.Write(header)
	file.Write(buffer)
	file.Sync()
}

func TestSharedMemory(t *testing.T) {
	tempPath := t.TempDir()
	PathProvider := TestPathProvider{path: tempPath}
	t.Run("Test shared memory to read", func(t *testing.T) {
		receiver, err := NewSharedMemoryReceiverWithPath("test_non_existent_shm", PathProvider)
		defer receiver.Close()
		if err != nil {
			t.Fatal("Failed to create SharedMemoryReceiver:", err)
		}
		_, i, err := receiver.readFrameFromShm()
		if i != -1 {
			t.Error("Expected index -1 when no shared memory file exists, got:", i)
		}
		if err == nil {
			t.Error("Expected an error when reading from non-existent shared memory file, got nil")
		}
		if err.Error() != "no valid shared memory file found" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})
	t.Run("DetectedFrame", func(t *testing.T) {
		data := []byte("test data")
		createFrame(data, 0, "test_shm")
		defer os.Remove("/dev/shm/test_shm")
		receiver, _ := NewSharedMemoryReceiverWithPath("test_shm", PathProvider)
		defer receiver.Close()
		frame, detected, err := receiver.readFrameFromShm()
		if err != nil {
			t.Fatal("Failed to read frame from shared memory:", err)
		}
		if frame == nil {
			t.Errorf("Expected frame data %s, got %s", data, frame)
		}
		if detected != 0 {
			t.Errorf("Expected detected value 0, got %d", detected)
		}
	})

	t.Run("TestWatchSharedMemoryReceivedFrame", func(t *testing.T) {
		data := []byte("test data")
		defer os.Remove("/dev/shm/test_shm")

		receiver, _ := NewSharedMemoryReceiverWithPath("test_shm", PathProvider)
		defer receiver.Close()
		go receiver.WatchSharedMemory()
		time.Sleep(10 * time.Millisecond)
		createFrame(data, 1, "test_shm")
		timeout := time.After(2 * time.Second)
		select {
		case frame := <-receiver.Frames:
			if string(frame) != string(data) {
				t.Errorf("Expected frame data %s, got %s", data, frame)
			}
		case <-timeout:
			t.Fatal("Timeout waiting for frame")
		}
	})
}
func TestSaveSignificantFrameForLaterWhenDirIsEmpty(t *testing.T) {
	tempPath := t.TempDir()
	PathProvider := TestPathProvider{path: tempPath}
	data := []byte("test data")
	defer os.Remove("/dev/shm/test_shm")

	receiver, _ := NewSharedMemoryReceiverWithPath("test_shm", PathProvider)
	defer receiver.Close()
	go receiver.WatchSharedMemory()

	t.Run("DetectedFrame", func(t *testing.T) {
		createFrame(data, 1, "test_shm")
		timeout := time.After(2 * time.Second)
		hasFrames := make(chan bool, 1)
		hasSignificant := make(chan bool, 1)
		go func() {
			select {
			case frame := <-receiver.Frames:
				if string(frame) != string(data) {
					t.Errorf("Expected frame data %s, got %s", data, frame)
				}
				hasFrames <- true
			case <-timeout:
				hasFrames <- false
			}
		}()
		go func() {
			select {
			case sf := <-receiver.SignificantFrames:
				fmt.Printf("Received significant frame: %s\n", sf.Data)
				if sf.Data == nil {
					t.Error("Expected frame data")
				}
				dirs, _ := os.ReadDir(tempPath)
				if len(dirs) != 0 {
					t.Errorf("Expected temp directory to be empty, got %d files", len(dirs))
				}
				hasSignificant <- true
			case <-timeout:
				hasSignificant <- false
			}
		}()
		if !<-hasFrames {
			t.Fatal("Timeout waiting for regular frame")
		}
		if !<-hasSignificant {
			t.Fatal("Timeout waiting for significant frame")
		}
	})
}
