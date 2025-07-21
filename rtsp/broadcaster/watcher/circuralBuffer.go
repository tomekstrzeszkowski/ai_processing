package watcher

// CircularBuffer holds a fixed number of []byte elements
type CircularBuffer struct {
	data     [][]byte
	size     int
	capacity int
	head     int
}

// NewCircularBuffer creates a new circular buffer with given capacity
func NewCircularBuffer(capacity int) *CircularBuffer {
	return &CircularBuffer{
		data:     make([][]byte, capacity),
		capacity: capacity,
		head:     0,
		size:     0,
	}
}

// Add appends a new []byte element, replacing the oldest if at capacity
func (cb *CircularBuffer) Add(item []byte) {
	cb.data[cb.head] = item
	cb.head = (cb.head + 1) % cb.capacity

	if cb.size < cb.capacity {
		cb.size++
	}
}

// GetAll returns all current elements in insertion order (oldest first)
func (cb *CircularBuffer) GetAll() [][]byte {
	if cb.size == 0 {
		return nil
	}

	result := make([][]byte, cb.size)

	if cb.size < cb.capacity {
		// Buffer not full yet, items are from index 0 to size-1
		copy(result, cb.data[:cb.size])
	} else {
		// Buffer is full, oldest item is at head position
		tail := cb.head
		copy(result, cb.data[tail:])
		copy(result[cb.capacity-tail:], cb.data[:tail])
	}

	return result
}

// Size returns current number of elements
func (cb *CircularBuffer) Size() int {
	return cb.size
}
func (cb *CircularBuffer) Clear() {
	// Reset all slice elements to nil to help GC
	for i := range cb.data {
		cb.data[i] = nil
	}
	cb.size = 0
	cb.head = 0
}
