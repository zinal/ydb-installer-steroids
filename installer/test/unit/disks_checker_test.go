package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/disks"
)

func TestDiskConsistencyCheck(t *testing.T) {
	checker := disks.NewChecker()

	tests := []struct {
		name     string
		results  []models.HostDisksResult
		expected bool
	}{
		{
			name: "consistent disks",
			results: []models.HostDisksResult{
				{
					Host: "host1",
					Disks: []models.DiskInfo{
						{Name: "vda", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
						{Name: "vdb", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
					},
				},
				{
					Host: "host2",
					Disks: []models.DiskInfo{
						{Name: "vda", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
						{Name: "vdb", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
					},
				},
			},
			expected: true,
		},
		{
			name: "inconsistent disk count",
			results: []models.HostDisksResult{
				{
					Host: "host1",
					Disks: []models.DiskInfo{
						{Name: "vda", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
					},
				},
				{
					Host: "host2",
					Disks: []models.DiskInfo{
						{Name: "vda", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
						{Name: "vdb", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
					},
				},
			},
			expected: false,
		},
		{
			name: "inconsistent disk types",
			results: []models.HostDisksResult{
				{
					Host: "host1",
					Disks: []models.DiskInfo{
						{Name: "vda", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
					},
				},
				{
					Host: "host2",
					Disks: []models.DiskInfo{
						{Name: "vda", Type: "HDD", SizeBytes: 500 * 1024 * 1024 * 1024},
					},
				},
			},
			expected: false,
		},
		{
			name: "inconsistent disk sizes",
			results: []models.HostDisksResult{
				{
					Host: "host1",
					Disks: []models.DiskInfo{
						{Name: "vda", Type: "SSD", SizeBytes: 500 * 1024 * 1024 * 1024},
					},
				},
				{
					Host: "host2",
					Disks: []models.DiskInfo{
						{Name: "vda", Type: "SSD", SizeBytes: 1000 * 1024 * 1024 * 1024},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CheckDiskConsistency(tt.results)
			if result.IsConsistent != tt.expected {
				t.Errorf("CheckDiskConsistency() = %v, want %v", result.IsConsistent, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	checker := disks.NewChecker()

	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500B"},
		{1024, "1.0K"},
		{1024 * 1024, "1.0M"},
		{1024 * 1024 * 1024, "1.0G"},
		{1024 * 1024 * 1024 * 1024, "1.0T"},
		{500 * 1024 * 1024 * 1024, "500.0G"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// formatBytes is a private method, so we can't test it directly
			// This test is a placeholder for when we make it public
			_ = checker // Use checker to avoid "declared and not used" error
		})
	}
}

func TestCheckDisks_KeyPathValidation(t *testing.T) {
	checker := disks.NewChecker()

	t.Run("missing absolute key returns not found", func(t *testing.T) {
		_, err := checker.CheckDisks(&models.CheckDisksRequest{
			Hosts: []string{"host1.example.com"},
			SSH: &models.SSHConfig{
				AuthType: "key",
				KeyPath:  "/tmp/ydb-installer-missing-key-test-noexist",
				Login:    "root",
			},
		})
		if err == nil {
			t.Fatal("expected error for missing key path")
		}
		if !strings.Contains(err.Error(), "SSH key file not found") {
			t.Errorf("error = %q, want message containing 'SSH key file not found'", err.Error())
		}
		// Must not use the old misleading message
		if strings.Contains(err.Error(), "SSH key path is required") {
			t.Errorf("got legacy error message: %v", err)
		}
	})

	t.Run("missing tilde key returns not found", func(t *testing.T) {
		// Regression: raw os.Stat("~/.…") used to fail before SSH; ValidateKeyPath expands ~/.
		_, err := checker.CheckDisks(&models.CheckDisksRequest{
			Hosts: []string{"host1.example.com"},
			SSH: &models.SSHConfig{
				AuthType: "key",
				KeyPath:  "~/ydb-installer-missing-key-test-noexist",
				Login:    "root",
			},
		})
		if err == nil {
			t.Fatal("expected error for missing tilde key path")
		}
		if !strings.Contains(err.Error(), "SSH key file not found") {
			t.Errorf("error = %q, want message containing 'SSH key file not found'", err.Error())
		}
	})

	t.Run("existing tilde key passes top-level validation", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("home directory unavailable")
		}
		name := ".ydb-installer-disks-key-test"
		full := filepath.Join(home, name)
		if err := os.WriteFile(full, []byte("test-key"), 0o600); err != nil {
			t.Fatalf("failed to create temp key: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(full) })

		// Empty hosts returns before SSH work, but also before key validation today.
		// Use one host with an unresolvable name; key validation runs first and must
		// not return a top-level error. Per-host SSH failure is OK and finishes via
		// ConnectTimeout (bounded). To keep the suite fast we only assert that a
		// *missing* tilde path errors (above) and that ValidateKeyPath accepts this
		// file — covered in sshclient tests. Here we only need: existing tilde path
		// does not produce the legacy Stat failure at the CheckDisks entrypoint.
		//
		// Call with empty hosts is insufficient (early return). Call with hosts and
		// cancel via unreachable blackhole IP with short path: we accept per-host
		// errors. Use 0 hosts is wrong. Fast approach: invoke CheckDisks; if top-level
		// err mentions key — fail. Use a host that fails DNS immediately.
		resp, err := checker.CheckDisks(&models.CheckDisksRequest{
			Hosts: []string{"invalid.invalid"},
			SSH: &models.SSHConfig{
				AuthType: "key",
				KeyPath:  "~/" + name,
				Login:    "root",
			},
		})
		if err != nil {
			t.Fatalf("CheckDisks with existing tilde key returned top-level error: %v", err)
		}
		if resp == nil || len(resp.Results) != 1 {
			t.Fatalf("expected 1 host result, got %+v", resp)
		}
	})

	t.Run("empty key path with empty hosts is ok", func(t *testing.T) {
		resp, err := checker.CheckDisks(&models.CheckDisksRequest{
			Hosts: []string{},
			SSH: &models.SSHConfig{
				AuthType: "key",
				KeyPath:  "",
				Login:    "root",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Results) != 0 {
			t.Errorf("expected empty results, got %d", len(resp.Results))
		}
	})
}

func TestValidateDisks_KeyPathValidation(t *testing.T) {
	checker := disks.NewChecker()

	t.Run("missing tilde key returns not found", func(t *testing.T) {
		_, err := checker.ValidateDisks(&models.ValidateDisksRequest{
			Hosts: []string{"host1.example.com"},
			Paths: []string{"/dev/vdb"},
			SSH: &models.SSHConfig{
				AuthType: "key",
				KeyPath:  "~/ydb-installer-missing-key-test-noexist",
				Login:    "root",
			},
		})
		if err == nil {
			t.Fatal("expected error for missing tilde key path")
		}
		if !strings.Contains(err.Error(), "SSH key file not found") {
			t.Errorf("error = %q, want message containing 'SSH key file not found'", err.Error())
		}
	})

	t.Run("existing tilde key passes top-level validation", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("home directory unavailable")
		}
		name := ".ydb-installer-disks-validate-key-test"
		full := filepath.Join(home, name)
		if err := os.WriteFile(full, []byte("test-key"), 0o600); err != nil {
			t.Fatalf("failed to create temp key: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(full) })

		resp, err := checker.ValidateDisks(&models.ValidateDisksRequest{
			Hosts: []string{"invalid.invalid"},
			Paths: []string{"/dev/vdb"},
			SSH: &models.SSHConfig{
				AuthType: "key",
				KeyPath:  "~/" + name,
				Login:    "root",
			},
		})
		if err != nil {
			t.Fatalf("ValidateDisks with existing tilde key returned top-level error: %v", err)
		}
		if resp == nil || len(resp.Results) != 1 {
			t.Fatalf("expected 1 host result, got %+v", resp)
		}
	})
}
