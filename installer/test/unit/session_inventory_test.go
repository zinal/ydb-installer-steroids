package unit_test

import (
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/session"
)

func TestSessionStore_SetAndGetInventory(t *testing.T) {
	s := session.New("en")

	if s.Get().Inventory != nil {
		t.Error("expected nil inventory initially")
	}
	if s.Get().InventoryFound {
		t.Error("expected inventory_found false initially")
	}

	inv := &models.InventoryData{
		Hosts: []string{"host1", "host2"},
		SSH: &models.SSHConfig{
			Login: "root",
		},
	}
	s.SetInventory(inv, true, true)

	got := s.Get()
	if got.Inventory == nil {
		t.Fatal("expected inventory after SetInventory")
	}
	if len(got.Inventory.Hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(got.Inventory.Hosts))
	}
	if !got.InventoryFound {
		t.Error("expected inventory_found true")
	}
	if !got.ConfigFound {
		t.Error("expected config_found true")
	}
}

func TestSessionStore_UpdateInventory(t *testing.T) {
	s := session.New("en")

	inv := &models.InventoryData{
		Hosts: []string{"new-host"},
		YDBConfig: &models.YDBConfig{
			Domain: "NewDomain",
		},
	}
	s.UpdateInventory(inv, true, false)

	got := s.Get()
	if got.Inventory == nil {
		t.Fatal("expected inventory after UpdateInventory")
	}
	if len(got.Inventory.Hosts) != 1 || got.Inventory.Hosts[0] != "new-host" {
		t.Errorf("unexpected hosts: %v", got.Inventory.Hosts)
	}
	if got.Inventory.YDBConfig.Domain != "NewDomain" {
		t.Errorf("unexpected domain: %s", got.Inventory.YDBConfig.Domain)
	}
}

func TestSessionStore_GetInventoryReturnsCopy(t *testing.T) {
	s := session.New("en")
	inv := &models.InventoryData{
		Hosts: []string{"host1"},
	}
	s.SetInventory(inv, true, true)

	got := s.GetInventory()
	got.Hosts = append(got.Hosts, "host2")

	original := s.GetInventory()
	if len(original.Hosts) != 1 {
		t.Errorf("expected original to have 1 host, got %d", len(original.Hosts))
	}
}

func TestSessionStore_SetInventoryFound(t *testing.T) {
	s := session.New("en")
	s.SetInventoryFound(true)
	if !s.Get().InventoryFound {
		t.Error("expected inventory_found true after SetInventoryFound(true)")
	}
}

func TestSessionStore_SetConfigFound(t *testing.T) {
	s := session.New("en")
	s.SetConfigFound(true)
	if !s.Get().ConfigFound {
		t.Error("expected config_found true after SetConfigFound(true)")
	}
}
