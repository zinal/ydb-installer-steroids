// Package handlers contains the HTTP handler functions exposed by the
// installer's REST API.
package handlers

import (
	"encoding/json"
	"net/http"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/session"
)

// Session returns a handler for the /api/session resource.
//
//   - GET    → returns the current session as JSON.
//   - PUT    → applies a SessionUpdate and returns the updated session.
//   - DELETE → resets the session.
func Session(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, store.Get())
		case http.MethodPut, http.MethodPatch:
			var upd models.SessionUpdate
			if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
				return
			}
			if upd.Flow != nil && !models.IsValidFlow(*upd.Flow) {
				writeError(w, http.StatusBadRequest, "unknown flow: "+string(*upd.Flow))
				return
			}
			if upd.Inventory != nil {
				store.UpdateInventory(upd.Inventory, true, true)
			}
			writeJSON(w, http.StatusOK, store.Update(upd))
		case http.MethodDelete:
			store.Reset()
			writeJSON(w, http.StatusOK, store.Get())
		default:
			w.Header().Set("Allow", "GET, PUT, PATCH, DELETE")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
