package unit_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"ydb-installer/internal/models"
	config "ydb-installer/internal/services/config"
)

// TestConfigGenerator_BuildHostsFromDCLists verifies that hosts are correctly
// assigned to their DC zones when DC-specific lists are provided.
// With mirror-3-dc topology and N=5 (<9) hosts, only the first host per DC
// gets all disks in fail_domains; extra hosts appear only in the hosts section.
func TestConfigGenerator_BuildHostsFromDCLists(t *testing.T) {
	inv := &models.InventoryData{
		DC1: []string{"host-a1.ydb.test", "host-a2.ydb.test"},
		DC2: []string{"host-b1.ydb.test"},
		DC3: []string{"host-d1.ydb.test", "host-d2.ydb.test"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
				{Path: "/dev/vdc", Label: "ydb_disk_2"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	tests := []struct {
		host     string
		wantZone string
		wantBody int
	}{
		{"host-a1.ydb.test", "dc-1", 1},
		{"host-a2.ydb.test", "dc-1", 2},
		{"host-b1.ydb.test", "dc-2", 3},
		{"host-d1.ydb.test", "dc-3", 4},
		{"host-d2.ydb.test", "dc-3", 5},
	}

	for _, tt := range tests {
		zoneMarker := "data_center: '" + tt.wantZone + "'"
		bodyMarker := "body: " + strconv.Itoa(tt.wantBody)
		if !strings.Contains(content, zoneMarker) {
			t.Errorf("host %s: expected zone %s, not found in config", tt.host, tt.wantZone)
		}
		if !strings.Contains(content, bodyMarker) {
			t.Errorf("host %s: expected body %d, not found in config", tt.host, tt.wantBody)
		}
	}

	rings := strings.Count(content, "fail_domains:")
	if rings != 3 {
		t.Errorf("expected 3 fail_domain groups (one per DC), got %d", rings)
	}

	zoneAVDiskCount := strings.Count(content, "node_id: host-a")
	zoneBVDiskCount := strings.Count(content, "node_id: host-b")
	zoneDVDiskCount := strings.Count(content, "node_id: host-d")

	if zoneAVDiskCount != 2 {
		t.Errorf("dc-1: expected 2 vdisk_locations (host-a1 × 2 disks), got %d", zoneAVDiskCount)
	}
	if zoneBVDiskCount != 2 {
		t.Errorf("dc-2: expected 2 vdisk_locations (host-b1 × 2 disks), got %d", zoneBVDiskCount)
	}
	if zoneDVDiskCount != 2 {
		t.Errorf("dc-3: expected 2 vdisk_locations (host-d1 × 2 disks), got %d", zoneDVDiskCount)
	}

	if strings.Contains(content, "node_id: host-a2.ydb.test") {
		t.Error("host-a2 should NOT appear in fail_domains (extra host, only in hosts section)")
	}
	if strings.Contains(content, "node_id: host-d2.ydb.test") {
		t.Error("host-d2 should NOT appear in fail_domains (extra host, only in hosts section)")
	}
}

// TestConfigGenerator_RoundRobinFallback verifies backward-compatible round-robin
// distribution when DC-specific lists are not provided.
// With mirror-3-dc topology and N=3 (<9) hosts, Case 1 applies:
// each host (1 per DC) gets all disks in its fail domain.
func TestConfigGenerator_RoundRobinFallback(t *testing.T) {
	inv := &models.InventoryData{
		Hosts: []string{"host1.ydb.test", "host2.ydb.test", "host3.ydb.test"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, _ := gen.Read()

	if !strings.Contains(content, "host1") || !strings.Contains(content, "host3") {
		t.Error("expected all 3 hosts in config")
	}

	vdiskCount := strings.Count(content, "vdisk_locations:")
	if vdiskCount != 3 {
		t.Errorf("expected 3 vdisk_locations (3 hosts × 1 disk), got %d", vdiskCount)
	}
}

// TestConfigGenerator_NoDisks verifies generation succeeds with no disks configured.
func TestConfigGenerator_NoDisks(t *testing.T) {
	inv := &models.InventoryData{
		DC1: []string{"host1.ydb.test"},
		DC2: []string{"host2.ydb.test"},
		DC3: []string{"host3.ydb.test"},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, _ := gen.Read()
	if !strings.Contains(content, "host1.ydb.test") {
		t.Error("expected host in config even without disks")
	}
}

// TestConfigGenerator_EmptyHosts verifies generation handles empty hosts gracefully.
func TestConfigGenerator_EmptyHosts(t *testing.T) {
	inv := &models.InventoryData{}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed even with empty hosts: %v", err)
	}

	content, _ := gen.Read()
	if !strings.Contains(content, "static_erasure: mirror-3-dc") {
		t.Error("expected default topology in output")
	}
}

// TestConfigGenerator_Backup verifies backup file creation.
func TestConfigGenerator_Backup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialContent := "# initial config"
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	gen := config.NewGenerator(configPath)
	backupPath, err := gen.Backup()
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if backupPath == "" {
		t.Fatal("expected non-empty backup path")
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file not found at %s", backupPath)
	}

	gen2 := config.NewGenerator(filepath.Join(tmpDir, "nonexistent.yaml"))
	emptyPath, err := gen2.Backup()
	if err != nil {
		t.Fatalf("Backup on non-existent file should not error: %v", err)
	}
	if emptyPath != "" {
		t.Errorf("expected empty path for non-existent file, got %s", emptyPath)
	}
}

// TestConfigGenerator_ReadNonExistent verifies Read returns empty string for non-existent file.
func TestConfigGenerator_ReadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "nonexistent.yaml"))
	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read should not error on non-existent file: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty content, got %q", content)
	}
}

// TestAllHostsMethod verifies AllHosts() correctly combines DC lists.
func TestAllHostsMethod(t *testing.T) {
	inv := &models.InventoryData{
		Hosts: []string{"legacy1", "legacy2"},
	}
	hosts := inv.AllHosts()
	if len(hosts) != 2 {
		t.Errorf("AllHosts from flat list: expected 2, got %d", len(hosts))
	}

	inv.DC1 = []string{"dc1-host1", "dc1-host2"}
	inv.DC2 = []string{"dc2-host1"}
	hosts = inv.AllHosts()
	if len(hosts) != 3 {
		t.Errorf("AllHosts from DC lists: expected 3, got %d: %v", len(hosts), hosts)
	}

	inv.SyncHostsFromDC()
	if len(inv.Hosts) != 3 {
		t.Errorf("SyncHostsFromDC: expected 3, got %d: %v", len(inv.Hosts), inv.Hosts)
	}
}

// TestConfigGenerator_GenerateConfigFile verifies the generated config.yaml
// is written to the correct output path and contains all required sections.
func TestConfigGenerator_GenerateConfigFile(t *testing.T) {
	inv := &models.InventoryData{
		DC1: []string{"node1.ydb.test"},
		DC2: []string{"node2.ydb.test"},
		DC3: []string{"node3.ydb.test"},
		YDBConfig: &models.YDBConfig{
			Domain:                      "Root",
			CoresStatic:                 4,
			EnforceUserTokenRequirement: true,
			RequestClientCertificate:    true,
			PoolKind:                    "ssd",
			Password:                    "test-password",
		},
		Disks: &models.DisksConfig{
			AllowFormat: true,
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/sda", Label: "disk1"},
				{Path: "/dev/sdb", Label: "disk2"},
			},
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "files", "config.yaml")
	gen := config.NewGenerator(configPath)

	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config.yaml not found at %s", configPath)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	sections := []string{
		"static_erasure: mirror-3-dc",
		"host_config_id: 1",
		"data_center: 'dc-1'",
		"data_center: 'dc-2'",
		"data_center: 'dc-3'",
		"domain:",
		"- name: Root",
		"blob_storage_config:",
		"service_set:",
		"groups:",
		"interconnect_config:",
		"grpc_config:",
		"auth_config:",
		"feature_flags:",
	}

	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("expected section %q in generated config", section)
		}
	}
}

// TestConfigGenerator_RegenerationRemovesExtraSections verifies that
// regeneration completely overwrites the config file, removing any extra
// sections that were present in a prior manual config.
func TestConfigGenerator_RegenerationRemovesExtraSections(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	oldConfig := `# Custom user config
storage_config_generation: 0
extra_section_that_should_not_survive:
		some_key: some_value
another_extra:
		- item1
		- item2
hosts: []
`
	if err := os.WriteFile(configPath, []byte(oldConfig), 0644); err != nil {
		t.Fatalf("failed to write old config: %v", err)
	}

	inv := &models.InventoryData{
		DC1: []string{"host-a1.ydb.test"},
		DC2: []string{"host-b1.ydb.test"},
		DC3: []string{"host-d1.ydb.test"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}

	gen := config.NewGenerator(configPath)
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if strings.Contains(content, "extra_section_that_should_not_survive") {
		t.Error("regeneration did NOT remove extra section 'extra_section_that_should_not_survive'")
	}
	if strings.Contains(content, "another_extra") {
		t.Error("regeneration did NOT remove extra section 'another_extra'")
	}

	sections := []string{
		"storage_config_generation:",
		"host_configs:",
		"hosts:",
		"domains_config:",
		"blob_storage_config:",
		"interconnect_config:",
		"grpc_config:",
		"auth_config:",
		"feature_flags:",
	}
	for _, s := range sections {
		if !strings.Contains(content, s) {
			t.Errorf("expected standard section %q after regeneration, not found", s)
		}
	}
}

// TestConfigGenerator_Mirror3DC_LessThan9Hosts verifies that for mirror-3-dc
// with N=6 (<9) hosts, only the first host per DC gets all 3 disks in
// fail_domains, producing exactly 9 vdisk_locations (3 hosts × 3 disks).
// Extra hosts (a2, b2, d2) appear only in the hosts section.
func TestConfigGenerator_Mirror3DC_LessThan9Hosts(t *testing.T) {
	inv := &models.InventoryData{
		Topology: &models.TopologyConfig{Type: "mirror-3-dc"},
		DC1:      []string{"host-a1", "host-a2"},
		DC2:      []string{"host-b1", "host-b2"},
		DC3:      []string{"host-d1", "host-d2"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/disk1", Label: "d1"},
				{Path: "/dev/disk2", Label: "d2"},
				{Path: "/dev/disk3", Label: "d3"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if rings := strings.Count(content, "fail_domains:"); rings != 3 {
		t.Errorf("expected 3 fail_domain groups, got %d", rings)
	}

	if vdc := strings.Count(content, "vdisk_locations:"); vdc != 9 {
		t.Errorf("expected 9 vdisk_locations (first host per DC × 3 disks), got %d", vdc)
	}

	for _, h := range []string{"host-a1", "host-a2", "host-b1", "host-b2", "host-d1", "host-d2"} {
		if !strings.Contains(content, "host: "+h) {
			t.Errorf("expected host %q in hosts section", h)
		}
	}

	for _, h := range []string{"host-a2", "host-b2", "host-d2"} {
		if strings.Contains(content, "node_id: "+h) {
			t.Errorf("host %q should NOT appear in fail_domains (extra host, only in hosts)", h)
		}
	}

	for _, h := range []string{"host-a1", "host-b1", "host-d1"} {
		count := strings.Count(content, "node_id: "+h)
		if count != 3 {
			t.Errorf("host %q expected 3 vdisk_locations in fail_domains, got %d", h, count)
		}
	}
}

// TestConfigGenerator_Mirror3DC_9Hosts verifies that for mirror-3-dc
// with N=9 hosts, the first 3 hosts per DC each get 1 disk, producing
// exactly 9 vdisk_locations (9 hosts × 1 disk).
func TestConfigGenerator_Mirror3DC_9Hosts(t *testing.T) {
	inv := &models.InventoryData{
		Topology: &models.TopologyConfig{Type: "mirror-3-dc"},
		DC1:      []string{"host-a1", "host-a2", "host-a3"},
		DC2:      []string{"host-b1", "host-b2", "host-b3"},
		DC3:      []string{"host-d1", "host-d2", "host-d3"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/disk1", Label: "d1"},
				{Path: "/dev/disk2", Label: "d2"},
				{Path: "/dev/disk3", Label: "d3"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if rings := strings.Count(content, "fail_domains:"); rings != 3 {
		t.Errorf("expected 3 fail_domain groups, got %d", rings)
	}

	if vdc := strings.Count(content, "vdisk_locations:"); vdc != 9 {
		t.Errorf("expected 9 vdisk_locations (9 hosts × 1 disk), got %d", vdc)
	}

	allHosts := []string{"host-a1", "host-a2", "host-a3", "host-b1", "host-b2", "host-b3", "host-d1", "host-d2", "host-d3"}
	for _, h := range allHosts {
		if !strings.Contains(content, "host: "+h) {
			t.Errorf("expected host %q in hosts section", h)
		}
	}

	for _, h := range allHosts {
		count := strings.Count(content, "node_id: "+h)
		if count != 1 {
			t.Errorf("host %q expected 1 vdisk_location in fail_domains, got %d", h, count)
		}
	}

	blob := blobStorageSection(content)
	if got := strings.Count(blob, "path: /dev/disk/by-partlabel/d1"); got != 9 {
		t.Errorf("expected 9 vdisk paths on first disk, got %d", got)
	}
	if strings.Contains(blob, "path: /dev/disk/by-partlabel/d2") || strings.Contains(blob, "path: /dev/disk/by-partlabel/d3") {
		t.Error("expected fail_domains to use only the first selected disk (d1)")
	}
}

// TestConfigGenerator_Mirror3DC_MoreThan9Hosts verifies that for mirror-3-dc
// with N=12 (>9) hosts, the first 3 hosts per DC each get 1 disk (9 hosts),
// and the 4th host per DC appears only in the hosts section.
func TestConfigGenerator_Mirror3DC_MoreThan9Hosts(t *testing.T) {
	inv := &models.InventoryData{
		Topology: &models.TopologyConfig{Type: "mirror-3-dc"},
		DC1:      []string{"host-a1", "host-a2", "host-a3", "host-a4"},
		DC2:      []string{"host-b1", "host-b2", "host-b3", "host-b4"},
		DC3:      []string{"host-d1", "host-d2", "host-d3", "host-d4"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/disk1", Label: "d1"},
				{Path: "/dev/disk2", Label: "d2"},
				{Path: "/dev/disk3", Label: "d3"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if rings := strings.Count(content, "fail_domains:"); rings != 3 {
		t.Errorf("expected 3 fail_domain groups, got %d", rings)
	}

	if vdc := strings.Count(content, "vdisk_locations:"); vdc != 9 {
		t.Errorf("expected 9 vdisk_locations (first 3 hosts per DC × 1 disk), got %d", vdc)
	}

	allHosts := []string{"host-a1", "host-a2", "host-a3", "host-a4", "host-b1", "host-b2", "host-b3", "host-b4", "host-d1", "host-d2", "host-d3", "host-d4"}
	for _, h := range allHosts {
		if !strings.Contains(content, "host: "+h) {
			t.Errorf("expected host %q in hosts section", h)
		}
	}

	for _, h := range []string{"host-a1", "host-a2", "host-a3", "host-b1", "host-b2", "host-b3", "host-d1", "host-d2", "host-d3"} {
		count := strings.Count(content, "node_id: "+h)
		if count != 1 {
			t.Errorf("host %q expected 1 vdisk_location in fail_domains, got %d", h, count)
		}
	}

	for _, h := range []string{"host-a4", "host-b4", "host-d4"} {
		if strings.Contains(content, "node_id: "+h) {
			t.Errorf("host %q should NOT appear in fail_domains (extra host, only in hosts)", h)
		}
	}

	blob := blobStorageSection(content)
	if got := strings.Count(blob, "path: /dev/disk/by-partlabel/d1"); got != 9 {
		t.Errorf("expected 9 vdisk paths on first disk, got %d", got)
	}
	if strings.Contains(blob, "path: /dev/disk/by-partlabel/d2") || strings.Contains(blob, "path: /dev/disk/by-partlabel/d3") {
		t.Error("expected fail_domains to use only the first selected disk (d1)")
	}
}

// TestConfigGenerator_Mirror3DC_9Hosts_SingleDisk verifies that mirror-3-dc
// with N=9 hosts and a single selected disk still produces 9 vdisk_locations,
// all pointing at the first disk's by-partlabel path.
func TestConfigGenerator_Mirror3DC_9Hosts_SingleDisk(t *testing.T) {
	inv := &models.InventoryData{
		Topology: &models.TopologyConfig{Type: "mirror-3-dc"},
		DC1:      []string{"host-a1", "host-a2", "host-a3"},
		DC2:      []string{"host-b1", "host-b2", "host-b3"},
		DC3:      []string{"host-d1", "host-d2", "host-d3"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/disk1", Label: "ydb_disk_1"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if rings := strings.Count(content, "fail_domains:"); rings != 3 {
		t.Errorf("expected 3 fail_domain groups, got %d", rings)
	}

	if vdc := strings.Count(content, "vdisk_locations:"); vdc != 9 {
		t.Errorf("expected 9 vdisk_locations (3 hosts per DC x 1 disk), got %d", vdc)
	}

	blob := blobStorageSection(content)
	if got := strings.Count(blob, "path: /dev/disk/by-partlabel/ydb_disk_1"); got != 9 {
		t.Errorf("expected 9 vdisk paths on first disk, got %d", got)
	}
}

// TestConfigGenerator_NonMirrorTopology verifies the block-4-2 fail-domain
// layout produced by buildBlock42FailDomains.
//
// block-4-2 emits a single fail_domains group: one vdisk per host in body
// order, each pinned to the FIRST selected disk. With 6 hosts and 2 selected
// disks this yields 6 vdisk_locations, all on the first disk's path.
func TestConfigGenerator_NonMirrorTopology(t *testing.T) {
	inv := &models.InventoryData{
		Topology: &models.TopologyConfig{Type: "block-4-2"},
		DC1:      []string{"host-a1", "host-a2"},
		DC2:      []string{"host-b1", "host-b2"},
		DC3:      []string{"host-d1", "host-d2"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/disk1", Label: "d1"},
				{Path: "/dev/disk2", Label: "d2"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	vdiskCount := strings.Count(content, "vdisk_locations:")
	if vdiskCount != 6 {
		t.Errorf("expected 6 vdisk_locations (block-4-2: one per host), got %d", vdiskCount)
	}

	if !strings.Contains(content, "static_erasure: block-4-2") {
		t.Error("expected block-4-2 erasure in config")
	}

	// Count inside blob_storage_config only — host_configs also lists every
	// selected disk and would inflate the numbers.
	blobSection := content
	if idx := strings.Index(content, "blob_storage_config:"); idx >= 0 {
		blobSection = content[idx:]
	}
	if n := strings.Count(blobSection, "path: /dev/disk/by-partlabel/d1"); n != 6 {
		t.Errorf("expected all 6 vdisks on the first selected disk, got %d", n)
	}
	if n := strings.Count(blobSection, "path: /dev/disk/by-partlabel/d2"); n != 0 {
		t.Errorf("expected no vdisks on the second selected disk, got %d", n)
	}
	if strings.Contains(content, "data_center:") {
		t.Error("block-4-2 must not emit data_center in walle_location")
	}
}

// TestConfigGenerator_Block42_8Hosts_SingleDisk verifies the canonical
// block-4-2 layout: 8 hosts (3/3/2 across DCs) and 1 selected disk produce
// exactly 8 vdisk_locations — one per host, all on the first disk — and a
// state_storage ring of bodies 1..8.
func TestConfigGenerator_Block42_8Hosts_SingleDisk(t *testing.T) {
	inv := &models.InventoryData{
		Topology: &models.TopologyConfig{Type: "block-4-2"},
		DC1:      []string{"static-node-1", "static-node-2", "static-node-3"},
		DC2:      []string{"static-node-4", "static-node-5", "static-node-6"},
		DC3:      []string{"static-node-7", "static-node-8"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if vdiskCount := strings.Count(content, "vdisk_locations:"); vdiskCount != 8 {
		t.Errorf("expected 8 vdisk_locations, got %d", vdiskCount)
	}
	// Count inside blob_storage_config only — host_configs also lists every
	// selected disk and would inflate the numbers.
	blobSection := content
	if idx := strings.Index(content, "blob_storage_config:"); idx >= 0 {
		blobSection = content[idx:]
	}
	if n := strings.Count(blobSection, "path: /dev/disk/by-partlabel/ydb_disk_1"); n != 8 {
		t.Errorf("expected all 8 vdisks on ydb_disk_1, got %d", n)
	}
	for i := 1; i <= 8; i++ {
		if !strings.Contains(content, "node_id: static-node-"+strconv.Itoa(i)+"\n") {
			t.Errorf("expected static-node-%d in fail_domains", i)
		}
	}
	if !strings.Contains(content, "node: [1, 2, 3, 4, 5, 6, 7, 8]") {
		t.Error("expected state_storage ring with bodies 1..8")
	}
	if !strings.Contains(content, "nto_select: 8") {
		t.Error("expected nto_select: 8")
	}
	if strings.Contains(content, "data_center:") {
		t.Error("block-4-2 must not emit data_center in walle_location")
	}
}

// TestConfigGenerator_Block42_MoreThan8Hosts verifies the 8-vdisk cap: with
// 10 hosts only the first 8 (body order) contribute vdisks; hosts 9-10 still
// appear in the hosts section.
func TestConfigGenerator_Block42_MoreThan8Hosts(t *testing.T) {
	inv := &models.InventoryData{
		Topology: &models.TopologyConfig{Type: "block-4-2"},
		DC1:      []string{"node-1", "node-2", "node-3", "node-4"},
		DC2:      []string{"node-5", "node-6", "node-7"},
		DC3:      []string{"node-8", "node-9", "node-10"},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}

	tmpDir := t.TempDir()
	gen := config.NewGenerator(filepath.Join(tmpDir, "config.yaml"))
	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if vdiskCount := strings.Count(content, "vdisk_locations:"); vdiskCount != 8 {
		t.Errorf("expected 8 vdisk_locations (cap), got %d", vdiskCount)
	}
	if strings.Contains(content, "node_id: node-9") {
		t.Error("node-9 is beyond the 8-vdisk cap and must not appear in fail_domains")
	}
	if strings.Contains(content, "node_id: node-10") {
		t.Error("node-10 is beyond the 8-vdisk cap and must not appear in fail_domains")
	}
	if !strings.Contains(content, "host: node-9") || !strings.Contains(content, "host: node-10") {
		t.Error("hosts beyond the cap must still appear in the hosts section")
	}
	if !strings.Contains(content, "nto_select: 8") {
		t.Error("expected nto_select: 8")
	}
}

// TestGenerator_Render_MatchesGenerateFileContent verifies that Render returns
// the same content that Generate writes to disk — ensuring the two paths are
// identical and the comparison in ConfigStatus is meaningful.
func TestGenerator_Render_MatchesGenerateFileContent(t *testing.T) {
	inv := &models.InventoryData{
		DC1: []string{"render-a.ydb.test"},
		DC2: []string{"render-b.ydb.test"},
		DC3: []string{"render-c.ydb.test"},
		YDBConfig: &models.YDBConfig{
			Domain:                      "Root",
			CoresStatic:                 2,
			EnforceUserTokenRequirement: false,
			RequestClientCertificate:    true,
		},
		Disks: &models.DisksConfig{
			SelectedDisks: []models.DiskEntry{
				{Path: "/dev/vdb", Label: "ydb_disk_1"},
			},
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	gen := config.NewGenerator(configPath)

	if err := gen.Generate(inv); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	diskContent, err := gen.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	rendered, err := gen.Render(inv)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if rendered != diskContent {
		t.Errorf("Render output differs from written file:\nRender: %q\nFile:   %q", rendered[:min(80, len(rendered))], diskContent[:min(80, len(diskContent))])
	}
}

// min is a local helper for Go < 1.21.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// blobStorageSection returns the blob_storage_config portion of the generated
// config so path assertions target vdisk_locations rather than host_configs.
func blobStorageSection(content string) string {
	start := strings.Index(content, "blob_storage_config:")
	if start == -1 {
		return ""
	}
	end := strings.Index(content[start:], "channel_profile_config:")
	if end == -1 {
		return content[start:]
	}
	return content[start : start+end]
}
