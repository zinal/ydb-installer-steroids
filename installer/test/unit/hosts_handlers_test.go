package unit_test

import (
	"net/http"
	"testing"

	"ydb-installer/internal/api"
	"ydb-installer/internal/models"
)

// newHostsServer builds a router for the hosts endpoints.
func newHostsServer(t *testing.T) string {
	t.Helper()
	h := api.New(api.Options{OperationStates: models.NewOperationStates()})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestCheckHostsHandler_MethodNotAllowed(t *testing.T) {
	base := newHostsServer(t)
	resp, err := http.Get(base + "/api/hosts/check")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestCheckHostsHandler_MalformedJSON(t *testing.T) {
	base := newHostsServer(t)
	resp := postRaw(t, "/api/hosts/check", base, "{not-json")
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCheckHostsHandler_NoHosts(t *testing.T) {
	base := newHostsServer(t)
	resp := postJSON(t, base+"/api/hosts/check", map[string]interface{}{"hosts": []string{}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
