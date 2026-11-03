package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ydb-installer/internal/api"
	"ydb-installer/internal/api/handlers"
	"ydb-installer/internal/models"
	"ydb-installer/internal/services/checker"
	"ydb-installer/internal/services/session"
)

// newRuntimeTestServer builds a server with explicit runtime/deps options.
func newRuntimeTestServer(t *testing.T, info handlers.RuntimeInfo, bundled *checker.BundledInfo) *httptest.Server {
	t.Helper()
	h := api.New(api.Options{
		SessionStore:     session.New("en"),
		OperationStates:  models.NewOperationStates(),
		SPA:              nil,
		RuntimeInfo:      info,
		BundledDeps:      bundled,
	})
	return httptest.NewServer(h)
}

func TestSystemRuntime_Shape(t *testing.T) {
	info := handlers.RuntimeInfo{
		Embedded:    true,
		Extracted:   true,
		Python:      "3.12.5",
		Ansible:     "10.3.0",
		AnsibleCore: "2.17.3",
		Collection:  "1.2.3",
		Path:        "/tmp/rt/python",
	}
	srv := newRuntimeTestServer(t, info, nil)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/system/runtime")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var got handlers.RuntimeInfo
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Embedded || !got.Extracted {
		t.Error("expected embedded+extracted true")
	}
	if got.Python != "3.12.5" || got.Ansible != "10.3.0" || got.Collection != "1.2.3" {
		t.Errorf("version mismatch: %+v", got)
	}
	if got.ShowEMScenarios {
		t.Error("expected show_em_scenarios to default false")
	}
}

func TestSystemRuntime_ShowEMScenarios(t *testing.T) {
	info := handlers.RuntimeInfo{
		Embedded:        true,
		Extracted:       true,
		ShowEMScenarios: true,
	}
	srv := newRuntimeTestServer(t, info, nil)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/system/runtime")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var got handlers.RuntimeInfo
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.ShowEMScenarios {
		t.Error("expected show_em_scenarios true")
	}
}

func TestSystemRuntime_NotEmbedded(t *testing.T) {
	srv := newRuntimeTestServer(t, handlers.RuntimeInfo{Embedded: false}, nil)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/system/runtime")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var got handlers.RuntimeInfo
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Embedded {
		t.Error("expected embedded=false")
	}
}

func TestSystemRuntime_MethodNotAllowed(t *testing.T) {
	srv := newRuntimeTestServer(t, handlers.RuntimeInfo{}, nil)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/system/runtime", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestSystemDeps_BundledFlag(t *testing.T) {
	bundled := &checker.BundledInfo{Python: "3.12.5", Ansible: "10.3.0", Collection: "1.2.3"}
	srv := newRuntimeTestServer(t, handlers.RuntimeInfo{Embedded: true, Extracted: true}, bundled)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/system/deps")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var got struct {
		Dependencies []checker.Dependency `json:"dependencies"`
		Bundled      bool                 `json:"bundled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Bundled {
		t.Error("expected bundled=true in deps response")
	}
	for _, d := range got.Dependencies {
		if d.Source != checker.SourceBundled {
			t.Errorf("dep %q source = %q, want bundled", d.Name, d.Source)
		}
	}
}
