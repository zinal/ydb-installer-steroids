package unit_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ydb-installer/internal/api"
	"ydb-installer/internal/models"
	"ydb-installer/internal/services/session"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := session.New("en")
	h := api.New(api.Options{
		SessionStore:    store,
		OperationStates: models.NewOperationStates(),
		SPA:             nil,
	})
	return httptest.NewServer(h)
}

func TestGetSession(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/session")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got models.Session
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.DefaultLang != "en" {
		t.Errorf("default_lang = %q, want en", got.DefaultLang)
	}
	if got.CurrentStep != "1" {
		t.Errorf("current_step = %q, want 1", got.CurrentStep)
	}
}

func TestPutSession_FlowSwitch(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{"flow": "ydb"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got models.Session
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Flow != models.FlowYDB {
		t.Errorf("flow = %q, want ydb", got.Flow)
	}
	if len(got.Steps) != 11 {
		t.Errorf("ydb flow should have 11 steps, got %d", len(got.Steps))
	}
}

func TestPutSession_UIStateRoundTrip(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	payload := map[string]any{
		"ui_state": map[string]any{
			"current_step":     "5",
			"visited":          []string{"1", "2", "3", "4", "5"},
			"errored":          []string{"3"},
			"show_em_step":     true,
			"config_generated": false,
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got models.Session
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.UIState == nil {
		t.Fatal("expected ui_state in response")
	}
	if got.UIState.CurrentStep != "5" {
		t.Errorf("ui_state.current_step = %q, want 5", got.UIState.CurrentStep)
	}
	if len(got.UIState.Visited) != 5 {
		t.Errorf("ui_state.visited len = %d, want 5", len(got.UIState.Visited))
	}
	if !got.UIState.ShowEMStep {
		t.Error("ui_state.show_em_step should be true")
	}

	// GET must return the same state — verifying it's actually persisted, not
	// just echoed back from the PUT.
	resp2, err := http.Get(srv.URL + "/api/session")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var got2 models.Session
	if err := json.NewDecoder(resp2.Body).Decode(&got2); err != nil {
		t.Fatal(err)
	}
	if got2.UIState == nil || got2.UIState.CurrentStep != "5" {
		t.Errorf("GET after PUT should retain ui_state, got %+v", got2.UIState)
	}
}

func TestPutSession_RejectsUnknownFlow(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body := strings.NewReader(`{"flow":"bogus"}`)
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/session", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestPutSession_RejectsMalformedJSON(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/session", strings.NewReader("not-json"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSession_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/session", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
	if got := resp.Header.Get("Allow"); !strings.Contains(got, "GET") {
		t.Errorf("Allow header should advertise GET, got %q", got)
	}
}

// TestSystemDeps verifies the /api/system/deps endpoint returns 4 dependencies
// (python, ansible, ydb_platform.ydb, ydb_platform.ydb_em).
// The real DepsChecker is used — it probes the system, so we only check shape.
func TestSystemDeps_Shape(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/system/deps")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var got struct {
		Dependencies []struct {
			Name      string `json:"name"`
			OK        bool   `json:"ok"`
			Installed bool   `json:"installed"`
		} `json:"dependencies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Dependencies) != 4 {
		t.Errorf("expected 4 deps, got %d", len(got.Dependencies))
	}
	wantNames := map[string]bool{
		"python": false, "ansible": false,
		"ydb_platform.ydb": false, "ydb_platform.ydb_em": false,
	}
	for _, d := range got.Dependencies {
		if _, ok := wantNames[d.Name]; ok {
			wantNames[d.Name] = true
		}
	}
	for n, seen := range wantNames {
		if !seen {
			t.Errorf("missing dep %q", n)
		}
	}
}

func TestSystemConfigsShape(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/system/configs")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["inventory"]; !ok {
		t.Error("response missing 'inventory' key")
	}
	if _, ok := got["config"]; !ok {
		t.Error("response missing 'config' key")
	}
}

func TestInstallStatus_ViaRouter(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/install/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "idle" {
		t.Errorf("status = %v, want idle", got["status"])
	}
}

func TestSPA_FallbackPlaceholderWhenNoBundle(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	for _, path := range []string{"/", "/screen/3", "/anything/here"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := readAll(resp)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("path=%s status=%d", path, resp.StatusCode)
		}
		if !strings.Contains(body, "YDB Installer") {
			t.Errorf("path=%s placeholder body missing marker", path)
		}
	}
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func readAll(resp *http.Response) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	return buf.String(), err
}
