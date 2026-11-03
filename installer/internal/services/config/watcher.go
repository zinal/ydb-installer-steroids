// Package config provides services for parsing and managing configuration files
// (inventory.yaml, config.yaml) used by the YDB installer.
package config

import (
	"os"
	"path/filepath"

	"ydb-installer/internal/models"
)

// ConfigDiscovery represents the result of searching for configuration files.
type ConfigDiscovery struct {
	InventoryFound bool
	InventoryPath  string
	InventoryData  *models.InventoryData

	ConfigFound bool
	ConfigPath  string
}

// ConfigWatcher handles discovery and parsing of configuration files
// in the current working directory.
type ConfigWatcher struct {
	inventoryService *InventoryService
}

func NewConfigWatcher() *ConfigWatcher {
	return &ConfigWatcher{
		inventoryService: NewInventoryService(),
	}
}

// Discover searches for inventory.yaml and config.yaml files in the given directory.
func (w *ConfigWatcher) Discover(cwd string) ConfigDiscovery {
	result := ConfigDiscovery{}

	invPath := filepath.Join(cwd, "inventory", "50-inventory.yaml")
	if _, err := os.Stat(invPath); err == nil {
		result.InventoryFound = true
		result.InventoryPath = invPath

		// If parsing fails, the file is still reported as found but with nil data.
		if invData, parseErr := w.inventoryService.Parse(invPath); parseErr == nil {
			result.InventoryData = invData
		}
	}

	configPath := filepath.Join(cwd, "files", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		result.ConfigFound = true
		result.ConfigPath = configPath
	}

	return result
}

func (w *ConfigWatcher) GetInventoryData(discovery ConfigDiscovery) *models.InventoryData {
	return discovery.InventoryData
}

func (w *ConfigWatcher) HasInventory(discovery ConfigDiscovery) bool {
	return discovery.InventoryFound && discovery.InventoryData != nil
}

func (w *ConfigWatcher) HasConfig(discovery ConfigDiscovery) bool {
	return discovery.ConfigFound
}
