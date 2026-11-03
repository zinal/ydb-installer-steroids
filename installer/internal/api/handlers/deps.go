package handlers

import (
	"net/http"

	"ydb-installer/internal/services/checker"
)

// DepsResponse is the body returned by GET /api/system/deps.
type DepsResponse struct {
	Dependencies []checker.Dependency `json:"dependencies"`
	// Bundled is true when the dependencies are satisfied by the embedded
	// runtime rather than probed from the host system.
	Bundled bool `json:"bundled"`
}

// SystemDeps handles GET /api/system/deps.
//
// When bundled is non-nil, the three dependencies are reported as satisfied by
// the embedded runtime. Otherwise the real host probes run (python3, ansible,
// ansible-galaxy).
func SystemDeps(bundled *checker.BundledInfo) http.HandlerFunc {
	c := &checker.DepsChecker{Bundled: bundled}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, DepsResponse{
			Dependencies: c.Check(),
			Bundled:      bundled != nil,
		})
	}
}

// SystemConfigs handles GET /api/system/configs.
// Returns a stub config-discovery report (real implementation in a later stage).
func SystemConfigs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"inventory": map[string]any{"found": false, "path": ""},
			"config":    map[string]any{"found": false, "path": ""},
		})
	}
}
