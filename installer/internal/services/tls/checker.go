package tls

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

type Checker struct{}

func NewChecker() *Checker {
	return &Checker{}
}

type ValidationResult struct {
	Valid      bool
	CAValid    bool
	NodeFiles  []NodeValidation
	Missing    []string
	Errors     []string
	TotalNodes int
	FoundNodes int
}

// NodeValidation represents the validation status of a single node's certificates.
// Only the 3 files required by YDB are checked: node.crt, node.key, and their pair match.
type NodeValidation struct {
	Index     int    // Sequential index (1-based)
	Hostname  string // Original FQDN / hostname as provided
	ShortName string // inventory_hostname_short (part before first dot)
	KeyFound  bool   // node.key exists
	CertFound bool   // node.crt exists
	CertValid bool   // node.crt is a valid X.509 certificate
	KeyValid  bool   // node.key is a valid RSA private key
	PairMatch bool   // node.crt public key matches node.key
	Match     bool   // All checks passed (CertFound && KeyFound && CertValid && KeyValid && PairMatch)
	Error     string // First validation error encountered
}

// ValidateRequest contains parameters for certificate validation.
type ValidateRequest struct {
	CertPath string   // Path to the certs-per-host directory (= ydb_tls_dir)
	Hosts    []string // Expected list of FQDNs / hostnames
}

// Validate checks existing certificates in the certs-per-host directory.
//
// For each host it checks the 3 files required by YDB:
//
//	<certPath>/ca.crt                    — CA certificate
//	<certPath>/<shortName>/node.crt      — node certificate
//	<certPath>/<shortName>/node.key      — private key (must match node.crt)
//
// where shortName = inventory_hostname_short (part of FQDN before the first dot).
func (c *Checker) Validate(req ValidateRequest) ValidationResult {
	result := ValidationResult{
		TotalNodes: len(req.Hosts),
	}

	if req.CertPath == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "certificate path is required")
		return result
	}

	info, err := os.Stat(req.CertPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("certificate directory not found: %v", err))
		return result
	}
	if !info.IsDir() {
		result.Valid = false
		result.Errors = append(result.Errors, "certificate path is not a directory")
		return result
	}

	caFile := filepath.Join(req.CertPath, "ca.crt")
	caData, err := os.ReadFile(caFile)
	if err != nil {
		result.CAValid = false
		result.Errors = append(result.Errors, "CA certificate (ca.crt) not found in certificate directory")
	} else {
		if _, err := parseCertificate(caData); err != nil {
			result.CAValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("CA certificate is invalid: %v", err))
		} else {
			result.CAValid = true
		}
	}

	for i, host := range req.Hosts {
		short := shortHostname(host)
		nv := NodeValidation{
			Index:     i + 1,
			Hostname:  host,
			ShortName: short,
		}

		nodeDir := filepath.Join(req.CertPath, short)

		certFile := filepath.Join(nodeDir, "node.crt")
		certData, err := os.ReadFile(certFile)
		if err != nil {
			nv.CertFound = false
			nv.Error = fmt.Sprintf("certificate not found at %s/node.crt", short)
			result.Missing = append(result.Missing, fmt.Sprintf("%s/node.crt", short))
		} else {
			nv.CertFound = true
			if _, err := parseCertificate(certData); err != nil {
				nv.CertValid = false
				if nv.Error == "" {
					nv.Error = fmt.Sprintf("certificate is invalid: %v", err)
				}
			} else {
				nv.CertValid = true
			}
		}

		keyFile := filepath.Join(nodeDir, "node.key")
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			nv.KeyFound = false
			if nv.Error == "" {
				nv.Error = fmt.Sprintf("private key not found at %s/node.key", short)
			}
			result.Missing = append(result.Missing, fmt.Sprintf("%s/node.key", short))
		} else {
			nv.KeyFound = true
			parsedKey, err := parsePrivateKey(keyData)
			if err != nil {
				nv.KeyValid = false
				if nv.Error == "" {
					nv.Error = fmt.Sprintf("key is invalid: %v", err)
				}
			} else {
				nv.KeyValid = true
				if nv.CertFound && nv.CertValid {
					nv.PairMatch = matchCertAndKey(certData, parsedKey)
					if !nv.PairMatch && nv.Error == "" {
						nv.Error = "node.crt public key does not match node.key"
					}
				}
			}
		}

		nv.Match = nv.CertFound && nv.KeyFound && nv.CertValid && nv.KeyValid && nv.PairMatch
		if nv.Match {
			result.FoundNodes++
		}

		result.NodeFiles = append(result.NodeFiles, nv)
	}

	result.Valid = result.CAValid && result.FoundNodes == result.TotalNodes && result.TotalNodes > 0

	return result
}

func (c *Checker) ValidateSingle(certPath string) (bool, string) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return false, fmt.Sprintf("failed to read file: %v", err)
	}

	_, err = parseCertificate(data)
	if err != nil {
		return false, fmt.Sprintf("invalid certificate: %v", err)
	}

	return true, ""
}

// GetExpectedFiles returns the list of expected relative paths for a given set
// of hosts in the certs-per-host directory layout.
//
// Returns only the 3 files required by YDB: ca.crt, <shortName>/node.crt, <shortName>/node.key.
//
// Example for hosts ["static-node-1.ydb-cluster.com"]:
//
//	["ca.crt",
//	 "static-node-1/node.crt",
//	 "static-node-1/node.key"]
func GetExpectedFiles(hosts []string) []string {
	files := []string{"ca.crt"}
	for _, host := range hosts {
		short := shortHostname(host)
		files = append(files,
			short+"/node.crt",
			short+"/node.key",
		)
	}
	return files
}

func CheckFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// matchCertAndKey checks whether the X.509 certificate's public key matches
// the given parsed private key. Supports RSA keys (PKCS1 and PKCS8).
func matchCertAndKey(certPEMData []byte, privateKey interface{}) bool {
	cert, err := parseCertificate(certPEMData)
	if err != nil {
		return false
	}

	certPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return false
	}

	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return certPubKey.Equal(&key.PublicKey)
	default:
		return false
	}
}

func parseCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

// parsePrivateKey parses a PEM-encoded private key (RSA PKCS1 or PKCS8).
func parsePrivateKey(data []byte) (interface{}, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("unsupported key format")
}
