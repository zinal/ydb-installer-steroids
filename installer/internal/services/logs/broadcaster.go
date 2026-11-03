package logs

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var subIDCounter uint64

// Subscriber represents a single log consumer (one WebSocket or SSE client).
// Each subscriber gets its own buffered channel so slow consumers do not
// block fast ones.
type Subscriber struct {
	ID  string
	Ch  chan string
	ctx *LogBroadcaster
}

func (s *Subscriber) Close() {
	s.ctx.Unsubscribe(s.ID)
}

// LogBroadcaster is a thread-safe pub/sub hub that fans out log lines to
// all subscribers, maintains an in-memory ring buffer for late joiners, and
// writes all lines to a persistent log file. Each subscriber gets a buffered
// channel (cap 2048); slow subscribers silently drop lines.
type LogBroadcaster struct {
	mu          sync.RWMutex
	fileMu      sync.Mutex
	subscribers map[string]*Subscriber
	buffer      *ringBuffer
	logFile     *os.File
	logWriter   *bufio.Writer
	closed      bool
}

const SubscriberChanCap = 2048

const RingBufferCapacity = 10000

// NewBroadcaster creates a broadcaster and opens a log file for persistent
// storage. If the log file cannot be opened, the broadcaster still works —
// it logs to stderr and continues without file persistence.
func NewBroadcaster(operationType string) *LogBroadcaster {
	b := &LogBroadcaster{
		subscribers: make(map[string]*Subscriber),
		buffer:      newRingBuffer(RingBufferCapacity),
	}

	logDir := "/tmp/ydb-installer/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("[broadcaster] Warning: cannot create log dir %s: %v", logDir, err)
		return b
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.log", operationType, timestamp)
	filePath := filepath.Join(logDir, filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[broadcaster] Warning: cannot open log file %s: %v", filePath, err)
		return b
	}

	b.logFile = file
	b.logWriter = bufio.NewWriter(file)
	log.Printf("[broadcaster] Log file: %s", filePath)
	return b
}

// Subscribe creates a new subscriber and seeds it with a snapshot of the
// ring buffer so it can catch up on recent history. The caller MUST call
// Subscriber.Close() when done.
func (b *LogBroadcaster) Subscribe() *Subscriber {
	sub := &Subscriber{
		ID:  fmt.Sprintf("sub-%d", atomic.AddUint64(&subIDCounter, 1)),
		Ch:  make(chan string, SubscriberChanCap),
		ctx: b,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, line := range b.buffer.Snapshot() {
		select {
		case sub.Ch <- line:
		default:
			break
		}
	}

	b.subscribers[sub.ID] = sub
	log.Printf("[broadcaster] Subscriber %s joined (%d active)", sub.ID, len(b.subscribers))
	return sub
}

func (b *LogBroadcaster) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, ok := b.subscribers[id]
	if !ok {
		return
	}
	delete(b.subscribers, id)
	close(sub.Ch)
	log.Printf("[broadcaster] Subscriber %s left (%d active)", id, len(b.subscribers))
}

// Publish sends a log line to all active subscribers (non-blocking per
// subscriber), appends it to the ring buffer, and writes it to the log file.
// Slow subscribers whose channel is full simply miss that line.
func (b *LogBroadcaster) Publish(line string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	b.buffer.Append(line)

	// fileMu allows concurrent Publish() calls without blocking subscriber delivery.
	b.fileMu.Lock()
	if b.logWriter != nil {
		_, _ = b.logWriter.WriteString(line + "\n")
		_ = b.logWriter.Flush()
	}
	b.fileMu.Unlock()

	for _, sub := range b.subscribers {
		select {
		case sub.Ch <- line:
		default:
		}
	}
}

func (b *LogBroadcaster) Snapshot() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buffer.Snapshot()
}

// Close shuts down the broadcaster: closes all subscriber channels, flushes
// and closes the log file. After Close() returns, Publish() is a no-op.
func (b *LogBroadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true

	for id, sub := range b.subscribers {
		delete(b.subscribers, id)
		close(sub.Ch)
	}

	b.fileMu.Lock()
	if b.logWriter != nil {
		_ = b.logWriter.Flush()
		b.logWriter = nil
	}
	if b.logFile != nil {
		_ = b.logFile.Close()
		b.logFile = nil
	}
	b.fileMu.Unlock()

	log.Printf("[broadcaster] Closed (%d subscribers disconnected)", len(b.subscribers))
}

func (b *LogBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// Reset clears the ring buffer and rotates the log file so that a new
// operation starts with a clean slate. Existing subscribers are NOT affected
// — they continue receiving live lines from the new operation.
func (b *LogBroadcaster) Reset(operationType string) {
	b.mu.Lock()
	b.buffer = newRingBuffer(RingBufferCapacity)
	b.mu.Unlock()

	b.fileMu.Lock()
	defer b.fileMu.Unlock()

	if b.logWriter != nil {
		_ = b.logWriter.Flush()
		b.logWriter = nil
	}
	if b.logFile != nil {
		_ = b.logFile.Close()
		b.logFile = nil
	}

	logDir := "/tmp/ydb-installer/logs"
	_ = os.MkdirAll(logDir, 0755)
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.log", operationType, timestamp)
	filePath := filepath.Join(logDir, filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[broadcaster] Warning: cannot open new log file %s: %v", filePath, err)
		return
	}

	b.logFile = file
	b.logWriter = bufio.NewWriter(file)
	log.Printf("[broadcaster] Log file rotated: %s", filePath)
}
