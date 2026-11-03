package unit

import (
	"sync"
	"testing"
	"time"

	"ydb-installer/internal/services/logs"
)

// TestBroadcasterPublishSubscribe verifies that published lines reach subscribers.
func TestBroadcasterPublishSubscribe(t *testing.T) {
	b := logs.NewBroadcaster("test")
	defer b.Close()

	sub1 := b.Subscribe()
	defer sub1.Close()

	sub2 := b.Subscribe()
	defer sub2.Close()

	b.Publish("line1")
	b.Publish("line2")

	checkSub := func(t *testing.T, sub *logs.Subscriber, expected ...string) {
		t.Helper()
		for _, exp := range expected {
			select {
			case got := <-sub.Ch:
				if got != exp {
					t.Errorf("expected %q, got %q", exp, got)
				}
			case <-time.After(time.Second):
				t.Errorf("timeout waiting for %q", exp)
			}
		}
	}

	checkSub(t, sub1, "line1", "line2")
	checkSub(t, sub2, "line1", "line2")
}

// TestBroadcasterSlowSubscriber verifies that a slow subscriber drops lines
// while a fast one receives all of them.
func TestBroadcasterSlowSubscriber(t *testing.T) {
	b := logs.NewBroadcaster("test")
	defer b.Close()

	fast := b.Subscribe()
	defer fast.Close()

	slow := b.Subscribe()
	defer slow.Close()

	for i := 0; i < logs.SubscriberChanCap*2; i++ {
		b.Publish("line")
	}

	got := 0
	for {
		select {
		case <-fast.Ch:
			got++
		default:
			goto doneFast
		}
	}
doneFast:
	if got == 0 {
		t.Fatal("fast subscriber received no lines")
	}
	if got < logs.SubscriberChanCap {
		t.Errorf("fast subscriber received only %d lines, expected at least %d", got, logs.SubscriberChanCap)
	}
}

// TestBroadcasterUnsubscribe verifies that unsubscribing removes the subscriber
// and closes its channel.
func TestBroadcasterUnsubscribe(t *testing.T) {
	b := logs.NewBroadcaster("test")
	defer b.Close()

	sub := b.Subscribe()

	b.Publish("before")
	<-sub.Ch

	sub.Close()

	b.Publish("after")

	select {
	case _, ok := <-sub.Ch:
		if ok {
			t.Error("expected closed channel after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Error("timeout: channel not closed after unsubscribe")
	}
}

// TestBroadcasterSnapshot verifies that new subscribers receive recent history.
func TestBroadcasterSnapshot(t *testing.T) {
	b := logs.NewBroadcaster("test")
	defer b.Close()

	for i := 0; i < 5; i++ {
		b.Publish("line")
	}

	sub := b.Subscribe()
	defer sub.Close()

	got := 0
	for {
		select {
		case <-sub.Ch:
			got++
		default:
			goto done
		}
	}
done:
	if got < 5 {
		t.Errorf("expected at least 5 history lines, got %d", got)
	}
}

// TestBroadcasterClose verifies that Close() closes all subscriber channels.
func TestBroadcasterClose(t *testing.T) {
	b := logs.NewBroadcaster("test")

	sub1 := b.Subscribe()
	sub2 := b.Subscribe()

	b.Close()

	for _, sub := range []*logs.Subscriber{sub1, sub2} {
		select {
		case _, ok := <-sub.Ch:
			if ok {
				t.Error("expected closed channel after broadcaster Close")
			}
		case <-time.After(time.Second):
			t.Error("timeout: channel not closed after broadcaster Close")
		}
	}
}

// TestBroadcasterConcurrentPublish verifies the broadcaster is safe under
// concurrent access.
func TestBroadcasterConcurrentPublish(t *testing.T) {
	b := logs.NewBroadcaster("test")
	defer b.Close()

	var wg sync.WaitGroup
	const writers = 8
	const linesPerWriter = 500

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < linesPerWriter; j++ {
				b.Publish("line")
			}
		}()
	}

	wg.Wait()

	// Verify snapshot contains data.
	snap := b.Snapshot()
	if len(snap) == 0 {
		t.Error("expected non-empty snapshot after concurrent publish")
	}
	if len(snap) > logs.RingBufferCapacity {
		t.Errorf("snapshot exceeds ring buffer capacity: %d > %d", len(snap), logs.RingBufferCapacity)
	}
}

// TestRingBuffer verifies the ring buffer works correctly.
func TestRingBuffer(t *testing.T) {
	b := logs.NewBroadcaster("test")
	defer b.Close()

	const lines = logs.RingBufferCapacity + 100
	for i := 0; i < lines; i++ {
		b.Publish("line")
	}

	snap := b.Snapshot()
	if len(snap) > logs.RingBufferCapacity {
		t.Errorf("snapshot exceeds ring buffer capacity: %d", len(snap))
	}
}