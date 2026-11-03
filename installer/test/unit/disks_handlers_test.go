package unit_test

import (
	"net/http"
	"testing"

	"ydb-installer/internal/api"
	"ydb-installer/internal/models"
)

// newDisksServer builds a router for the disks endpoints.
func newDisksServer(t *testing.T) string {
	t.Helper()
	h := api.New(api.Options{OperationStates: models.NewOperationStates()})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestDisksHandlers_MethodNotAllowed(t *testing.T) {
	base := newDisksServer(t)
	for _, path := range []string{"/api/disks/check", "/api/disks/validate"} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("GET %s status = %d, want 405", path, resp.StatusCode)
		}
	}
}

func TestDisksHandlers_MalformedJSON(t *testing.T) {
	base := newDisksServer(t)
	for _, path := range []string{"/api/disks/check", "/api/disks/validate"} {
		resp := postRaw(t, path, base, "{not-json")
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("POST %s malformed status = %d, want 400", path, resp.StatusCode)
		}
	}
}

func TestCheckDisksHandler_NoHosts(t *testing.T) {
	base := newDisksServer(t)
	resp := postJSON(t, base+"/api/disks/check", map[string]interface{}{"hosts": []string{}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestValidateDisksHandler_NoHosts(t *testing.T) {
	base := newDisksServer(t)
	resp := postJSON(t, base+"/api/disks/validate", map[string]interface{}{
		"hosts": []string{},
		"paths": []string{"/dev/sda"},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestValidateDisksHandler_NoPaths(t *testing.T) {
	base := newDisksServer(t)
	resp := postJSON(t, base+"/api/disks/validate", map[string]interface{}{
		"hosts": []string{"node-a.example.com"},
		"paths": []string{},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
