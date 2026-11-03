package handlers

import (
	"encoding/json"
	"net/http"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/disks"
)

// HandleCheckDisks handles POST /api/disks/check
func HandleCheckDisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CheckDisksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Hosts) == 0 {
		http.Error(w, "No hosts provided", http.StatusBadRequest)
		return
	}

	checker := disks.NewChecker()
	response, err := checker.CheckDisks(&req)
	if err != nil {
		http.Error(w, "Failed to check disks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleValidateDisks handles POST /api/disks/validate
func HandleValidateDisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ValidateDisksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Hosts) == 0 {
		http.Error(w, "No hosts provided", http.StatusBadRequest)
		return
	}

	if len(req.Paths) == 0 {
		http.Error(w, "No paths provided", http.StatusBadRequest)
		return
	}

	checker := disks.NewChecker()
	response, err := checker.ValidateDisks(&req)
	if err != nil {
		http.Error(w, "Failed to validate disks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
