// Package handlers provides HTTP handlers for the YDB Installer API.
package handlers

import (
	"encoding/json"
	"net/http"

	"ydb-installer/internal/services/tls"
)

// TLSGenerator is the certificate generation service.
var TLSGenerator = tls.NewGenerator()

// TLSChecker is the certificate validation service.
var TLSChecker = tls.NewChecker()

// tlsGenerateRequest is the request body for POST /api/tls/generate.
type tlsGenerateRequest struct {
	Hosts []string `json:"hosts"`
	Days  int      `json:"days,omitempty"`
	Force bool     `json:"force,omitempty"`
}

// tlsGenerateResponse is the response for POST /api/tls/generate.
type tlsGenerateResponse struct {
	Success   bool              `json:"success"`
	OutputDir string            `json:"output_dir,omitempty"`
	CAFile    string            `json:"ca_file,omitempty"`
	NodeFiles []tlsNodeFileInfo `json:"node_files,omitempty"`
	Error     string            `json:"error,omitempty"`
	ErrorCode string            `json:"error_code,omitempty"`
	Existing  []string          `json:"existing,omitempty"`
}

// tlsExistsRequest is the request body for POST /api/tls/exists.
type tlsExistsRequest struct {
	CertPath string   `json:"cert_path,omitempty"`
	Hosts    []string `json:"hosts"`
}

// tlsExistsResponse is the response for POST /api/tls/exists.
type tlsExistsResponse struct {
	Exists        bool     `json:"exists"`
	Path          string   `json:"path,omitempty"`
	Reasons       []string `json:"reasons,omitempty"`
	CAPresent     bool     `json:"ca_present"`
	HostConflicts []string `json:"host_conflicts,omitempty"`
	BlockGenerate bool     `json:"block_generate"`
}

// tlsNodeFileInfo represents certificate and key files for a single node.
type tlsNodeFileInfo struct {
	Hostname  string `json:"hostname"`
	ShortName string `json:"short_name"`
	CertFile  string `json:"cert_file"`
	KeyFile   string `json:"key_file"`
	WebPem    string `json:"web_pem"`
}

// tlsValidateRequest is the request body for POST /api/tls/validate.
type tlsValidateRequest struct {
	CertPath string   `json:"cert_path"`
	Hosts    []string `json:"hosts"`
}

// tlsValidateResponse is the response for POST /api/tls/validate.
type tlsValidateResponse struct {
	Valid      bool                `json:"valid"`
	CAValid    bool                `json:"ca_valid"`
	NodeFiles  []tlsNodeValidation `json:"node_files"`
	Missing    []string            `json:"missing"`
	Errors     []string            `json:"errors"`
	TotalNodes int                 `json:"total_nodes"`
	FoundNodes int                 `json:"found_nodes"`
}

// tlsNodeValidation represents the validation status of a node's certificates.
// Only the 3 files required by YDB are checked: node.crt, node.key, and their pair match.
type tlsNodeValidation struct {
	Index     int    `json:"index"`
	Hostname  string `json:"hostname"`
	ShortName string `json:"short_name"`
	KeyFound  bool   `json:"key_found"`
	CertFound bool   `json:"cert_found"`
	CertValid bool   `json:"cert_valid"`
	KeyValid  bool   `json:"key_valid"`
	PairMatch bool   `json:"pair_match"`
	Match     bool   `json:"match"`
	Error     string `json:"error,omitempty"`
}

// HandleTLGenerate handles POST /api/tls/generate.
//
// Generates self-signed TLS certificates for the provided hosts using the
// per-hostname directory structure expected by Ansible playbooks.
// The output directory is always fixed at ./TLS/CA/certs/certs-per-host
// relative to the working directory.
func HandleTLGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tlsGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Hosts) == 0 {
		sendJSONError(w, "Hosts are required", http.StatusBadRequest)
		return
	}

	result := TLSGenerator.Generate(tls.GenerateRequest{
		Hosts: req.Hosts,
		Days:  req.Days,
		Force: req.Force,
	})

	resp := tlsGenerateResponse{
		Success:   result.Success,
		OutputDir: result.OutputDir,
		CAFile:    result.CAFile,
		Error:     result.Error,
		ErrorCode: result.ErrorCode,
		Existing:  result.Existing,
	}

	for _, nf := range result.NodeFiles {
		resp.NodeFiles = append(resp.NodeFiles, tlsNodeFileInfo{
			Hostname:  nf.Hostname,
			ShortName: nf.ShortName,
			CertFile:  nf.CertFile,
			KeyFile:   nf.KeyFile,
			WebPem:    nf.WebPem,
		})
	}

	sendJSON(w, resp, http.StatusOK)
}

// HandleTLSExists handles POST /api/tls/exists.
//
// Reports whether generating certs for the requested hosts would overwrite
// existing per-host material (host-only block, Option E).
// CA presence is informational only and does not block generate.
func HandleTLSExists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tlsExistsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result := tls.CheckExistingCerts(req.CertPath, req.Hosts)
	sendJSON(w, tlsExistsResponse{
		Exists:        result.Exists,
		Path:          result.Path,
		Reasons:       result.Reasons,
		CAPresent:     result.CAPresent,
		HostConflicts: result.HostConflicts,
		BlockGenerate: result.BlockGenerate,
	}, http.StatusOK)
}

// HandleTLValidate handles POST /api/tls/validate.
//
// Validates existing certificates in the certs-per-host directory structure.
// For each host it checks the 3 files required by YDB:
//   - <certPath>/ca.crt                          — CA certificate
//   - <certPath>/<shortName>/node.crt             — node certificate
//   - <certPath>/<shortName>/node.key             — private key (matched against node.crt)
func HandleTLValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tlsValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CertPath == "" {
		sendJSONError(w, "Certificate path is required", http.StatusBadRequest)
		return
	}

	result := TLSChecker.Validate(tls.ValidateRequest{
		CertPath: req.CertPath,
		Hosts:    req.Hosts,
	})

	resp := tlsValidateResponse{
		Valid:      result.Valid,
		CAValid:    result.CAValid,
		Missing:    result.Missing,
		Errors:     result.Errors,
		TotalNodes: result.TotalNodes,
		FoundNodes: result.FoundNodes,
	}

	for _, nv := range result.NodeFiles {
		resp.NodeFiles = append(resp.NodeFiles, tlsNodeValidation{
			Index:     nv.Index,
			Hostname:  nv.Hostname,
			ShortName: nv.ShortName,
			KeyFound:  nv.KeyFound,
			CertFound: nv.CertFound,
			CertValid: nv.CertValid,
			KeyValid:  nv.KeyValid,
			PairMatch: nv.PairMatch,
			Match:     nv.Match,
			Error:     nv.Error,
		})
	}

	sendJSON(w, resp, http.StatusOK)
}

// sendJSON sends a JSON response with the given status code.
func sendJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// sendJSONError sends a JSON error response.
func sendJSONError(w http.ResponseWriter, message string, status int) {
	sendJSON(w, map[string]string{"error": message}, status)
}
