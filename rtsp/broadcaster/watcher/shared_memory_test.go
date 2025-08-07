package watcher

import (
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type TestPathProvider struct {
	path string
}

func (tp TestPathProvider) GetSavePath() string {
	return tp.path
}

func createFrameWithDelay(buffer []byte, detected int, shmName string) {
	header := make([]byte, 5)
	header[0] = byte(detected)
	binary.LittleEndian.PutUint32(header[1:], uint32(len(buffer)))

	totalSize := len(header) + len(buffer)
	filePath := "/dev/shm/" + shmName

	// Create and size the file atomically
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		panic("Failed to create file: " + err.Error())
	}
	defer file.Close()

	err = file.Truncate(int64(totalSize))
	if err != nil {
		panic("Failed to truncate file: " + err.Error())
	}

	// Memory map the file
	data, err := unix.Mmap(int(file.Fd()), 0, totalSize, unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		panic("Failed to mmap file: " + err.Error())
	}
	defer unix.Munmap(data)

	// Copy data directly to memory
	copy(data[:len(header)], header)
	copy(data[len(header):], buffer)

	// Sync to disk
	err = unix.Msync(data, unix.MS_SYNC)
	if err != nil {
		panic("Failed to msync file: " + err.Error())
	}
	fmt.Printf("Created shm file /dev/shm/%v %v", shmName, string(buffer))
	time.Sleep(10 * time.Millisecond)
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
		createFrameWithDelay(data, 0, "test_shm")
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
		createFrameWithDelay(data, 1, "test_shm")
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
	t.Run("DetectedFrame", func(t *testing.T) {
		receiver, _ := NewSharedMemoryReceiverWithPath("test_shm", PathProvider)
		defer receiver.Close()
		go receiver.WatchSharedMemory()
		createFrameWithDelay(data, 1, "test_shm")
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
	t.Run("SendFramesBeforeDetection", func(t *testing.T) {
		receiver, _ := NewSharedMemoryReceiverWithPath("test_shm", PathProvider)
		defer receiver.Close()
		go receiver.WatchSharedMemory()
		createFrameWithDelay([]byte("nothing 1"), -1, "test_shm")
		createFrameWithDelay([]byte("nothing 2"), -1, "test_shm")
		createFrameWithDelay(data, 0, "test_shm")
		timeout := time.After(10 * time.Second)
		hasSignificant := make(chan bool, 1)
		go func() {
			select {
			case <-receiver.Frames:
			}
		}()
		go func() {
			select {
			case sf := <-receiver.SignificantFrames:
				fmt.Printf("Received significant frame: %s\n", sf.Data)
				if sf.Data == nil {
					t.Error("Expected frame data")
				}
				if sf.Before.Size() != 2 {
					t.Error("Buffer size is incorrectf")
				}
				if sf.After != nil {
					t.Errorf("Buffer size is incorrect %v", sf.After.Size())
				}
				hasSignificant <- true
			case <-timeout:
				hasSignificant <- false
			}
		}()
		if !<-hasSignificant {
			t.Fatal("Timeout waiting for significant frame")
		}
	})
	t.Run("SendFramesAfterDetection", func(t *testing.T) {
		receiver, _ := NewSharedMemoryReceiverWithPath("test_shm", PathProvider)
		defer receiver.Close()
		go receiver.WatchSharedMemory()
		createFrameWithDelay(data, 0, "test_shm")
		createFrameWithDelay([]byte("nothing 1"), -1, "test_shm")
		createFrameWithDelay([]byte("nothing 2"), -1, "test_shm")
		createFrameWithDelay(data, 0, "test_shm")
		timeout := time.After(10 * time.Second)
		called := 0
		hasSignificant := make(chan bool, 1)
		go func() {
			select {
			case fr := <-receiver.Frames:
				fmt.Printf("FRA %v\n", fr)
			}
		}()
		go func() {
			for {
				select {
				case sf := <-receiver.SignificantFrames:
					called++
					fmt.Printf("Received %v\n", sf)
					if called == 2 {
						fmt.Printf("Received significant frame: %s\n", sf.Data)
						if sf.Data == nil {
							t.Error("Expected frame data")
						}
						if sf.Before.Size() != 0 {
							t.Error("Buffer size is incorrectf")
						}
						if sf.After.Size() != 2 {
							t.Errorf("Buffer size is incorrect %v", sf.After.Size())
						}
						hasSignificant <- true
					}
				case <-timeout:
					hasSignificant <- false
				}
			}
		}()
		if !<-hasSignificant {
			t.Fatal("Timeout waiting for significant frame")
		}
	})
}
