package config

import (
	"fmt"
	"os"
	"strconv"

	"ydb-installer/internal/models"

	"gopkg.in/yaml.v3"
)

// EMInventoryService handles parsing and writing of the EM inventory file
// (90-inventory.yaml). This is separate from the main InventoryService which
// handles 50-inventory.yaml.
type EMInventoryService struct{}

func NewEMInventoryService() *EMInventoryService {
	return &EMInventoryService{}
}

// Parse reads and parses an EM Ansible inventory file (90-inventory.yaml),
// transforming it into EMConfig.
func (s *EMInventoryService) Parse(path string) (*models.EMConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read EM inventory file: %w", err)
	}

	var emInv models.EMAnsibleInventory
	if err := yaml.Unmarshal(data, &emInv); err != nil {
		return nil, fmt.Errorf("failed to parse EM Ansible inventory: %w", err)
	}

	emConfig, err := s.transformFromAnsible(&emInv)
	if err != nil {
		return nil, fmt.Errorf("failed to transform EM inventory: %w", err)
	}

	return emConfig, nil
}

// transformFromAnsible converts EMAnsibleInventory structure to EMConfig.
func (s *EMInventoryService) transformFromAnsible(emInv *models.EMAnsibleInventory) (*models.EMConfig, error) {
	emConfig := &models.EMConfig{}

	emHosts := s.extractEMHosts(&emInv.All.Children.YDB.Children.YdbEM)
	if len(emHosts) > 0 {
		emConfig.Hosts = emHosts
	}

	vars := emInv.All.Children.YDB.Vars
	if vars == nil {
		return emConfig, nil
	}

	if name, ok := vars["ydb_em_cluster_name"].(string); ok {
		emConfig.ClusterName = name
	}

	if root, ok := vars["ydb_em_db_connection_db_root"].(string); ok {
		emConfig.DbConnectionDBRoot = root
	}
	if db, ok := vars["ydb_em_db_connection_db"].(string); ok {
		emConfig.DbConnectionDB = db
	}
	if ep, ok := vars["ydb_em_db_connection_endpoint"].(string); ok {
		emConfig.DbConnectionEndpoint = ep
	}

	if archive, ok := vars["ydb_em_archive"].(string); ok {
		emConfig.ArchivePath = archive
	}
	if archive, ok := vars["ydb_em_meta_ui_archive"].(string); ok {
		emConfig.MetaUIArchive = archive
	}

	if binary, ok := vars["ydb_em_meta_ui_binary"].(string); ok {
		emConfig.MetaUIBinary = binary
	}
	if binary, ok := vars["ydb_em_cp_binary"].(string); ok {
		emConfig.CPBinary = binary
	}
	if binary, ok := vars["ydb_em_cp_agent_binary"].(string); ok {
		emConfig.CPAgentBinary = binary
	}

	if cpu, ok := vars["ydb_em_agent_cpu"]; ok {
		emConfig.AgentCPU = parseIntAny(cpu)
	}
	if mem, ok := vars["ydb_em_agent_memory"]; ok {
		emConfig.AgentMemory = parseIntAny(mem)
	}

	if raw, ok := vars["ydb_em_az_locations"]; ok {
		emConfig.AZLocations = ParseAZLocations(raw)
	}

	// ydb_em_cp_presets is intentionally NOT parsed into the model: it is a
	// file-level default injected via EmAlwaysPresentKeys and never managed by
	// the UI, so the patcher leaves any user customisation untouched.

	if udc, ok := vars["ydb_use_dynamic_config"].(bool); ok {
		emConfig.UseDynamicConfig = udc
	}

	if tlsDir, ok := vars["ydb_em_tls_dir"].(string); ok {
		emConfig.EmTLSCertPath = tlsDir
	}

	return emConfig, nil
}

// extractEMHosts extracts EM hosts with their locations from the ydb_em group.
func (s *EMInventoryService) extractEMHosts(group *models.EMGroup) []models.EMHost {
	var hosts []models.EMHost
	for fqdn, hostVars := range group.Hosts {
		host := models.EMHost{FQDN: fqdn}
		if loc, ok := hostVars["location"].(string); ok {
			host.Location = loc
		}
		hosts = append(hosts, host)
	}
	return hosts
}

// parseIntAny safely parses an integer from various types.
func parseIntAny(value interface{}) int {
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
