package handlers

import (
	"log"
	"net/http"
	"time"
)

// ConnectivityResponse is the body returned by GET /api/system/connectivity.
type ConnectivityResponse struct {
	Online    bool   `json:"online"`
	CheckedAt string `json:"checked_at"`
	Error     string `json:"error,omitempty"`
}

// ConnectivityAPIURL is the endpoint probed to detect internet access.
// It is a variable (not a constant) so tests can point it at a stub server;
// production behavior is unchanged.
var ConnectivityAPIURL = "https://api.github.com"

// HandleConnectivity probes a stable endpoint to detect whether the installer
// machine has internet access. Any 2xx response counts as online; network
// errors, timeouts, and non-2xx statuses count as offline.
func HandleConnectivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ConnectivityAPIURL)
	if err != nil {
		log.Printf("[HandleConnectivity] Probe failed: %v", err)
		writeJSON(w, http.StatusOK, ConnectivityResponse{
			Online:    false,
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
			Error:     err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[HandleConnectivity] Probe returned status %d", resp.StatusCode)
		writeJSON(w, http.StatusOK, ConnectivityResponse{
			Online:    false,
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
			Error:     "probe returned status " + http.StatusText(resp.StatusCode),
		})
		return
	}

	writeJSON(w, http.StatusOK, ConnectivityResponse{
		Online:    true,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	})
}
