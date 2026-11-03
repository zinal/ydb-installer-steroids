package unit_test

// Tests for GET /api/install/ydb/config-status — dry-run compare logic.
// Scenarios: missing file, matches, outdated, method-not-allowed, bad inventory.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ydb-installer/internal/api/handlers"
	"ydb-installer/internal/models"
	cfgsvc "ydb-installer/internal/services/config"
	"ydb-installer/internal/services/session"
)

// minimalInv returns a small but complete InventoryData suitable for
// config generation tests (mirror-3-dc, 3 hosts, 1 disk per host).
func minimalInv() *models.InventoryData {
	return &models.InventoryData{
		DC1: []string{"host-a.test"},
		DC2: []string{"host-b.test"},
		DC3: []string{"host-c.test"},
		YDBConfig: &models.YDBConfig{
			Domain:                      "Root",
			CoresStatic:                 2,
			EnforceUserTokenRequirement: false,
			RequestClientCertificate:    true,
		},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}
}

// writeInventory writes inv to a YAML file at path using InventoryWriter.
func writeInventory(t *testing.T, path string, inv *models.InventoryData) {
	t.Helper()
	w := cfgsvc.NewInventoryWriter()
	if err := w.Write(path, inv); err != nil {
		t.Fatalf("writeInventory: %v", err)
	}
}

// configStatusBody decodes the response body of GET /api/install/ydb/config-status.
type configStatusBody struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Error   string `json:"error"`
}

func callConfigStatus(t *testing.T, store *session.Store, invPath, cfgPath string) (int, configStatusBody) {
	t.Helper()
	h := handlers.ConfigStatus(store, invPath, cfgPath)
	req := httptest.NewRequest(http.MethodGet, "/api/install/ydb/config-status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body configStatusBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return rec.Code, body
}

// TestConfigStatus_MethodNotAllowed verifies POST is rejected with 405.
func TestConfigStatus_MethodNotAllowed(t *testing.T) {
	h := handlers.ConfigStatus(nil, "", "/tmp/config.yaml")
	req := httptest.NewRequest(http.MethodPost, "/api/install/ydb/config-status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// TestConfigStatus_MissingFile returns status=missing when config.yaml does not exist.
func TestConfigStatus_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	invPath := filepath.Join(tmpDir, "inventory.yaml")
	writeInventory(t, invPath, minimalInv())

	code, body := callConfigStatus(t, nil, invPath, cfgPath)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !body.Success {
		t.Errorf("expected success=true for missing file, got false (error: %s)", body.Error)
	}
	if body.Status != "missing" {
		t.Errorf("status = %q, want %q", body.Status, "missing")
	}
	if body.Content != "" {
		t.Errorf("content should be empty for missing file")
	}
}

// TestConfigStatus_Matches returns status=matches after generate with the same inventory.
// DC1/DC2/DC3 live only in the session (not in inventory.yaml), so we must pass them
// via the session store — identical to how GenerateConfig enriches the inventory.
func TestConfigStatus_Matches(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	invPath := filepath.Join(tmpDir, "inventory.yaml")

	// Write a minimal inventory WITHOUT DC lists (as the real inventory.yaml looks).
	invFlat := &models.InventoryData{
		YDBConfig: &models.YDBConfig{
			Domain:                      "Root",
			CoresStatic:                 2,
			EnforceUserTokenRequirement: false,
			RequestClientCertificate:    true,
		},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}
	writeInventory(t, invPath, invFlat)

	// Session carries the DC lists (as in the real UI flow).
	store := session.New("en")
	sessionInv := &models.InventoryData{
		DC1: []string{"host-a.test"},
		DC2: []string{"host-b.test"},
		DC3: []string{"host-c.test"},
	}
	store.SetInventory(sessionInv, true, false)

	// Generate config.yaml using the same merged inventory as ConfigStatus will use.
	invForGen := &models.InventoryData{
		YDBConfig: invFlat.YDBConfig,
		Disks:     invFlat.Disks,
		DC1:       sessionInv.DC1,
		DC2:       sessionInv.DC2,
		DC3:       sessionInv.DC3,
	}
	gen := cfgsvc.NewGenerator(cfgPath)
	if err := gen.Generate(invForGen); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code, body := callConfigStatus(t, store, invPath, cfgPath)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !body.Success {
		t.Errorf("expected success=true, got false (error: %s)", body.Error)
	}
	if body.Status != "matches" {
		t.Errorf("status = %q, want %q", body.Status, "matches")
	}
	if body.Content == "" {
		t.Errorf("expected non-empty content for matches")
	}
}

// TestConfigStatus_Outdated returns status=outdated when inventory has changed after generation.
// As with Matches, DC lists come from session; changing the domain triggers outdated.
func TestConfigStatus_Outdated(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	invPath := filepath.Join(tmpDir, "inventory.yaml")

	// Write flat inventory with domain "Root"
	invFlat := &models.InventoryData{
		YDBConfig: &models.YDBConfig{
			Domain:                      "Root",
			CoresStatic:                 2,
			EnforceUserTokenRequirement: false,
			RequestClientCertificate:    true,
		},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}
	writeInventory(t, invPath, invFlat)

	store := session.New("en")
	sessionInv := &models.InventoryData{
		DC1: []string{"host-a.test"},
		DC2: []string{"host-b.test"},
		DC3: []string{"host-c.test"},
	}
	store.SetInventory(sessionInv, true, false)

	// Generate config using domain "Root"
	invForGenA := &models.InventoryData{
		YDBConfig: invFlat.YDBConfig,
		Disks:     invFlat.Disks,
		DC1:       sessionInv.DC1,
		DC2:       sessionInv.DC2,
		DC3:       sessionInv.DC3,
	}
	gen := cfgsvc.NewGenerator(cfgPath)
	if err := gen.Generate(invForGenA); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Now change the inventory domain to "ChangedDomain" (simulate user going back)
	invFlat.YDBConfig.Domain = "ChangedDomain"
	writeInventory(t, invPath, invFlat)

	code, body := callConfigStatus(t, store, invPath, cfgPath)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !body.Success {
		t.Errorf("expected success=true, got false (error: %s)", body.Error)
	}
	if body.Status != "outdated" {
		t.Errorf("status = %q, want %q", body.Status, "outdated")
	}
	// Content = on-disk (old) config, not the new one
	if body.Content == "" {
		t.Errorf("expected on-disk content in outdated response")
	}
}

// TestConfigStatus_SessionDCsEnrich verifies that session DC lists are merged
// before comparison so mirror-3-dc layouts that live only in session don't
// cause false outdated when they haven't changed.
func TestConfigStatus_SessionDCsEnrich(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	invPath := filepath.Join(tmpDir, "inventory.yaml")

	// Inventory has flat hosts (no DC breakdown) — session provides DC lists.
	invFlat := &models.InventoryData{
		Hosts: []string{"host-a.test", "host-b.test", "host-c.test"},
		YDBConfig: &models.YDBConfig{
			Domain:                      "Root",
			CoresStatic:                 2,
			EnforceUserTokenRequirement: false,
			RequestClientCertificate:    true,
		},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}
	writeInventory(t, invPath, invFlat)

	// Session with DC1/DC2/DC3
	store := session.New("en")
	sessionInv := &models.InventoryData{
		DC1: []string{"host-a.test"},
		DC2: []string{"host-b.test"},
		DC3: []string{"host-c.test"},
	}
	store.SetInventory(sessionInv, true, false)

	// Generate using the same session DC merge
	invForGen := &models.InventoryData{
		Hosts: invFlat.Hosts,
		DC1:   sessionInv.DC1,
		DC2:   sessionInv.DC2,
		DC3:   sessionInv.DC3,
		YDBConfig: &models.YDBConfig{
			Domain:                      "Root",
			CoresStatic:                 2,
			EnforceUserTokenRequirement: false,
			RequestClientCertificate:    true,
		},
		Disks: invFlat.Disks,
	}
	gen := cfgsvc.NewGenerator(cfgPath)
	if err := gen.Generate(invForGen); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code, body := callConfigStatus(t, store, invPath, cfgPath)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !body.Success {
		t.Errorf("expected success=true, got false (error: %s)", body.Error)
	}
	if body.Status != "matches" {
		t.Errorf("status = %q, want %q (session DC enrichment should yield match)", body.Status, "matches")
	}
}

// TestConfigStatus_BadInventory returns success=false on inventory parse error.
func TestConfigStatus_BadInventory(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	invPath := filepath.Join(tmpDir, "inventory.yaml")

	// Write a non-YAML file as inventory
	if err := os.WriteFile(invPath, []byte("not: valid: yaml: :::"), 0644); err != nil {
		t.Fatalf("write bad inventory: %v", err)
	}

	// Write a valid config.yaml so the "missing" early return doesn't trigger
	gen := cfgsvc.NewGenerator(cfgPath)
	if err := gen.Generate(minimalInv()); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code, body := callConfigStatus(t, nil, invPath, cfgPath)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if body.Success {
		t.Errorf("expected success=false on bad inventory, got true")
	}
	if body.Error == "" {
		t.Errorf("expected non-empty error on bad inventory")
	}
}

// TestConfigStatus_Block42_Matches verifies that the dry-run honours the
// session topology: a block-4-2 config generated from session state compares
// as "matches" instead of falling back to the mirror-3-dc default.
func TestConfigStatus_Block42_Matches(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	invPath := filepath.Join(tmpDir, "inventory.yaml")

	// Inventory on disk carries no DC lists and no ydb_topology (as in the
	// real flow — both live only in the session).
	invFlat := &models.InventoryData{
		YDBConfig: &models.YDBConfig{
			Domain:                      "Root",
			CoresStatic:                 2,
			EnforceUserTokenRequirement: false,
			RequestClientCertificate:    true,
		},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}
	writeInventory(t, invPath, invFlat)

	store := session.New("en")
	sessionInv := &models.InventoryData{
		DC1:      []string{"node-1", "node-2", "node-3"},
		DC2:      []string{"node-4", "node-5", "node-6"},
		DC3:      []string{"node-7", "node-8"},
		Topology: &models.TopologyConfig{Type: "block-4-2"},
	}
	store.SetInventory(sessionInv, true, false)

	// Generate the block-4-2 config the same way GenerateConfig would.
	invForGen := &models.InventoryData{
		YDBConfig: invFlat.YDBConfig,
		Disks:     invFlat.Disks,
		DC1:       sessionInv.DC1,
		DC2:       sessionInv.DC2,
		DC3:       sessionInv.DC3,
		Topology:  sessionInv.Topology,
	}
	gen := cfgsvc.NewGenerator(cfgPath)
	if err := gen.Generate(invForGen); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code, body := callConfigStatus(t, store, invPath, cfgPath)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !body.Success {
		t.Errorf("expected success=true, got false (error: %s)", body.Error)
	}
	if body.Status != "matches" {
		t.Errorf("status = %q, want %q (session topology must drive the dry-run)", body.Status, "matches")
	}
}
