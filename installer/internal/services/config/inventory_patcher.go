// Package config provides services for parsing and managing configuration files.
package config

import (
	"bytes"
	"fmt"
	"sort"

	"ydb-installer/internal/models"

	"gopkg.in/yaml.v3"
)

// ManagedVarsKeySet defines the set of variable keys that the installer manages.
// Keys not in this set are preserved as-is from the original inventory file.
var ManagedVarsKeySet = map[string]bool{
	"ansible_user":                           true,
	"ansible_ssh_private_key_file":           true,
	"ansible_ssh_common_args":                true,
	"system_timezone":                        true,
	"system_ntp_servers":                     true,
	"ydb_version":                            true,
	"ydb_archive":                            true,
	"ydbd_binary":                            true,
	"ydb_cli_binary":                         true,
	"ydbops_binary":                          true,
	"ydb_dstool_binary":                      true,
	"ydb_domain":                             true,
	"ydb_password":                           true,
	"ydb_user":                               true,
	"ydb_cores_static":                       true,
	"ydb_cores_dynamic":                      true,
	"ydb_enforce_user_token_requirement":     true,
	"ydb_request_client_certificate":         true,
	"ydb_pool_kind":                          true,
	"ydb_dbname":                             true,
	"ydb_database_groups":                    true,
	"ydb_databases":                          true,
	"ydb_tls_dir":                            true,
	"ydb_tls_mode":                           true,
	"ydb_tls_days":                           true,
	"ydb_disks":                              true,
	"ydb_allow_format_drives":                true,
	"ydb_brokers":                            true,
	"ydb_dynnodes":                           true,
	"ydbops_local":                           true,
	"ydb_config":                             true,
	"ydb_skip_data_loss_confirmation_prompt": true,
	"ydb_topology":                           true,
}

// EMManagedVarsKeySet defines the set of variable keys that the installer manages
// in the EM inventory file (90-inventory.yaml). Keys not in this set are
// preserved as-is from the original file.
var EMManagedVarsKeySet = map[string]bool{
	"ydb_em_frontend":               true,
	"ydb_em_cluster_name":           true,
	"ydb_em_cluster_balancer":       true,
	"ydb_em_db_connection_db_root":  true,
	"ydb_em_db_connection_db":       true,
	"ydb_em_db_connection_endpoint": true,
	"ydb_em_cp_connection_endpoint": true,
	"ydb_em_archive":                true,
	"ydb_em_meta_ui_archive":        true,
	"ydb_em_meta_ui_binary":         true,
	"ydb_em_cp_binary":              true,
	"ydb_em_cp_agent_binary":        true,
	"ydb_em_agent_cpu":              true,
	"ydb_em_agent_memory":           true,
	"ydb_em_az_locations":           true,
	"ydb_em_cp_presets":             true,
	"ydb_use_dynamic_config":        true,
}

// alwaysPresentEntry represents a key-value default that the installer
// ensures is present in the inventory vars section.
type alwaysPresentEntry struct {
	Key   string
	Value interface{}
}

// YdbAlwaysPresentKeys defines defaults injected into 50-inventory.yaml.
// The 50-inventory writer (InventoryWriter.buildVars) already includes these
// keys in managedVars; the patcher adds them only if they are missing from
// both the existing file and the managed vars map.
var YdbAlwaysPresentKeys = []alwaysPresentEntry{
	{Key: "ydb_config", Value: "{{ ansible_config_file | dirname }}/files/config.yaml"},
	{Key: "ydb_skip_data_loss_confirmation_prompt", Value: true},
	{Key: "ydb_user", Value: "root"},
	{Key: "system_timezone", Value: "UTC"},
	{Key: "system_ntp_servers", Value: []interface{}{
		"time.cloudflare.com",
		"time.google.com",
		"ntp.ripe.net",
		"pool.ntp.org",
	}},
	{Key: "ydbops_local", Value: true},
}

// EmAlwaysPresentKeys defines defaults injected into 90-inventory.yaml.
// Most EM defaults are produced by the writer in buildVars(); this list carries
// the ones that must exist on disk but are NOT part of the EMConfig model, and
// therefore must never be overwritten once the user edits them.
var EmAlwaysPresentKeys = []alwaysPresentEntry{
	{Key: "ydb_em_cp_presets", Value: DefaultCPPresets()},
}

// ApplyAlwaysPresentToVars injects always-present defaults into a plain vars
// map, adding each key only when it is not already present.
//
// The fresh-write paths marshal a vars map directly and never invoke the
// patcher, so without this the always-present defaults would only ever appear
// in files that already existed.
func ApplyAlwaysPresentToVars(vars map[string]interface{}, entries []alwaysPresentEntry) map[string]interface{} {
	if vars == nil {
		vars = make(map[string]interface{})
	}
	for _, entry := range entries {
		if _, exists := vars[entry.Key]; !exists {
			vars[entry.Key] = entry.Value
		}
	}
	return vars
}

// InventoryPatcher performs surgical updates to an existing inventory.yaml file,
// preserving all user comments, unknown variables, and structural elements.
type InventoryPatcher struct{}

func NewInventoryPatcher() *InventoryPatcher {
	return &InventoryPatcher{}
}

// Patch applies managed variables and hosts from InventoryData to the raw
// inventory YAML, preserving all non-managed keys, comments, and structure.
//
// The algorithm:
//  1. Parse rawYAML into a yaml.Node tree
//  2. Find or create the all → children → ydb → vars mapping node
//  3. For each managed var key: update if present, add if absent
//  4. For each non-managed key: preserve as-is
//  5. Inject always-present keys
//  6. Update the hosts section from InventoryData (only if host list changed)
//  7. Serialize the modified tree back to YAML
//
// Returns the patched YAML bytes.
func (p *InventoryPatcher) Patch(rawYAML []byte, managedVars map[string]interface{}, data *models.InventoryData, alwaysPresent ...alwaysPresentEntry) ([]byte, error) {
	if len(bytes.TrimSpace(rawYAML)) == 0 {
		return nil, fmt.Errorf("inventory YAML is empty")
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse inventory YAML: %w", err)
	}

	if !p.hasRootMapping(&doc) {
		return nil, fmt.Errorf("inventory document has no root mapping node — cannot patch; content: %.200s", string(rawYAML))
	}

	varsNode := p.findOrCreateVarsNode(&doc)

	p.applyManagedVars(varsNode, managedVars)

	p.applyAlwaysPresent(varsNode, managedVars, alwaysPresent)

	if data != nil && len(data.Hosts) > 0 {
		p.applyHosts(&doc, data)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return nil, fmt.Errorf("failed to marshal patched YAML: %w", err)
	}
	encoder.Close()

	return buf.Bytes(), nil
}

// findOrCreateVarsNode walks the document tree to find
// all → children → ydb → vars and returns the vars MappingNode.
// If any intermediate node is missing, it is created.
func (p *InventoryPatcher) findOrCreateVarsNode(doc *yaml.Node) *yaml.Node {
	rootMapping := getRootMapping(doc)
	if rootMapping == nil {
		return nil
	}

	allMapping := p.findOrCreateChildMapping(rootMapping, "all")
	childrenMapping := p.findOrCreateChildMapping(allMapping, "children")
	ydbMapping := p.findOrCreateChildMapping(childrenMapping, "ydb")
	varsNode := p.findOrCreateChildMapping(ydbMapping, "vars")

	return varsNode
}

// hasRootMapping checks whether the parsed YAML document has a root
// MappingNode (i.e. the document is a mapping like {all: {children: ...}}).
// Ansible inventory must be a mapping, so this guards against empty or
// invalid documents that would produce "null" output on serialize.
func (p *InventoryPatcher) hasRootMapping(doc *yaml.Node) bool {
	return getRootMapping(doc) != nil
}

// findOrCreateChildMapping finds a key in a mapping node and returns its
// value as a MappingNode. If the key doesn't exist, it creates both the
// key and an empty MappingNode value, then returns the new MappingNode.
func (p *InventoryPatcher) findOrCreateChildMapping(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil {
		return nil
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			val := mapping.Content[i+1]
			if val.Kind == yaml.MappingNode {
				return val
			}
			// Value exists but is not a mapping — replace with mapping.
			newMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			mapping.Content[i+1] = newMapping
			return newMapping
		}
	}

	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: key,
	}
	emptyMapping := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
	mapping.Content = append(mapping.Content, keyNode, emptyMapping)
	return emptyMapping
}

// applyManagedVars updates the vars mapping node with the managed variables.
// For each key in managedVars:
//   - If the key already exists in the mapping → update its value node
//   - If the key doesn't exist → append a new key-value pair
//   - Keys in the original mapping that are NOT in managedVars → preserved unchanged
func (p *InventoryPatcher) applyManagedVars(varsNode *yaml.Node, managedVars map[string]interface{}) {
	if varsNode == nil || managedVars == nil {
		return
	}

	for key, value := range managedVars {
		p.setVar(varsNode, key, value)
	}
}

// setVar sets or creates a key-value pair in a mapping node.
// If value is nil, the key is deleted from the mapping (used to signal
// that a previously managed key should be removed from the YAML).
func (p *InventoryPatcher) setVar(mapping *yaml.Node, key string, value interface{}) {
	if value == nil {
		p.deleteVar(mapping, key)
		return
	}

	newValue := p.interfaceToNode(value)

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			// Replace the value node, preserving the key node's comments.
			mapping.Content[i+1] = newValue
			return
		}
	}

	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: key,
	}
	mapping.Content = append(mapping.Content, keyNode, newValue)
}

// deleteVar removes a key-value pair from a mapping node.
func (p *InventoryPatcher) deleteVar(mapping *yaml.Node, key string) {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

// applyAlwaysPresent ensures that caller-specified always-present keys exist
// in the vars mapping. Each key is only added if it is not already present
// (either from the original YAML or from managedVars).
//
// Callers control which defaults are applied by passing the appropriate
// alwaysPresent slice (e.g. ydbAlwaysPresentKeys for 50-inventory,
// emAlwaysPresentKeys for 90-inventory).
func (p *InventoryPatcher) applyAlwaysPresent(varsNode *yaml.Node, managedVars map[string]interface{}, alwaysPresent []alwaysPresentEntry) {
	if varsNode == nil {
		return
	}

	for _, entry := range alwaysPresent {
		if _, isManaged := managedVars[entry.Key]; isManaged {
			continue
		}
		if keyExistsInMapping(varsNode, entry.Key) {
			continue
		}
		p.setVar(varsNode, entry.Key, entry.Value)
	}
}

// buildHostLocationMap creates a map of FQDN → location ("dc-1", "dc-2", "dc-3")
// from InventoryData DC lists. Falls back to round-robin from flat Hosts.
func (p *InventoryPatcher) buildHostLocationMap(data *models.InventoryData) map[string]string {
	locMap := make(map[string]string)

	dcLists := [][]string{data.DC1, data.DC2, data.DC3}
	locationNames := []string{"dc-1", "dc-2", "dc-3"}

	for dcIdx, hosts := range dcLists {
		loc := locationNames[dcIdx]
		for _, host := range hosts {
			if host != "" {
				locMap[host] = loc
			}
		}
	}

	// Fallback: round-robin from flat Hosts for hosts not in any DC list.
	for i, host := range data.Hosts {
		if _, ok := locMap[host]; !ok && host != "" {
			locMap[host] = locationNames[i%3]
		}
	}

	return locMap
}

// applyHosts updates the hosts section under the children-based format:
//
//	all → children → ydb → children → ydbd_static → hosts
//	all → children → ydb → children → ydbd_dynamic → hosts
//
// The SAME host list is written to both groups (identity mapping).
// Each host gets a "location" var derived from DC membership (dc-1/dc-2/dc-3).
//
// Migration: the old ydb → hosts key is ALWAYS deleted. If it carried
// host-specific vars, they are deep-copied into both groups before deletion.
func (p *InventoryPatcher) applyHosts(doc *yaml.Node, data *models.InventoryData) {
	hosts := data.Hosts
	rootMapping := getRootMapping(doc)
	if rootMapping == nil {
		return
	}

	allMapping := p.findOrCreateChildMapping(rootMapping, "all")
	childrenMapping := p.findOrCreateChildMapping(allMapping, "children")
	ydbMapping := p.findOrCreateChildMapping(childrenMapping, "ydb")

	hostLocations := p.buildHostLocationMap(data)

	// Snapshot old ydb → hosts for migration (host-specific vars may be present)
	// AND always delete it afterwards — the old flat key must NOT survive.
	oldHosts := p.findHostsInMapping(ydbMapping)
	oldHostsMatch := oldHosts != nil && p.hostsMatch(oldHosts, hosts)
	oldHasVars := oldHosts != nil && p.hasHostVars(oldHosts)

	ydbChildren := p.findOrCreateChildMapping(ydbMapping, "children")

	for _, group := range []string{"ydbd_static", "ydbd_dynamic"} {
		p.writeGroupHosts(ydbChildren, group, hosts, oldHosts, oldHostsMatch && oldHasVars, hostLocations)
	}

	// Migration: remove old ydb → hosts key.
	p.deleteVar(ydbMapping, "hosts")
}

// writeGroupHosts updates a single child group's hosts section.
// If existing hosts in the group match, preserves them as-is but ensures
// location host-var is up-to-date via ensureHostLocations.
// If oldLegacyHosts is non-nil and matches (migration from ydb → hosts),
// deep-copies the old node to preserve host-specific vars.
func (p *InventoryPatcher) writeGroupHosts(ydbChildren *yaml.Node, groupName string, hosts []string, oldLegacyHosts *yaml.Node, useLegacy bool, hostLocations map[string]string) {
	if ydbChildren == nil {
		return
	}
	hosts = p.dedupHosts(hosts)

	groupMapping := p.findOrCreateChildMapping(ydbChildren, groupName)

	// If existing group hosts match — preserve but ensure location vars are correct.
	existing := p.findHostsInMapping(groupMapping)
	if existing != nil && p.hostsMatch(existing, hosts) {
		p.ensureHostLocations(existing, hostLocations)
		return
	}

	var newHostsNode *yaml.Node
	if useLegacy && oldLegacyHosts != nil {
		newHostsNode = p.deepCopyNode(oldLegacyHosts)
		p.ensureHostLocations(newHostsNode, hostLocations)
	} else {
		newHostsNode = p.buildHostsNode(hosts, hostLocations)
	}

	for i := 0; i < len(groupMapping.Content)-1; i += 2 {
		if groupMapping.Content[i].Value == "hosts" {
			groupMapping.Content[i+1] = newHostsNode
			return
		}
	}
	hostKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "hosts"}
	groupMapping.Content = append(groupMapping.Content, hostKey, newHostsNode)
}

// dedupHosts returns a deduplicated copy of the host list.
func (p *InventoryPatcher) dedupHosts(hosts []string) []string {
	seen := make(map[string]bool, len(hosts))
	var out []string
	for _, h := range hosts {
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	return out
}

// hostsMatch checks whether an existing hosts mapping node contains exactly the desired host keys.
func (p *InventoryPatcher) hostsMatch(existingHosts *yaml.Node, hosts []string) bool {
	if existingHosts == nil {
		return false
	}
	desired := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		desired[h] = true
	}
	existingKeys := make(map[string]bool)
	for i := 0; i < len(existingHosts.Content)-1; i += 2 {
		key := existingHosts.Content[i].Value
		existingKeys[key] = true
		if !desired[key] {
			return false
		}
	}
	for h := range desired {
		if !existingKeys[h] {
			return false
		}
	}
	return true
}

// hasHostVars checks whether any host entry in the mapping has a non-null value.
func (p *InventoryPatcher) hasHostVars(hostsNode *yaml.Node) bool {
	if hostsNode == nil {
		return false
	}
	for i := 0; i < len(hostsNode.Content)-1; i += 2 {
		val := hostsNode.Content[i+1]
		if val != nil && (val.Kind != yaml.ScalarNode || val.Tag != "!!null") {
			return true
		}
	}
	return false
}

// buildHostsNode creates a fresh hosts mapping node from a list of hostnames,
// each with a "location" host-var derived from DC membership (dc-1/dc-2/dc-3).
func (p *InventoryPatcher) buildHostsNode(hosts []string, hostLocations map[string]string) *yaml.Node {
	entries := make([]*yaml.Node, 0, len(hosts)*2)
	for _, host := range hosts {
		loc := hostLocations[host]
		if loc == "" {
			loc = "dc-1"
		}
		hostVars := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "location"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: loc},
			},
		}
		entries = append(entries,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: host},
			hostVars,
		)
	}
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: entries}
}

// ensureHostLocations walks an existing hosts mapping node and sets/updates
// the "location" var on each host entry, preserving all other existing vars.
func (p *InventoryPatcher) ensureHostLocations(hostsNode *yaml.Node, hostLocations map[string]string) {
	if hostsNode == nil || hostsNode.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(hostsNode.Content)-1; i += 2 {
		hostKey := hostsNode.Content[i].Value
		hostVal := hostsNode.Content[i+1]

		loc, ok := hostLocations[hostKey]
		if !ok || loc == "" {
			continue
		}

		if hostVal == nil || (hostVal.Kind == yaml.ScalarNode && hostVal.Tag == "!!null") {
			hostsNode.Content[i+1] = &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "location"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: loc},
				},
			}
			continue
		}

		if hostVal.Kind == yaml.MappingNode {
			found := false
			for j := 0; j < len(hostVal.Content)-1; j += 2 {
				if hostVal.Content[j].Value == "location" {
					hostVal.Content[j+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: loc}
					found = true
					break
				}
			}
			if !found {
				hostVal.Content = append(hostVal.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "location"},
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: loc},
				)
			}
		}
	}
}

// deepCopyNode performs a recursive deep copy of a yaml.Node tree.
func (p *InventoryPatcher) deepCopyNode(src *yaml.Node) *yaml.Node {
	if src == nil {
		return nil
	}
	dst := &yaml.Node{
		Kind:        src.Kind,
		Tag:         src.Tag,
		Value:       src.Value,
		LineComment: src.LineComment,
		HeadComment: src.HeadComment,
		FootComment: src.FootComment,
		Line:        src.Line,
		Column:      src.Column,
	}
	for _, child := range src.Content {
		dst.Content = append(dst.Content, p.deepCopyNode(child))
	}
	return dst
}

// findHostsInMapping returns the "hosts" mapping node from a parent mapping, or nil.
// Used for both old ydb → hosts and new ydb → children → group → hosts.
func (p *InventoryPatcher) findHostsInMapping(mapping *yaml.Node) *yaml.Node {
	if mapping == nil {
		return nil
	}
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == "hosts" {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// interfaceToNode converts a Go value to a yaml.Node for use in the YAML tree.
func (p *InventoryPatcher) interfaceToNode(value interface{}) *yaml.Node {
	switch v := value.(type) {
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
	case *yaml.Node:
		return v
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
	case bool:
		tag := "!!bool"
		strVal := "false"
		if v {
			strVal = "true"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: strVal}
	case int:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", v)}
	case float64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: fmt.Sprintf("%v", v)}
	case []string:
		return p.stringSequenceNode(v)
	case []interface{}:
		return p.sequenceNode(v)
	case []map[string]interface{}:
		return p.mappingSequenceNode(v)
	case map[string]interface{}:
		return p.mapToNode(v)
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprintf("%v", v)}
	}
}

// stringSequenceNode converts a []string to a yaml SequenceNode.
//
// Without this case, []string values fall through interfaceToNode's default
// branch and get stringified via fmt.Sprintf("%v"), producing a corrupt scalar
// like "[dc-1 dc-2]" instead of a real YAML list.
func (p *InventoryPatcher) stringSequenceNode(items []string) *yaml.Node {
	content := make([]*yaml.Node, 0, len(items))
	for _, item := range items {
		content = append(content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: item})
	}
	return &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: content,
	}
}

// sequenceNode converts a []interface{} to a yaml.Node.
func (p *InventoryPatcher) sequenceNode(items []interface{}) *yaml.Node {
	content := make([]*yaml.Node, 0, len(items))
	for _, item := range items {
		content = append(content, p.interfaceToNode(item))
	}
	return &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: content,
	}
}

// mappingSequenceNode converts a []map[string]interface{} to a yaml.Node.
func (p *InventoryPatcher) mappingSequenceNode(items []map[string]interface{}) *yaml.Node {
	content := make([]*yaml.Node, 0, len(items))
	for _, item := range items {
		content = append(content, p.mapToNode(item))
	}
	return &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: content,
	}
}

// mapToNode converts a map[string]interface{} to a yaml MappingNode.
func (p *InventoryPatcher) mapToNode(m map[string]interface{}) *yaml.Node {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	content := make([]*yaml.Node, 0, len(m)*2)
	for _, k := range keys {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: k,
		}
		valueNode := p.interfaceToNode(m[k])
		content = append(content, keyNode, valueNode)
	}

	return &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: content,
	}
}

// keyExistsInMap checks if a key exists in a Go map.
func (p *InventoryPatcher) keyExistsInMap(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	_, exists := m[key]
	return exists
}

// clearManagedKeys removes all managed keys from the vars mapping node.
func (p *InventoryPatcher) clearManagedKeys(varsNode *yaml.Node, keysToClear []string) {
	if varsNode == nil {
		return
	}
	clearSet := make(map[string]bool, len(keysToClear))
	for _, k := range keysToClear {
		clearSet[k] = true
	}
	var newContent []*yaml.Node
	for i := 0; i < len(varsNode.Content)-1; i += 2 {
		if !clearSet[varsNode.Content[i].Value] {
			newContent = append(newContent, varsNode.Content[i], varsNode.Content[i+1])
		}
	}
	varsNode.Content = newContent
}
