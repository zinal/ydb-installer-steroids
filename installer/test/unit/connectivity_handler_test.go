package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ydb-installer/internal/api"
	"ydb-installer/internal/api/handlers"
	"ydb-installer/internal/models"
)

// stubConnectivityAPI replaces handlers.ConnectivityAPIURL with a local stub
// for the duration of the test and restores the original value on cleanup.
func stubConnectivityAPI(t *testing.T, status int) {
	t.Helper()
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	t.Cleanup(stub.Close)

	old := handlers.ConnectivityAPIURL
	handlers.ConnectivityAPIURL = stub.URL
	t.Cleanup(func() { handlers.ConnectivityAPIURL = old })
}

func TestConnectivityHandler_Online(t *testing.T) {
	stubConnectivityAPI(t, http.StatusOK)

	h := api.New(api.Options{OperationStates: models.NewOperationStates()})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/system/connectivity")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body handlers.ConnectivityResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.Online {
		t.Errorf("online = false, want true")
	}
	if body.CheckedAt == "" {
		t.Errorf("checked_at is empty")
	}
}

func TestConnectivityHandler_OfflineOnNon2xx(t *testing.T) {
	stubConnectivityAPI(t, http.StatusForbidden)

	h := api.New(api.Options{OperationStates: models.NewOperationStates()})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/system/connectivity")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body handlers.ConnectivityResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Online {
		t.Errorf("online = true, want false")
	}
	if body.Error == "" {
		t.Errorf("error is empty, want a reason")
	}
}

func TestConnectivityHandler_OfflineOnNetworkError(t *testing.T) {
	// Point the probe at a closed port so the request fails at the network layer.
	old := handlers.ConnectivityAPIURL
	handlers.ConnectivityAPIURL = "http://127.0.0.1:1"
	t.Cleanup(func() { handlers.ConnectivityAPIURL = old })

	h := api.New(api.Options{OperationStates: models.NewOperationStates()})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/system/connectivity")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body handlers.ConnectivityResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Online {
		t.Errorf("online = true, want false")
	}
	if body.Error == "" {
		t.Errorf("error is empty, want a reason")
	}
}

func TestConnectivityHandler_MethodNotAllowed(t *testing.T) {
	h := api.New(api.Options{OperationStates: models.NewOperationStates()})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)

	resp := postJSON(t, srv.URL+"/api/system/connectivity", map[string]string{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}
