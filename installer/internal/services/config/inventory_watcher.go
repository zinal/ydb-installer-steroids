// Package config provides services for parsing and managing configuration files.
package config

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// InventoryFileWatcher watches the inventory.yaml file for external changes
// and notifies subscribers via a channel.
type InventoryFileWatcher struct {
	watcher   *fsnotify.Watcher
	filePath  string
	mu        sync.RWMutex
	listeners []chan struct{}
	done      chan struct{}
}

// NewInventoryFileWatcher creates a new file watcher for the inventory file.
func NewInventoryFileWatcher(cwd string) (*InventoryFileWatcher, error) {
	fw := &InventoryFileWatcher{
		filePath: filepath.Join(cwd, "inventory", "50-inventory.yaml"),
		done:     make(chan struct{}),
	}

	if _, err := os.Stat(fw.filePath); err != nil {
		log.Printf("inventory file not found at %s, watcher will monitor directory", fw.filePath)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw.watcher = w

	dir := filepath.Dir(fw.filePath)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, err
	}

	go fw.watchLoop()
	return fw, nil
}

// AddListener registers a channel that will receive notifications on file changes.
func (fw *InventoryFileWatcher) AddListener() chan struct{} {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	ch := make(chan struct{}, 1)
	fw.listeners = append(fw.listeners, ch)
	return ch
}

// RemoveListener unregisters a listener channel.
func (fw *InventoryFileWatcher) RemoveListener(ch chan struct{}) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	for i, l := range fw.listeners {
		if l == ch {
			fw.listeners = append(fw.listeners[:i], fw.listeners[i+1:]...)
			close(ch)
			return
		}
	}
}

// Close stops the file watcher and cleans up resources.
func (fw *InventoryFileWatcher) Close() {
	close(fw.done)
	if fw.watcher != nil {
		fw.watcher.Close()
	}
	fw.mu.Lock()
	for _, ch := range fw.listeners {
		close(ch)
	}
	fw.listeners = nil
	fw.mu.Unlock()
}

// watchLoop monitors the directory for file changes.
func (fw *InventoryFileWatcher) watchLoop() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			baseName := filepath.Base(event.Name)
			if event.Name == fw.filePath || baseName == "50-inventory.yaml" || baseName == "90-inventory.yaml" {
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
					fw.notifyListeners()
				}
			}
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("inventory file watcher error: %v", err)
		case <-fw.done:
			return
		}
	}
}

// notifyListeners sends a notification to all registered listeners.
func (fw *InventoryFileWatcher) notifyListeners() {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	for _, ch := range fw.listeners {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
