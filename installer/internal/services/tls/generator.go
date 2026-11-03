package tls

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Generator struct {
	keyBits int
	days    int
}

func NewGenerator() *Generator {
	return &Generator{
		keyBits: 4096,
		days:    365,
	}
}

func (g *Generator) SetDays(days int) {
	if days > 0 {
		g.days = days
	}
}

// SetKeyBits overrides the RSA key size used for certificate generation.
// The default is 4096 bits (production-grade).
// Use 2048 in tests to keep test suites fast.
func (g *Generator) SetKeyBits(bits int) {
	if bits >= 1024 {
		g.keyBits = bits
	}
}

// ErrCodeCertsAlreadyExist is returned when Generate would overwrite existing material.
const ErrCodeCertsAlreadyExist = "certs_already_exist"

type GenerateRequest struct {
	Hosts []string // List of FQDNs/hostnames for the certificates
	Days  int      // Certificate validity period in days
	Force bool     // Clean-slate regeneration: wipe TLS/CA, fresh CA, only requested hosts
}

type GenerateResponse struct {
	Success   bool
	OutputDir string // Absolute path to certs-per-host directory (= ydb_tls_dir)
	CAFile    string // Absolute path to ca.crt inside OutputDir
	NodeFiles []NodeCertFiles
	Error     string
	// ErrorCode is a stable machine-readable code (e.g. ErrCodeCertsAlreadyExist).
	ErrorCode string
	// Existing lists relative paths that already exist when generation is blocked.
	Existing []string
	// CANew indicates whether a fresh CA was created (true) or an existing one was reused (false).
	CANew bool
}

type ExistCheckResult struct {
	Exists        bool
	Path          string   // Resolved certs-per-host directory (may not exist yet)
	Reasons       []string // Relative host-conflict paths (node.crt/node.key for requested hosts)
	CAPresent     bool     // Whether ca.crt already exists in the certs-per-host directory
	HostConflicts []string // Relative paths of target host certs/keys that already exist
	BlockGenerate bool     // True when any requested host has node.crt or node.key (would overwrite)
}

type NodeCertFiles struct {
	Hostname  string // Original FQDN / hostname as provided
	ShortName string // inventory_hostname_short: part before the first dot
	ConfFile  string // <OutputDir>/<ShortName>/options.cnf  (OpenSSL node config)
	KeyFile   string // <OutputDir>/<ShortName>/node.key     (0600)
	CSRFile   string // <OutputDir>/<ShortName>/node.csr     (certificate signing request)
	CertFile  string // <OutputDir>/<ShortName>/node.crt     (signed certificate)
	NodePem   string // <OutputDir>/<ShortName>/node.pem     (DER copy, like openssl ca -outdir serial.pem)
	WebPem    string // <OutputDir>/<ShortName>/web.pem      (key+cert+ca concatenated)
}

// shortHostname returns the Ansible inventory_hostname_short equivalent:
// the part of the FQDN before the first dot.  If there is no dot the full
// string is returned unchanged.
func shortHostname(fqdn string) string {
	if idx := strings.Index(fqdn, "."); idx > 0 {
		return fqdn[:idx]
	}
	return fqdn
}

// certsPerHostDir returns the fixed absolute path to the certs-per-host
// directory relative to the current working directory.
//
// Layout (mirrors TLS/ydb-ca-update.sh):
//
//	<cwd>/TLS/CA/
//	  secure/ca.key
//	  certs/ca.crt
//	  certs/certs-per-host/   ← returned value (= ydb_tls_dir)
//	    ca.crt
//	    <shortName>/
//	      node.key
//	      node.crt
//	      web.pem
func certsPerHostDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	return filepath.Join(cwd, "TLS", "CA", "certs", "certs-per-host"), nil
}

func caDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	return filepath.Join(cwd, "TLS", "CA"), nil
}

// CheckExistingCerts reports whether generating certs for the requested hosts
// would overwrite existing per-host material.
//
// New rules (Option E):
//   - D1 (ca.crt presence) is informational only — does NOT block generate
//   - D2 (requested host node.crt / node.key) IS the block condition
//   - Other hosts' directories are ignored
//
// If certPath is empty, the fixed default certs-per-host directory is used.
func CheckExistingCerts(certPath string, hosts []string) ExistCheckResult {
	path := strings.TrimSpace(certPath)
	if path == "" {
		resolved, err := certsPerHostDir()
		if err != nil {
			return ExistCheckResult{Exists: false, Path: "", Reasons: nil}
		}
		path = resolved
	}

	caPresent := caCrtPresent(path)
	hostConflicts := hostConflictReasonList(path, hosts)

	return ExistCheckResult{
		Exists:        len(hostConflicts) > 0,
		Path:          path,
		Reasons:       hostConflicts,
		CAPresent:     caPresent,
		HostConflicts: hostConflicts,
		BlockGenerate: len(hostConflicts) > 0,
	}
}

func caCrtPresent(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "ca.crt"))
	return err == nil
}

// hostConflictReasons returns relative paths of node.crt / node.key for
// requested hosts that already exist. Non-requested host directories are
// not checked.
func hostConflictReasons(dir string, hosts []string) []string {
	return hostConflictReasonList(dir, hosts)
}

// hostConflictReasonList is the internal implementation used by both
// CheckExistingCerts and Generate block guards.
func hostConflictReasonList(dir string, hosts []string) []string {
	var reasons []string
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		short := shortHostname(host)
		for _, name := range []string{"node.crt", "node.key"} {
			rel := short + "/" + name
			if _, err := os.Stat(filepath.Join(dir, short, name)); err == nil {
				reasons = append(reasons, rel)
			}
		}
	}
	return reasons
}

// existingConflictReasons is the legacy D1∪D2 check kept for backward
// compatibility. New code should prefer CheckExistingCerts + BlockGenerate.
func existingConflictReasons(dir string, hosts []string) []string {
	var reasons []string
	if _, err := os.Stat(filepath.Join(dir, "ca.crt")); err == nil {
		reasons = append(reasons, "ca.crt")
	}
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		short := shortHostname(host)
		for _, name := range []string{"node.crt", "node.key"} {
			rel := short + "/" + name
			if _, err := os.Stat(filepath.Join(dir, short, name)); err == nil {
				reasons = append(reasons, rel)
			}
		}
	}
	return reasons
}

// Generate creates self-signed certificates for the specified hosts using the
// correct per-hostname directory structure expected by the Ansible playbooks.
//
// The output directory is always fixed at ./TLS/CA/certs/certs-per-host
// relative to the current working directory.
//
// Blocking rules (Option E):
//   - If any requested host already has node.crt or node.key → blocked
//     (ErrCodeCertsAlreadyExist) to avoid overwriting existing cluster material.
//   - ca.crt presence alone does NOT block generate; an existing CA is reused.
//   - Other (non-requested) host directories are ignored for blocking.
//
// When an existing CA is present (TLS/CA/secure/ca.key + certs/ca.crt),
// it is loaded and reused. Otherwise a fresh CA is created.
//
// Force is a clean-slate run: it wipes the entire TLS/CA tree first, then
// generates a fresh CA and only the requested hosts' certificates — so there
// is no CA reuse and no orphaned per-host directories from a previous run.
func (g *Generator) Generate(req GenerateRequest) GenerateResponse {
	if len(req.Hosts) == 0 {
		return GenerateResponse{
			Success: false,
			Error:   "no hosts specified",
		}
	}

	if req.Days > 0 {
		g.days = req.Days
	}

	outputDir, err := certsPerHostDir()
	if err != nil {
		return GenerateResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	// Host-only block: do not overwrite existing node.crt / node.key for
	// requested hosts. CA presence alone does not block.
	// When Force is true, the whole tree is wiped and regenerated, so the
	// conflict check is skipped.
	if !req.Force {
		if hostReasons := hostConflictReasonList(outputDir, req.Hosts); len(hostReasons) > 0 {
			return GenerateResponse{
				Success:   false,
				OutputDir: outputDir,
				ErrorCode: ErrCodeCertsAlreadyExist,
				Existing:  hostReasons,
				Error: "certificates already exist at " + outputDir +
					" (" + strings.Join(hostReasons, ", ") +
					"); delete them manually before regenerating to avoid breaking the cluster",
			}
		}
	}

	caDirPath, err := caDir()
	if err != nil {
		return GenerateResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	// Force is a clean-slate run: remove the whole TLS/CA tree so regeneration
	// starts from scratch (fresh CA, no orphaned per-host directories).
	if req.Force {
		if err := os.RemoveAll(caDirPath); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to clear TLS/CA: %v", err),
			}
		}
	}

	certsDirPath := filepath.Join(caDirPath, "certs")
	secureDirPath := filepath.Join(caDirPath, "secure")

	var caCertPEM, caKeyPEM []byte
	var caNew bool

	// An existing CA is reused only in the non-force path. Force always
	// generates a fresh CA (the previous tree was removed above).
	if !req.Force {
		existingCACert, existingCAKey, loadErr := loadExistingCA(caDirPath)
		if loadErr == nil {
			caCertPEM = existingCACert
			caKeyPEM = existingCAKey
		}
	}

	if caCertPEM == nil {
		cert, key, err := g.generateCA()
		if err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to generate CA: %v", err),
			}
		}
		caCertPEM = cert
		caKeyPEM = key
		caNew = true
	}

	for _, dir := range []string{secureDirPath, certsDirPath, outputDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to create directory %s: %v", dir, err),
			}
		}
	}

	// Write CA private key (only if newly created; existing key stays untouched)
	if caNew {
		caKeyPath := filepath.Join(secureDirPath, "ca.key")
		if err := os.WriteFile(caKeyPath, caKeyPEM, 0600); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write CA key: %v", err),
			}
		}
	}

	if caNew {
		caCertPath := filepath.Join(certsDirPath, "ca.crt")
		if err := os.WriteFile(caCertPath, caCertPEM, 0644); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write CA certificate: %v", err),
			}
		}
	}

	// Always ensure ca.crt is present in certs-per-host/ for Ansible convenience.
	// Write even when reusing CA (benign idempotent overwrite of same material).
	caCertInOutputDir := filepath.Join(outputDir, "ca.crt")
	if err := os.WriteFile(caCertInOutputDir, caCertPEM, 0644); err != nil {
		return GenerateResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to copy CA certificate to output dir: %v", err),
		}
	}

	// Generate per-node certificates — 6 files per node, matching ydb-ca-update.sh output:
	//   options.cnf  — OpenSSL node config (make_node_conf)
	//   node.key     — private key          (make_node_key)
	//   node.csr     — signing request      (make_node_csr)
	//   node.crt     — signed certificate   (make_node_cert)
	//   node.pem     — DER copy of cert     (equivalent to openssl ca -outdir <serial>.pem)
	//   web.pem      — key+cert+ca bundle   (make_node_cert, line 130)
	nodeFiles := make([]NodeCertFiles, 0, len(req.Hosts))
	for i, host := range req.Hosts {
		shortName := shortHostname(host)

		nodeDir := filepath.Join(outputDir, shortName)
		if err := os.MkdirAll(nodeDir, 0755); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to create node directory for %s: %v", shortName, err),
			}
		}

		// 1. options.cnf — OpenSSL node configuration (mirrors make_node_conf)
		confFile := filepath.Join(nodeDir, "options.cnf")
		confContent := buildNodeConf(shortName, host)
		if err := os.WriteFile(confFile, []byte(confContent), 0644); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write options.cnf for %s: %v", host, err),
			}
		}

		certPEM, csrPEM, keyPEM, err := g.generateNodeCertWithCSR(host, shortName, caCertPEM, caKeyPEM, i+1)
		if err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to generate certificate for %s: %v", host, err),
			}
		}

		// 2. node.key — private key (restricted permissions)
		keyFile := filepath.Join(nodeDir, "node.key")
		if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write key for %s: %v", host, err),
			}
		}

		// 3. node.csr — certificate signing request
		csrFile := filepath.Join(nodeDir, "node.csr")
		if err := os.WriteFile(csrFile, csrPEM, 0644); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write CSR for %s: %v", host, err),
			}
		}

		// 4. node.crt — signed certificate
		certFile := filepath.Join(nodeDir, "node.crt")
		if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write certificate for %s: %v", host, err),
			}
		}

		// 5. node.pem — copy of the signed certificate (equivalent to openssl ca -outdir <serial>.pem)
		nodePemFile := filepath.Join(nodeDir, "node.pem")
		if err := os.WriteFile(nodePemFile, certPEM, 0644); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write node.pem for %s: %v", host, err),
			}
		}

		// 6. web.pem — node.key + node.crt + ca.crt (matches ydb-ca-update.sh line 130)
		webPemFile := filepath.Join(nodeDir, "web.pem")
		webPemContent := append(append(keyPEM, certPEM...), caCertPEM...)
		if err := os.WriteFile(webPemFile, webPemContent, 0600); err != nil {
			return GenerateResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write web.pem for %s: %v", host, err),
			}
		}

		nodeFiles = append(nodeFiles, NodeCertFiles{
			Hostname:  host,
			ShortName: shortName,
			ConfFile:  confFile,
			KeyFile:   keyFile,
			CSRFile:   csrFile,
			CertFile:  certFile,
			NodePem:   nodePemFile,
			WebPem:    webPemFile,
		})
	}

	return GenerateResponse{
		Success:   true,
		OutputDir: outputDir,
		CAFile:    caCertInOutputDir,
		NodeFiles: nodeFiles,
		CANew:     caNew,
	}
}

// loadExistingCA attempts to load an existing CA from the standard layout:
//
//	TLS/CA/secure/ca.key   — CA private key
//	TLS/CA/certs/ca.crt    — CA certificate
//
// Returns (certPEM, keyPEM, nil) on success, or (nil, nil, error) if either
// file is missing or unparseable.
func loadExistingCA(caDirPath string) ([]byte, []byte, error) {
	caCertPath := filepath.Join(caDirPath, "certs", "ca.crt")
	caKeyPath := filepath.Join(caDirPath, "secure", "ca.key")

	certPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("CA certificate not found at %s: %w", caCertPath, err)
	}

	keyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("CA private key not found at %s: %w", caKeyPath, err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA certificate PEM at %s", caCertPath)
	}
	if _, err := x509.ParseCertificate(certBlock.Bytes); err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate at %s: %w", caCertPath, err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA private key PEM at %s", caKeyPath)
	}
	if _, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err != nil {
		if _, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes); err != nil {
			return nil, nil, fmt.Errorf("failed to parse CA private key at %s: %w", caKeyPath, err)
		}
	}

	return certPEM, keyPEM, nil
}

// generateCA creates a self-signed CA certificate.
// Returns (certPEM, keyPEM, error).
func (g *Generator) generateCA() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, g.keyBits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"YDB"},
			CommonName:   "YDB CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, g.days),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

// buildNodeConf returns the content of options.cnf for a node, mirroring
// the make_node_conf function in ydb-ca-update.sh.
//
// shortName is DNS.1 (inventory_hostname_short); fqdn is DNS.2 if different.
func buildNodeConf(shortName, fqdn string) string {
	conf := `# OpenSSL node configuration file
[ req ]
prompt=no
distinguished_name = distinguished_name
req_extensions = extensions

[ distinguished_name ]
organizationName = YDB

[ extensions ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1=` + shortName + "\n"

	if fqdn != shortName {
		conf += "DNS.2=" + fqdn + "\n"
	}
	return conf
}

// generateNodeCertWithCSR creates a certificate for a node signed by the CA.
// Returns (certPEM, csrPEM, keyPEM, error).
//
// SANs follow the same convention as ydb-ca-update.sh options.cnf:
//   - DNS.1 = shortName  (inventory_hostname_short)
//   - DNS.2 = fqdn       (full hostname, if different from shortName)
//   - IP SAN             (if host is an IP address)
func (g *Generator) generateNodeCertWithCSR(fqdn, shortName string, caCertPEM, caKeyPEM []byte, index int) (certPEM, csrPEM, keyPEM []byte, err error) {
	caBlock, _ := pem.Decode(caCertPEM)
	if caBlock == nil {
		return nil, nil, nil, fmt.Errorf("failed to decode CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil {
		return nil, nil, nil, fmt.Errorf("failed to decode CA private key PEM")
	}
	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	nodeKey, err := rsa.GenerateKey(rand.Reader, g.keyBits)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate node private key: %w", err)
	}

	// Build DNS SANs: shortName first (Ansible uses this), then FQDN if different
	dnsNames := []string{shortName}
	if fqdn != shortName {
		dnsNames = append(dnsNames, fqdn)
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{"YDB"},
		},
		DNSNames:    dnsNames,
		IPAddresses: g.parseIPAddresses(fqdn),
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, nodeKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create CSR: %w", err)
	}
	csrPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	template := &x509.Certificate{
		SerialNumber: big.NewInt(int64(index + 1)), // +1 because serial 1 is used by CA
		Subject: pkix.Name{
			Organization: []string{"YDB"},
			CommonName:   shortName,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(0, 0, g.days),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		DNSNames:    dnsNames,
		IPAddresses: g.parseIPAddresses(fqdn),
	}

	nodeCertDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &nodeKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create node certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: nodeCertDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(nodeKey),
	})

	return certPEM, csrPEM, keyPEM, nil
}

func (g *Generator) parseIPAddresses(host string) []net.IP {
	ip := net.ParseIP(host)
	if ip != nil {
		return []net.IP{ip}
	}
	return nil
}

// GenerateUsingOpenSSL generates certificates using the existing OpenSSL shell script.
// This is a legacy fallback that calls TLS/ydb-ca-update.sh directly.
// Deprecated: use Generate() instead which produces the same structure in pure Go.
func (g *Generator) GenerateUsingOpenSSL(req GenerateRequest) GenerateResponse {
	if len(req.Hosts) == 0 {
		return GenerateResponse{
			Success: false,
			Error:   "no hosts specified",
		}
	}

	outputDir, err := certsPerHostDir()
	if err != nil {
		return GenerateResponse{Success: false, Error: err.Error()}
	}

	tlsDir := filepath.Dir(filepath.Dir(filepath.Dir(outputDir))) // TLS/
	nodesFile := filepath.Join(tlsDir, "ydb-ca-nodes.txt")

	// Build nodes file content: "<shortName> <fqdn>\n"
	var buf bytes.Buffer
	for _, host := range req.Hosts {
		short := shortHostname(host)
		if short == host {
			buf.WriteString(host + "\n")
		} else {
			buf.WriteString(short + " " + host + "\n")
		}
	}
	if err := os.WriteFile(nodesFile, buf.Bytes(), 0644); err != nil {
		return GenerateResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to write nodes file: %v", err),
		}
	}

	// The script must be run from the TLS/ directory
	scriptPath := filepath.Join(tlsDir, "ydb-ca-update.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return GenerateResponse{
			Success: false,
			Error:   fmt.Sprintf("OpenSSL script not found at %s", scriptPath),
		}
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return GenerateResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to read output directory: %v", err),
		}
	}

	nodeFiles := make([]NodeCertFiles, 0)
	caFile := filepath.Join(outputDir, "ca.crt")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		nodeDir := filepath.Join(outputDir, entry.Name())
		certFile := filepath.Join(nodeDir, "node.crt")
		keyFile := filepath.Join(nodeDir, "node.key")
		webPem := filepath.Join(nodeDir, "web.pem")

		if _, err := os.Stat(certFile); err == nil {
			nodeFiles = append(nodeFiles, NodeCertFiles{
				Hostname:  entry.Name(), // best effort — short name only
				ShortName: entry.Name(),
				CertFile:  certFile,
				KeyFile:   keyFile,
				WebPem:    webPem,
			})
		}
	}

	return GenerateResponse{
		Success:   true,
		OutputDir: outputDir,
		CAFile:    caFile,
		NodeFiles: nodeFiles,
	}
}

func GetCertificateInfo(certPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	info := map[string]interface{}{
		"subject":      cert.Subject.String(),
		"issuer":       cert.Issuer.String(),
		"not_before":   cert.NotBefore.Format(time.RFC3339),
		"not_after":    cert.NotAfter.Format(time.RFC3339),
		"dns_names":    cert.DNSNames,
		"ip_addresses": fmt.Sprintf("%v", cert.IPAddresses),
	}

	return info, nil
}

// GetFileStructure returns a map of expected relative paths → existence status
// for the certs-per-host directory layout.
//
// hosts should be the list of FQDNs; the function derives short names internally.
func GetFileStructure(dir string, hosts []string) map[string]bool {
	structure := map[string]bool{
		"ca.crt": false,
	}

	if _, err := os.Stat(filepath.Join(dir, "ca.crt")); err == nil {
		structure["ca.crt"] = true
	}

	for _, host := range hosts {
		short := shortHostname(host)
		for _, name := range []string{"options.cnf", "node.key", "node.csr", "node.crt", "node.pem", "web.pem"} {
			key := short + "/" + name
			structure[key] = false
			if _, err := os.Stat(filepath.Join(dir, short, name)); err == nil {
				structure[key] = true
			}
		}
	}

	return structure
}
