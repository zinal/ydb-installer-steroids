package unit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/config"
)

// buildManagedVars is a test helper that extracts the managed vars map
// from the InventoryWriter's BuildInventory output.
func buildManagedVars(t *testing.T, writer *config.InventoryWriter, data *models.InventoryData) map[string]interface{} {
	t.Helper()
	inv := writer.BuildInventory(data)
	all, ok := inv["all"].(map[string]interface{})
	if !ok {
		t.Fatal("BuildInventory: 'all' is not a map")
	}
	children, ok := all["children"].(map[string]interface{})
	if !ok {
		t.Fatal("BuildInventory: 'children' is not a map")
	}
	ydb, ok := children["ydb"].(map[string]interface{})
	if !ok {
		t.Fatal("BuildInventory: 'ydb' is not a map")
	}
	vars, ok := ydb["vars"].(map[string]interface{})
	if !ok {
		t.Fatal("BuildInventory: 'vars' is not a map")
	}
	return vars
}

func TestPatcher_PreservesComments(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        # This is a comment about ansible_user
        ansible_user: root
        # This comment should survive
        ydb_domain: Root
        # Custom user var with comment
        ydb_install_fq_connector: false
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		SSH: &models.SSHConfig{
			Login: "admin",
		},
		YDBConfig: &models.YDBConfig{
			Domain: "NewRoot",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "This is a comment about ansible_user") {
		t.Error("expected comment to be preserved")
	}
	if !strings.Contains(patchedStr, "This comment should survive") {
		t.Error("expected comment to be preserved")
	}
	if !strings.Contains(patchedStr, "Custom user var with comment") {
		t.Error("expected comment to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_install_fq_connector: false") {
		t.Error("expected unknown var to be preserved")
	}
	if !strings.Contains(patchedStr, "ansible_user: admin") {
		t.Error("expected ansible_user to be updated")
	}
	if !strings.Contains(patchedStr, "ydb_domain: NewRoot") {
		t.Error("expected ydb_domain to be updated")
	}
}

func TestPatcher_UpdatesExistingKey(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_domain: OldDomain
        ydb_cores_static: 4
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		YDBConfig: &models.YDBConfig{
			Domain:      "NewDomain",
			CoresStatic: 8,
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_domain: NewDomain") {
		t.Error("expected ydb_domain to be updated to 'NewDomain'")
	}
	if strings.Contains(patchedStr, "ydb_domain: OldDomain") {
		t.Error("expected no 'OldDomain' in output")
	}
	if !strings.Contains(patchedStr, "ydb_cores_static: 8") {
		t.Errorf("expected ydb_cores_static to be updated to 8, got:\n%s", patchedStr)
	}
}

func TestPatcher_AddsNewKey(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_domain: Root
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		YDBConfig: &models.YDBConfig{
			Domain:      "Root",
			CoresStatic: 8,
			PoolKind:    "ssd",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_pool_kind: ssd") {
		t.Errorf("expected new key 'ydb_pool_kind: ssd' to be added, got:\n%s", patchedStr)
	}
	if !strings.Contains(patchedStr, "ydb_cores_static: 8") {
		t.Errorf("expected new key 'ydb_cores_static: 8' to be added, got:\n%s", patchedStr)
	}
}

func TestPatcher_PreservesUnknownKeys(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_domain: Root
        ydb_install_fq_connector: false
        ydb_fq_connector_version: "v0.7.2-rc.5"
        ydb_load_blob_from_file_or_ds: true
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		YDBConfig: &models.YDBConfig{
			Domain: "NewDomain",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_install_fq_connector: false") {
		t.Error("expected unknown var to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_fq_connector_version:") {
		t.Error("expected unknown var to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_load_blob_from_file_or_ds: true") {
		t.Error("expected unknown var to be preserved")
	}
}

func TestPatcher_PreservesExtraGroups(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_domain: Root
    ungrouped:
      hosts:
        extra-host:
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		YDBConfig: &models.YDBConfig{
			Domain: "NewDomain",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ungrouped:") {
		t.Error("expected 'ungrouped' group to be preserved")
	}
	if !strings.Contains(patchedStr, "extra-host:") {
		t.Error("expected 'extra-host' to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_domain: NewDomain") {
		t.Error("expected ydb_domain to be updated")
	}
}

func TestPatcher_HandlesEmptyVars(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		YDBConfig: &models.YDBConfig{
			Domain: "NewDomain",
		},
		SSH: &models.SSHConfig{
			Login: "root",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_domain: NewDomain") {
		t.Errorf("expected ydb_domain to be added to empty vars, got:\n%s", patchedStr)
	}
	if !strings.Contains(patchedStr, "host1") {
		t.Error("expected host1 to remain in hosts section")
	}
}

func TestPatcher_HandlesMissingVars(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		YDBConfig: &models.YDBConfig{
			Domain: "NewDomain",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_domain: NewDomain") {
		t.Errorf("expected vars section to be created, got:\n%s", patchedStr)
	}
}

func TestPatcher_AddsHostsSection(t *testing.T) {
	raw := `all:
  children:
    ydb:
      vars:
        ydb_domain: Root
`

	data := &models.InventoryData{
		Hosts: []string{"new-host"},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "new-host:") {
		t.Errorf("expected hosts section with 'new-host' to be created, got:\n%s", patchedStr)
	}
	if !strings.Contains(patchedStr, "ydb_domain: Root") {
		t.Error("expected existing vars to be preserved")
	}
}

func TestPatcher_AlwaysPresentKeys(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_domain: Root
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		YDBConfig: &models.YDBConfig{
			Domain: "Root",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_config:") {
		t.Error("expected always-present key 'ydb_config' to be injected")
	}
	if !strings.Contains(patchedStr, "ydb_user:") {
		t.Error("expected default 'ydb_user' to be injected")
	}
	if !strings.Contains(patchedStr, "ydb_skip_data_loss_confirmation_prompt: true") {
		t.Error("expected 'ydb_skip_data_loss_confirmation_prompt: true' to be injected")
	}
	if !strings.Contains(patchedStr, "system_timezone: UTC") {
		t.Error("expected default 'system_timezone: UTC' to be injected")
	}
}

func TestPatcher_RoundTripIdentity(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
          some_host_var: value
      vars:
        ansible_user: root
        ydb_domain: Root
        ydb_cores_static: 4
        # Keep this comment
        custom_var: hello
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		SSH: &models.SSHConfig{
			Login: "root",
		},
		YDBConfig: &models.YDBConfig{
			Domain:      "Root",
			CoresStatic: 4,
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "Keep this comment") {
		t.Error("expected comment to be preserved in round-trip")
	}
	if !strings.Contains(patchedStr, "custom_var: hello") {
		t.Error("expected custom_var to be preserved")
	}
	if !strings.Contains(patchedStr, "some_host_var: value") {
		t.Error("expected host-specific var to be preserved")
	}
	if !strings.Contains(patchedStr, "location: dc-1") {
		t.Errorf("expected location: dc-1 on host1, got:\n%s", patchedStr)
	}
	if !strings.Contains(patchedStr, "ansible_user: root") {
		t.Error("expected ansible_user to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_domain: Root") {
		t.Error("expected ydb_domain to be preserved")
	}
}

func TestPatcher_DistributionOfficial(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_version: "25.3.1.25"
        ydb_archive: /tmp/old-archive.tar.gz
        ydbd_binary: /tmp/old-ydbd
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		Distribution: &models.DistributionConfig{
			Source:  "official",
			Version: "25.4.1.15",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_version: 25.4.1.15") {
		t.Errorf("expected ydb_version to be updated, got:\n%s", patchedStr)
	}
	if strings.Contains(patchedStr, "ydb_archive:") || strings.Contains(patchedStr, "ydbd_binary:") || strings.Contains(patchedStr, "ydb_cli_binary:") || strings.Contains(patchedStr, "ydbops_binary:") || strings.Contains(patchedStr, "ydb_dstool_binary:") {
		t.Errorf("expected local-specific distribution keys to be deleted when switching to official, got:\n%s", patchedStr)
	}
}

func TestPatcher_DistributionLocal(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_version: "25.3.1.25"
        ydb_archive: /tmp/old-archive.tar.gz
        ydbd_binary: /tmp/old-ydbd
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		Distribution: &models.DistributionConfig{
			Source:      "local",
			ArchivePath: "/tmp/new-archive.tar.gz",
			BinaryPath:  "/tmp/new-ydbd",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_archive: /tmp/new-archive.tar.gz") {
		t.Errorf("expected ydb_archive to be updated, got:\n%s", patchedStr)
	}
	if strings.Contains(patchedStr, "ydb_version:") {
		t.Errorf("expected ydb_version to be deleted when switching to local, got:\n%s", patchedStr)
	}
}

func TestPatcher_DatabaseSingleClearsYdbDatabases(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_domain: "/Root"
        ydb_database_groups: 8
        ydb_databases:
          - name: database
            groups: 8
          - name: analytics
            groups: 4
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
		Database: &models.DatabaseConfig{
			ConfigVariant: "single",
			Entries: []models.DatabaseEntry{
				{Name: "database", Groups: 8},
			},
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if strings.Contains(patchedStr, "ydb_databases:") {
		t.Errorf("expected ydb_databases to be deleted when configVariant is 'single', got:\n%s", patchedStr)
	}
	if !strings.Contains(patchedStr, "ydb_dbname: database") {
		t.Errorf("expected ydb_dbname to remain, got:\n%s", patchedStr)
	}
}

func TestPatcher_TLSClearDeletesKeys(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_tls_dir: /tmp/certs
        ydb_tls_mode: generate
        ydb_tls_days: "365"
`

	data := &models.InventoryData{
		Hosts: []string{"host1"},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if strings.Contains(patchedStr, "ydb_tls_dir") || strings.Contains(patchedStr, "ydb_tls_mode") || strings.Contains(patchedStr, "ydb_tls_days") {
		t.Errorf("expected ydb_tls_* keys to be deleted when TLS is not configured, got:\n%s", patchedStr)
	}
}

func TestWriteWithPatch_PreservesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "50-inventory.yaml")

	originalContent := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        # User SSH config
        ansible_user: root
        # Domain setting
        ydb_domain: OriginalDomain
        # Custom user var
        ydb_install_fq_connector: false
        ydb_fq_connector_version: "v0.7.2"
`

	if err := os.WriteFile(path, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	writer := config.NewInventoryWriter()
	updated := &models.InventoryData{
		Hosts: []string{"host1"},
		SSH: &models.SSHConfig{
			Login: "admin",
		},
		YDBConfig: &models.YDBConfig{
			Domain: "UpdatedDomain",
		},
	}

	if err := writer.WriteWithPatch(path, updated); err != nil {
		t.Fatalf("WriteWithPatch failed: %v", err)
	}

	patchedBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read patched file: %v", err)
	}

	patchedStr := string(patchedBytes)

	if !strings.Contains(patchedStr, "User SSH config") {
		t.Error("expected comment 'User SSH config' to be preserved")
	}
	if !strings.Contains(patchedStr, "Domain setting") {
		t.Error("expected comment 'Domain setting' to be preserved")
	}
	if !strings.Contains(patchedStr, "Custom user var") {
		t.Error("expected comment 'Custom user var' to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_install_fq_connector: false") {
		t.Error("expected unknown var to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_fq_connector_version:") {
		t.Error("expected fq_connector_version to be preserved")
	}
	if !strings.Contains(patchedStr, "ansible_user: admin") {
		t.Error("expected ansible_user to be updated")
	}
	if !strings.Contains(patchedStr, "ydb_domain: UpdatedDomain") {
		t.Error("expected ydb_domain to be updated")
	}
}

func TestWriteWithPatch_NoFile_FallsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-such-file.yaml")

	writer := config.NewInventoryWriter()
	data := &models.InventoryData{
		Hosts: []string{"host1"},
		SSH: &models.SSHConfig{
			Login: "root",
		},
		YDBConfig: &models.YDBConfig{
			Domain:      "NewDomain",
			CoresStatic: 4,
		},
	}

	if err := writer.WriteWithPatch(path, data); err != nil {
		t.Fatalf("WriteWithPatch (no file) failed: %v", err)
	}

	svc := config.NewInventoryService()
	parsed, err := svc.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse written file: %v", err)
	}

	if len(parsed.Hosts) != 1 || parsed.Hosts[0] != "host1" {
		t.Errorf("unexpected hosts: %v", parsed.Hosts)
	}
	if parsed.YDBConfig == nil || parsed.YDBConfig.Domain != "NewDomain" {
		t.Errorf("unexpected domain: %v", parsed.YDBConfig)
	}
}

func TestWriteWithPatch_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "50-inventory.yaml")

	content := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ydb_domain: Root
        ansible_user: root
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	writer := config.NewInventoryWriter()
	data := &models.InventoryData{
		Hosts: []string{"host1"},
		SSH: &models.SSHConfig{
			Login: "root",
		},
		YDBConfig: &models.YDBConfig{
			Domain: "Root",
		},
	}

	if err := writer.WriteWithPatch(path, data); err != nil {
		t.Fatalf("first WriteWithPatch failed: %v", err)
	}
	firstBytes, _ := os.ReadFile(path)

	if err := writer.WriteWithPatch(path, data); err != nil {
		t.Fatalf("second WriteWithPatch failed: %v", err)
	}
	secondBytes, _ := os.ReadFile(path)

	if string(firstBytes) != string(secondBytes) {
		t.Error("expected idempotent writes to produce identical output")
	}
}

func TestPatcher_ComplexInventory(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        node-1:
        node-2:
        node-3:
      vars:
        ansible_user: root
        ydb_domain: Root
        ydb_dc1_hosts:
          - node-1
        ydb_dc2_hosts:
          - node-2
        ydb_cores_static: 4
        ydb_disks:
          - name: /dev/vdb
            label: ydb_disk_1
          - name: /dev/vdc
            label: ydb_disk_2
`

	data := &models.InventoryData{
		Hosts: []string{"node-1", "node-2", "node-3"},
		DC1:   []string{"node-1"},
		DC2:   []string{"node-2"},
		DC3:   []string{"node-3"},
		YDBConfig: &models.YDBConfig{
			Domain:      "Root",
			CoresStatic: 8,
		},
		SSH: &models.SSHConfig{
			Login: "admin",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ansible_user: admin") {
		t.Error("expected ansible_user to be updated")
	}
	if !strings.Contains(patchedStr, "ydb_cores_static: 8") {
		t.Error("expected ydb_cores_static to be updated")
	}
	if !strings.Contains(patchedStr, "ydb_dc1_hosts:") {
		t.Error("expected ydb_dc1_hosts to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_dc2_hosts:") {
		t.Error("expected ydb_dc2_hosts to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_disks:") {
		t.Error("expected ydb_disks to be present")
	}
	if !strings.Contains(patchedStr, "node-3:") {
		t.Error("expected node-3 to be preserved in hosts")
	}
}

func TestPatcher_DCHostsPreserved(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        node-1:
        node-2:
        node-3:
      vars:
        ydb_domain: Root
        ydb_dc1_hosts:
          - node-1
        ydb_dc2_hosts:
          - node-2
        ydb_dc3_hosts:
          - node-3
`

	data := &models.InventoryData{
		Hosts: []string{"node-1", "node-2", "node-3"},
		DC1:   []string{"node-1"},
		DC2:   []string{"node-2"},
		DC3:   []string{"node-3"},
		YDBConfig: &models.YDBConfig{
			Domain: "Updated",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if !strings.Contains(patchedStr, "ydb_dc1_hosts:") {
		t.Error("expected ydb_dc1_hosts to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_dc2_hosts:") {
		t.Error("expected ydb_dc2_hosts to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_dc3_hosts:") {
		t.Error("expected ydb_dc3_hosts to be preserved")
	}
	if !strings.Contains(patchedStr, "ydb_domain: Updated") {
		t.Error("expected ydb_domain to be updated")
	}
}

func TestPatcher_UpdatesHosts(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        old-host-1:
        old-host-2:
      vars:
        ydb_domain: Root
`

	data := &models.InventoryData{
		Hosts: []string{"new-host-1", "new-host-2"},
		YDBConfig: &models.YDBConfig{
			Domain: "Root",
		},
	}

	writer := config.NewInventoryWriter()
	managedVars := buildManagedVars(t, writer, data)

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, data)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	patchedStr := string(patched)

	if strings.Contains(patchedStr, "old-host-1") {
		t.Error("expected old hosts to be removed")
	}
	if !strings.Contains(patchedStr, "new-host-1:") {
		t.Error("expected new-host-1 to be in hosts")
	}
	if !strings.Contains(patchedStr, "new-host-2:") {
		t.Error("expected new-host-2 to be in hosts")
	}
}

// TestBuildDynamicNodes_SingleDb_Default verifies that a single database
// without pre-configured nodes gets a default {a, 1} node.
func TestBuildDynamicNodes_SingleDb_Default(t *testing.T) {
	w := config.NewInventoryWriter()
	inv := &models.InventoryData{
		Database: &models.DatabaseConfig{
			Entries: []models.DatabaseEntry{
				{Name: "database", Groups: 8},
			},
		},
	}
	result := w.BuildInventory(inv)
	vars := result["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})
	dynNodes := vars["ydb_dynnodes"].([]interface{})
	if len(dynNodes) != 1 {
		t.Fatalf("expected 1 dynnode, got %d", len(dynNodes))
	}
	node := dynNodes[0].(map[string]interface{})
	if node["instance"] != "a" || node["offset"] != 1 {
		t.Errorf("expected {a, 1}, got {%v, %v}", node["instance"], node["offset"])
	}
	if _, hasDbname := node["dbname"]; hasDbname {
		t.Error("default node should not have dbname")
	}
}

// TestBuildDynamicNodes_MultiDb_AutoGenerated verifies that multiple
// databases auto-generate sequential instance letters with dbname.
func TestBuildDynamicNodes_MultiDb_AutoGenerated(t *testing.T) {
	w := config.NewInventoryWriter()
	inv := &models.InventoryData{
		Database: &models.DatabaseConfig{
			Entries: []models.DatabaseEntry{
				{Name: "database", Groups: 8},
				{Name: "analytics", Groups: 4},
				{Name: "testing", Groups: 2},
			},
		},
	}
	result := w.BuildInventory(inv)
	vars := result["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})
	dynNodes := vars["ydb_dynnodes"].([]interface{})
	if len(dynNodes) != 3 {
		t.Fatalf("expected 3 dynnodes, got %d", len(dynNodes))
	}
	n0 := dynNodes[0].(map[string]interface{})
	if n0["instance"] != "a" || n0["offset"] != 1 {
		t.Errorf("node 0: expected {a, 1}, got {%v, %v}", n0["instance"], n0["offset"])
	}
	if _, ok := n0["dbname"]; ok {
		t.Error("node 0 should not have dbname")
	}
	n1 := dynNodes[1].(map[string]interface{})
	if n1["instance"] != "b" || n1["offset"] != 2 || n1["dbname"] != "analytics" {
		t.Errorf("node 1: expected {b, 2, analytics}, got {%v, %v, %v}", n1["instance"], n1["offset"], n1["dbname"])
	}
	n2 := dynNodes[2].(map[string]interface{})
	if n2["instance"] != "c" || n2["offset"] != 3 || n2["dbname"] != "testing" {
		t.Errorf("node 2: expected {c, 3, testing}, got {%v, %v, %v}", n2["instance"], n2["offset"], n2["dbname"])
	}
}

// TestBuildDynamicNodes_PreconfiguredIgnored verifies that
// pre-configured instance/offset mappings are IGNORED in favor
// of pure position-based sequential assignment.
func TestBuildDynamicNodes_PreconfiguredIgnored(t *testing.T) {
	w := config.NewInventoryWriter()
	inv := &models.InventoryData{
		DynamicNodes: []models.DynamicNode{
			{Instance: "x", Offset: 10},
			{Instance: "y", Offset: 20, Dbname: "analytics"},
		},
		Database: &models.DatabaseConfig{
			Entries: []models.DatabaseEntry{
				{Name: "database", Groups: 8},
				{Name: "analytics", Groups: 4},
				{Name: "testing", Groups: 2},
			},
		},
	}
	result := w.BuildInventory(inv)
	vars := result["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})
	dynNodes := vars["ydb_dynnodes"].([]interface{})
	if len(dynNodes) != 3 {
		t.Fatalf("expected 3 dynnodes, got %d", len(dynNodes))
	}
	n0 := dynNodes[0].(map[string]interface{})
	if n0["instance"] != "a" || n0["offset"] != 1 {
		t.Errorf("node 0: expected {a, 1}, got {%v, %v}", n0["instance"], n0["offset"])
	}
	n1 := dynNodes[1].(map[string]interface{})
	if n1["instance"] != "b" || n1["offset"] != 2 || n1["dbname"] != "analytics" {
		t.Errorf("node 1: expected {b, 2, analytics}, got {%v, %v, %v}", n1["instance"], n1["offset"], n1["dbname"])
	}
	n2 := dynNodes[2].(map[string]interface{})
	if n2["instance"] != "c" || n2["offset"] != 3 || n2["dbname"] != "testing" {
		t.Errorf("node 2: expected {c, 3, testing}, got {%v, %v, %v}", n2["instance"], n2["offset"], n2["dbname"])
	}
}

// TestBuildDynamicNodes_NoDatabaseEntries verifies behavior when there
// are no Database.Entries — existing DynamicNodes should be preserved.
func TestBuildDynamicNodes_NoDatabaseEntries(t *testing.T) {
	w := config.NewInventoryWriter()
	inv := &models.InventoryData{
		DynamicNodes: []models.DynamicNode{
			{Instance: "a", Offset: 1},
			{Instance: "b", Offset: 2, Dbname: "sales"},
		},
	}
	result := w.BuildInventory(inv)
	vars := result["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})
	dynNodes := vars["ydb_dynnodes"].([]interface{})
	if len(dynNodes) != 2 {
		t.Fatalf("expected 2 dynnodes, got %d", len(dynNodes))
	}
	n0 := dynNodes[0].(map[string]interface{})
	if n0["instance"] != "a" || n0["offset"] != 1 {
		t.Errorf("node 0: expected {a, 1}, got {%v, %v}", n0["instance"], n0["offset"])
	}
	n1 := dynNodes[1].(map[string]interface{})
	if n1["instance"] != "b" || n1["offset"] != 2 || n1["dbname"] != "sales" {
		t.Errorf("node 1: expected {b, 2, sales}, got {%v, %v, %v}", n1["instance"], n1["offset"], n1["dbname"])
	}
}

// TestBuildDynamicNodes_RenamedDatabase verifies that when a database is
// renamed, the old node's instance does NOT block the new name. The old
// node (whose dbname no longer matches any entry) should be ignored, and
// the new name should get the next sequential instance letter.
func TestBuildDynamicNodes_RenamedDatabase(t *testing.T) {
	w := config.NewInventoryWriter()
	// Simulate: user had "database" + "database2", then renamed "database2" → "analytics".
	// The session still has the old DynamicNodes with dbname "database2".
	inv := &models.InventoryData{
		DynamicNodes: []models.DynamicNode{
			{Instance: "a", Offset: 1},
			{Instance: "b", Offset: 2, Dbname: "database2"},
		},
		Database: &models.DatabaseConfig{
			ConfigVariant: "multiple",
			Entries: []models.DatabaseEntry{
				{Name: "database", Groups: 8},
				{Name: "analytics", Groups: 4},
			},
		},
	}
	result := w.BuildInventory(inv)
	vars := result["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})
	dynNodes := vars["ydb_dynnodes"].([]interface{})
	if len(dynNodes) != 2 {
		t.Fatalf("expected 2 dynnodes, got %d", len(dynNodes))
	}
	n0 := dynNodes[0].(map[string]interface{})
	if n0["instance"] != "a" || n0["offset"] != 1 {
		t.Errorf("node 0: expected {a, 1}, got {%v, %v}", n0["instance"], n0["offset"])
	}
	if _, hasDbname := n0["dbname"]; hasDbname {
		t.Error("node 0 should not have dbname")
	}
	n1 := dynNodes[1].(map[string]interface{})
	if n1["instance"] != "b" || n1["offset"] != 2 || n1["dbname"] != "analytics" {
		t.Errorf("node 1: expected {b, 2, analytics}, got {%v, %v, %v}", n1["instance"], n1["offset"], n1["dbname"])
	}
}

// TestBuildDynamicNodes_Entry0MatchedByName verifies that Entry 0 matches
// an existing node by dbname when the first existing node has a dbname
// (e.g., after a previous installation added dbname to the first node).
func TestBuildDynamicNodes_Entry0MatchedByName(t *testing.T) {
	w := config.NewInventoryWriter()
	// Simulate: after a previous installation, both nodes have dbname.
	inv := &models.InventoryData{
		DynamicNodes: []models.DynamicNode{
			{Instance: "a", Offset: 1, Dbname: "database"},
			{Instance: "b", Offset: 2, Dbname: "database2"},
		},
		Database: &models.DatabaseConfig{
			ConfigVariant: "multiple",
			Entries: []models.DatabaseEntry{
				{Name: "database", Groups: 8},
				{Name: "database2", Groups: 4},
			},
		},
	}
	result := w.BuildInventory(inv)
	vars := result["all"].(map[string]interface{})["children"].(map[string]interface{})["ydb"].(map[string]interface{})["vars"].(map[string]interface{})
	dynNodes := vars["ydb_dynnodes"].([]interface{})
	if len(dynNodes) != 2 {
		t.Fatalf("expected 2 dynnodes, got %d", len(dynNodes))
	}
	n0 := dynNodes[0].(map[string]interface{})
	if n0["instance"] != "a" || n0["offset"] != 1 {
		t.Errorf("node 0: expected {a, 1}, got {%v, %v}", n0["instance"], n0["offset"])
	}
	if _, hasDbname := n0["dbname"]; hasDbname {
		t.Error("node 0 should not have dbname")
	}
	n1 := dynNodes[1].(map[string]interface{})
	if n1["instance"] != "b" || n1["offset"] != 2 || n1["dbname"] != "database2" {
		t.Errorf("node 1: expected {b, 2, database2}, got {%v, %v, %v}", n1["instance"], n1["offset"], n1["dbname"])
	}
}

func TestPatcher_YDBDefaultsApplied(t *testing.T) {
	raw := `all:
  children:
    ydb:
      hosts:
        host1:
      vars:
        ansible_user: root
`

	managedVars := map[string]interface{}{
		"ansible_user": "root",
	}

	patcher := config.NewInventoryPatcher()
	patched, err := patcher.Patch([]byte(raw), managedVars, nil, config.YdbAlwaysPresentKeys...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	patchedStr := string(patched)

	requiredKeys := []string{
		"ydb_user:",
		"system_timezone:",
		"system_ntp_servers:",
		"ydbops_local:",
		"ydb_config:",
		"ydb_skip_data_loss_confirmation_prompt:",
	}

	for _, key := range requiredKeys {
		if !strings.Contains(patchedStr, key) {
			t.Errorf("YDB default %q missing from patched output:\n%s", key, patchedStr)
		}
	}

	if !strings.Contains(patchedStr, "ansible_user:") {
		t.Error("existing ansible_user should be preserved")
	}
}
