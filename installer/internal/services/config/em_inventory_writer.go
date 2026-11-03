package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	"ydb-installer/internal/models"

	"gopkg.in/yaml.v3"
)

// azLocationNames maps a DC index (0-based) to its availability-zone name.
var azLocationNames = []string{"dc-1", "dc-2", "dc-3"}

// DefaultAZLocations returns the default dc-1/dc-2/dc-3 list.
func DefaultAZLocations() []string {
	return []string{"dc-1", "dc-2", "dc-3"}
}

// NormalizeAZLocations cleans a list of availability-zone names for writing to
// ydb_em_az_locations: entries are trimmed, empties dropped, and an empty
// result falls back to the defaults.
//
// The value is written as a native YAML list:
//
//	ydb_em_az_locations:
//	  - dc-1
//	  - dc-2
//	  - dc-3
func NormalizeAZLocations(locs []string) []string {
	out := make([]string, 0, len(locs))
	for _, loc := range locs {
		if trimmed := strings.TrimSpace(loc); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return DefaultAZLocations()
	}
	return out
}

// ParseAZLocations parses the ydb_em_az_locations YAML value, which is a
// sequence of location names.
//
// Returns nil for empty or non-sequence input. The bracket-string form
// ("[dc-1 dc-2 dc-3]") previously written by this installer is NOT supported:
// such a value yields nil, after which the writer re-emits the derived (or
// default) list as a proper YAML sequence.
func ParseAZLocations(raw interface{}) []string {
	var items []string

	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
	case []string:
		items = v
	default:
		return nil
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DeriveAZFromDCs derives availability zone locations from inventory DC lists.
// Each non-empty DC contributes its corresponding zone (dc-1, dc-2, dc-3).
// Falls back to defaults if all DCs are empty.
func DeriveAZFromDCs(dc1, dc2, dc3 []string) []string {
	var out []string
	for i, hosts := range [][]string{dc1, dc2, dc3} {
		for _, h := range hosts {
			if h != "" {
				out = append(out, azLocationNames[i])
				break
			}
		}
	}
	if len(out) == 0 {
		return DefaultAZLocations()
	}
	return out
}

// cpPreset is the on-disk shape of a single ydb_em_cp_presets entry.
// Values are strings (cpu: "1", memory: "2") to match the YDB EM ansible role.
type cpPreset struct {
	ID     string
	CPU    string
	Memory string
}

// defaultCPPresets is the canonical set of Control Plane presets written into
// 90-inventory.yaml when the file does not already define them.
var defaultCPPresets = []cpPreset{
	{ID: "m1.medium", CPU: "1", Memory: "2"},
	{ID: "m1.tiny", CPU: "1", Memory: "1"},
	{ID: "m1.large", CPU: "2", Memory: "8"},
}

// DefaultCPPresets returns the default ydb_em_cp_presets value as a YAML node.
//
// A *yaml.Node is returned (rather than a plain Go slice) so each preset renders
// in flow style — {id: "m1.medium", cpu: "1", memory: "2"} — keeping this
// user-facing inventory block compact. Scalars are force-quoted so cpu/memory
// remain strings rather than being re-read as integers.
//
// Presets are NOT part of the EMConfig model: they are injected via
// EmAlwaysPresentKeys, so they are written only when absent from the inventory
// and are never overwritten once a user customises them.
func DefaultCPPresets() *yaml.Node {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, p := range defaultCPPresets {
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}
		for _, kv := range []struct{ key, value string }{
			{"id", p.ID},
			{"cpu", p.CPU},
			{"memory", p.Memory},
		} {
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: kv.key},
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: kv.value,
					Style: yaml.DoubleQuotedStyle,
				},
			)
		}
		seq.Content = append(seq.Content, node)
	}
	return seq
}

// EMInventoryWriter handles serializing EMConfig back to YAML
// and writing it to the 90-inventory.yaml file.
type EMInventoryWriter struct {
	// mu serializes the read-modify-write cycle so concurrent PUTs cannot
	// interleave and drop each other's changes.
	mu sync.Mutex
}

func NewEMInventoryWriter() *EMInventoryWriter {
	return &EMInventoryWriter{}
}

// Write serializes EMConfig to YAML and writes it to the specified path.
// This creates a fresh 90-inventory.yaml from scratch.
func (w *EMInventoryWriter) Write(path string, data *models.EMConfig) error {
	if data == nil {
		return fmt.Errorf("EM config data is nil")
	}

	inv := w.buildAnsibleInventory(data)
	// buildVars uses nil for patch deletes — omit them from a fresh marshal.
	stripNilFromEMVars(inv)

	yamlBytes, err := yaml.Marshal(&inv)
	if err != nil {
		return fmt.Errorf("failed to marshal EM inventory to YAML: %w", err)
	}

	if err := ensureDir(path); err != nil {
		return err
	}

	if err := os.WriteFile(path, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write EM inventory file: %w", err)
	}

	return nil
}

// WriteWithPatch writes EM config data to the specified path, preserving
// the existing file's comments, unknown variables, and structure.
// Falls back to Write if the file doesn't exist.
func (w *EMInventoryWriter) WriteWithPatch(path string, data *models.EMConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if data == nil {
		return fmt.Errorf("EM config data is nil")
	}

	existingBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return w.Write(path, data)
		}
		return fmt.Errorf("failed to read existing EM inventory: %w", err)
	}

	// buildVars includes nil entries for XOR dist-key deletes (patch only).
	// Fresh Write() strips nils before marshal.
	managedVars := w.buildVars(data)

	patcher := NewInventoryPatcher()
	patchedBytes, err := patcher.Patch(existingBytes, managedVars, nil, EmAlwaysPresentKeys...)
	if err != nil {
		return fmt.Errorf("failed to patch EM inventory: %w", err)
	}

	patchedBytes, err = w.patchEMHosts(patchedBytes, data)
	if err != nil {
		return fmt.Errorf("failed to patch EM hosts: %w", err)
	}

	if !bytes.Equal(existingBytes, patchedBytes) {
		if err := ensureDir(path); err != nil {
			return err
		}
		if err := os.WriteFile(path, patchedBytes, 0644); err != nil {
			return fmt.Errorf("failed to write patched EM inventory: %w", err)
		}
	}

	return nil
}

// PatchAZLocations updates only the ydb_em_az_locations key in an existing
// 90-inventory.yaml file. Does nothing if the file does not exist.
// Uses the InventoryPatcher to preserve all other keys and comments.
//
// The value is written as a native YAML list.
func (w *EMInventoryWriter) PatchAZLocations(path string, locs []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	existingBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	patcher := NewInventoryPatcher()
	patchedBytes, err := patcher.Patch(existingBytes, map[string]interface{}{
		"ydb_em_az_locations": NormalizeAZLocations(locs),
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to patch EM inventory AZ: %w", err)
	}

	if !bytes.Equal(existingBytes, patchedBytes) {
		if err := ensureDir(path); err != nil {
			return err
		}
		if err := os.WriteFile(path, patchedBytes, 0644); err != nil {
			return fmt.Errorf("failed to write EM inventory AZ: %w", err)
		}
	}
	return nil
}

// buildAnsibleInventory converts EMConfig to the Ansible inventory YAML structure.
//
// Only ydb_em hosts and EM vars are written. ydbd_static / ydbd_dynamic are owned
// exclusively by 50-inventory.yaml and MUST NOT appear in 90-inventory.yaml.
func (w *EMInventoryWriter) buildAnsibleInventory(data *models.EMConfig) map[string]interface{} {
	emHosts := make(map[string]interface{})
	for _, host := range data.Hosts {
		hostVars := map[string]interface{}{}
		if host.Location != "" {
			hostVars["location"] = host.Location
		}
		emHosts[host.FQDN] = hostVars
	}

	// buildVars uses nil sentinels for patch-deletes; those are stripped by
	// stripNilFromEMVars below. Always-present defaults must be injected here
	// too, because a fresh marshal never goes through the patcher.
	vars := ApplyAlwaysPresentToVars(w.buildVars(data), EmAlwaysPresentKeys)

	inv := map[string]interface{}{
		"all": map[string]interface{}{
			"children": map[string]interface{}{
				"ydb": map[string]interface{}{
					"vars": vars,
					"children": map[string]interface{}{
						"ydb_em": map[string]interface{}{
							"hosts": emHosts,
						},
					},
				},
			},
		},
	}
	return inv
}

// buildVars creates the vars section from EMConfig.
// Only ydb_em_* vars are managed here — SSH, ydb_user, ydb_password are
// inherited from 50-inventory.yaml and not duplicated.
func (w *EMInventoryWriter) buildVars(data *models.EMConfig) map[string]interface{} {
	vars := make(map[string]interface{})

	if len(data.Hosts) > 0 {
		vars["ydb_em_frontend"] = data.Hosts[0].FQDN
	}

	if data.ClusterName != "" {
		vars["ydb_em_cluster_name"] = data.ClusterName
	}

	// Computed connection endpoints (Jinja2 templates referencing ydb_em_frontend).
	vars["ydb_em_cluster_balancer"] = "https://{{ ydb_em_frontend }}:8765"
	vars["ydb_em_cp_connection_endpoint"] = "grpcs://{{ ydb_em_frontend }}:8787"
	// DB endpoint: prefer explicit value (from UI check), else first host, else Jinja template.
	if data.DbConnectionEndpoint != "" {
		vars["ydb_em_db_connection_endpoint"] = data.DbConnectionEndpoint
	} else if len(data.Hosts) > 0 && data.Hosts[0].FQDN != "" {
		vars["ydb_em_db_connection_endpoint"] = "grpcs://" + data.Hosts[0].FQDN + ":2135"
	} else {
		vars["ydb_em_db_connection_endpoint"] = "grpcs://{{ ydb_em_frontend }}:2135"
	}

	if data.DbConnectionDBRoot != "" {
		vars["ydb_em_db_connection_db_root"] = data.DbConnectionDBRoot
	} else {
		vars["ydb_em_db_connection_db_root"] = "/Root"
	}
	if data.DbConnectionDB != "" {
		vars["ydb_em_db_connection_db"] = data.DbConnectionDB
	} else {
		vars["ydb_em_db_connection_db"] = "database"
	}

	// Distribution paths — no Jinja defaults. XOR on disk:
	//   archives only  → write archives, nil-delete binaries
	//   binaries only  → write binaries, nil-delete archives
	//   both (legacy)  → prefer archives, nil-delete binaries
	//   neither        → nil-delete all five (clear stale)
	// nil values are stripped in Write(); WriteWithPatch uses them as deletes.
	hasArchive := data.ArchivePath != "" || data.MetaUIArchive != ""
	hasBinary := data.MetaUIBinary != "" || data.CPBinary != "" || data.CPAgentBinary != ""

	switch {
	case hasArchive && !hasBinary, hasArchive && hasBinary:
		if data.ArchivePath != "" {
			vars["ydb_em_archive"] = data.ArchivePath
		} else {
			vars["ydb_em_archive"] = nil
		}
		if data.MetaUIArchive != "" {
			vars["ydb_em_meta_ui_archive"] = data.MetaUIArchive
		} else {
			vars["ydb_em_meta_ui_archive"] = nil
		}
		vars["ydb_em_meta_ui_binary"] = nil
		vars["ydb_em_cp_binary"] = nil
		vars["ydb_em_cp_agent_binary"] = nil
	case hasBinary:
		vars["ydb_em_archive"] = nil
		vars["ydb_em_meta_ui_archive"] = nil
		if data.MetaUIBinary != "" {
			vars["ydb_em_meta_ui_binary"] = data.MetaUIBinary
		} else {
			vars["ydb_em_meta_ui_binary"] = nil
		}
		if data.CPBinary != "" {
			vars["ydb_em_cp_binary"] = data.CPBinary
		} else {
			vars["ydb_em_cp_binary"] = nil
		}
		if data.CPAgentBinary != "" {
			vars["ydb_em_cp_agent_binary"] = data.CPAgentBinary
		} else {
			vars["ydb_em_cp_agent_binary"] = nil
		}
	default:
		vars["ydb_em_archive"] = nil
		vars["ydb_em_meta_ui_archive"] = nil
		vars["ydb_em_meta_ui_binary"] = nil
		vars["ydb_em_cp_binary"] = nil
		vars["ydb_em_cp_agent_binary"] = nil
	}

	if data.AgentCPU > 0 {
		vars["ydb_em_agent_cpu"] = data.AgentCPU
	} else {
		vars["ydb_em_agent_cpu"] = 1
	}
	if data.AgentMemory > 0 {
		vars["ydb_em_agent_memory"] = data.AgentMemory
	} else {
		vars["ydb_em_agent_memory"] = 2
	}

	vars["ydb_em_az_locations"] = NormalizeAZLocations(data.AZLocations)

	// ydb_em_cp_presets is intentionally NOT written here. It is a file-level
	// default injected via EmAlwaysPresentKeys, so user customisations in
	// 90-inventory.yaml are never overwritten.

	vars["ydb_use_dynamic_config"] = data.UseDynamicConfig

	// EM TLS directory — always write (value or nil) to match YDB pattern.
	// When empty, nil-delete ensures stale key is removed from inventory.
	if data.EmTLSCertPath != "" {
		vars["ydb_em_tls_dir"] = data.EmTLSCertPath
	} else {
		vars["ydb_em_tls_dir"] = nil
	}

	return vars
}

// patchEMHosts updates the ydb_em hosts group in the patched YAML.
func (w *EMInventoryWriter) patchEMHosts(rawYAML []byte, data *models.EMConfig) ([]byte, error) {
	if len(data.Hosts) == 0 {
		return rawYAML, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return rawYAML, err
	}

	rootMapping := getRootMapping(&doc)
	if rootMapping == nil {
		return rawYAML, nil
	}

	allMapping := findChildMappingNode(rootMapping, "all")
	if allMapping == nil {
		return rawYAML, nil
	}

	childrenMapping := findChildMappingNode(allMapping, "children")
	if childrenMapping == nil {
		return rawYAML, nil
	}

	ydbMapping := findChildMappingNode(childrenMapping, "ydb")
	if ydbMapping == nil {
		return rawYAML, nil
	}

	ydbChildren := findChildMappingNode(ydbMapping, "children")
	if ydbChildren == nil {
		return rawYAML, nil
	}

	emGroup := findChildMappingNode(ydbChildren, "ydb_em")
	if emGroup == nil {
		emGroup = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		emKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ydb_em"}
		ydbChildren.Content = append(ydbChildren.Content, emKey, emGroup)
	}

	hostEntries := make([]*yaml.Node, 0, len(data.Hosts)*2)
	for _, host := range data.Hosts {
		hostKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: host.FQDN}
		if host.Location != "" {
			hostVal := &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "location"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: host.Location},
				},
			}
			hostEntries = append(hostEntries, hostKey, hostVal)
		} else {
			hostVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
			hostEntries = append(hostEntries, hostKey, hostVal)
		}
	}

	newHosts := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: hostEntries,
	}

	for i := 0; i < len(emGroup.Content)-1; i += 2 {
		if emGroup.Content[i].Value == "hosts" {
			emGroup.Content[i+1] = newHosts
			break
		}
	}
	if !keyExistsInMapping(emGroup, "hosts") {
		hostKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "hosts"}
		emGroup.Content = append(emGroup.Content, hostKey, newHosts)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return rawYAML, err
	}
	encoder.Close()

	return buf.Bytes(), nil
}

// stripNilFromEMVars removes nil-valued keys from ydb.vars so a fresh
// yaml.Marshal does not emit "key: null" for patch-delete sentinels.
func stripNilFromEMVars(inv map[string]interface{}) {
	all, _ := inv["all"].(map[string]interface{})
	if all == nil {
		return
	}
	children, _ := all["children"].(map[string]interface{})
	if children == nil {
		return
	}
	ydb, _ := children["ydb"].(map[string]interface{})
	if ydb == nil {
		return
	}
	vars, _ := ydb["vars"].(map[string]interface{})
	if vars == nil {
		return
	}
	for k, v := range vars {
		if v == nil {
			delete(vars, k)
		}
	}
}
