package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ydb-installer/internal/api"
	"ydb-installer/internal/api/handlers"
	"ydb-installer/internal/models"
)

// newVersionsServer builds a router for the versions endpoints.
func newVersionsServer(t *testing.T) string {
	t.Helper()
	h := api.New(api.Options{OperationStates: models.NewOperationStates()})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

// stubGitHubAPI replaces handlers.VersionsAPIURL with a local stub for the
// duration of the test and restores the original value on cleanup.
func stubGitHubAPI(t *testing.T, status int, body string) {
	t.Helper()
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(stub.Close)

	old := handlers.VersionsAPIURL
	handlers.VersionsAPIURL = stub.URL
	t.Cleanup(func() { handlers.VersionsAPIURL = old })
}

func TestVersionsHandler_FiltersAndMarksRecommended(t *testing.T) {
	stubGitHubAPI(t, http.StatusOK, `[
		{"tag_name": "25.1.0", "prerelease": false},
		{"tag_name": "25.2.0-alpha", "prerelease": true},
		{"tag_name": "CLI_2.8.0", "prerelease": false},
		{"tag_name": "24.9.9", "prerelease": false}
	]`)

	base := newVersionsServer(t)
	resp, err := http.Get(base + "/api/versions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body handlers.YdbVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	// Prerelease and CLI_ entries are filtered out.
	if len(body.Versions) != 2 {
		t.Fatalf("versions = %+v, want exactly 2 entries", body.Versions)
	}
	if body.Versions[0].Version != "25.1.0" || !body.Versions[0].Recommended {
		t.Errorf("first version = %+v, want 25.1.0 recommended", body.Versions[0])
	}
	if body.Versions[1].Version != "24.9.9" || body.Versions[1].Recommended {
		t.Errorf("second version = %+v, want 24.9.9 not recommended", body.Versions[1])
	}
}

func TestVersionsHandler_UpstreamErrorReturns502(t *testing.T) {
	stubGitHubAPI(t, http.StatusForbidden, `{"message":"rate limited"}`)

	base := newVersionsServer(t)
	resp, err := http.Get(base + "/api/versions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
}

func TestVersionsHandler_MethodNotAllowed(t *testing.T) {
	base := newVersionsServer(t)
	resp := postJSON(t, base+"/api/versions", map[string]string{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestValidateFileHandler_Variants(t *testing.T) {
	dir := chdirTemp(t)
	base := newVersionsServer(t)

	// A real file → valid with a formatted size.
	filePath := filepath.Join(dir, "pkg.tar.gz")
	if err := os.WriteFile(filePath, make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		path      string
		wantValid bool
	}{
		{name: "existing absolute file", path: filePath, wantValid: true},
		{name: "existing relative file", path: "pkg.tar.gz", wantValid: true},
		{name: "missing file", path: filepath.Join(dir, "nope.tar.gz"), wantValid: false},
		{name: "directory instead of file", path: dir, wantValid: false},
		{name: "empty path", path: "", wantValid: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := postJSON(t, base+"/api/versions/validate-file",
				handlers.ValidateFileRequest{Path: tc.path})
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			var body handlers.ValidateFileResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Valid != tc.wantValid {
				t.Errorf("valid = %v, want %v (error=%q)", body.Valid, tc.wantValid, body.Error)
			}
			if tc.wantValid && body.Size == "" {
				t.Error("expected non-empty size for a valid file")
			}
		})
	}
}

func TestValidateFileHandler_MethodNotAllowed(t *testing.T) {
	base := newVersionsServer(t)
	resp, err := http.Get(base + "/api/versions/validate-file")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestValidateFileHandler_MalformedJSON(t *testing.T) {
	base := newVersionsServer(t)
	resp := postRaw(t, "/api/versions/validate-file", base, "{not-json")
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
