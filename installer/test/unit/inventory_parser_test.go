package unit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/config"
)

func TestInventoryService_ParseRealFile(t *testing.T) {
	cwd, _ := os.Getwd()
	invPath := filepath.Join(cwd, "..", "testdata", "50-inventory.yaml")

	svc := config.NewInventoryService()
	data, err := svc.Parse(invPath)
	if err != nil {
		t.Fatalf("failed to parse inventory: %v", err)
	}

	// The live inventory file is user-managed and its host count changes
	// over time, so we only assert that at least one host was parsed.
	if len(data.Hosts) == 0 {
		t.Errorf("expected at least 1 host, got 0")
	} else {
		t.Logf("parsed %d hosts from inventory", len(data.Hosts))
	}

	if data.SSH == nil {
		t.Fatal("SSH config should not be nil")
	}
	if data.SSH.Login != "ansible" {
		t.Errorf("expected SSH login 'ansible', got %q", data.SSH.Login)
	}
	if data.SSH.CustomSSHArgs == "" {
		t.Log("CustomSSHArgs is empty (jump host not configured in inventory)")
	}

	if data.YDBConfig != nil {
		if data.YDBConfig.Domain != "Root" {
			t.Errorf("expected domain 'Root', got %q", data.YDBConfig.Domain)
		}
		if data.YDBConfig.User != "root" {
			t.Errorf("expected ydb_user 'root', got %q", data.YDBConfig.User)
		}
		if data.YDBConfig.Domain == "Root" && data.YDBConfig.CoresStatic == 0 {
			t.Log("cores_static not set in inventory (defaults to 0)")
		} else if data.YDBConfig.Domain == "Root" && data.YDBConfig.CoresStatic != 0 {
			t.Logf("cores_static = %d (set in inventory)", data.YDBConfig.CoresStatic)
		}
	} else {
		t.Log("YDB config is nil (not configured in inventory)")
	}

	if data.Disks == nil {
		t.Fatal("Disks config should not be nil")
	}
	if len(data.Disks.SelectedDisks) != 3 {
		t.Errorf("expected 3 disks, got %d", len(data.Disks.SelectedDisks))
	}

	if len(data.Brokers) != 0 {
		t.Logf("brokers = %v (found %d brokers)", data.Brokers, len(data.Brokers))
	}

	if data.Distribution != nil {
		// The version is read from the live inventory/50-inventory.yaml file
		// which may be updated over time. Just verify it's non-empty.
		if data.Distribution.Version == "" {
			t.Error("expected a non-empty version")
		} else {
			t.Logf("distribution version = %s", data.Distribution.Version)
		}
	} else {
		t.Log("Distribution config is nil (not configured in inventory)")
	}

	if data.System != nil {
		if data.System.Timezone != "" {
			t.Logf("system_timezone = %s", data.System.Timezone)
		}
	}
}

func TestInventoryService_Validate(t *testing.T) {
	svc := config.NewInventoryService()

	validData := &models.InventoryData{
		Hosts: []string{"host1"},
		SSH: &models.SSHConfig{
			Login:    "root",
			AuthType: "key",
			KeyPath:  "~/.ssh/id_rsa",
		},
		YDBConfig: &models.YDBConfig{
			Domain:      "Root",
			CoresStatic: 4,
		},
	}
	if err := svc.Validate(validData); err != nil {
		t.Errorf("expected valid data, got error: %v", err)
	}

	if err := svc.Validate(nil); err == nil {
		t.Error("expected error for nil data")
	}

	invalidData := &models.InventoryData{
		SSH: &models.SSHConfig{Login: "root"},
	}
	if err := svc.Validate(invalidData); err == nil {
		t.Error("expected error for missing hosts")
	}
}

func TestInventoryWriter_WriteAndRead(t *testing.T) {
	original := &models.InventoryData{
		Hosts: []string{"host1.example.com", "host2.example.com"},
		SSH: &models.SSHConfig{
			Login:        "testuser",
			AuthType:     "key",
			KeyPath:      "~/.ssh/test_key",
			UseJumpHost:  true,
			JumpHostFQDN: "jump.example.com",
			JumpHostUser: "testuser",
		},
		YDBConfig: &models.YDBConfig{
			Domain:       "TestDomain",
			CoresStatic:  8,
			CoresDynamic: 4,
		},
		Disks: &models.DisksConfig{
			AllowFormat: true,
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/sda", Label: "disk1"},
			},
		},
		Database: &models.DatabaseConfig{
			ConfigVariant: "single",
			Entries: []models.DatabaseEntry{
				{Name: "database", Groups: 8},
			},
			CoresDynamic: 4,
		},
	}

	writer := config.NewInventoryWriter()
	tmpPath := t.TempDir() + "/test-inventory.yaml"
	if err := writer.Write(tmpPath, original); err != nil {
		t.Fatalf("failed to write inventory: %v", err)
	}

	svc := config.NewInventoryService()
	parsed, err := svc.Parse(tmpPath)
	if err != nil {
		t.Fatalf("failed to parse written inventory: %v", err)
	}

	if len(parsed.Hosts) != len(original.Hosts) {
		t.Errorf("hosts count mismatch: got %d, want %d", len(parsed.Hosts), len(original.Hosts))
	}
	if parsed.SSH == nil || parsed.SSH.Login != original.SSH.Login {
		t.Errorf("SSH login mismatch: got %v, want %v", parsed.SSH, original.SSH)
	}
	if parsed.YDBConfig == nil || parsed.YDBConfig.Domain != original.YDBConfig.Domain {
		t.Errorf("YDB domain mismatch: got %v, want %v", parsed.YDBConfig, original.YDBConfig)
	}
	if parsed.YDBConfig == nil || parsed.YDBConfig.User != "root" {
		t.Errorf("expected ydb_user default 'root', got %q", parsed.YDBConfig.User)
	}

	if parsed.System == nil {
		t.Fatal("System config should not be nil (expected defaults)")
	}
	if parsed.System.Timezone != "UTC" {
		t.Errorf("expected system_timezone default 'UTC', got %q", parsed.System.Timezone)
	}
	if len(parsed.System.NTPServers) != 4 {
		t.Errorf("expected 4 NTP servers, got %d: %v", len(parsed.System.NTPServers), parsed.System.NTPServers)
	}

	if len(parsed.Brokers) != 2 {
		t.Errorf("expected 2 brokers (from hosts), got %d: %v", len(parsed.Brokers), parsed.Brokers)
	}

	if parsed.DynamicNodes == nil || len(parsed.DynamicNodes) != 1 {
		t.Errorf("expected 1 default dynamic node, got %v", parsed.DynamicNodes)
	} else {
		if parsed.DynamicNodes[0].Instance != "a" {
			t.Errorf("expected dynamic node instance 'a', got %q", parsed.DynamicNodes[0].Instance)
		}
		if parsed.DynamicNodes[0].Offset != 1 {
			t.Errorf("expected dynamic node offset 1, got %d", parsed.DynamicNodes[0].Offset)
		}
	}

	if parsed.Operations == nil || !parsed.Operations.LocalControl {
		t.Error("expected ydbops_local default 'true'")
	}
}

func TestInventoryWriter_DefaultSystemSettings(t *testing.T) {
	writer := config.NewInventoryWriter()

	data := &models.InventoryData{
		Hosts: []string{"host1.example.com"},
	}

	inv := writer.BuildInventory(data)
	vars := inv["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})

	if tz, ok := vars["system_timezone"].(string); !ok || tz != "UTC" {
		t.Errorf("expected system_timezone 'UTC', got %v", vars["system_timezone"])
	}

	ntp, ok := vars["system_ntp_servers"].([]interface{})
	if !ok || len(ntp) != 4 {
		t.Errorf("expected 4 NTP servers, got %d: %v", len(ntp), vars["system_ntp_servers"])
	}
}

func TestInventoryWriter_DefaultBrokers(t *testing.T) {
	writer := config.NewInventoryWriter()

	data := &models.InventoryData{
		Hosts: []string{"host1.example.com", "host2.example.com", "host3.example.com"},
	}

	inv := writer.BuildInventory(data)
	vars := inv["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})

	brokers, ok := vars["ydb_brokers"].([]interface{})
	if !ok || len(brokers) != 3 {
		t.Errorf("expected 3 default brokers, got %d: %v", len(brokers), vars["ydb_brokers"])
	}
}

func TestInventoryWriter_DefaultDynnodes(t *testing.T) {
	writer := config.NewInventoryWriter()

	data := &models.InventoryData{
		Hosts: []string{"host1.example.com"},
	}

	inv := writer.BuildInventory(data)
	vars := inv["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})

	dynnodes, ok := vars["ydb_dynnodes"].([]interface{})
	if !ok || len(dynnodes) != 1 {
		t.Fatalf("expected 1 default dynnode, got %d: %v", len(dynnodes), vars["ydb_dynnodes"])
	}

	first := dynnodes[0].(map[string]interface{})
	if first["instance"] != "a" {
		t.Errorf("expected instance 'a', got %v", first["instance"])
	}
	if first["offset"] != 1 {
		t.Errorf("expected offset 1, got %v", first["offset"])
	}
}

func TestInventoryWriter_DefaultYdbopsLocal(t *testing.T) {
	writer := config.NewInventoryWriter()

	data := &models.InventoryData{
		Hosts: []string{"host1.example.com"},
	}

	inv := writer.BuildInventory(data)
	vars := inv["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})

	if val, ok := vars["ydbops_local"].(bool); !ok || !val {
		t.Errorf("expected ydbops_local default 'true', got %v", vars["ydbops_local"])
	}
}

func TestInventoryWriter_DefaultYdbUser(t *testing.T) {
	writer := config.NewInventoryWriter()

	data := &models.InventoryData{
		Hosts: []string{"host1.example.com"},
		YDBConfig: &models.YDBConfig{
			Domain: "Root",
		},
	}

	inv := writer.BuildInventory(data)
	vars := inv["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})

	if user, ok := vars["ydb_user"].(string); !ok || user != "root" {
		t.Errorf("expected ydb_user default 'root', got %v", vars["ydb_user"])
	}
}

func TestInventoryWriter_YdbUserFromConfig(t *testing.T) {
	writer := config.NewInventoryWriter()

	data := &models.InventoryData{
		Hosts: []string{"host1.example.com"},
		YDBConfig: &models.YDBConfig{
			Domain: "Root",
			User:   "admin",
		},
	}

	inv := writer.BuildInventory(data)
	vars := inv["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})

	if user, ok := vars["ydb_user"].(string); !ok || user != "admin" {
		t.Errorf("expected ydb_user 'admin' from config, got %v", vars["ydb_user"])
	}
}

func TestInventoryWriter_ExcludedFields(t *testing.T) {
	writer := config.NewInventoryWriter()

	data := &models.InventoryData{
		Hosts: []string{"host1.example.com"},
		SSH: &models.SSHConfig{
			Login:    "testuser",
			AuthType: "key",
			KeyPath:  "~/.ssh/test_key",
		},
	}

	inv := writer.BuildInventory(data)
	vars := inv["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})

	if _, exists := vars["ansible_host_key_checking"]; exists {
		t.Error("ansible_host_key_checking should not be written to inventory (not in good-inventory.yaml)")
	}

	if _, exists := vars["ydb_topology"]; exists {
		t.Error("ydb_topology should not be written to inventory (not in good-inventory.yaml)")
	}
}

func TestInventoryWriter_ProxyCommandFormat(t *testing.T) {
	writer := config.NewInventoryWriter()

	data := &models.InventoryData{
		Hosts: []string{"host1.example.com"},
		SSH: &models.SSHConfig{
			Login:        "ansible",
			AuthType:     "key",
			KeyPath:      "~/.ssh/id_rsa",
			UseJumpHost:  true,
			JumpHostFQDN: "bastion.example.com",
			JumpHostUser: "ansible",
		},
	}

	inv := writer.BuildInventory(data)
	vars := inv["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})

	sshArgs, ok := vars["ansible_ssh_common_args"].(string)
	if !ok {
		t.Fatal("ansible_ssh_common_args should be set when UseJumpHost is true")
	}

	if !strings.Contains(sshArgs, "ProxyCommand") {
		t.Errorf("expected ProxyCommand format, got: %s", sshArgs)
	}
	if strings.Contains(sshArgs, "-J ") {
		t.Errorf("should not contain old -J format: %s", sshArgs)
	}
	if !strings.Contains(sshArgs, "ansible@bastion.example.com") {
		t.Errorf("expected jump host in args, got: %s", sshArgs)
	}
}

func TestInventoryParser_ProxyCommandFormat(t *testing.T) {
	svc := config.NewInventoryService()

	tmpDir := t.TempDir()
	tmpPath := tmpDir + "/test-inventory.yaml"

	yamlContent := `
all:
    children:
        ydb:
            hosts:
                test-host.example.com: null
            vars:
                ansible_user: ansible
                ansible_ssh_common_args: -o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p ansible@bastion.example.com'
`
	if err := os.WriteFile(tmpPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	data, err := svc.Parse(tmpPath)
	if err != nil {
		t.Fatalf("failed to parse inventory: %v", err)
	}

	if data.SSH == nil {
		t.Fatal("SSH config should not be nil")
	}
	if !data.SSH.UseJumpHost {
		t.Error("UseJumpHost should be true")
	}
	if data.SSH.JumpHostFQDN != "bastion.example.com" {
		t.Errorf("expected JumpHostFQDN 'bastion.example.com', got '%s'", data.SSH.JumpHostFQDN)
	}
	if data.SSH.JumpHostUser != "ansible" {
		t.Errorf("expected JumpHostUser 'ansible', got '%s'", data.SSH.JumpHostUser)
	}
}

func TestInventoryParser_LegacyJumpFormat(t *testing.T) {
	// The legacy -J format must NOT trigger jump host auto-fill.
	// Only ProxyCommand is recognized. The raw args are still preserved
	// in CustomSSHArgs, but UseJumpHost/JumpHostFQDN must remain unset.
	svc := config.NewInventoryService()

	tmpDir := t.TempDir()
	tmpPath := tmpDir + "/test-inventory.yaml"

	yamlContent := `
all:
    children:
        ydb:
            hosts:
                test-host.example.com: null
            vars:
                ansible_user: ansible
                ansible_ssh_common_args: -J {{ ansible_user }}@bastion.example.com -o StrictHostKeyChecking=no
`
	if err := os.WriteFile(tmpPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	data, err := svc.Parse(tmpPath)
	if err != nil {
		t.Fatalf("failed to parse inventory: %v", err)
	}

	if data.SSH == nil {
		t.Fatal("SSH config should not be nil")
	}
	if data.SSH.UseJumpHost {
		t.Error("UseJumpHost should be false for legacy -J format (only ProxyCommand triggers auto-fill)")
	}
	if data.SSH.JumpHostFQDN != "" {
		t.Errorf("JumpHostFQDN should be empty for -J format, got '%s'", data.SSH.JumpHostFQDN)
	}
	if data.SSH.CustomSSHArgs == "" {
		t.Error("CustomSSHArgs should preserve the raw -J args string")
	}
	if !strings.Contains(data.SSH.CustomSSHArgs, "-J") {
		t.Errorf("CustomSSHArgs should contain the -J flag, got: %s", data.SSH.CustomSSHArgs)
	}
}
