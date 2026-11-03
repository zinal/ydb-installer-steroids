package unit_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ydb-installer/internal/services/config"
)

// newFileWatcher creates a watcher over a fresh temp dir with the
// inventory/ subdirectory in place. The watcher (not the test) owns the
// goroutine; Close is registered as cleanup.
func newFileWatcher(t *testing.T) (*config.InventoryFileWatcher, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "inventory"), 0o755); err != nil {
		t.Fatal(err)
	}
	fw, err := config.NewInventoryFileWatcher(dir)
	if err != nil {
		t.Fatalf("NewInventoryFileWatcher: %v", err)
	}
	t.Cleanup(fw.Close)
	return fw, dir
}

// waitForNotification blocks until the listener channel fires or times out.
func waitForNotification(t *testing.T, ch chan struct{}, d time.Duration) bool {
	t.Helper()
	select {
	case _, ok := <-ch:
		return ok
	case <-time.After(d):
		return false
	}
}

func TestInventoryFileWatcher_NotifiesOnInventoryWrite(t *testing.T) {
	fw, dir := newFileWatcher(t)
	ch := fw.AddListener()

	target := filepath.Join(dir, "inventory", "50-inventory.yaml")
	if err := os.WriteFile(target, []byte("all: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !waitForNotification(t, ch, 3*time.Second) {
		t.Fatal("timed out waiting for inventory change notification")
	}
}

func TestInventoryFileWatcher_IgnoresUnrelatedFiles(t *testing.T) {
	fw, dir := newFileWatcher(t)
	ch := fw.AddListener()

	unrelated := filepath.Join(dir, "inventory", "other.txt")
	if err := os.WriteFile(unrelated, []byte("noise"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Give inotify a moment to deliver any (unexpected) event.
	if waitForNotification(t, ch, 300*time.Millisecond) {
		t.Fatal("unexpected notification for an unrelated file")
	}
}

func TestInventoryFileWatcher_RemoveListenerClosesChannel(t *testing.T) {
	fw, _ := newFileWatcher(t)
	ch := fw.AddListener()
	fw.RemoveListener(ch)

	if _, ok := <-ch; ok {
		t.Fatal("listener channel still open after RemoveListener")
	}
}

func TestInventoryFileWatcher_MissingDirIsError(t *testing.T) {
	dir := t.TempDir()
	// No inventory/ subdirectory → watching the dir must fail cleanly.
	if _, err := config.NewInventoryFileWatcher(dir); err == nil {
		t.Fatal("expected error when inventory directory does not exist")
	}
}
