package handlers

import (
	"net/http"
)

// RuntimeInfo is the JSON body for GET /api/system/runtime.
type RuntimeInfo struct {
	// Embedded is true when the binary ships a bundled Python+Ansible runtime.
	Embedded bool `json:"embedded"`
	// Extracted is true when the bundled runtime has been extracted and is
	// ready to use on this machine.
	Extracted bool `json:"extracted"`
	// Python is the bundled CPython version (empty if not embedded).
	Python string `json:"python,omitempty"`
	// Ansible is the bundled Ansible version (empty if not embedded).
	Ansible string `json:"ansible,omitempty"`
	// AnsibleCore is the bundled ansible-core version.
	AnsibleCore string `json:"ansible_core,omitempty"`
	// Collection is the bundled ydb_platform.ydb collection version.
	Collection string `json:"collection,omitempty"`
	// EMCollection is the bundled ydb_platform.ydb_em collection version
	// (used by the YDB EM install flow).
	EMCollection string `json:"em_collection,omitempty"`
	// ShowEMScenarios enables the YDB EM installation scenarios (ydb-em
	// wizard and EM delete) in the UI. Off by default, hiding their entry points.
	ShowEMScenarios bool `json:"show_em_scenarios"`
	// Path is the resolved runtime root on disk (empty if not extracted).
	Path string `json:"path,omitempty"`
	// Error carries a non-fatal extraction/resolution error message, if any.
	Error string `json:"error,omitempty"`
}

// SystemRuntime handles GET /api/system/runtime.
//
// The handler is given a snapshot of the runtime state assembled at startup, so
// it performs no extraction work itself (extraction happens once in main).
func SystemRuntime(info RuntimeInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}
