package handlers

import (
	"encoding/json"
	"net/http"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/hosts"
)

// HandleCheckHosts handles POST /api/hosts/check
func HandleCheckHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CheckHostsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Hosts) == 0 {
		http.Error(w, "No hosts provided", http.StatusBadRequest)
		return
	}

	checker := hosts.NewChecker()
	response, err := checker.CheckHosts(&req)
	if err != nil {
		http.Error(w, "Failed to check hosts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
