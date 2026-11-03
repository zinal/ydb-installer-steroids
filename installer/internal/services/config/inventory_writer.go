// Package config provides services for parsing and managing configuration files.
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
	"ydb-installer/internal/models"
)

// InventoryWriter handles serializing InventoryData back to YAML
// and writing it to the inventory file.
type InventoryWriter struct {
	// mu serializes the read-modify-write cycle so concurrent PUTs cannot
	// interleave and drop each other's changes.
	mu sync.Mutex
}

func NewInventoryWriter() *InventoryWriter {
	return &InventoryWriter{}
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// Write serializes InventoryData to YAML and writes it to the specified path.
// This creates a fresh inventory file from scratch. For patching an existing
// file (preserving user content), use WriteWithPatch instead.
func (w *InventoryWriter) Write(path string, data *models.InventoryData) error {
	if data == nil {
		return fmt.Errorf("inventory data is nil")
	}

	inv := w.buildAnsibleInventory(data)

	yamlBytes, err := yaml.Marshal(&inv)
	if err != nil {
		return fmt.Errorf("failed to marshal inventory to YAML: %w", err)
	}

	if err := ensureDir(path); err != nil {
		return err
	}

	if err := os.WriteFile(path, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write inventory file: %w", err)
	}

	return nil
}

// WriteWithPatch writes inventory data to the specified path, preserving
// the existing file's comments, unknown variables, and structure.
//
// If the file already exists, it uses YAML Node patching to surgically
// update only the installer-managed variables. If the file doesn't exist,
// it falls back to the standard Write (fresh build).
//
// This prevents user data loss when the installer is run against an
// existing inventory that contains custom variables or comments.
func (w *InventoryWriter) WriteWithPatch(path string, data *models.InventoryData) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if data == nil {
		return fmt.Errorf("inventory data is nil")
	}

	existingBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return w.Write(path, data)
		}
		return fmt.Errorf("failed to read existing inventory: %w", err)
	}

	managedVars := w.buildVars(data)

	patcher := NewInventoryPatcher()
	patchedBytes, err := patcher.Patch(existingBytes, managedVars, data, YdbAlwaysPresentKeys...)
	if err != nil {
		return fmt.Errorf("failed to patch inventory: %w", err)
	}

	// Only write if content actually changed — prevents sync loop.
	if !bytes.Equal(existingBytes, patchedBytes) {
		if err := ensureDir(path); err != nil {
			return err
		}
		if err := os.WriteFile(path, patchedBytes, 0644); err != nil {
			return fmt.Errorf("failed to write patched inventory: %w", err)
		}
	}

	return nil
}

// BuildInventory builds the Ansible inventory structure from InventoryData.
// Returns the raw map structure without writing to file.
func (w *InventoryWriter) BuildInventory(data *models.InventoryData) map[string]interface{} {
	return w.buildAnsibleInventory(data)
}

// buildAnsibleInventory converts InventoryData to the Ansible inventory YAML structure.
// Writes the children-based format: ydb → children → ydbd_static / ydbd_dynamic → hosts.
// The old ydb → hosts key is NEVER written.
func (w *InventoryWriter) buildAnsibleInventory(data *models.InventoryData) map[string]interface{} {
	hosts := w.BuildHosts(data)
	inv := map[string]interface{}{
		"all": map[string]interface{}{
			"children": map[string]interface{}{
				"ydb": map[string]interface{}{
					"children": map[string]interface{}{
						"ydbd_static": map[string]interface{}{
							"hosts": hosts,
						},
						"ydbd_dynamic": map[string]interface{}{
							"hosts": hosts,
						},
					},
					"vars": w.buildVars(data),
				},
			},
		},
	}
	return inv
}

// BuildHosts creates a hosts map from InventoryData.
// Returns a map of hostname → {location: dc-N} suitable for both ydbd_static and ydbd_dynamic.
// Location is derived from DC membership or round-robin from flat hosts.
// Exported for testing purposes.
func (w *InventoryWriter) BuildHosts(data *models.InventoryData) map[string]interface{} {
	locMap := w.buildHostLocationMap(data)

	hosts := make(map[string]interface{})
	for _, host := range data.Hosts {
		loc := locMap[host]
		if loc == "" {
			loc = "dc-1"
		}
		hosts[host] = map[string]interface{}{
			"location": loc,
		}
	}

	return hosts
}

// buildHostLocationMap creates a map of FQDN → location ("dc-1", "dc-2", "dc-3")
// from InventoryData DC lists. Falls back to round-robin from flat Hosts.
func (w *InventoryWriter) buildHostLocationMap(data *models.InventoryData) map[string]string {
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

	// Fallback: round-robin from flat Hosts for hosts not in any DC list
	for i, host := range data.Hosts {
		if _, ok := locMap[host]; !ok && host != "" {
			locMap[host] = locationNames[i%3]
		}
	}

	return locMap
}

// buildVars creates the vars section from InventoryData.
func (w *InventoryWriter) buildVars(data *models.InventoryData) map[string]interface{} {
	vars := make(map[string]interface{})

	if data.SSH != nil {
		if data.SSH.Login != "" {
			vars["ansible_user"] = data.SSH.Login
		}
		if data.SSH.KeyPath != "" {
			vars["ansible_ssh_private_key_file"] = data.SSH.KeyPath
		} else {
			// nil signals deletion in patcher — key path was explicitly cleared.
			vars["ansible_ssh_private_key_file"] = nil
		}

		var sshArgs string
		customArgs := data.SSH.CustomSSHArgs

		hasJumpHostInCustom := strings.Contains(customArgs, "ProxyCommand") || strings.Contains(customArgs, "-J")

		if data.SSH.UseJumpHost && data.SSH.JumpHostFQDN != "" && !hasJumpHostInCustom {
			// ProxyCommand is more stable than -J which has known issues.
			jumpUser := data.SSH.JumpHostUser
			if jumpUser == "" {
				jumpUser = "{{ ansible_user }}"
			}
			proxyCmd := fmt.Sprintf("-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %%h:%%p %s@%s'", jumpUser, data.SSH.JumpHostFQDN)

			if customArgs != "" {
				cleanCustom := strings.ReplaceAll(customArgs, "-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p {{ ansible_user }}@"+data.SSH.JumpHostFQDN+"'", "")
				cleanCustom = strings.ReplaceAll(cleanCustom, "-J {{ ansible_user }}@"+data.SSH.JumpHostFQDN, "")
				cleanCustom = strings.TrimSpace(cleanCustom)
				sshArgs = cleanCustom + " " + proxyCmd
			} else {
				sshArgs = proxyCmd
			}
		} else if customArgs != "" {
			sshArgs = customArgs
		}

		if sshArgs != "" {
			vars["ansible_ssh_common_args"] = sshArgs
		}
	}

	if data.System != nil {
		if data.System.Timezone != "" {
			vars["system_timezone"] = data.System.Timezone
		}
		if len(data.System.NTPServers) > 0 {
			vars["system_ntp_servers"] = data.System.NTPServers
		}
	}

	// Default system settings (not exposed in UI).
	if _, has := vars["system_timezone"]; !has {
		vars["system_timezone"] = "UTC"
	}
	if _, has := vars["system_ntp_servers"]; !has {
		vars["system_ntp_servers"] = []interface{}{
			"time.cloudflare.com",
			"time.google.com",
			"ntp.ripe.net",
			"pool.ntp.org",
		}
	}

	// Distribution: official and local sources are mutually exclusive.
	// nil values signal deletion to the patcher.
	if data.Distribution != nil {
		if data.Distribution.Source == "official" {
			if data.Distribution.Version != "" {
				vars["ydb_version"] = data.Distribution.Version
			}
			vars["ydb_archive"] = nil
			vars["ydbd_binary"] = nil
			vars["ydb_cli_binary"] = nil
			vars["ydbops_binary"] = nil
			vars["ydb_dstool_binary"] = nil
		} else if data.Distribution.Source == "local" {
			if data.Distribution.ArchivePath != "" {
				vars["ydb_archive"] = data.Distribution.ArchivePath
			}
			vars["ydb_version"] = nil
			if data.Distribution.BinaryPath != "" {
				vars["ydbd_binary"] = data.Distribution.BinaryPath
			}
			if data.Distribution.CLIBinary != "" {
				vars["ydb_cli_binary"] = data.Distribution.CLIBinary
			}
			if data.Distribution.OpsBinary != "" {
				vars["ydbops_binary"] = data.Distribution.OpsBinary
			}
			if data.Distribution.ToolBinary != "" {
				vars["ydb_dstool_binary"] = data.Distribution.ToolBinary
			}
		}
	}

	if data.YDBConfig != nil {
		if data.YDBConfig.Domain != "" {
			vars["ydb_domain"] = data.YDBConfig.Domain
		}
		vars["ydb_password"] = data.YDBConfig.Password
		if data.YDBConfig.User != "" {
			vars["ydb_user"] = data.YDBConfig.User
		}
		if data.YDBConfig.CoresStatic > 0 {
			vars["ydb_cores_static"] = data.YDBConfig.CoresStatic
		}
		if data.YDBConfig.CoresDynamic > 0 {
			vars["ydb_cores_dynamic"] = data.YDBConfig.CoresDynamic
		}
		vars["ydb_enforce_user_token_requirement"] = data.YDBConfig.EnforceUserTokenRequirement
		vars["ydb_request_client_certificate"] = data.YDBConfig.RequestClientCertificate
		if data.YDBConfig.PoolKind != "" {
			vars["ydb_pool_kind"] = data.YDBConfig.PoolKind
		}
	}

	if _, has := vars["ydb_user"]; !has {
		vars["ydb_user"] = "root"
	}

	if data.Database != nil {
		if len(data.Database.Entries) > 0 {
			// Primary DB is always the first entry — map to legacy flat fields.
			primary := data.Database.Entries[0]
			if primary.Name != "" {
				vars["ydb_dbname"] = primary.Name
			}
			if primary.Groups > 0 {
				vars["ydb_database_groups"] = primary.Groups
			}

			// Structured ydb_databases field for the multiple-database variant.
			if data.Database.ConfigVariant != "single" && len(data.Database.Entries) > 1 {
				dbs := make([]map[string]interface{}, len(data.Database.Entries))
				for i, entry := range data.Database.Entries {
					dbMap := map[string]interface{}{
						"name":   entry.Name,
						"groups": entry.Groups,
					}
					if len(entry.Hosts) > 0 {
						dbMap["hosts"] = entry.Hosts
					}
					if entry.Affinity != "" {
						dbMap["affinity"] = entry.Affinity
					}
					dbs[i] = dbMap
				}
				vars["ydb_databases"] = dbs
			} else {
				vars["ydb_databases"] = nil
			}
		}
		if data.Database.CoresDynamic > 0 {
			vars["ydb_cores_dynamic"] = data.Database.CoresDynamic
		}
	}

	if data.TLS != nil && data.TLS.CertPath != "" {
		vars["ydb_tls_dir"] = data.TLS.CertPath
	} else {
		// Clear TLS vars to avoid stale data.
		vars["ydb_tls_dir"] = nil
		vars["ydb_tls_mode"] = nil
		vars["ydb_tls_days"] = nil
	}

	if data.Disks != nil && len(data.Disks.SelectedDisks) > 0 {
		disks := make([]map[string]interface{}, len(data.Disks.SelectedDisks))
		for i, disk := range data.Disks.SelectedDisks {
			diskMap := map[string]interface{}{"name": disk.Path}
			if disk.Label != "" {
				diskMap["label"] = disk.Label
			}
			disks[i] = diskMap
		}
		vars["ydb_disks"] = disks
		vars["ydb_allow_format_drives"] = data.Disks.AllowFormat
	}

	// Brokers: fallback to first host from each DC (deduplicated, max 3).
	if len(data.Brokers) > 0 {
		vars["ydb_brokers"] = data.Brokers
	} else {
		var brokers []interface{}
		seen := make(map[string]bool)

		dcLists := [][]string{data.DC1, data.DC2, data.DC3}
		for _, dc := range dcLists {
			for _, host := range dc {
				if host != "" && !seen[host] {
					seen[host] = true
					brokers = append(brokers, host)
					break
				}
			}
		}

		if len(brokers) == 0 && len(data.Hosts) > 0 {
			for _, host := range data.Hosts {
				if !seen[host] {
					seen[host] = true
					brokers = append(brokers, host)
					if len(brokers) >= 3 {
						break
					}
				}
			}
		}

		if len(brokers) > 0 {
			vars["ydb_brokers"] = brokers
		}
	}

	// ydb_dynnodes is auto-generated from Database.Entries merged with
	// any user-pre-configured DynamicNodes. Preserves custom instance/offset
	// mappings while ensuring every database has a corresponding node.
	vars["ydb_dynnodes"] = w.buildDynamicNodes(data.DynamicNodes, data.Database)

	if data.Operations != nil {
		vars["ydbops_local"] = data.Operations.LocalControl
	}
	if _, has := vars["ydbops_local"]; !has {
		vars["ydbops_local"] = true
	}

	// Required for format_cluster playbook.
	vars["ydb_config"] = "{{ ansible_config_file | dirname }}/files/config.yaml"

	// Skip data loss confirmation prompt in format_cluster playbook.
	vars["ydb_skip_data_loss_confirmation_prompt"] = true

	return vars
}

// buildDynamicNodes generates the ydb_dynnodes list from Database.Entries.
//
// Instance letters and offsets are ALWAYS determined by position in the
// entries list — existing DynamicNodes are only consulted when there are
// no database entries at all (preservation fallback). This prevents
// instance letters from "following" database names when they are renamed
// or reordered, which caused unpredictable letter swapping.
//
// Rules:
//   - Entry 0 (default DB) → node with no dbname {a, 1}
//   - Entry 1 → node with dbname {b, 2, dbname: entry.Name}
//   - Entry 2 → node with dbname {c, 3, dbname: entry.Name}
//   - …and so on sequentially.
//   - Empty entry.Name is skipped (no node generated for it).
func (w *InventoryWriter) buildDynamicNodes(
	existing []models.DynamicNode,
	dbConfig *models.DatabaseConfig,
) []interface{} {
	if dbConfig == nil || len(dbConfig.Entries) == 0 {
		if len(existing) > 0 {
			var result []interface{}
			for _, node := range existing {
				m := map[string]interface{}{"instance": node.Instance, "offset": node.Offset}
				if node.Dbname != "" {
					m["dbname"] = node.Dbname
				}
				result = append(result, m)
			}
			return result
		}
		return []interface{}{map[string]interface{}{"instance": "a", "offset": 1}}
	}

	// Position-based assignment: instance = 'a' + idx, offset = idx + 1.
	// idx is incremented only for entries with non-empty names to avoid gaps.
	var result []interface{}
	idx := 0
	for _, entry := range dbConfig.Entries {
		if entry.Name == "" {
			continue
		}
		m := map[string]interface{}{
			"instance": string(rune('a' + idx)),
			"offset":   idx + 1,
		}
		if idx > 0 {
			m["dbname"] = entry.Name
		}
		result = append(result, m)
		idx++
	}

	if len(result) == 0 {
		result = append(result, map[string]interface{}{"instance": "a", "offset": 1})
	}

	return result
}
