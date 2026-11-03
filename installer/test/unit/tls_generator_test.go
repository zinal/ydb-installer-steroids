package unit

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ydb-installer/internal/services/tls"
)

// generateInTempDir runs Generate with a temporary working directory so the
// fixed path (TLS/CA/certs/certs-per-host) is created inside a temp dir.
func generateInTempDir(t *testing.T, hosts []string, days int) (tls.GenerateResponse, string) {
	t.Helper()

	tmpCwd, err := os.MkdirTemp("", "tls-cwd-*")
	if err != nil {
		t.Fatalf("failed to create temp cwd: %v", err)
	}

	// Change to the temp dir so certsPerHostDir() resolves inside it
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(tmpCwd); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = os.RemoveAll(tmpCwd)
	})

	g := tls.NewGenerator()
	// Use 2048-bit keys in tests — functionally identical but ~8x faster than 4096.
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: hosts,
		Days:  days,
	})
	return result, tmpCwd
}

func TestGenerator_NewGenerator(t *testing.T) {
	g := tls.NewGenerator()
	if g == nil {
		t.Fatal("NewGenerator returned nil")
	}
}

func TestGenerator_SetDays(t *testing.T) {
	g := tls.NewGenerator()
	g.SetDays(0)
	g.SetDays(-10)
	g.SetDays(365)
}

func TestGenerator_GenerateNoHosts(t *testing.T) {
	g := tls.NewGenerator()
	result := g.Generate(tls.GenerateRequest{Hosts: []string{}})

	if result.Success {
		t.Error("expected failure for empty hosts list")
	}
	if result.Error == "" {
		t.Error("expected error message for empty hosts list")
	}
}

func TestGenerator_DirectoryStructure(t *testing.T) {
	hosts := []string{
		"static-node-1.ydb-cluster.com",
		"static-node-2.ydb-cluster.com",
		"static-node-3.ydb-cluster.com",
	}
	result, tmpCwd := generateInTempDir(t, hosts, 365)

	if !result.Success {
		t.Fatalf("Generate failed: %s", result.Error)
	}

	// OutputDir must be the fixed certs-per-host path
	expectedOutputDir := filepath.Join(tmpCwd, "TLS", "CA", "certs", "certs-per-host")
	if result.OutputDir != expectedOutputDir {
		t.Errorf("OutputDir = %q, want %q", result.OutputDir, expectedOutputDir)
	}

	caInOutput := filepath.Join(result.OutputDir, "ca.crt")
	if _, err := os.Stat(caInOutput); os.IsNotExist(err) {
		t.Errorf("ca.crt not found in OutputDir: %s", caInOutput)
	}

	// CA key must be in TLS/CA/secure/ca.key (not exposed in OutputDir)
	caKeyPath := filepath.Join(tmpCwd, "TLS", "CA", "secure", "ca.key")
	if _, err := os.Stat(caKeyPath); os.IsNotExist(err) {
		t.Errorf("CA key not found at expected path: %s", caKeyPath)
	}

	shortNames := []string{"static-node-1", "static-node-2", "static-node-3"}
	for _, short := range shortNames {
		nodeDir := filepath.Join(result.OutputDir, short)
		for _, fname := range []string{"options.cnf", "node.key", "node.csr", "node.crt", "node.pem", "web.pem"} {
			fpath := filepath.Join(nodeDir, fname)
			if _, err := os.Stat(fpath); os.IsNotExist(err) {
				t.Errorf("expected file not found: %s", fpath)
			}
		}
	}
}

func TestGenerator_ShortNameInNodeFiles(t *testing.T) {
	hosts := []string{
		"node1.example.com",
		"node2.example.com",
	}
	result, _ := generateInTempDir(t, hosts, 365)

	if !result.Success {
		t.Fatalf("Generate failed: %s", result.Error)
	}

	if len(result.NodeFiles) != 2 {
		t.Fatalf("expected 2 NodeFiles, got %d", len(result.NodeFiles))
	}

	for i, nf := range result.NodeFiles {
		expectedShort := strings.SplitN(hosts[i], ".", 2)[0]
		if nf.ShortName != expectedShort {
			t.Errorf("NodeFiles[%d].ShortName = %q, want %q", i, nf.ShortName, expectedShort)
		}
		if nf.Hostname != hosts[i] {
			t.Errorf("NodeFiles[%d].Hostname = %q, want %q", i, nf.Hostname, hosts[i])
		}
	}
}

func TestGenerator_BareHostname(t *testing.T) {
	// Host without a dot — shortName should equal the full hostname
	hosts := []string{"mynode"}
	result, _ := generateInTempDir(t, hosts, 365)

	if !result.Success {
		t.Fatalf("Generate failed: %s", result.Error)
	}
	if len(result.NodeFiles) != 1 {
		t.Fatalf("expected 1 NodeFile, got %d", len(result.NodeFiles))
	}
	if result.NodeFiles[0].ShortName != "mynode" {
		t.Errorf("ShortName = %q, want %q", result.NodeFiles[0].ShortName, "mynode")
	}
}

func TestGenerator_WebPemCreated(t *testing.T) {
	hosts := []string{"node1.example.com"}
	result, _ := generateInTempDir(t, hosts, 365)

	if !result.Success {
		t.Fatalf("Generate failed: %s", result.Error)
	}

	webPemPath := result.NodeFiles[0].WebPem
	data, err := os.ReadFile(webPemPath)
	if err != nil {
		t.Fatalf("web.pem not readable: %v", err)
	}
	if len(data) == 0 {
		t.Error("web.pem is empty")
	}

	// web.pem must contain at least two PEM blocks (key + cert; ca is appended too)
	var blockCount int
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		blockCount++
	}
	if blockCount < 2 {
		t.Errorf("web.pem should contain at least 2 PEM blocks (key+cert+ca), got %d", blockCount)
	}
}

func TestGenerator_FQDNSANs(t *testing.T) {
	fqdn := "static-node-1.ydb-cluster.com"
	result, _ := generateInTempDir(t, []string{fqdn}, 365)

	if !result.Success {
		t.Fatalf("Generate failed: %s", result.Error)
	}

	certData, err := os.ReadFile(result.NodeFiles[0].CertFile)
	if err != nil {
		t.Fatalf("cannot read cert: %v", err)
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		t.Fatal("failed to decode cert PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse cert: %v", err)
	}

	dnsSet := make(map[string]bool)
	for _, name := range cert.DNSNames {
		dnsSet[name] = true
	}

	// Must contain the short name (Ansible uses this)
	if !dnsSet["static-node-1"] {
		t.Errorf("cert DNSNames missing short name 'static-node-1'; got %v", cert.DNSNames)
	}
	if !dnsSet[fqdn] {
		t.Errorf("cert DNSNames missing FQDN %q; got %v", fqdn, cert.DNSNames)
	}
}

func TestGenerator_GenerateWithIPAddress(t *testing.T) {
	hosts := []string{"192.168.1.1", "10.0.0.1"}
	result, _ := generateInTempDir(t, hosts, 365)

	if !result.Success {
		t.Fatalf("Generate failed for IP addresses: %s", result.Error)
	}
	if len(result.NodeFiles) != 2 {
		t.Errorf("expected 2 NodeFiles, got %d", len(result.NodeFiles))
	}
}

func TestGenerator_GenerateMultipleHosts(t *testing.T) {
	hosts := []string{
		"node1.example.com",
		"node2.example.com",
		"node3.example.com",
	}
	result, _ := generateInTempDir(t, hosts, 365)

	if !result.Success {
		t.Fatalf("Generate failed: %s", result.Error)
	}
	if len(result.NodeFiles) != 3 {
		t.Errorf("expected 3 NodeFiles, got %d", len(result.NodeFiles))
	}

	if _, err := os.Stat(result.CAFile); os.IsNotExist(err) {
		t.Error("CA file does not exist")
	}

	for i, nf := range result.NodeFiles {
		for _, fpath := range []string{nf.ConfFile, nf.KeyFile, nf.CSRFile, nf.CertFile, nf.NodePem, nf.WebPem} {
			if _, err := os.Stat(fpath); os.IsNotExist(err) {
				t.Errorf("NodeFiles[%d]: file not found: %s", i, fpath)
			}
		}
	}
}

func TestGetCertificateInfo(t *testing.T) {
	result, _ := generateInTempDir(t, []string{"test.example.com"}, 365)
	if !result.Success {
		t.Fatalf("Generate failed: %s", result.Error)
	}

	info, err := tls.GetCertificateInfo(result.CAFile)
	if err != nil {
		t.Fatalf("GetCertificateInfo failed: %v", err)
	}

	for _, key := range []string{"subject", "issuer", "not_before", "not_after"} {
		if v, ok := info[key]; !ok || v == "" {
			t.Errorf("GetCertificateInfo: key %q is missing or empty", key)
		}
	}
}

func TestGetCertificateInfoInvalidFile(t *testing.T) {
	_, err := tls.GetCertificateInfo("/nonexistent/file.crt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGetFileStructure(t *testing.T) {
	hosts := []string{"node1.example.com", "node2.example.com"}
	result, _ := generateInTempDir(t, hosts, 365)
	if !result.Success {
		t.Fatalf("Generate failed: %s", result.Error)
	}

	structure := tls.GetFileStructure(result.OutputDir, hosts)

	if !structure["ca.crt"] {
		t.Error("expected ca.crt in structure")
	}

	for _, host := range hosts {
		short := strings.SplitN(host, ".", 2)[0]
		for _, fname := range []string{"options.cnf", "node.key", "node.csr", "node.crt", "node.pem", "web.pem"} {
			key := short + "/" + fname
			if !structure[key] {
				t.Errorf("expected %q in structure", key)
			}
		}
	}
}

// generateYdbCerts runs full Generate for YDB static nodes (creates CA + hosts).
// Returns the response and temp cwd.
func generateYdbCerts(t *testing.T) (tls.GenerateResponse, string) {
	t.Helper()
	return generateInTempDir(t, []string{
		"static-node-1.ydb-cluster.com",
		"static-node-2.ydb-cluster.com",
		"static-node-3.ydb-cluster.com",
	}, 365)
}

func TestGenerator_CAOnlyTreeDoesNotBlockEMGenerate(t *testing.T) {
	_, ydbCwd := generateYdbCerts(t)

	// CA already exists from step 1 — must NOT block.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(ydbCwd); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{"em.example.com"},
		Days:  365,
	})

	if !result.Success {
		t.Fatalf("EM generate blocked by CA-only tree: %s (error_code=%s)", result.Error, result.ErrorCode)
	}
	if result.CANew {
		t.Error("expected existing CA to be reused (CANew=false), got CANew=true")
	}

	outputDir := filepath.Join(ydbCwd, "TLS", "CA", "certs", "certs-per-host")
	emDir := filepath.Join(outputDir, "em")
	for _, fname := range []string{"options.cnf", "node.key", "node.csr", "node.crt", "node.pem", "web.pem"} {
		if _, err := os.Stat(filepath.Join(emDir, fname)); os.IsNotExist(err) {
			t.Errorf("expected EM host file not found: %s", fname)
		}
	}

	for _, short := range []string{"static-node-1", "static-node-2", "static-node-3"} {
		if _, err := os.Stat(filepath.Join(outputDir, short, "node.crt")); os.IsNotExist(err) {
			t.Errorf("YDB host file missing after EM generate: %s/node.crt", short)
		}
	}
}

func TestGenerator_HostConflictBlocksGenerate(t *testing.T) {
	genResult, ydbCwd := generateInTempDir(t, []string{
		"em.example.com",
	}, 365)
	if !genResult.Success {
		t.Fatalf("initial generate failed: %s", genResult.Error)
	}

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{"em.example.com"},
		Days:  365,
	})

	if result.Success {
		t.Fatal("expected generate to be blocked when host node.crt already exists")
	}
	if result.ErrorCode != tls.ErrCodeCertsAlreadyExist {
		t.Errorf("expected error_code=%s, got %s", tls.ErrCodeCertsAlreadyExist, result.ErrorCode)
	}
	if len(result.Existing) == 0 {
		t.Error("expected existing list to contain conflicting paths")
	}
}

func TestGenerator_YDBHostsPresentEMHostMissing_NoBlock(t *testing.T) {
	genResult, ydbCwd := generateInTempDir(t, []string{
		"static-node-1.ydb-cluster.com",
		"static-node-2.ydb-cluster.com",
	}, 365)
	if !genResult.Success {
		t.Fatalf("initial generate failed: %s", genResult.Error)
	}

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{"em.example.com"},
		Days:  365,
	})

	if !result.Success {
		t.Fatalf("EM generate blocked when only YDB hosts present: %s", result.Error)
	}
	if result.CANew {
		t.Error("expected existing CA reuse, got CANew=true")
	}
}

func TestGenerator_CreatesCAWhenNonePresent(t *testing.T) {
	result, _ := generateInTempDir(t, []string{"node1.example.com"}, 365)
	if !result.Success {
		t.Fatalf("generate failed: %s", result.Error)
	}
	if !result.CANew {
		t.Error("expected CANew=true when creating from scratch")
	}
}

func TestGenerator_CAOnlyTreeDoesNotBlockMultiHostGenerate(t *testing.T) {
	_, ydbCwd := generateInTempDir(t, []string{"static-node-1.ydb-cluster.com"}, 365)

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{"em1.example.com", "em2.example.com"},
		Days:  365,
	})

	if !result.Success {
		t.Fatalf("multi-host EM generate blocked by CA-only tree: %s", result.Error)
	}
	if len(result.NodeFiles) != 2 {
		t.Errorf("expected 2 node files, got %d", len(result.NodeFiles))
	}

	outputDir := filepath.Join(ydbCwd, "TLS", "CA", "certs", "certs-per-host")
	for _, short := range []string{"em1", "em2"} {
		if _, err := os.Stat(filepath.Join(outputDir, short, "node.crt")); os.IsNotExist(err) {
			t.Errorf("expected %s/node.crt to exist", short)
		}
	}
}

func TestCheckExistingCerts_CAOnly_NotBlocked(t *testing.T) {
	_, ydbCwd := generateYdbCerts(t)

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	result := tls.CheckExistingCerts("", []string{"em.example.com"})
	if result.BlockGenerate {
		t.Error("expected BlockGenerate=false when only CA and YDB hosts exist")
	}
	if !result.CAPresent {
		t.Error("expected CAPresent=true")
	}
	if len(result.HostConflicts) != 0 {
		t.Errorf("expected no host conflicts, got %v", result.HostConflicts)
	}
}

func TestCheckExistingCerts_HostConflict_Blocked(t *testing.T) {
	genResult, ydbCwd := generateInTempDir(t, []string{"em.example.com"}, 365)
	if !genResult.Success {
		t.Fatalf("initial generate failed: %s", genResult.Error)
	}

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	result := tls.CheckExistingCerts("", []string{"em.example.com"})
	if !result.BlockGenerate {
		t.Error("expected BlockGenerate=true when EM host node.crt exists")
	}
	if len(result.HostConflicts) == 0 {
		t.Error("expected host conflicts to include em/node.crt or em/node.key")
	}
}

func TestCheckExistingCerts_NoCerts_NoBlock(t *testing.T) {
	tmpCwd, err := os.MkdirTemp("", "tls-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpCwd)

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpCwd)
	defer func() { _ = os.Chdir(origDir) }()

	result := tls.CheckExistingCerts("", []string{"node1.example.com"})
	if result.BlockGenerate {
		t.Error("expected BlockGenerate=false on empty tree")
	}
	if result.CAPresent {
		t.Error("expected CAPresent=false on empty tree")
	}
}

func TestCheckExistingCerts_CustomPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tls-custom-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create ca.crt only — should not block
	if err := os.WriteFile(filepath.Join(tmpDir, "ca.crt"), []byte("dummy"), 0644); err != nil {
		t.Fatalf("failed to write ca.crt: %v", err)
	}

	result := tls.CheckExistingCerts(tmpDir, []string{"node1.example.com"})
	if result.BlockGenerate {
		t.Error("expected BlockGenerate=false on CA-only custom path")
	}
	if !result.CAPresent {
		t.Error("expected CAPresent=true")
	}
	if result.Path != tmpDir {
		t.Errorf("expected path=%s, got %s", tmpDir, result.Path)
	}
}

func TestGenerator_DoesNotOverwriteExistingCAKey(t *testing.T) {
	_, ydbCwd := generateYdbCerts(t)

	caKeyPath := filepath.Join(ydbCwd, "TLS", "CA", "secure", "ca.key")
	origKeyBytes, err := os.ReadFile(caKeyPath)
	if err != nil {
		t.Fatalf("failed to read original CA key: %v", err)
	}

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{"em.example.com"},
		Days:  365,
	})
	if !result.Success {
		t.Fatalf("EM generate failed: %s", result.Error)
	}

	currentKeyBytes, err := os.ReadFile(caKeyPath)
	if err != nil {
		t.Fatalf("failed to read CA key after EM generate: %v", err)
	}
	if string(currentKeyBytes) != string(origKeyBytes) {
		t.Fatal("CA private key was overwritten during incremental EM generate")
	}
}

func TestGenerator_ForceSkipsHostConflict(t *testing.T) {
	_, ydbCwd := generateInTempDir(t, []string{"em.example.com"}, 365)

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{"em.example.com"},
		Days:  365,
		Force: true,
	})

	if !result.Success {
		t.Fatalf("Force=true should skip host conflict: %s (error_code=%s)", result.Error, result.ErrorCode)
	}
	if !result.CANew {
		t.Error("expected a fresh CA (CANew=true) with force")
	}

	outputDir := filepath.Join(ydbCwd, "TLS", "CA", "certs", "certs-per-host")
	for _, fname := range []string{"options.cnf", "node.key", "node.csr", "node.crt", "node.pem", "web.pem"} {
		if _, err := os.Stat(filepath.Join(outputDir, "em", fname)); os.IsNotExist(err) {
			t.Errorf("expected %s to exist after force overwrite", fname)
		}
	}
}

func TestGenerator_ForceMultiHostOverwrite(t *testing.T) {
	_, ydbCwd := generateInTempDir(t, []string{
		"static-node-1.ydb-cluster.com",
		"static-node-2.ydb-cluster.com",
	}, 365)

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{
			"static-node-1.ydb-cluster.com",
			"static-node-2.ydb-cluster.com",
			"static-node-3.ydb-cluster.com",
		},
		Days:  365,
		Force: true,
	})

	if !result.Success {
		t.Fatalf("Force=true multi-host overwrite failed: %s", result.Error)
	}
	if len(result.NodeFiles) != 3 {
		t.Errorf("expected 3 node files, got %d", len(result.NodeFiles))
	}

	outputDir := filepath.Join(ydbCwd, "TLS", "CA", "certs", "certs-per-host")
	for _, short := range []string{"static-node-1", "static-node-2", "static-node-3"} {
		for _, fname := range []string{"node.crt", "node.key"} {
			if _, err := os.Stat(filepath.Join(outputDir, short, fname)); os.IsNotExist(err) {
				t.Errorf("expected %s/%s to exist after force overwrite", short, fname)
			}
		}
	}
}

func TestGenerator_ForceRegeneratesCAKey(t *testing.T) {
	_, ydbCwd := generateInTempDir(t, []string{"em.example.com"}, 365)

	origKey, err := os.ReadFile(filepath.Join(ydbCwd, "TLS", "CA", "secure", "ca.key"))
	if err != nil {
		t.Fatalf("failed to read original CA key: %v", err)
	}

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{"em.example.com"},
		Days:  365,
		Force: true,
	})
	if !result.Success {
		t.Fatalf("Force generate failed: %s", result.Error)
	}
	if !result.CANew {
		t.Error("expected a fresh CA (CANew=true) with force")
	}

	newKey, err := os.ReadFile(filepath.Join(ydbCwd, "TLS", "CA", "secure", "ca.key"))
	if err != nil {
		t.Fatalf("failed to read regenerated CA key: %v", err)
	}
	if string(newKey) == string(origKey) {
		t.Error("expected CA private key to change after force regeneration")
	}
}

func TestGenerator_ForceClearsOrphanHostDirs(t *testing.T) {
	_, ydbCwd := generateInTempDir(t, []string{"node1.example.com", "node2.example.com"}, 365)

	origDir, _ := os.Getwd()
	_ = os.Chdir(ydbCwd)
	defer func() { _ = os.Chdir(origDir) }()

	g := tls.NewGenerator()
	g.SetKeyBits(2048)
	result := g.Generate(tls.GenerateRequest{
		Hosts: []string{"node1.example.com"},
		Days:  365,
		Force: true,
	})
	if !result.Success {
		t.Fatalf("Force generate failed: %s", result.Error)
	}

	outputDir := filepath.Join(ydbCwd, "TLS", "CA", "certs", "certs-per-host")
	if _, err := os.Stat(filepath.Join(outputDir, "node1", "node.crt")); os.IsNotExist(err) {
		t.Errorf("expected node1/node.crt to exist after force")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "node2")); !os.IsNotExist(err) {
		t.Error("expected orphaned node2 directory to be removed after force")
	}
}
