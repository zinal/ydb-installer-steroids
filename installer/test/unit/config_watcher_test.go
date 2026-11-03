package unit_test

import (
	"os"
	"path/filepath"
	"testing"

	"ydb-installer/internal/services/config"
)

// watcherFixtureDir returns the controlled testdata fixture directory that
// mirrors the installer layout (inventory/50-inventory.yaml + files/config.yaml).
func watcherFixtureDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(cwd, "..", "testdata", "watcher")
}

func TestConfigWatcher_DiscoverWithRealFile(t *testing.T) {
	watcher := config.NewConfigWatcher()
	discovery := watcher.Discover(watcherFixtureDir(t))

	if !discovery.InventoryFound {
		t.Error("expected to find inventory file")
	}
	if discovery.InventoryData == nil {
		t.Error("expected parsed inventory data")
	}

	if !discovery.ConfigFound {
		t.Error("expected to find config.yaml")
	}
}

func TestConfigWatcher_DiscoverEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	watcher := config.NewConfigWatcher()
	discovery := watcher.Discover(tmpDir)

	if discovery.InventoryFound {
		t.Error("expected no inventory in empty dir")
	}
	if discovery.ConfigFound {
		t.Error("expected no config in empty dir")
	}
	if discovery.InventoryData != nil {
		t.Error("expected nil inventory data")
	}
}

func TestConfigWatcher_HasInventory(t *testing.T) {
	watcher := config.NewConfigWatcher()
	discovery := watcher.Discover(watcherFixtureDir(t))

	if !watcher.HasInventory(discovery) {
		t.Error("expected HasInventory to return true")
	}
}

func TestConfigWatcher_HasConfig(t *testing.T) {
	watcher := config.NewConfigWatcher()
	discovery := watcher.Discover(watcherFixtureDir(t))

	if !watcher.HasConfig(discovery) {
		t.Error("expected HasConfig to return true")
	}
}
