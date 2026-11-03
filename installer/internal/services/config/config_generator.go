// Package config provides services for generating config.yaml from inventory.
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"ydb-installer/internal/models"
)

// Generator generates config.yaml from inventory data.
type Generator struct {
	templatePath string
	outputPath   string
}

func NewGenerator(outputPath string) *Generator {
	return &Generator{
		templatePath: "",
		outputPath:   outputPath,
	}
}

// ConfigTemplateData holds the data needed to render the config.yaml template.
type ConfigTemplateData struct {
	Topology          string
	WithDataCenter    bool // false for block-4-2, which has no DC concept
	Hosts             []HostData
	FailDomains       []FailDomainGroup // Per-DC fail domain groups for blob_storage_config
	HostConfigDisks   []DiskData        // Single disk config entry for host_config_id: 1
	StateStorageNodes []int             // Body numbers of hosts used in fail_domains (for state_storage ring)
	Domain            string
	PoolKind          string
	EnforceUserToken  bool
	RequestClientCert bool
	CoresStatic       int
}

// HostData holds per-host configuration data.
type HostData struct {
	Hostname     string
	HostConfigID int
	Body         int
	DataCenter   string
	Rack         string
	Disks        []DiskData
}

// DiskData holds per-disk configuration data.
type DiskData struct {
	Path string
	Type string // SSD, HDD, NVME
}

// FailDomainGroup holds vdisk locations for a single fail domain (one per DC).
type FailDomainGroup struct {
	Zone   string
	VDisks []VDiskLocation
}

// VDiskLocation represents a single vdisk location entry in fail_domains.
type VDiskLocation struct {
	NodeID        string
	PDiskCategory string
	Path          string
}

// dcLocationConfig maps each DC index to its location name used in config data_center.
// These replace the old zone-a/zone-b/zone-d naming.
var dcLocationConfig = []string{"dc-1", "dc-2", "dc-3"}

// Render generates config.yaml content from inventory data without writing to disk.
// Generate calls Render then writes. Use Render for in-memory comparison (config-status dry-run).
func (g *Generator) Render(inv *models.InventoryData) (string, error) {
	data := g.buildTemplateData(inv)
	content, err := g.renderTemplate(data)
	if err != nil {
		return "", fmt.Errorf("failed to render config template: %w", err)
	}
	return content, nil
}

// Generate creates config.yaml from inventory data.
func (g *Generator) Generate(inv *models.InventoryData) error {
	content, err := g.Render(inv)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(g.outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(g.outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config.yaml: %w", err)
	}

	return nil
}

// Backup creates a timestamped backup of the existing config.yaml.
// Returns the backup file path, or empty string if no file exists.
func (g *Generator) Backup() (string, error) {
	if _, err := os.Stat(g.outputPath); os.IsNotExist(err) {
		return "", nil
	}

	ts := time.Now().Format("20060102-150405")
	backupPath := g.outputPath + "." + ts + ".bak"

	input, err := os.ReadFile(g.outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read existing config for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, input, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup to %s: %w", backupPath, err)
	}

	return backupPath, nil
}

// Read returns the current content of config.yaml.
func (g *Generator) Read() (string, error) {
	content, err := os.ReadFile(g.outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read config.yaml: %w", err)
	}
	return string(content), nil
}

// buildTemplateData converts inventory data to template data.
func (g *Generator) buildTemplateData(inv *models.InventoryData) *ConfigTemplateData {
	data := &ConfigTemplateData{
		Topology:          "mirror-3-dc",
		Domain:            "Root",
		PoolKind:          "ssd",
		EnforceUserToken:  true,
		RequestClientCert: true,
	}

	if inv.Topology != nil {
		data.Topology = inv.Topology.Type
	}

	// block-4-2 has no DC concept — omit walle_location.data_center.
	data.WithDataCenter = data.Topology != "block-4-2"

	if inv.YDBConfig != nil {
		data.Domain = inv.YDBConfig.Domain
		data.EnforceUserToken = inv.YDBConfig.EnforceUserTokenRequirement
		data.RequestClientCert = inv.YDBConfig.RequestClientCertificate
		data.CoresStatic = inv.YDBConfig.CoresStatic
		if inv.YDBConfig.PoolKind != "" {
			data.PoolKind = inv.YDBConfig.PoolKind
		}
	}

	// TLS paths are hardcoded to /opt/ydb/certs — these are runtime paths on
	// the target hosts, NOT local paths from inventory. The playbooks handle
	// copying certificates from the local ydb_tls_dir to /opt/ydb/certs/ on
	// each target host.
	data.HostConfigDisks = g.buildHostConfigDisks(inv)

	data.Hosts = g.buildHostsData(inv)

	data.FailDomains = g.buildFailDomains(inv, data.Hosts, data.Topology)

	// Only hosts with vdisk_locations in blob_storage_config should appear in state_storage ring.
	data.StateStorageNodes = extractStateStorageNodes(data.FailDomains, data.Hosts)

	return data
}

// buildHostConfigDisks builds the single host_config disk entry using stable
// by-partlabel paths (e.g. /dev/disk/by-partlabel/ydb_disk_1) instead of
// unstable kernel device names (e.g. /dev/vdb).
func (g *Generator) buildHostConfigDisks(inv *models.InventoryData) []DiskData {
	if inv.Disks == nil || len(inv.Disks.SelectedDisks) == 0 {
		return nil
	}
	disks := make([]DiskData, len(inv.Disks.SelectedDisks))
	for i, d := range inv.Disks.SelectedDisks {
		path := d.Path
		if d.Label != "" {
			path = "/dev/disk/by-partlabel/" + d.Label
		}
		disks[i] = DiskData{
			Path: path,
			Type: "SSD",
		}
	}
	return disks
}

// buildHostsData creates host configuration from inventory, using DC-specific
// host lists (DC1, DC2, DC3) when available, falling back to round-robin
// distribution from the flat Hosts list for backward compatibility.
func (g *Generator) buildHostsData(inv *models.InventoryData) []HostData {
	var disks []DiskData
	if inv.Disks != nil && len(inv.Disks.SelectedDisks) > 0 {
		for _, d := range inv.Disks.SelectedDisks {
			path := d.Path
			if d.Label != "" {
				path = "/dev/disk/by-partlabel/" + d.Label
			}
			disks = append(disks, DiskData{
				Path: path,
				Type: "SSD",
			})
		}
	}

	dcHosts := [][]string{inv.DC1, inv.DC2, inv.DC3}
	hasDCLists := len(inv.DC1) > 0 || len(inv.DC2) > 0 || len(inv.DC3) > 0

	if hasDCLists {
		return g.buildHostsFromDCLists(dcHosts, disks)
	}

	return g.buildHostsRoundRobin(inv.Hosts, disks)
}

// buildHostsFromDCLists assigns hosts from DC-specific lists to their correct data centers.
func (g *Generator) buildHostsFromDCLists(dcHosts [][]string, disks []DiskData) []HostData {
	var result []HostData
	bodyCounter := 1

	for dcIdx := 0; dcIdx < 3; dcIdx++ {
		dataCenter := dcLocationConfig[dcIdx]
		hosts := dcHosts[dcIdx]
		for _, hostname := range hosts {
			if hostname == "" {
				continue
			}
			// Rack follows same sequential numbering as body (see good-config.yaml).
			result = append(result, HostData{
				Hostname:     hostname,
				HostConfigID: 1,
				Body:         bodyCounter,
				DataCenter:   dataCenter,
				Rack:         strconv.Itoa(bodyCounter),
				Disks:        disks,
			})
			bodyCounter++
		}
	}

	return result
}

// buildHostsRoundRobin distributes flat hosts across 3 DCs in round-robin order.
// Used as fallback when DC-specific lists are not available.
func (g *Generator) buildHostsRoundRobin(hosts []string, disks []DiskData) []HostData {
	if len(hosts) == 0 {
		return nil
	}

	result := make([]HostData, len(hosts))
	for i, hostname := range hosts {
		dcIndex := i % 3
		dataCenter := dcLocationConfig[dcIndex]

		result[i] = HostData{
			Hostname:     hostname,
			HostConfigID: 1,
			Body:         i + 1,
			DataCenter:   dataCenter,
			Rack:         strconv.Itoa(i + 1),
			Disks:        disks,
		}
	}

	return result
}

// buildFailDomains builds fail domain groups for blob_storage_config.
//
// For mirror-3-dc topology, exactly 3 fail domains (one per DC) are created
// with exactly 3 vdisk_locations each (9 disks total). The distribution
// depends on the total number of hosts:
//
//   - 3 <= N < 9 hosts: first host in each DC gets all 3 selected disks.
//     Remaining hosts (>3) appear only in the "hosts" section.
//   - N >= 9 hosts: first 3 hosts in each DC get 1 vdisk each, all on the
//     first selected disk. Remaining hosts (>9) appear only in the "hosts" section.
//
// For block-4-2, a single fail domain holds up to 8 vdisk_locations (one per
// host, each pinned to the first selected disk); see buildBlock42FailDomains.
//
// For other topologies, all hosts and all disks are included (legacy behavior).
func (g *Generator) buildFailDomains(inv *models.InventoryData, hosts []HostData, topology string) []FailDomainGroup {
	if len(hosts) == 0 {
		return nil
	}

	dcGroups := make(map[string][]HostData)
	for _, h := range hosts {
		dcGroups[h.DataCenter] = append(dcGroups[h.DataCenter], h)
	}

	zoneOrder := dcLocationConfig
	var groups []FailDomainGroup

	if topology == "mirror-3-dc" {
		groups = g.buildMirror3DCFailDomains(dcGroups, zoneOrder, hosts)
	} else if topology == "block-4-2" {
		groups = g.buildBlock42FailDomains(hosts)
	} else {
		for _, zone := range zoneOrder {
			groupHosts, ok := dcGroups[zone]
			if !ok || len(groupHosts) == 0 {
				continue
			}

			var vdisks []VDiskLocation
			for _, host := range groupHosts {
				for _, disk := range host.Disks {
					cat := pdiskCategory(disk.Type)
					vdisks = append(vdisks, VDiskLocation{
						NodeID:        host.Hostname,
						PDiskCategory: cat,
						Path:          disk.Path,
					})
				}
			}

			if len(vdisks) > 0 {
				groups = append(groups, FailDomainGroup{
					Zone:   zone,
					VDisks: vdisks,
				})
			}
		}
	}

	return groups
}

// buildMirror3DCFailDomains builds exactly 3 fail domains (one per DC) with
// exactly 3 vdisk_locations each (9 total) for mirror-3-dc topology.
//
// Distribution rules:
//   - 3 <= N < 9 hosts: first host per DC gets all 3 disks
//   - N >= 9 hosts: first 3 hosts per DC get 1 vdisk each, all on the first disk
func (g *Generator) buildMirror3DCFailDomains(dcGroups map[string][]HostData, zoneOrder []string, hosts []HostData) []FailDomainGroup {
	totalHosts := len(hosts)
	var groups []FailDomainGroup

	for _, zone := range zoneOrder {
		groupHosts, ok := dcGroups[zone]
		if !ok || len(groupHosts) == 0 {
			continue
		}

		var vdisks []VDiskLocation

		if totalHosts < 9 {
			firstHost := groupHosts[0]
			for _, disk := range firstHost.Disks {
				vdisks = append(vdisks, VDiskLocation{
					NodeID:        firstHost.Hostname,
					PDiskCategory: pdiskCategory(disk.Type),
					Path:          disk.Path,
				})
			}
		} else {
			for i := 0; i < 3 && i < len(groupHosts); i++ {
				if len(groupHosts[i].Disks) > 0 {
					disk := groupHosts[i].Disks[0]
					vdisks = append(vdisks, VDiskLocation{
						NodeID:        groupHosts[i].Hostname,
						PDiskCategory: pdiskCategory(disk.Type),
						Path:          disk.Path,
					})
				}
			}
		}

		if len(vdisks) > 0 {
			groups = append(groups, FailDomainGroup{
				Zone:   zone,
				VDisks: vdisks,
			})
		}
	}

	return groups
}

// buildBlock42FailDomains builds a single fail domain with up to 8
// vdisk_locations (one per host) for block-4-2 topology.
//
// Hosts are taken in body order (DC1 → DC2 → DC3); each contributing host is
// pinned to the FIRST selected disk — the same rule mirror-3-dc applies at
// N >= 9. Hosts beyond the 8-vdisk cap appear only in the "hosts" section.
func (g *Generator) buildBlock42FailDomains(hosts []HostData) []FailDomainGroup {
	var vdisks []VDiskLocation

	for _, host := range hosts {
		if len(vdisks) >= 8 {
			break
		}
		if len(host.Disks) == 0 {
			continue
		}
		disk := host.Disks[0]
		vdisks = append(vdisks, VDiskLocation{
			NodeID:        host.Hostname,
			PDiskCategory: pdiskCategory(disk.Type),
			Path:          disk.Path,
		})
	}

	if len(vdisks) == 0 {
		return nil
	}
	return []FailDomainGroup{{Zone: "", VDisks: vdisks}}
}

// extractStateStorageNodes extracts body numbers of hosts that appear in
// fail_domains. Only these hosts should be listed in the state_storage ring.
// For legacy topologies (non mirror-3-dc) all hosts have fail_domain entries,
// so all hosts are returned — preserving backward compatibility.
func extractStateStorageNodes(failDomains []FailDomainGroup, hosts []HostData) []int {
	if len(failDomains) == 0 {
		return nil
	}

	hostnames := make(map[string]bool)
	for _, fd := range failDomains {
		for _, vdisk := range fd.VDisks {
			hostnames[vdisk.NodeID] = true
		}
	}

	var bodies []int
	for _, h := range hosts {
		if hostnames[h.Hostname] {
			bodies = append(bodies, h.Body)
		}
	}

	return bodies
}

// pdiskCategory returns the PDisk category string from a disk type.
func pdiskCategory(diskType string) string {
	switch strings.ToUpper(diskType) {
	case "SSD":
		return "SSD"
	case "HDD", "ROT":
		return "ROT"
	case "NVME":
		return "NVME"
	default:
		return "SSD"
	}
}

// renderTemplate renders the config.yaml template with the given data.
func (g *Generator) renderTemplate(data *ConfigTemplateData) (string, error) {
	tmpl := `storage_config_generation: 0
static_erasure: {{ .Topology }}
host_configs:
- drive:
{{- range .HostConfigDisks }}
  - path: {{ .Path }}
    type: {{ .Type }}
{{- end }}
  host_config_id: 1
hosts:
{{- range $i, $host := .Hosts }}
- host: {{ $host.Hostname }}
  host_config_id: {{ $host.HostConfigID }}
  walle_location:
    body: {{ $host.Body }}
{{- if $.WithDataCenter }}
    data_center: '{{ $host.DataCenter }}'
{{- end }}
    rack: '{{ $host.Rack }}'
{{- end }}
domains_config:
  domain:
  - name: {{ .Domain }}
    storage_pool_types:
    - kind: {{ .PoolKind }}
      pool_config:
        box_id: 1
        erasure_species: {{ .Topology }}
        kind: {{ .PoolKind }}
        geometry:
          realm_level_begin: 10
          realm_level_end: 20
          domain_level_begin: 10
          domain_level_end: 256
        pdisk_filter:
        - property:
          - type: SSD
        vdisk_kind: Default
  state_storage:
  - ring:
      node: [{{ range $i, $body := .StateStorageNodes }}{{ if $i }}, {{ end }}{{ $body }}{{ end }}]
      nto_select: {{ len .StateStorageNodes }}
    ssid: 1
  security_config:
    enforce_user_token_requirement: {{ .EnforceUserToken }}
    monitoring_allowed_sids:
    - "root"
    - "ADMINS"
    - "DATABASE-ADMINS"
    administration_allowed_sids:
    - "root"
    - "ADMINS"
    - "DATABASE-ADMINS"
    viewer_allowed_sids:
    - "root"
    - "ADMINS"
    - "DATABASE-ADMINS"
    register_dynamic_node_allowed_sids:
    - databaseNodes@cert
    - root@builtin
blob_storage_config:
  service_set:
    groups:
    - erasure_species: {{ .Topology }}
      rings:
{{- range .FailDomains }}
      - fail_domains:
{{- range .VDisks }}
        - vdisk_locations:
          - node_id: {{ .NodeID }}
            pdisk_category: {{ .PDiskCategory }}
            path: {{ .Path }}
{{- end }}
{{- end }}
channel_profile_config:
  profile:
  - channel:
    - erasure_species: {{ .Topology }}
      pdisk_category: 1   # 0=ROT, 1=SSD, 2=NVME
      storage_pool_kind: {{ .PoolKind }}
    - erasure_species: {{ .Topology }}
      pdisk_category: 1
      storage_pool_kind: {{ .PoolKind }}
    - erasure_species: {{ .Topology }}
      pdisk_category: 1
      storage_pool_kind: {{ .PoolKind }}
    profile_id: 0
interconnect_config:
    start_tcp: true
    encryption_mode: OPTIONAL
    path_to_certificate_file: "/opt/ydb/certs/node.crt"
    path_to_private_key_file: "/opt/ydb/certs/node.key"
    path_to_ca_file: "/opt/ydb/certs/ca.crt"
grpc_config:
    cert: "/opt/ydb/certs/node.crt"
    key: "/opt/ydb/certs/node.key"
    ca: "/opt/ydb/certs/ca.crt"
    services_enabled:
    - legacy
    - discovery
auth_config:
  path_to_root_ca: /opt/ydb/certs/ca.crt
client_certificate_authorization:
  request_client_certificate: {{ .RequestClientCert }}
  client_certificate_definitions:
      - member_groups: ["databaseNodes@cert"]
        subject_terms:
        - short_name: "O"
          values: ["YDB"]
query_service_config:
  generic:
    connector:
      endpoint:
        host: localhost
        port: 19102
      use_ssl: false
    default_settings:
      - name: DateTimeFormat
        value: string
      - name: UsePredicatePushdown
        value: "true"
feature_flags:
  enable_external_data_sources: true
  enable_script_execution_operations: true
`

	t, err := template.New("config").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
