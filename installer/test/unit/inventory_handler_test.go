package unit_test

import (
	"bytes"
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

// chdirTemp moves the test process into a fresh temp directory for the
// duration of the test and restores the original working directory on
// cleanup. Required because handlers.Inventory resolves
// inventory/50-inventory.yaml relative to the process CWD.
//
// Tests in this package never call t.Parallel, so mutating the
// process-wide CWD is safe. Cleanup order (LIFO) guarantees the CWD is
// restored before t.TempDir's removal runs.
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	return dir
}

// putJSON sends a PUT request with a JSON body and returns the response.
func putJSON(t *testing.T, url string, body interface{}) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// postJSON sends a POST request with a JSON body and returns the response.
func postJSON(t *testing.T, url string, body interface{}) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// putRaw sends a PUT request with a raw (possibly malformed) body.
func putRaw(t *testing.T, url string, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// postRaw sends a POST request to base+path with a raw (possibly malformed)
// JSON body.
func postRaw(t *testing.T, path, base, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(base+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// newServerFromHandler wraps an http.Handler in an httptest server.
func newServerFromHandler(h http.Handler) *httptest.Server {
	return httptest.NewServer(h)
}

// newInventoryServer builds a router with the inventory writer wired.
func newInventoryServer(t *testing.T, store *session.Store) *httptest.Server {
	t.Helper()
	h := api.New(api.Options{
		SessionStore:    store,
		OperationStates: models.NewOperationStates(),
		SPA:             nil,
		InventoryWriter: config.NewInventoryWriter(),
	})
	return httptest.NewServer(h)
}

func TestInventoryHandler_GetEmpty(t *testing.T) {
	chdirTemp(t)
	store := session.New("en")
	srv := newInventoryServer(t, store)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/inventory")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		InventoryFound bool                  `json:"inventory_found"`
		ConfigFound    bool                  `json:"config_found"`
		Data           *models.InventoryData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.InventoryFound || body.ConfigFound {
		t.Errorf("found flags = %v/%v, want false/false", body.InventoryFound, body.ConfigFound)
	}
	if body.Data != nil {
		t.Errorf("data = %+v, want nil", body.Data)
	}
}

func TestInventoryHandler_PutWritesFileAndUpdatesSession(t *testing.T) {
	dir := chdirTemp(t)
	store := session.New("en")
	srv := newInventoryServer(t, store)
	defer srv.Close()

	payload := models.InventoryData{
		DC1:      []string{"node-a.example.com"},
		DC2:      []string{"node-b.example.com"},
		DC3:      []string{"node-c.example.com"},
		SSH:      &models.SSHConfig{Login: "root", AuthType: "key"},
		Topology: &models.TopologyConfig{Type: "mirror-3-dc"},
	}

	resp := putJSON(t, srv.URL+"/api/inventory", payload)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var putBody struct {
		Status string                `json:"status"`
		Data   *models.InventoryData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&putBody); err != nil {
		t.Fatal(err)
	}
	if putBody.Status != "ok" {
		t.Errorf("status = %q, want ok", putBody.Status)
	}
	if putBody.Data == nil || len(putBody.Data.DC1) != 1 || putBody.Data.DC1[0] != "node-a.example.com" {
		t.Errorf("response data DC1 = %+v, want [node-a.example.com]", putBody.Data)
	}

	// The file must exist on disk and contain the submitted hosts.
	invFile := filepath.Join(dir, "inventory", "50-inventory.yaml")
	raw, err := os.ReadFile(invFile)
	if err != nil {
		t.Fatalf("inventory file not written: %v", err)
	}
	for _, host := range []string{"node-a.example.com", "node-b.example.com", "node-c.example.com"} {
		if !strings.Contains(string(raw), host) {
			t.Errorf("inventory file missing host %q", host)
		}
	}

	// Session must be updated (frontend GET would see the same data).
	got := store.GetInventory()
	if got == nil || len(got.DC2) != 1 || got.DC2[0] != "node-b.example.com" {
		t.Errorf("session inventory DC2 = %+v, want [node-b.example.com]", got)
	}
	if !store.Get().InventoryFound {
		t.Error("session inventory_found = false, want true after PUT")
	}
}

// TestInventoryHandler_PutPreservesDynamicNodes locks the contract that
// backend-managed dynamic_nodes survive a frontend PUT that omits them.
func TestInventoryHandler_PutPreservesDynamicNodes(t *testing.T) {
	chdirTemp(t)
	store := session.New("en")
	srv := newInventoryServer(t, store)
	defer srv.Close()

	// Seed the session with backend-managed dynamic nodes.
	store.UpdateInventory(&models.InventoryData{
		DC1:          []string{"node-a.example.com"},
		DynamicNodes: []models.DynamicNode{{Instance: "node-a", Offset: 0, Dbname: "user_db"}},
	}, true, false)

	// Frontend PUT without dynamic_nodes.
	resp := putJSON(t, srv.URL+"/api/inventory", models.InventoryData{
		DC1: []string{"node-a.example.com"},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Data *models.InventoryData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data == nil || len(body.Data.DynamicNodes) == 0 {
		t.Fatalf("dynamic_nodes lost after PUT that omitted them: %+v", body.Data)
	}
}

func TestInventoryHandler_PutMalformedJSON(t *testing.T) {
	chdirTemp(t)
	store := session.New("en")
	srv := newInventoryServer(t, store)
	defer srv.Close()

	resp := putRaw(t, srv.URL+"/api/inventory", "{not-json")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

func TestInventoryHandler_MethodNotAllowed(t *testing.T) {
	chdirTemp(t)
	store := session.New("en")
	srv := newInventoryServer(t, store)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/inventory", nil)
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
