package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ydb-installer/internal/api"
	"ydb-installer/internal/models"
	"ydb-installer/internal/services/config"
	"ydb-installer/internal/services/session"
)

// newEMInventoryServer builds a router with the EM inventory writer wired.
// emPath is the absolute path of 90-inventory.yaml the handler should use.
func newEMInventoryServer(t *testing.T, store *session.Store, emPath string) *httptest.Server {
	t.Helper()
	h := api.New(api.Options{
		SessionStore:      store,
		OperationStates:   models.NewOperationStates(),
		SPA:               nil,
		EMInventoryWriter: config.NewEMInventoryWriter(),
		EMInventoryPath:   emPath,
	})
	return httptest.NewServer(h)
}

func TestEMInventoryHandler_GetEmpty(t *testing.T) {
	chdirTemp(t)
	store := session.New("en")
	srv := newEMInventoryServer(t, store, "")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/em-inventory")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		EMInventoryFound bool             `json:"em_inventory_found"`
		Data             *models.EMConfig `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.EMInventoryFound {
		t.Error("em_inventory_found = true, want false")
	}
	if body.Data != nil {
		t.Errorf("data = %+v, want nil", body.Data)
	}
}

func TestEMInventoryHandler_PutRoundTrip(t *testing.T) {
	dir := chdirTemp(t)
	emPath := filepath.Join(dir, "90-inventory.yaml")
	store := session.New("en")
	srv := newEMInventoryServer(t, store, emPath)
	defer srv.Close()

	payload := models.EMConfig{
		Hosts:       []models.EMHost{{FQDN: "em-1.example.com"}, {FQDN: "em-2.example.com"}},
		ClusterName: "em-cluster",
		ArchivePath: "/tmp/ydb-em.tar.gz",
	}

	resp := putJSON(t, srv.URL+"/api/em-inventory", payload)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Status string           `json:"status"`
		Data   *models.EMConfig `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Data == nil || len(body.Data.Hosts) != 2 {
		t.Fatalf("response data hosts = %+v, want 2 hosts", body.Data)
	}

	// File must exist and contain the EM hosts.
	raw, err := os.ReadFile(emPath)
	if err != nil {
		t.Fatalf("90-inventory not written: %v", err)
	}
	if !strings.Contains(string(raw), "em-1.example.com") {
		t.Error("90-inventory missing em-1.example.com")
	}

	// Session must reflect the write and raise the discovery flag.
	if !store.GetEMInventoryFound() {
		t.Error("em_inventory_found = false, want true after PUT")
	}
	got := store.GetEMConfig()
	if got == nil || got.ClusterName != "em-cluster" {
		t.Errorf("session EM config = %+v, want cluster_name em-cluster", got)
	}
}

// TestEMInventoryHandler_PutReparsesFile locks the GOTCHA: after PUT the
// session is re-parsed FROM THE FILE (file is the source of truth), not
// echoed from the request body. The writer derives
// ydb_em_db_connection_endpoint from the first host when the request leaves
// it empty — so a non-empty endpoint in the response proves the re-parse.
func TestEMInventoryHandler_PutReparsesFile(t *testing.T) {
	dir := chdirTemp(t)
	emPath := filepath.Join(dir, "90-inventory.yaml")
	store := session.New("en")
	srv := newEMInventoryServer(t, store, emPath)
	defer srv.Close()

	hosts := []models.EMHost{{FQDN: "em-1.example.com"}}

	// First PUT: archive-based distribution.
	resp1 := putJSON(t, srv.URL+"/api/em-inventory", models.EMConfig{
		Hosts:       hosts,
		ArchivePath: "/tmp/ydb-em.tar.gz",
	})
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first PUT status = %d, want 200", resp1.StatusCode)
	}

	// Second PUT: binary-only distribution (archive removed — XOR).
	resp2 := putJSON(t, srv.URL+"/api/em-inventory", models.EMConfig{
		Hosts:    hosts,
		CPBinary: "/tmp/cp-binary",
	})
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second PUT status = %d, want 200", resp2.StatusCode)
	}
	var body struct {
		Data *models.EMConfig `json:"data"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data == nil {
		t.Fatal("response data is nil")
	}
	if body.Data.ArchivePath != "" {
		t.Errorf("archive_path = %q, want empty after XOR switch to binaries", body.Data.ArchivePath)
	}
	if body.Data.CPBinary != "/tmp/cp-binary" {
		t.Errorf("cp_binary = %q, want /tmp/cp-binary", body.Data.CPBinary)
	}
	// Derived by the writer, present only via file re-parse.
	wantEndpoint := "grpcs://em-1.example.com:2135"
	if body.Data.DbConnectionEndpoint != wantEndpoint {
		t.Errorf("db_connection_endpoint = %q, want %q (proves session re-parsed from file)",
			body.Data.DbConnectionEndpoint, wantEndpoint)
	}
}

// TestEMInventoryHandler_PutDefaultPath covers the fallback branch: when no
// EMInventoryPath is configured the handler writes to
// <cwd>/inventory/90-inventory.yaml.
func TestEMInventoryHandler_PutDefaultPath(t *testing.T) {
	dir := chdirTemp(t)
	if err := os.MkdirAll(filepath.Join(dir, "inventory"), 0o755); err != nil {
		t.Fatal(err)
	}
	store := session.New("en")
	srv := newEMInventoryServer(t, store, "")
	defer srv.Close()

	resp := putJSON(t, srv.URL+"/api/em-inventory", models.EMConfig{
		Hosts: []models.EMHost{{FQDN: "em-1.example.com"}},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	defaultPath := filepath.Join(dir, "inventory", "90-inventory.yaml")
	if _, err := os.Stat(defaultPath); err != nil {
		t.Fatalf("default 90-inventory path not written: %v", err)
	}
}

func TestEMInventoryHandler_PutMalformedJSON(t *testing.T) {
	chdirTemp(t)
	store := session.New("en")
	srv := newEMInventoryServer(t, store, "")
	defer srv.Close()

	resp := putRaw(t, srv.URL+"/api/em-inventory", "{not-json")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestEMInventoryHandler_MethodNotAllowed(t *testing.T) {
	chdirTemp(t)
	store := session.New("en")
	srv := newEMInventoryServer(t, store, "")
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/em-inventory", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
	if got := resp.Header.Get("Allow"); got != "GET, PUT" {
		t.Errorf("Allow = %q, want \"GET, PUT\"", got)
	}
}
