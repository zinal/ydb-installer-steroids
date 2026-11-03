package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/config"
	"ydb-installer/internal/services/session"
)

const emInventoryPath = "inventory/90-inventory.yaml"

// EMInventory handles GET/PUT /api/em-inventory.
//
//   - GET  → returns the current EM config data from the session.
//   - PUT  → applies EM config updates, writes to 90-inventory.yaml,
//     re-parses the file into the session (file is source of truth), and returns it.
func EMInventory(store *session.Store, writer *config.EMInventoryWriter, emInvPath string) http.HandlerFunc {
	parser := config.NewEMInventoryService()
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			em := store.GetEMConfig()
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"em_inventory_found": store.GetEMInventoryFound(),
				"data":               em,
			})

		case http.MethodPut:
			var data models.EMConfig
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
				return
			}

			path := emInvPath
			if path == "" {
				cwd, _ := os.Getwd()
				path = filepath.Join(cwd, emInventoryPath)
			}

			if err := writer.WriteWithPatch(path, &data); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to write EM inventory: "+err.Error())
				return
			}

			// Re-parse so session matches file (XOR deletes, no phantom defaults).
			parsed, err := parser.Parse(path)
			if err != nil {
				// Write succeeded; fall back to request body if parse fails.
				store.SetEMConfig(&data, true)
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"status": "ok",
					"data":   store.GetEMConfig(),
				})
				return
			}
			store.SetEMConfig(parsed, true)

			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status": "ok",
				"data":   store.GetEMConfig(),
			})

		default:
			w.Header().Set("Allow", "GET, PUT")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}
