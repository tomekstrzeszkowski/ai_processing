package watcher

import (
	"reflect"
	"testing"
)

func TestInitialState(t *testing.T) {
	buffer := NewCircularBuffer(10)
	if buffer.IsFull() {
		t.Error("Buffer should not be full")
	}
}

func TestIsFull(t *testing.T) {
	buffer := NewCircularBuffer(3)
	buffer.Add([]byte("test1"))
	buffer.Add([]byte("test1"))
	buffer.Add([]byte("test1"))

	if !buffer.IsFull() {
		t.Error("Buffer should be full")
	}
}

func TestClearBuffer(t *testing.T) {
	buffer := NewCircularBuffer(3)
	buffer.Add([]byte("test1"))
	buffer.Add([]byte("test1"))
	buffer.Add([]byte("test1"))
	buffer.Clear()
	buffer.Add([]byte("test1"))
	if buffer.IsFull() {
		t.Error("Buffer should not be full")
	}
}

func TestBufferCanReturnSize(t *testing.T) {
	buffer := NewCircularBuffer(3)
	buffer.Add([]byte("test1"))
	buffer.Add([]byte("test1"))
	if buffer.Size() != 2 {
		t.Errorf("Expected size 2, got %d", buffer.Size())
	}
	buffer.Add([]byte("test1"))

	if buffer.Size() != 3 {
		t.Errorf("Expected size 3, got %d", buffer.Size())
	}
	buffer.Add([]byte("test1"))

	if buffer.Size() != 3 {
		t.Errorf("Expected size 3, got %d", buffer.Size())
	}
}

func TestBufferSizeResetAfterClear(t *testing.T) {
	buffer := NewCircularBuffer(3)
	buffer.Add([]byte("test1"))
	buffer.Add([]byte("test1"))
	buffer.Clear()
	if buffer.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", buffer.Size())
	}
}

func TestBufferCanReturnValues(t *testing.T) {
	buffer := NewCircularBuffer(3)
	buffer.Add([]byte("test1"))
	buffer.Add([]byte("test2"))
	buffer.Add([]byte("test3"))

	values := buffer.GetAll()
	expected := [][]byte{[]byte("test1"), []byte("test2"), []byte("test3")}

	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Expected values %v, got %v", expected, values)
	}
}
