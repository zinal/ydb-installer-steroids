package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ydb-installer/internal/services/tls"
)

func TestChecker_NewChecker(t *testing.T) {
	c := tls.NewChecker()
	if c == nil {
		t.Fatal("NewChecker returned nil")
	}
}

func TestChecker_ValidateValidCertificates(t *testing.T) {
	hosts := []string{
		"static-node-1.ydb-cluster.com",
		"static-node-2.ydb-cluster.com",
	}

	genResult, _ := generateInTempDir(t, hosts, 365)
	if !genResult.Success {
		t.Fatalf("failed to generate certificates: %s", genResult.Error)
	}

	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: genResult.OutputDir,
		Hosts:    hosts,
	})

	if !result.Valid {
		t.Errorf("expected valid certificates, got errors: %v", result.Errors)
	}
	if !result.CAValid {
		t.Error("expected CA to be valid")
	}
	if result.TotalNodes != 2 {
		t.Errorf("TotalNodes = %d, want 2", result.TotalNodes)
	}
	if result.FoundNodes != 2 {
		t.Errorf("FoundNodes = %d, want 2", result.FoundNodes)
	}
	if len(result.Missing) != 0 {
		t.Errorf("unexpected missing files: %v", result.Missing)
	}
}

func TestChecker_ValidateByShortName(t *testing.T) {
	// Generate for FQDNs; checker must find certs by short name
	hosts := []string{"node1.example.com", "node2.example.com"}
	genResult, _ := generateInTempDir(t, hosts, 365)
	if !genResult.Success {
		t.Fatalf("generate failed: %s", genResult.Error)
	}

	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: genResult.OutputDir,
		Hosts:    hosts,
	})

	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}

	for i, nv := range result.NodeFiles {
		expectedShort := strings.SplitN(hosts[i], ".", 2)[0]
		if nv.ShortName != expectedShort {
			t.Errorf("NodeFiles[%d].ShortName = %q, want %q", i, nv.ShortName, expectedShort)
		}
		if nv.Hostname != hosts[i] {
			t.Errorf("NodeFiles[%d].Hostname = %q, want %q", i, nv.Hostname, hosts[i])
		}
	}
}

func TestChecker_ValidateBareHostname(t *testing.T) {
	// Bare hostname (no dot) — shortName == hostname
	hosts := []string{"mynode"}
	genResult, _ := generateInTempDir(t, hosts, 365)
	if !genResult.Success {
		t.Fatalf("generate failed: %s", genResult.Error)
	}

	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: genResult.OutputDir,
		Hosts:    hosts,
	})

	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
	if len(result.NodeFiles) != 1 {
		t.Fatalf("expected 1 NodeFile, got %d", len(result.NodeFiles))
	}
	if result.NodeFiles[0].ShortName != "mynode" {
		t.Errorf("ShortName = %q, want %q", result.NodeFiles[0].ShortName, "mynode")
	}
}

func TestChecker_ValidateCertKeyPairMatch(t *testing.T) {
	hosts := []string{"node1.example.com"}
	genResult, _ := generateInTempDir(t, hosts, 365)
	if !genResult.Success {
		t.Fatalf("generate failed: %s", genResult.Error)
	}

	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: genResult.OutputDir,
		Hosts:    hosts,
	})

	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
	if len(result.NodeFiles) != 1 {
		t.Fatalf("expected 1 NodeFile, got %d", len(result.NodeFiles))
	}
	if !result.NodeFiles[0].PairMatch {
		t.Error("expected PairMatch=true for matching cert and key")
	}
}

func TestChecker_ValidateCertKeyMismatch(t *testing.T) {
	// Generate certs for two hosts then swap their keys
	hosts := []string{"node-a.example.com", "node-b.example.com"}
	genResult, _ := generateInTempDir(t, hosts, 365)
	if !genResult.Success {
		t.Fatalf("generate failed: %s", genResult.Error)
	}

	// Swap the key files so cert and key don't match
	keyA := filepath.Join(genResult.OutputDir, "node-a", "node.key")
	keyB := filepath.Join(genResult.OutputDir, "node-b", "node.key")
	dataA, _ := os.ReadFile(keyA)
	dataB, _ := os.ReadFile(keyB)
	_ = os.WriteFile(keyA, dataB, 0600)
	_ = os.WriteFile(keyB, dataA, 0600)

	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: genResult.OutputDir,
		Hosts:    hosts,
	})

	if result.Valid {
		t.Error("expected invalid after key swap")
	}
	for _, nv := range result.NodeFiles {
		if nv.Match {
			t.Errorf("expected Match=false for %s after key swap, got true", nv.Hostname)
		}
		if nv.PairMatch {
			t.Errorf("expected PairMatch=false for %s after key swap, got true", nv.Hostname)
		}
	}
}

func TestChecker_ValidateOnlyThreeFiles(t *testing.T) {
	// Generate full structure then remove the 4 non-critical files
	hosts := []string{"node1.example.com"}
	genResult, _ := generateInTempDir(t, hosts, 365)
	if !genResult.Success {
		t.Fatalf("generate failed: %s", genResult.Error)
	}

	nodeDir := filepath.Join(genResult.OutputDir, "node1")

	for _, f := range []string{"options.cnf", "node.csr", "node.pem", "web.pem"} {
		os.Remove(filepath.Join(nodeDir, f))
	}

	// Validation should still pass with only node.crt and node.key
	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: genResult.OutputDir,
		Hosts:    hosts,
	})

	if !result.Valid {
		t.Errorf("expected valid with only 3 critical files, got errors: %v", result.Errors)
	}
	if len(result.NodeFiles) != 1 {
		t.Fatalf("expected 1 NodeFile, got %d", len(result.NodeFiles))
	}
	if !result.NodeFiles[0].Match {
		t.Error("expected Match=true with only node.crt, node.key and ca.crt")
	}
	if len(result.Missing) > 0 {
		t.Errorf("expected no missing files, got: %v", result.Missing)
	}
}

func TestChecker_ValidateEmptyCertPath(t *testing.T) {
	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: "",
		Hosts:    []string{"node1.example.com"},
	})

	if result.Valid {
		t.Error("expected invalid for empty cert path")
	}
	if len(result.Errors) == 0 {
		t.Error("expected error message for empty cert path")
	}
}

func TestChecker_ValidateMissingDirectory(t *testing.T) {
	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: "/nonexistent/directory",
		Hosts:    []string{"node1.example.com"},
	})

	if result.Valid {
		t.Error("expected invalid for missing directory")
	}
	if len(result.Errors) == 0 {
		t.Error("expected error messages for missing directory")
	}
}

func TestChecker_ValidateMissingShortNameDir(t *testing.T) {
	// Directory exists but has no per-host subdirectory
	tmpDir, err := os.MkdirTemp("", "tls-checker-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a valid CA cert so CA check passes
	genResult, _ := generateInTempDir(t, []string{"other-node.example.com"}, 365)
	if !genResult.Success {
		t.Fatalf("generate failed: %s", genResult.Error)
	}
	caData, _ := os.ReadFile(genResult.CAFile)
	_ = os.WriteFile(filepath.Join(tmpDir, "ca.crt"), caData, 0644)

	// Validate for a host whose directory does NOT exist
	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: tmpDir,
		Hosts:    []string{"missing-node.example.com"},
	})

	if result.Valid {
		t.Error("expected invalid when node directory is missing")
	}
	if len(result.Missing) == 0 {
		t.Error("expected missing files to be reported")
	}
	found := false
	for _, m := range result.Missing {
		if strings.Contains(m, "missing-node") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("missing list should reference 'missing-node', got: %v", result.Missing)
	}
}

func TestChecker_ValidateInvalidCA(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tls-checker-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "ca.crt"), []byte("not a valid certificate"), 0644); err != nil {
		t.Fatalf("failed to write invalid CA: %v", err)
	}

	c := tls.NewChecker()
	result := c.Validate(tls.ValidateRequest{
		CertPath: tmpDir,
		Hosts:    []string{"node1.example.com"},
	})

	if result.CAValid {
		t.Error("expected CA to be invalid")
	}
}

func TestChecker_ValidateMissingCertificates(t *testing.T) {
	// Directory exists with valid CA but no node subdirectories
	genResult, _ := generateInTempDir(t, []string{"seed.example.com"}, 365)
	if !genResult.Success {
		t.Fatalf("generate failed: %s", genResult.Error)
	}

	c := tls.NewChecker()
	// Ask for a host that was NOT generated
	result := c.Validate(tls.ValidateRequest{
		CertPath: genResult.OutputDir,
		Hosts:    []string{"nonexistent.example.com"},
	})

	if result.Valid {
		t.Error("expected invalid for missing node certificates")
	}
	if len(result.Missing) == 0 {
		t.Error("expected missing certificates to be reported")
	}
}

func TestChecker_ValidateSingle(t *testing.T) {
	genResult, _ := generateInTempDir(t, []string{"node1.example.com"}, 365)
	if !genResult.Success {
		t.Fatalf("generate failed: %s", genResult.Error)
	}

	c := tls.NewChecker()
	valid, errMsg := c.ValidateSingle(genResult.CAFile)

	if !valid {
		t.Errorf("expected valid CA certificate, got error: %s", errMsg)
	}
}

func TestChecker_ValidateSingleInvalidFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tls-checker-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invalidFile := filepath.Join(tmpDir, "invalid.crt")
	if err := os.WriteFile(invalidFile, []byte("not a valid certificate"), 0644); err != nil {
		t.Fatalf("failed to write invalid cert: %v", err)
	}

	c := tls.NewChecker()
	valid, errMsg := c.ValidateSingle(invalidFile)

	if valid {
		t.Error("expected invalid for invalid certificate file")
	}
	if errMsg == "" {
		t.Error("expected error message for invalid certificate")
	}
}

func TestGetExpectedFiles(t *testing.T) {
	hosts := []string{
		"static-node-1.ydb-cluster.com",
		"static-node-2.ydb-cluster.com",
		"static-node-3.ydb-cluster.com",
	}
	files := tls.GetExpectedFiles(hosts)

	if len(files) == 0 || files[0] != "ca.crt" {
		t.Errorf("first file should be ca.crt, got: %v", files)
	}

	// 1 (ca.crt) + 3 hosts × 2 files (node.crt, node.key) = 7
	expectedCount := 1 + 3*2
	if len(files) != expectedCount {
		t.Errorf("expected %d files, got %d: %v", expectedCount, len(files), files)
	}

	for _, short := range []string{"static-node-1", "static-node-2", "static-node-3"} {
		for _, fname := range []string{"node.crt", "node.key"} {
			want := short + "/" + fname
			found := false
			for _, f := range files {
				if f == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q in GetExpectedFiles result", want)
			}
		}
	}

	for _, short := range []string{"static-node-1", "static-node-2", "static-node-3"} {
		for _, fname := range []string{"options.cnf", "node.csr", "node.pem", "web.pem"} {
			want := short + "/" + fname
			for _, f := range files {
				if f == want {
					t.Errorf("unexpected non-critical file %q in GetExpectedFiles result", want)
				}
			}
		}
	}
}

func TestGetExpectedFilesZeroHosts(t *testing.T) {
	files := tls.GetExpectedFiles([]string{})

	if len(files) != 1 {
		t.Errorf("expected 1 file (ca.crt), got %d", len(files))
	}
	if files[0] != "ca.crt" {
		t.Errorf("expected ca.crt, got %s", files[0])
	}
}

func TestGetExpectedFilesBareHostname(t *testing.T) {
	files := tls.GetExpectedFiles([]string{"mynode"})

	// 1 (ca.crt) + 2 files for "mynode" (node.crt, node.key)
	expectedCount := 1 + 2
	if len(files) != expectedCount {
		t.Fatalf("expected %d files, got %d: %v", expectedCount, len(files), files)
	}
	for _, fname := range []string{"node.crt", "node.key"} {
		want := "mynode/" + fname
		found := false
		for _, f := range files {
			if f == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in GetExpectedFiles result", want)
		}
	}
}

func TestCheckFileExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tls-checker-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	exists, err := tls.CheckFileExists(testFile)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}

	exists, err = tls.CheckFileExists(filepath.Join(tmpDir, "nonexistent.txt"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected file to not exist")
	}
}
