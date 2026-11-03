// Package handlers provides HTTP handlers for the YDB Installer API.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// YdbVersion represents a single available YDB version.
type YdbVersion struct {
	Version     string `json:"version"`
	Recommended bool   `json:"recommended"`
}

// YdbVersionsResponse is the response for GET /api/versions.
type YdbVersionsResponse struct {
	Versions []YdbVersion `json:"versions"`
}

// ValidateFileRequest is the request body for POST /api/versions/validate-file.
type ValidateFileRequest struct {
	Path string `json:"path"`
}

// ValidateFileResponse is the response for POST /api/versions/validate-file.
type ValidateFileResponse struct {
	Valid bool   `json:"valid"`
	Size  string `json:"size,omitempty"`
	Error string `json:"error,omitempty"`
}

// VersionsAPIURL is the GitHub releases endpoint used by HandleVersions.
// It is a variable (not a constant) so tests can point it at a stub server;
// production behavior is unchanged.
var VersionsAPIURL = "https://api.github.com/repos/ydb-platform/ydb/releases?per_page=20"

// HandleVersions fetches YDB versions from GitHub releases API.
func HandleVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[HandleVersions] Fetching YDB versions from GitHub...")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(VersionsAPIURL)
	if err != nil {
		log.Printf("[HandleVersions] Failed to fetch from GitHub: %v", err)
		http.Error(w, "Failed to fetch versions from GitHub: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Printf("[HandleVersions] GitHub API returned status %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[HandleVersions] GitHub API returned status %d", resp.StatusCode)
		http.Error(w, fmt.Sprintf("GitHub API returned status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[HandleVersions] Failed to read response: %v", err)
		http.Error(w, "Failed to read GitHub response", http.StatusInternalServerError)
		return
	}

	log.Printf("[HandleVersions] Received %d bytes from GitHub", len(body))

	type GitHubRelease struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Prerelease  bool   `json:"prerelease"`
		PublishedAt string `json:"published_at"`
	}

	var githubReleases []GitHubRelease
	if err := json.Unmarshal(body, &githubReleases); err != nil {
		log.Printf("[HandleVersions] Failed to parse GitHub response: %v", err)
		http.Error(w, "Failed to parse GitHub response", http.StatusInternalServerError)
		return
	}

	log.Printf("[HandleVersions] Parsed %d releases from GitHub", len(githubReleases))

	// YDB versions start with a digit; CLI versions start with "CLI_" and must be skipped.
	var versions []YdbVersion
	for i, release := range githubReleases {
		if release.Prerelease {
			continue
		}
		version := release.TagName
		if version == "" {
			continue
		}
		if strings.HasPrefix(version, "CLI_") {
			continue
		}
		versions = append(versions, YdbVersion{
			Version:     version,
			Recommended: i == 0,
		})
		if len(versions) >= 20 {
			break
		}
	}

	log.Printf("[HandleVersions] Found %d stable versions", len(versions))

	if len(versions) == 0 {
		log.Printf("[HandleVersions] No stable versions found in GitHub releases")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(YdbVersionsResponse{Versions: []YdbVersion{}})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(YdbVersionsResponse{Versions: versions})
}

// HandleValidateFile validates that a file path exists and is accessible.
func HandleValidateFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ValidateFileResponse{
			Valid: false,
			Error: "Path not specified",
		})
		return
	}

	path := req.Path
	if !filepath.IsAbs(path) {
		cwd, _ := os.Getwd()
		path = filepath.Join(cwd, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ValidateFileResponse{
			Valid: false,
			Error: "File not found or inaccessible",
		})
		return
	}

	if info.IsDir() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ValidateFileResponse{
			Valid: false,
			Error: "Specified path is a directory, not a file",
		})
		return
	}

	size := info.Size()
	sizeStr := formatFileSize(size)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ValidateFileResponse{
		Valid: true,
		Size:  sizeStr,
	})
}

// formatFileSize formats bytes into human-readable string.
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
