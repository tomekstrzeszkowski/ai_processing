package watcher

import (
	"encoding/binary"
	"os"
	"testing"
	"time"
)

func createFrame(buffer []byte, detected byte) {
	header := make([]byte, 5)
	header[0] = detected
	binary.LittleEndian.PutUint32(header[1:], uint32(len(buffer)))
	file, err := os.Create("/dev/shm/test_shm")
	if err != nil {
		panic("Failed to create shared memory file: " + err.Error())
	}
	defer file.Close()
	file.Write(header)
	file.Write(buffer)
	file.Sync()
}

func TestNoSharedMemoryFileToRead(t *testing.T) {
	receiver, err := NewSharedMemoryReceiver("non_existent_shm")
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
}

func TestDetectedFrame(t *testing.T) {
	data := []byte("test data")
	createFrame(data, 0)
	defer os.Remove("/dev/shm/test_shm")
	receiver, _ := NewSharedMemoryReceiver("test_shm")
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
}

func TestWatchSharedMemoryReceivedFrame(t *testing.T) {
	data := []byte("test data")
	defer os.Remove("/dev/shm/test_shm")

	receiver, _ := NewSharedMemoryReceiver("test_shm")
	defer receiver.Close()
	go receiver.WatchSharedMemory()
	time.Sleep(10 * time.Millisecond)
	createFrame(data, 1)
	timeout := time.After(2 * time.Second)
	select {
	case frame := <-receiver.Frames:
		if string(frame) != string(data) {
			t.Errorf("Expected frame data %s, got %s", data, frame)
		}
	case <-timeout:
		t.Fatal("Timeout waiting for frame")
	}
}
