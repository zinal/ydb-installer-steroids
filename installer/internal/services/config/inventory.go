// Package config provides services for parsing and managing configuration files.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	"ydb-installer/internal/models"
)

// InventoryService handles parsing and validation of Ansible inventory files.
type InventoryService struct{}

func NewInventoryService() *InventoryService {
	return &InventoryService{}
}

// Parse reads and parses an Ansible inventory file, transforming it into InventoryData.
func (s *InventoryService) Parse(path string) (*models.InventoryData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory file: %w", err)
	}

	var ansibleInv models.AnsibleInventory
	if err := yaml.Unmarshal(data, &ansibleInv); err != nil {
		return nil, fmt.Errorf("failed to parse Ansible inventory: %w", err)
	}

	invData, err := s.transformFromAnsible(&ansibleInv)
	if err != nil {
		return nil, fmt.Errorf("failed to transform Ansible inventory: %w", err)
	}

	// Migration: if no hosts found via children, try legacy ydb.hosts (pre-static/dynamic).
	if len(invData.Hosts) == 0 {
		if legacyHosts := s.extractLegacyHosts(data); len(legacyHosts) > 0 {
			invData.Hosts = legacyHosts
		}
	}

	return invData, nil
}

// extractLegacyHosts reads hosts from the old ydb → hosts key in raw YAML
// as a one-shot migration. Returns nil if the key does not exist.
func (s *InventoryService) extractLegacyHosts(rawYAML []byte) []string {
	var doc yaml.Node
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return nil
	}
	root := getRootMappingNode(&doc)
	if root == nil {
		return nil
	}
	all := findChildMappingNode(root, "all")
	if all == nil {
		return nil
	}
	children := findChildMappingNode(all, "children")
	if children == nil {
		return nil
	}
	ydb := findChildMappingNode(children, "ydb")
	if ydb == nil {
		return nil
	}
	hostsNode := findChildMappingNode(ydb, "hosts")
	if hostsNode == nil {
		return nil
	}
	var hosts []string
	for i := 0; i < len(hostsNode.Content)-1; i += 2 {
		hosts = append(hosts, hostsNode.Content[i].Value)
	}
	return hosts
}

// getRootMappingNode returns the root MappingNode from a yaml.Node.
// Duplicates getRootMapping in yaml_helpers.go but is kept standalone for migration use.
func getRootMappingNode(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.MappingNode {
		return node
	}
	for _, child := range node.Content {
		if child != nil && child.Kind == yaml.MappingNode {
			return child
		}
	}
	return nil
}

// Validate checks if the parsed inventory data is valid.
func (s *InventoryService) Validate(invData *models.InventoryData) error {
	if invData == nil {
		return fmt.Errorf("inventory data is nil")
	}

	if invData.SSH != nil {
		if invData.SSH.Login == "" {
			return fmt.Errorf("SSH login is required")
		}
		if invData.SSH.AuthType == "key" && invData.SSH.KeyPath == "" {
			return fmt.Errorf("SSH key path is required when auth_type is 'key'")
		}
		if invData.SSH.AuthType == "password" && invData.SSH.Password == "" {
			return fmt.Errorf("SSH password is required when auth_type is 'password'")
		}
	}

	if len(invData.Hosts) == 0 {
		return fmt.Errorf("at least one host is required")
	}

	if invData.YDBConfig != nil {
		if invData.YDBConfig.Domain == "" {
			return fmt.Errorf("YDB domain is required")
		}
		if invData.YDBConfig.CoresStatic <= 0 {
			return fmt.Errorf("YDB static cores must be positive")
		}
	}

	return nil
}

// transformFromAnsible converts Ansible inventory structure to simplified InventoryData.
func (s *InventoryService) transformFromAnsible(ansibleInv *models.AnsibleInventory) (*models.InventoryData, error) {
	invData := &models.InventoryData{}

	hosts := s.extractHosts(ansibleInv)
	if len(hosts) > 0 {
		invData.Hosts = hosts
	}

	vars := ansibleInv.All.Children.YDB.Vars
	if vars == nil {
		return invData, nil
	}

	if sshConfig := s.extractSSHConfig(vars); sshConfig != nil {
		invData.SSH = sshConfig
	}

	if brokers := s.extractBrokers(vars); len(brokers) > 0 {
		invData.Brokers = brokers
	}

	if disksConfig := s.extractDisksConfig(vars); disksConfig != nil {
		invData.Disks = disksConfig
	}

	if distConfig := s.extractDistributionConfig(vars); distConfig != nil {
		invData.Distribution = distConfig
	}

	if ydbConfig := s.extractYDBConfig(vars); ydbConfig != nil {
		invData.YDBConfig = ydbConfig
	}

	if dbConfig := s.extractDatabaseConfig(vars); dbConfig != nil {
		invData.Database = dbConfig
	}

	if tlsConfig := s.extractTLSConfig(vars); tlsConfig != nil {
		invData.TLS = tlsConfig
	}

	if dynNodes := s.extractDynamicNodes(vars); len(dynNodes) > 0 {
		invData.DynamicNodes = dynNodes
	}

	if sysConfig := s.extractSystemConfig(vars); sysConfig != nil {
		invData.System = sysConfig
	}

	if opsConfig := s.extractOperationsConfig(vars); opsConfig != nil {
		invData.Operations = opsConfig
	}

	if dc1 := s.extractStringList(vars, "ydb_dc1_hosts"); len(dc1) > 0 {
		invData.DC1 = dc1
	}
	if dc2 := s.extractStringList(vars, "ydb_dc2_hosts"); len(dc2) > 0 {
		invData.DC2 = dc2
	}
	if dc3 := s.extractStringList(vars, "ydb_dc3_hosts"); len(dc3) > 0 {
		invData.DC3 = dc3
	}

	// Sync flat hosts from DC lists for backward compatibility.
	if len(invData.DC1) > 0 || len(invData.DC2) > 0 || len(invData.DC3) > 0 {
		invData.SyncHostsFromDC()
	}

	if topologyConfig := s.extractTopologyConfig(vars); topologyConfig != nil {
		invData.Topology = topologyConfig
	}

	return invData, nil
}

// extractHosts extracts host FQDNs from Ansible inventory children groups
// (ydbd_static.hosts ∪ ydbd_dynamic.hosts). Returns deduplicated flat list.
func (s *InventoryService) extractHosts(ansibleInv *models.AnsibleInventory) []string {
	ydb := ansibleInv.All.Children.YDB
	seen := make(map[string]bool)
	var hosts []string

	if ydb.Children != nil {
		for fqdn := range ydb.Children.YdbdStatic.Hosts {
			if !seen[fqdn] {
				seen[fqdn] = true
				hosts = append(hosts, fqdn)
			}
		}
		for fqdn := range ydb.Children.YdbdDynamic.Hosts {
			if !seen[fqdn] {
				seen[fqdn] = true
				hosts = append(hosts, fqdn)
			}
		}
	}

	return hosts
}

// extractSSHConfig extracts SSH configuration from Ansible variables.
func (s *InventoryService) extractSSHConfig(vars map[string]interface{}) *models.SSHConfig {
	sshConfig := &models.SSHConfig{}

	if login, ok := vars["ansible_user"].(string); ok && login != "" {
		sshConfig.Login = login
	}

	if keyPath, ok := vars["ansible_ssh_private_key_file"].(string); ok && keyPath != "" {
		sshConfig.KeyPath = keyPath
		sshConfig.AuthType = "key"
	}

	if sshArgs, ok := vars["ansible_ssh_common_args"].(string); ok && sshArgs != "" {
		sshConfig.CustomSSHArgs = sshArgs
		if jumpHost := s.parseJumpHost(sshArgs); jumpHost != nil {
			sshConfig.UseJumpHost = true
			sshConfig.JumpHostUser = jumpHost.user
			sshConfig.JumpHostFQDN = jumpHost.fqdn
		}
	}

	if sshConfig.Login != "" {
		return sshConfig
	}

	return nil
}

// parseJumpHost parses jump host configuration from SSH args.
// Only the ProxyCommand format is recognized; the legacy -J flag and any
// other SSH arguments do NOT trigger jump host auto-fill.
func (s *InventoryService) parseJumpHost(sshArgs string) *struct {
	user string
	fqdn string
} {
	reProxy := regexp.MustCompile(`-o\s+ProxyCommand='ssh[^']*?\s+(\S+)\s*@\s*(\S+)'`)
	if matches := reProxy.FindStringSubmatch(sshArgs); len(matches) >= 3 {
		user := strings.TrimSpace(matches[1])
		fqdn := strings.TrimSpace(matches[2])
		// Strip Jinja2 template markers from the captured username.
		// The regex `\S+` greedily captures `ansible_user}}` when the
		// template is `{{ ansible_user }}` because `}}` is contiguous
		// with the username (no space).  Real OS usernames never start
		// with `{` or end with `}`, so stripping is always safe.
		user = strings.TrimLeft(user, "{")
		user = strings.TrimRight(user, "}")
		user = strings.TrimSpace(user)
		return &struct {
			user string
			fqdn string
		}{
			user: user,
			fqdn: fqdn,
		}
	}
	return nil
}

// extractBrokers extracts broker hosts from Ansible variables.
func (s *InventoryService) extractBrokers(vars map[string]interface{}) []string {
	if brokers, ok := vars["ydb_brokers"].([]interface{}); ok {
		var brokerList []string
		for _, broker := range brokers {
			if brokerStr, ok := broker.(string); ok {
				brokerList = append(brokerList, brokerStr)
			}
		}
		return brokerList
	}
	return nil
}

// extractDisksConfig extracts disk configuration from Ansible variables.
func (s *InventoryService) extractDisksConfig(vars map[string]interface{}) *models.DisksConfig {
	disksConfig := &models.DisksConfig{}

	if allowFormat, ok := vars["ydb_allow_format_drives"].(bool); ok {
		disksConfig.AllowFormat = allowFormat
	}

	if disks, ok := vars["ydb_disks"].([]interface{}); ok {
		var diskEntries []models.DiskEntry
		for _, disk := range disks {
			if diskMap, ok := disk.(map[string]interface{}); ok {
				entry := models.DiskEntry{}
				if name, ok := diskMap["name"].(string); ok {
					entry.Path = name
				}
				if label, ok := diskMap["label"].(string); ok {
					entry.Label = label
				}
				if entry.Path != "" {
					diskEntries = append(diskEntries, entry)
				}
			}
		}
		if len(diskEntries) > 0 {
			disksConfig.SelectedDisks = diskEntries
		}
	}

	if len(disksConfig.SelectedDisks) > 0 || disksConfig.AllowFormat {
		return disksConfig
	}

	return nil
}

// extractDistributionConfig extracts distribution configuration from Ansible variables.
func (s *InventoryService) extractDistributionConfig(vars map[string]interface{}) *models.DistributionConfig {
	distConfig := &models.DistributionConfig{}

	// ydb_version presence is the most reliable indicator of official source.
	if _, ok := vars["ydb_version"]; ok {
		distConfig.Source = "official"
	} else if _, ok := vars["ydbd_binary"]; ok {
		distConfig.Source = "local"
	} else {
		return nil
	}

	if version, ok := vars["ydb_version"].(string); ok && version != "" {
		distConfig.Version = version
	}

	if archivePath, ok := vars["ydb_archive"].(string); ok && archivePath != "" {
		distConfig.ArchivePath = cleanJinja2Template(archivePath)
	}

	if binaryPath, ok := vars["ydbd_binary"].(string); ok && binaryPath != "" {
		distConfig.BinaryPath = cleanJinja2Template(binaryPath)
	}
	if cliBinary, ok := vars["ydb_cli_binary"].(string); ok && cliBinary != "" {
		distConfig.CLIBinary = cleanJinja2Template(cliBinary)
	}
	if opsBinary, ok := vars["ydbops_binary"].(string); ok && opsBinary != "" {
		distConfig.OpsBinary = cleanJinja2Template(opsBinary)
	}
	if toolBinary, ok := vars["ydb_dstool_binary"].(string); ok && toolBinary != "" {
		distConfig.ToolBinary = cleanJinja2Template(toolBinary)
	}

	return distConfig
}

// extractYDBConfig extracts YDB configuration from Ansible variables.
func (s *InventoryService) extractYDBConfig(vars map[string]interface{}) *models.YDBConfig {
	ydbConfig := &models.YDBConfig{}

	if domain, ok := vars["ydb_domain"].(string); ok && domain != "" {
		ydbConfig.Domain = domain
	}

	// ydb_password is a plaintext field in 50-inventory.yaml.
	if password, ok := vars["ydb_password"].(string); ok && password != "" {
		ydbConfig.Password = password
	}

	if user, ok := vars["ydb_user"].(string); ok && user != "" {
		ydbConfig.User = user
	}

	// YAML unmarshals numbers as float64, so parseInt handles type coercion.
	if coresStatic, ok := vars["ydb_cores_static"]; ok {
		ydbConfig.CoresStatic = s.parseInt(coresStatic)
	}
	if coresDynamic, ok := vars["ydb_cores_dynamic"]; ok {
		ydbConfig.CoresDynamic = s.parseInt(coresDynamic)
	}

	if enforceToken, ok := vars["ydb_enforce_user_token_requirement"].(bool); ok {
		ydbConfig.EnforceUserTokenRequirement = enforceToken
	}
	if requestCert, ok := vars["ydb_request_client_certificate"].(bool); ok {
		ydbConfig.RequestClientCertificate = requestCert
	}

	if poolKind, ok := vars["ydb_pool_kind"].(string); ok && poolKind != "" {
		ydbConfig.PoolKind = poolKind
	}

	if ydbConfig.Domain != "" {
		return ydbConfig
	}

	return nil
}

// extractDatabaseConfig extracts database configuration from Ansible variables.
func (s *InventoryService) extractDatabaseConfig(vars map[string]interface{}) *models.DatabaseConfig {
	dbConfig := &models.DatabaseConfig{
		ConfigVariant: "single",
	}

	if coresDynamic, ok := vars["ydb_cores_dynamic"]; ok {
		dbConfig.CoresDynamic = s.parseInt(coresDynamic)
	}

	hasPrimary := false
	entry := models.DatabaseEntry{}
	if dbname, ok := vars["ydb_dbname"].(string); ok && dbname != "" {
		entry.Name = dbname
		hasPrimary = true
	}
	if groups, ok := vars["ydb_database_groups"].(int); ok {
		entry.Groups = groups
	}

	if dbs, ok := vars["ydb_databases"].([]interface{}); ok && len(dbs) > 0 {
		if len(dbs) == 2 {
			// Heuristic: single-plus-em uses "ydb_em" as the second DB name.
			if db2, ok2 := dbs[1].(map[string]interface{}); ok2 {
				if name, _ := db2["name"].(string); name == "ydb_em" {
					dbConfig.ConfigVariant = "single-plus-em"
				} else {
					dbConfig.ConfigVariant = "multiple"
				}
			} else {
				dbConfig.ConfigVariant = "multiple"
			}
		} else {
			dbConfig.ConfigVariant = "multiple"
		}
		for _, db := range dbs {
			if dbMap, ok := db.(map[string]interface{}); ok {
				de := models.DatabaseEntry{}
				if name, ok := dbMap["name"].(string); ok && name != "" {
					de.Name = name
				}
				if groups, ok := dbMap["groups"]; ok {
					de.Groups = s.parseInt(groups)
				}
				if hosts, ok := dbMap["hosts"].([]interface{}); ok {
					for _, h := range hosts {
						if hs, ok := h.(string); ok {
							de.Hosts = append(de.Hosts, hs)
						}
					}
				}
				if affinity, ok := dbMap["affinity"].(string); ok {
					de.Affinity = affinity
				}
				if de.Name != "" {
					dbConfig.Entries = append(dbConfig.Entries, de)
				}
			}
		}
	}

	if len(dbConfig.Entries) == 0 && hasPrimary {
		dbConfig.ConfigVariant = "single"
		dbConfig.Entries = []models.DatabaseEntry{entry}
	}

	if len(dbConfig.Entries) > 0 {
		return dbConfig
	}

	return nil
}

// extractTLSConfig extracts TLS configuration from Ansible variables.
func (s *InventoryService) extractTLSConfig(vars map[string]interface{}) *models.TLSConfig {
	if tlsDir, ok := vars["ydb_tls_dir"].(string); ok && tlsDir != "" {
		cleanPath := cleanJinja2Template(tlsDir)
		if cleanPath != "" {
			return &models.TLSConfig{
				CertPath: cleanPath,
			}
		}
	}
	return nil
}

// extractDynamicNodes extracts dynamic nodes configuration from Ansible variables.
func (s *InventoryService) extractDynamicNodes(vars map[string]interface{}) []models.DynamicNode {
	if dynNodes, ok := vars["ydb_dynnodes"].([]interface{}); ok {
		var nodes []models.DynamicNode
		for _, node := range dynNodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				dynNode := models.DynamicNode{}
				if instance, ok := nodeMap["instance"].(string); ok {
					dynNode.Instance = instance
				}
				if offset, ok := nodeMap["offset"].(int); ok {
					dynNode.Offset = offset
				}
				if dbname, ok := nodeMap["dbname"].(string); ok {
					dynNode.Dbname = dbname
				}
				if dynNode.Instance != "" {
					nodes = append(nodes, dynNode)
				}
			}
		}
		return nodes
	}
	return nil
}

// extractSystemConfig extracts system configuration from Ansible variables.
func (s *InventoryService) extractSystemConfig(vars map[string]interface{}) *models.SystemConfig {
	sysConfig := &models.SystemConfig{}

	if timezone, ok := vars["system_timezone"].(string); ok && timezone != "" {
		sysConfig.Timezone = timezone
	}

	if ntpServers, ok := vars["system_ntp_servers"].([]interface{}); ok {
		var servers []string
		for _, server := range ntpServers {
			if serverStr, ok := server.(string); ok {
				servers = append(servers, serverStr)
			}
		}
		if len(servers) > 0 {
			sysConfig.NTPServers = servers
		}
	}

	if sysConfig.Timezone != "" {
		return sysConfig
	}

	return nil
}

// extractOperationsConfig extracts operations configuration from Ansible variables.
func (s *InventoryService) extractOperationsConfig(vars map[string]interface{}) *models.OperationsConfig {
	opsConfig := &models.OperationsConfig{}

	if localControl, ok := vars["ydbops_local"].(bool); ok {
		opsConfig.LocalControl = localControl
		return opsConfig
	}

	return nil
}

// extractTopologyConfig extracts topology configuration from Ansible variables.
func (s *InventoryService) extractTopologyConfig(vars map[string]interface{}) *models.TopologyConfig {
	if topology, ok := vars["ydb_topology"].(string); ok && topology != "" {
		return &models.TopologyConfig{Type: topology}
	}
	return nil
}

// cleanJinja2Template removes Jinja2 template syntax from a string.
func cleanJinja2Template(value string) string {
	re := regexp.MustCompile(`\{\{.*?\}\}`)
	cleaned := re.ReplaceAllString(value, "")

	cleaned = strings.ReplaceAll(cleaned, "//", "/")
	cleaned = strings.TrimSuffix(cleaned, "/")

	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// parseBool safely parses a boolean from various types.
func (s *InventoryService) parseBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true" || v == "1"
	case int:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

// extractStringList extracts a list of strings from a vars field by key.
func (s *InventoryService) extractStringList(vars map[string]interface{}, key string) []string {
	if raw, ok := vars[key]; ok {
		switch v := raw.(type) {
		case []interface{}:
			var result []string
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// parseInt safely parses an integer from various types.
func (s *InventoryService) parseInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	case float64:
		return int(v)
	case bool:
		if v {
			return 1
		}
	}
	return 0
}
