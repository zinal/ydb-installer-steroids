package unit_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/config"

	"gopkg.in/yaml.v3"
)

func TestEMInventoryWriter_Write_NoYdbdDynamic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts: []models.EMHost{
			{FQDN: "em-host.example.com", Location: "dc-1"},
		},
		ClusterName: "mycluster",
	}

	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	contentStr := string(content)

	if strings.Contains(contentStr, "ydbd_dynamic") {
		t.Errorf("ydbd_dynamic should NOT appear in 90-inventory.yaml:\n%s", contentStr)
	}

	if !strings.Contains(contentStr, "ydb_em") {
		t.Error("ydb_em should be present in 90-inventory.yaml")
	}

	if !strings.Contains(contentStr, "ydb_em_cluster_name: mycluster") {
		t.Error("ydb_em_cluster_name should be written")
	}
	if !strings.Contains(contentStr, "em-host.example.com") {
		t.Error("EM host FQDN should be written")
	}

	svc := config.NewEMInventoryService()
	parsed, err := svc.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse written file: %v", err)
	}

	if len(parsed.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(parsed.Hosts))
	}
	if parsed.Hosts[0].FQDN != "em-host.example.com" {
		t.Errorf("unexpected host FQDN: %s", parsed.Hosts[0].FQDN)
	}
	if parsed.Hosts[0].Location != "dc-1" {
		t.Errorf("unexpected host location: %s", parsed.Hosts[0].Location)
	}
	if parsed.ClusterName != "mycluster" {
		t.Errorf("unexpected cluster name: %s", parsed.ClusterName)
	}
}

func TestEMInventoryWriter_WriteWithPatch_NoYdbdDynamic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts: []models.EMHost{
			{FQDN: "em-host1.example.com"},
		},
		ClusterName: "test-cluster",
	}

	if err := writer.WriteWithPatch(path, data); err != nil {
		t.Fatalf("WriteWithPatch (no existing file) failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)

	if strings.Contains(contentStr, "ydbd_dynamic") {
		t.Errorf("ydbd_dynamic should NOT appear in 90-inventory.yaml:\n%s", contentStr)
	}

	if !strings.Contains(contentStr, "ydb_em") {
		t.Error("ydb_em should be present")
	}

	svc := config.NewEMInventoryService()
	parsed, err := svc.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}
	if len(parsed.Hosts) != 1 || parsed.Hosts[0].FQDN != "em-host1.example.com" {
		t.Errorf("unexpected hosts: %v", parsed.Hosts)
	}
}

// TestEMInventoryWriter_DefaultAZLocations verifies empty AZLocations writes
// the default dc-1/dc-2/dc-3 as a native YAML list (not a bracket string).
func TestEMInventoryWriter_DefaultAZLocations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts: []models.EMHost{{FQDN: "em.example.com"}},
	}
	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(content)

	if got := emAZFromFile(t, path); !reflect.DeepEqual(got, []string{"dc-1", "dc-2", "dc-3"}) {
		t.Errorf("ydb_em_az_locations = %v, want [dc-1 dc-2 dc-3] in:\n%s", got, s)
	}
	if strings.Contains(s, "[dc-1 dc-2 dc-3]") {
		t.Errorf("bracket-string AZ format must not be written:\n%s", s)
	}
	if strings.Contains(s, "db-dc-") {
		t.Errorf("legacy db-dc-* defaults must not appear:\n%s", s)
	}
	if !strings.Contains(s, "grpcs://em.example.com:2135") {
		t.Errorf("expected endpoint from first EM host:\n%s", s)
	}
}

// emVarsFromFile returns the ydb vars mapping from a written inventory file.
func emVarsFromFile(t *testing.T, path string) map[string]interface{} {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var doc struct {
		All struct {
			Children struct {
				YDB struct {
					Vars map[string]interface{} `yaml:"vars"`
				} `yaml:"ydb"`
			} `yaml:"children"`
		} `yaml:"all"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return doc.All.Children.YDB.Vars
}

// emAZFromFile extracts ydb_em_az_locations as a []string.
func emAZFromFile(t *testing.T, path string) []string {
	t.Helper()

	raw, ok := emVarsFromFile(t, path)["ydb_em_az_locations"]
	if !ok {
		t.Fatalf("ydb_em_az_locations missing in %s", path)
	}
	items, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("ydb_em_az_locations is not a YAML sequence: %#v", raw)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("AZ entry is not a string: %#v", item)
		}
		out = append(out, s)
	}
	return out
}

// emPresetsFromFile extracts ydb_em_cp_presets as a list of maps.
func emPresetsFromFile(t *testing.T, path string) []map[string]interface{} {
	t.Helper()

	raw, ok := emVarsFromFile(t, path)["ydb_em_cp_presets"]
	if !ok {
		t.Fatalf("ydb_em_cp_presets missing in %s", path)
	}
	items, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("ydb_em_cp_presets is not a sequence: %#v", raw)
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("preset entry is not a mapping: %#v", item)
		}
		out = append(out, m)
	}
	return out
}

// TestEMInventoryWriter_AZLocationsCustomList verifies an explicit AZ list is
// written verbatim as a YAML sequence and survives a re-parse.
func TestEMInventoryWriter_AZLocationsCustomList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts:       []models.EMHost{{FQDN: "em.example.com"}},
		AZLocations: []string{"dc-1", "dc-3"},
	}
	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if got := emAZFromFile(t, path); !reflect.DeepEqual(got, []string{"dc-1", "dc-3"}) {
		t.Errorf("AZ locations = %v, want [dc-1 dc-3]", got)
	}

	parsed, err := config.NewEMInventoryService().Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !reflect.DeepEqual(parsed.AZLocations, []string{"dc-1", "dc-3"}) {
		t.Errorf("parsed AZLocations = %v, want [dc-1 dc-3]", parsed.AZLocations)
	}
}

// TestEMInventoryWriter_PatchAZLocations verifies the targeted AZ patch writes
// a YAML list and leaves other keys intact.
func TestEMInventoryWriter_PatchAZLocations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	existing := `all:
  children:
    ydb:
      vars:
        ydb_em_cluster_name: keep-me
        ydb_em_az_locations: '[dc-1 dc-2 dc-3]'
`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	writer := config.NewEMInventoryWriter()
	if err := writer.PatchAZLocations(path, []string{"dc-1", "dc-2"}); err != nil {
		t.Fatalf("PatchAZLocations failed: %v", err)
	}

	if got := emAZFromFile(t, path); !reflect.DeepEqual(got, []string{"dc-1", "dc-2"}) {
		t.Errorf("AZ locations = %v, want [dc-1 dc-2]", got)
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "ydb_em_cluster_name: keep-me") {
		t.Errorf("unrelated key lost:\n%s", content)
	}
}

// TestEMInventoryWriter_PatchAZLocations_EmptyUsesDefaults verifies an empty
// list falls back to the defaults rather than writing an empty sequence.
func TestEMInventoryWriter_PatchAZLocations_EmptyUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	if err := os.WriteFile(path, []byte("all:\n  children:\n    ydb:\n      vars: {}\n"), 0644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	writer := config.NewEMInventoryWriter()
	if err := writer.PatchAZLocations(path, nil); err != nil {
		t.Fatalf("PatchAZLocations failed: %v", err)
	}

	if got := emAZFromFile(t, path); !reflect.DeepEqual(got, []string{"dc-1", "dc-2", "dc-3"}) {
		t.Errorf("AZ locations = %v, want defaults", got)
	}
}

// TestEMInventoryWriter_PatchAZLocations_NoFileIsNoop verifies a missing file
// is silently ignored (nothing is created).
func TestEMInventoryWriter_PatchAZLocations_NoFileIsNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	writer := config.NewEMInventoryWriter()
	if err := writer.PatchAZLocations(path, []string{"dc-1"}); err != nil {
		t.Fatalf("PatchAZLocations on missing file should be a no-op, got: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should not have been created")
	}
}

// TestEMInventoryWriter_DefaultCPPresets_FreshWrite verifies a brand-new
// 90-inventory.yaml carries the three default presets with id/cpu/memory.
func TestEMInventoryWriter_DefaultCPPresets_FreshWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{Hosts: []models.EMHost{{FQDN: "em.example.com"}}}
	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	presets := emPresetsFromFile(t, path)
	if len(presets) != 3 {
		t.Fatalf("expected 3 default presets, got %d: %v", len(presets), presets)
	}

	want := []struct{ id, cpu, memory string }{
		{"m1.medium", "1", "2"},
		{"m1.tiny", "1", "1"},
		{"m1.large", "2", "8"},
	}
	for i, w := range want {
		got := presets[i]
		if got["id"] != w.id {
			t.Errorf("preset[%d].id = %v, want %s", i, got["id"], w.id)
		}
		// cpu/memory must stay strings, not be re-read as ints.
		if got["cpu"] != w.cpu {
			t.Errorf("preset[%d].cpu = %#v, want %q", i, got["cpu"], w.cpu)
		}
		if got["memory"] != w.memory {
			t.Errorf("preset[%d].memory = %#v, want %q", i, got["memory"], w.memory)
		}
	}
}

// TestEMInventoryWriter_DefaultCPPresets_FlowStyle verifies presets render as
// compact flow-style mappings.
func TestEMInventoryWriter_DefaultCPPresets_FlowStyle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	if err := writer.Write(path, &models.EMConfig{}); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(content)

	for _, want := range []string{
		`{id: "m1.medium", cpu: "1", memory: "2"}`,
		`{id: "m1.tiny", cpu: "1", memory: "1"}`,
		`{id: "m1.large", cpu: "2", memory: "8"}`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected flow-style preset %s in:\n%s", want, s)
		}
	}
}

// TestEMInventoryWriter_DefaultCPPresets_PatchInjectsWhenMissing verifies
// patching a file that lacks the key adds the defaults.
func TestEMInventoryWriter_DefaultCPPresets_PatchInjectsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	existing := `all:
  children:
    ydb:
      vars:
        ydb_em_cluster_name: existing
      children:
        ydb_em:
          hosts:
            em.example.com:
`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{Hosts: []models.EMHost{{FQDN: "em.example.com"}}}
	if err := writer.WriteWithPatch(path, data); err != nil {
		t.Fatalf("WriteWithPatch failed: %v", err)
	}

	presets := emPresetsFromFile(t, path)
	if len(presets) != 3 {
		t.Fatalf("expected defaults injected, got %d presets", len(presets))
	}
	if presets[0]["id"] != "m1.medium" {
		t.Errorf("unexpected first preset: %v", presets[0])
	}
}

// TestEMInventoryWriter_DefaultCPPresets_PatchPreservesUserPresets is the key
// guarantee of injecting presets via EmAlwaysPresentKeys instead of managed
// vars: a user's customised presets must survive a write.
func TestEMInventoryWriter_DefaultCPPresets_PatchPreservesUserPresets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	existing := `all:
  children:
    ydb:
      vars:
        ydb_em_cp_presets:
          - {id: "custom.huge", cpu: "64", memory: "256"}
      children:
        ydb_em:
          hosts:
            em.example.com:
`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts:       []models.EMHost{{FQDN: "em.example.com"}},
		ClusterName: "changed",
	}
	if err := writer.WriteWithPatch(path, data); err != nil {
		t.Fatalf("WriteWithPatch failed: %v", err)
	}

	presets := emPresetsFromFile(t, path)
	if len(presets) != 1 {
		t.Fatalf("user presets must not be replaced, got %d: %v", len(presets), presets)
	}
	if presets[0]["id"] != "custom.huge" {
		t.Errorf("user preset clobbered: %v", presets[0])
	}

	// Sanity: the managed change still landed.
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "ydb_em_cluster_name: changed") {
		t.Errorf("managed var was not patched:\n%s", content)
	}
}

// TestEMInventoryWriter_MultipleHosts verifies multiple EM hosts
// are written correctly without ydbd_dynamic.
func TestEMInventoryWriter_MultipleHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts: []models.EMHost{
			{FQDN: "em1.example.com", Location: "dc-1"},
			{FQDN: "em2.example.com", Location: "dc-2"},
			{FQDN: "em3.example.com"},
		},
		ClusterName: "multi-cluster",
	}

	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)

	if strings.Contains(contentStr, "ydbd_dynamic") {
		t.Error("ydbd_dynamic should NOT appear")
	}

	for _, host := range data.Hosts {
		if !strings.Contains(contentStr, host.FQDN) {
			t.Errorf("host %q not found in output", host.FQDN)
		}
	}

	svc := config.NewEMInventoryService()
	parsed, err := svc.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(parsed.Hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(parsed.Hosts))
	}
}

// TestEMInventoryWriter_EmptyHosts verifies an empty EM config
// still produces a valid inventory without ydbd_dynamic.
func TestEMInventoryWriter_EmptyHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		ClusterName: "empty-cluster",
	}

	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)

	if strings.Contains(contentStr, "ydbd_dynamic") {
		t.Error("ydbd_dynamic should NOT appear even with empty hosts")
	}

	if !strings.Contains(contentStr, "ydb_em") {
		t.Error("ydb_em should still be present")
	}
}

// TestEMInventoryService_Parse_IgnoresLegacyYdbdDynamic verifies
// that the parser still works on legacy 90-inventory files that
// contain ydbd_dynamic (backward compatibility).
func TestEMInventoryService_Parse_IgnoresLegacyYdbdDynamic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	legacyContent := `all:
  children:
    ydb:
      vars:
        ydb_em_cluster_name: legacy-cluster
        ydb_em_frontend: em-host1
      children:
        ydb_em:
          hosts:
            em-host1:
              location: dc-1
            em-host2:
        ydbd_dynamic:
          hosts: {}
`

	if err := os.WriteFile(path, []byte(legacyContent), 0644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	svc := config.NewEMInventoryService()
	parsed, err := svc.Parse(path)
	if err != nil {
		t.Fatalf("failed to parse legacy file with ydbd_dynamic: %v", err)
	}

	if len(parsed.Hosts) != 2 {
		t.Fatalf("expected 2 EM hosts, got %d", len(parsed.Hosts))
	}
	// Build index by FQDN — YAML map key order is non-deterministic.
	hostByFQDN := make(map[string]models.EMHost, len(parsed.Hosts))
	for _, h := range parsed.Hosts {
		hostByFQDN[h.FQDN] = h
	}
	h1, ok1 := hostByFQDN["em-host1"]
	if !ok1 {
		t.Error("em-host1 not found in parsed hosts")
	} else if h1.Location != "dc-1" {
		t.Errorf("em-host1 location = %q, want dc-1", h1.Location)
	}
	if _, ok := hostByFQDN["em-host2"]; !ok {
		t.Error("em-host2 not found in parsed hosts")
	}
	if parsed.ClusterName != "legacy-cluster" {
		t.Errorf("unexpected cluster name: %s", parsed.ClusterName)
	}
}

func TestEMInventoryWriter_Write_ArchivePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts: []models.EMHost{
			{FQDN: "em.example.com"},
		},
		ArchivePath:   "/opt/dist/ydb-em.tar.gz",
		MetaUIArchive: "/opt/dist/ui.tar.gz",
	}

	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "ydb_em_archive: /opt/dist/ydb-em.tar.gz") {
		t.Errorf("ydb_em_archive missing:\n%s", s)
	}
	if !strings.Contains(s, "ydb_em_meta_ui_archive: /opt/dist/ui.tar.gz") {
		t.Errorf("ydb_em_meta_ui_archive missing:\n%s", s)
	}
	// XOR: archive mode must not emit binary keys or Jinja defaults.
	for _, key := range []string{
		"ydb_em_meta_ui_binary",
		"ydb_em_cp_binary",
		"ydb_em_cp_agent_binary",
		"ydb-em-gateway",
	} {
		if strings.Contains(s, key) {
			t.Errorf("archive-only write must not contain %q:\n%s", key, s)
		}
	}
	if strings.Contains(s, "ydb_em_archive: null") {
		t.Errorf("unexpected null archive:\n%s", s)
	}

	svc := config.NewEMInventoryService()
	parsed, err := svc.Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if parsed.ArchivePath != "/opt/dist/ydb-em.tar.gz" {
		t.Errorf("ArchivePath = %q", parsed.ArchivePath)
	}
	if parsed.MetaUIArchive != "/opt/dist/ui.tar.gz" {
		t.Errorf("MetaUIArchive = %q", parsed.MetaUIArchive)
	}
	if parsed.MetaUIBinary != "" || parsed.CPBinary != "" || parsed.CPAgentBinary != "" {
		t.Errorf("parsed binaries should be empty, got meta=%q cp=%q agent=%q",
			parsed.MetaUIBinary, parsed.CPBinary, parsed.CPAgentBinary)
	}
}

func TestEMInventoryWriter_Write_OmitsEmptyArchives(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts: []models.EMHost{{FQDN: "em.example.com"}},
	}
	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	s := string(content)
	for _, key := range []string{
		"ydb_em_archive",
		"ydb_em_meta_ui_archive",
		"ydb_em_meta_ui_binary",
		"ydb_em_cp_binary",
		"ydb_em_cp_agent_binary",
	} {
		if strings.Contains(s, key) {
			t.Errorf("empty dist payload must omit %q:\n%s", key, s)
		}
	}
}

func TestEMInventoryWriter_Write_NoJinjaBinaryDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")
	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts:       []models.EMHost{{FQDN: "em.example.com"}},
		ClusterName: "c",
	}
	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write: %v", err)
	}
	content, _ := os.ReadFile(path)
	s := string(content)
	if strings.Contains(s, "ansible_config_file") {
		t.Errorf("must not inject Jinja binary defaults:\n%s", s)
	}
}

func TestEMInventoryWriter_WriteWithPatch_ArchiveDeletesBinaries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")
	writer := config.NewEMInventoryWriter()

	binaries := &models.EMConfig{
		Hosts:         []models.EMHost{{FQDN: "em.example.com"}},
		MetaUIBinary:  "/opt/gateway",
		CPBinary:      "/opt/cp",
		CPAgentBinary: "/opt/agent",
	}
	if err := writer.Write(path, binaries); err != nil {
		t.Fatalf("Write binaries: %v", err)
	}

	archives := &models.EMConfig{
		Hosts:         []models.EMHost{{FQDN: "em.example.com"}},
		ArchivePath:   "/opt/em.tar.gz",
		MetaUIArchive: "/opt/ui.tar.gz",
	}
	if err := writer.WriteWithPatch(path, archives); err != nil {
		t.Fatalf("WriteWithPatch archives: %v", err)
	}

	content, _ := os.ReadFile(path)
	s := string(content)
	if !strings.Contains(s, "ydb_em_archive: /opt/em.tar.gz") {
		t.Errorf("archive missing:\n%s", s)
	}
	for _, key := range []string{"ydb_em_meta_ui_binary", "ydb_em_cp_binary", "ydb_em_cp_agent_binary"} {
		if strings.Contains(s, key) {
			t.Errorf("binary key %q should be deleted:\n%s", key, s)
		}
	}
}

func TestEMInventoryWriter_WriteWithPatch_DeletesArchivesWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	withArchives := &models.EMConfig{
		Hosts:         []models.EMHost{{FQDN: "em.example.com"}},
		ArchivePath:   "/opt/dist/ydb-em.tar.gz",
		MetaUIArchive: "/opt/dist/ui.tar.gz",
	}
	if err := writer.Write(path, withArchives); err != nil {
		t.Fatalf("initial Write failed: %v", err)
	}

	// Switch to binaries-only: clear archives, keep custom binary.
	binariesOnly := &models.EMConfig{
		Hosts:         []models.EMHost{{FQDN: "em.example.com"}},
		MetaUIBinary:  "/opt/dist/ydb-em-gateway",
		CPBinary:      "/opt/dist/ydb-em-cp",
		CPAgentBinary: "/opt/dist/ydb-em-agent",
	}
	if err := writer.WriteWithPatch(path, binariesOnly); err != nil {
		t.Fatalf("WriteWithPatch failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	s := string(content)
	if strings.Contains(s, "ydb_em_archive") {
		t.Errorf("ydb_em_archive should be deleted after patch:\n%s", s)
	}
	if strings.Contains(s, "ydb_em_meta_ui_archive") {
		t.Errorf("ydb_em_meta_ui_archive should be deleted after patch:\n%s", s)
	}
	if !strings.Contains(s, "ydb_em_cp_binary: /opt/dist/ydb-em-cp") {
		t.Errorf("cp binary should remain:\n%s", s)
	}

	svc := config.NewEMInventoryService()
	parsed, err := svc.Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if parsed.ArchivePath != "" || parsed.MetaUIArchive != "" {
		t.Errorf("parsed archives should be empty, got archive=%q meta=%q", parsed.ArchivePath, parsed.MetaUIArchive)
	}
	if parsed.CPBinary != "/opt/dist/ydb-em-cp" {
		t.Errorf("CPBinary = %q", parsed.CPBinary)
	}
}

// TestEMInventoryWriter_TLSCertPath verifies ydb_em_tls_dir write + re-parse.
func TestEMInventoryWriter_TLSCertPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts:         []models.EMHost{{FQDN: "em.example.com"}},
		EmTLSCertPath: "TLS/CA/certs/certs-per-host",
	}
	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "ydb_em_tls_dir: TLS/CA/certs/certs-per-host") {
		t.Errorf("expected ydb_em_tls_dir in output:\n%s", s)
	}

	svc := config.NewEMInventoryService()
	parsed, err := svc.Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if parsed.EmTLSCertPath != "TLS/CA/certs/certs-per-host" {
		t.Errorf("EmTLSCertPath = %q, want TLS/CA/certs/certs-per-host", parsed.EmTLSCertPath)
	}
}

// TestEMInventoryWriter_TLSCertPath_EmptyOmitsKey verifies empty path does not write the key.
func TestEMInventoryWriter_TLSCertPath_EmptyOmitsKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "90-inventory.yaml")

	writer := config.NewEMInventoryWriter()
	data := &models.EMConfig{
		Hosts: []models.EMHost{{FQDN: "em.example.com"}},
	}
	if err := writer.Write(path, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(content), "ydb_em_tls_dir") {
		t.Errorf("ydb_em_tls_dir must be omitted when empty:\n%s", content)
	}
}
