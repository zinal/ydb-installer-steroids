package unit_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"ydb-installer/internal/api"
	"ydb-installer/internal/models"
)

// newTLSServer builds a minimal router for the TLS endpoints.
func newTLSServer(t *testing.T) string {
	t.Helper()
	h := api.New(api.Options{
		OperationStates: models.NewOperationStates(),
	})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestTLSHandlers_GenerateRoundTrip(t *testing.T) {
	dir := chdirTemp(t)
	base := newTLSServer(t)

	hosts := []string{"static-node-1", "static-node-2"}
	resp := postJSON(t, base+"/api/tls/generate", map[string]interface{}{
		"hosts": hosts,
		"days":  30,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var gen struct {
		Success   bool   `json:"success"`
		OutputDir string `json:"output_dir"`
		CAFile    string `json:"ca_file"`
		NodeFiles []struct {
			Hostname string `json:"hostname"`
			CertFile string `json:"cert_file"`
			KeyFile  string `json:"key_file"`
		} `json:"node_files"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gen); err != nil {
		t.Fatal(err)
	}
	if !gen.Success {
		t.Fatalf("generate failed: %s", gen.Error)
	}
	if len(gen.NodeFiles) != len(hosts) {
		t.Fatalf("node_files = %d, want %d", len(gen.NodeFiles), len(hosts))
	}

	// Per-host material must exist on disk under the fixed output dir.
	certsRoot := filepath.Join(dir, "TLS", "CA", "certs", "certs-per-host")
	for _, host := range hosts {
		if _, err := os.Stat(filepath.Join(certsRoot, host, "node.crt")); err != nil {
			t.Errorf("node.crt for %s missing: %v", host, err)
		}
		if _, err := os.Stat(filepath.Join(certsRoot, host, "node.key")); err != nil {
			t.Errorf("node.key for %s missing: %v", host, err)
		}
	}

	// exists → must now report conflicts for the same hosts.
	respExists := postJSON(t, base+"/api/tls/exists", map[string]interface{}{
		"cert_path": certsRoot,
		"hosts":     hosts,
	})
	defer respExists.Body.Close()
	if respExists.StatusCode != http.StatusOK {
		t.Fatalf("exists status = %d, want 200", respExists.StatusCode)
	}
	var exists struct {
		Exists        bool     `json:"exists"`
		BlockGenerate bool     `json:"block_generate"`
		HostConflicts []string `json:"host_conflicts"`
	}
	if err := json.NewDecoder(respExists.Body).Decode(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists.Exists || !exists.BlockGenerate {
		t.Errorf("exists/block = %v/%v, want true/true", exists.Exists, exists.BlockGenerate)
	}
	// Conflicts are reported per file (node.crt + node.key per host).
	if len(exists.HostConflicts) != 2*len(hosts) {
		t.Errorf("host_conflicts = %v, want %d entries", exists.HostConflicts, 2*len(hosts))
	}

	// validate → freshly generated certs must be valid.
	respValidate := postJSON(t, base+"/api/tls/validate", map[string]interface{}{
		"cert_path": certsRoot,
		"hosts":     hosts,
	})
	defer respValidate.Body.Close()
	if respValidate.StatusCode != http.StatusOK {
		t.Fatalf("validate status = %d, want 200", respValidate.StatusCode)
	}
	var validate struct {
		Valid      bool `json:"valid"`
		CAValid    bool `json:"ca_valid"`
		TotalNodes int  `json:"total_nodes"`
		FoundNodes int  `json:"found_nodes"`
	}
	if err := json.NewDecoder(respValidate.Body).Decode(&validate); err != nil {
		t.Fatal(err)
	}
	if !validate.Valid || !validate.CAValid {
		t.Errorf("valid/ca_valid = %v/%v, want true/true", validate.Valid, validate.CAValid)
	}
	if validate.TotalNodes != len(hosts) || validate.FoundNodes != len(hosts) {
		t.Errorf("nodes = %d/%d, want %d/%d",
			validate.FoundNodes, validate.TotalNodes, len(hosts), len(hosts))
	}
}

func TestTLSHandlers_GenerateEmptyHosts(t *testing.T) {
	base := newTLSServer(t)
	resp := postJSON(t, base+"/api/tls/generate", map[string]interface{}{"hosts": []string{}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestTLSHandlers_ValidateMissingPath(t *testing.T) {
	base := newTLSServer(t)
	resp := postJSON(t, base+"/api/tls/validate", map[string]interface{}{"hosts": []string{"a"}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestTLSHandlers_MethodNotAllowed(t *testing.T) {
	base := newTLSServer(t)
	for _, path := range []string{"/api/tls/generate", "/api/tls/validate", "/api/tls/exists"} {
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

func TestTLSHandlers_GenerateForce(t *testing.T) {
	dir := chdirTemp(t)
	base := newTLSServer(t)

	hosts := []string{"static-node-1", "static-node-2"}

	resp1 := postJSON(t, base+"/api/tls/generate", map[string]interface{}{
		"hosts": hosts,
		"days":  30,
	})
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first generate status = %d, want 200", resp1.StatusCode)
	}

	caKeyPath := filepath.Join(dir, "TLS", "CA", "secure", "ca.key")
	origKey, err := os.ReadFile(caKeyPath)
	if err != nil {
		t.Fatalf("failed to read CA key after first generate: %v", err)
	}

	// Step 2: Regenerate same hosts with force=true — must succeed (200).
	resp2 := postJSON(t, base+"/api/tls/generate", map[string]interface{}{
		"hosts": hosts,
		"days":  30,
		"force": true,
	})
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("force generate status = %d, want 200", resp2.StatusCode)
	}

	var gen struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&gen); err != nil {
		t.Fatal(err)
	}
	if !gen.Success {
		t.Fatalf("force generate failed: %s", gen.Error)
	}

	certsRoot := filepath.Join(dir, "TLS", "CA", "certs", "certs-per-host")
	for _, host := range hosts {
		if _, err := os.Stat(filepath.Join(certsRoot, host, "node.crt")); err != nil {
			t.Errorf("node.crt for %s missing after force: %v", host, err)
		}
		if _, err := os.Stat(filepath.Join(certsRoot, host, "node.key")); err != nil {
			t.Errorf("node.key for %s missing after force: %v", host, err)
		}
	}

	newKey, err := os.ReadFile(caKeyPath)
	if err != nil {
		t.Fatalf("failed to read CA key after force: %v", err)
	}
	if string(newKey) == string(origKey) {
		t.Error("expected CA private key to change after force regeneration")
	}
}

func TestTLSHandlers_GenerateWithoutForceBlocks(t *testing.T) {
	chdirTemp(t)
	base := newTLSServer(t)

	hosts := []string{"node1"}

	resp1 := postJSON(t, base+"/api/tls/generate", map[string]interface{}{
		"hosts": hosts,
	})
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first generate status = %d, want 200", resp1.StatusCode)
	}

	// Step 2: Regenerate without force — must be blocked (409 or 200 with success=false).
	resp2 := postJSON(t, base+"/api/tls/generate", map[string]interface{}{
		"hosts": hosts,
	})
	defer resp2.Body.Close()

	var gen struct {
		Success   bool   `json:"success"`
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&gen); err != nil {
		t.Fatal(err)
	}
	if gen.Success {
		t.Fatal("expected generate without force to be blocked when host certs exist")
	}
	if gen.ErrorCode != "certs_already_exist" {
		t.Errorf("expected error_code=certs_already_exist, got %s", gen.ErrorCode)
	}
}
