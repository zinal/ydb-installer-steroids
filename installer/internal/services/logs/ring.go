// Package logs provides a pub/sub log broadcasting system for streaming
// ansible-playbook output to multiple WebSocket/SSE consumers.
package logs

import "sync"

// ringBuffer is a thread-safe fixed-capacity circular buffer that stores the
// most recent log lines in memory so late-joining WebSocket clients can catch
// up on recent history without reading the log file from disk.
type ringBuffer struct {
	mu    sync.RWMutex
	lines []string
	start int
	count int
	cap   int
}

func newRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{
		lines: make([]string, capacity),
		cap:   capacity,
	}
}

func (rb *ringBuffer) Append(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	idx := (rb.start + rb.count) % rb.cap
	rb.lines[idx] = line
	if rb.count < rb.cap {
		rb.count++
	} else {
		rb.start = (rb.start + 1) % rb.cap
	}
}

func (rb *ringBuffer) Snapshot() []string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return []string{}
	}

	result := make([]string, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.start + i) % rb.cap
		result[i] = rb.lines[idx]
	}
	return result
}

func (rb *ringBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}
