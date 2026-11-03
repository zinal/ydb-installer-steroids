package models

type YDBHostGroup struct {
	Hosts map[string]interface{} `yaml:"hosts,omitempty"`
}

// AnsibleInventory mirrors the raw Ansible inventory YAML for parsing only.
//
//	all:
//	  children:
//	    ydb:
//	      children:
//	        ydbd_static:
//	          hosts:
//	            node01:
//	        ydbd_dynamic:
//	          hosts:
//	            node01:
//	      vars: ...
//
// The top-level ydb → hosts key is NO LONGER supported; it is deleted
// automatically on the first write (migration) and never written.
type AnsibleInventory struct {
	All struct {
		Children struct {
			YDB struct {
				Children *struct {
					YdbdStatic  YDBHostGroup `yaml:"ydbd_static"`
					YdbdDynamic YDBHostGroup `yaml:"ydbd_dynamic"`
				} `yaml:"children,omitempty"`
				Vars map[string]interface{} `yaml:"vars,omitempty"`
			} `yaml:"ydb"`
		} `yaml:"children"`
	} `yaml:"all"`
}

// EMAnsibleInventory mirrors 90-inventory.yaml for parsing only.
//
// ydbd_static / ydbd_dynamic are owned exclusively by 50-inventory.yaml and
// are NEVER written into 90-inventory.yaml. The YdbDynamic field is retained
// only for backward-compatible parsing of legacy 90 files that may still
// contain it.
type EMAnsibleInventory struct {
	All struct {
		Children struct {
			YDB struct {
				Vars     map[string]interface{} `yaml:"vars,omitempty"`
				Children struct {
					YdbEM      EMGroup `yaml:"ydb_em"`
					YdbDynamic EMGroup `yaml:"ydbd_dynamic"`
				} `yaml:"children,omitempty"`
			} `yaml:"ydb"`
		} `yaml:"children"`
	} `yaml:"all"`
}

type EMGroup struct {
	Hosts map[string]map[string]interface{} `yaml:"hosts,omitempty"`
}

type InventoryData struct {
	Hosts        []string            `json:"hosts,omitempty"`
	DC1          []string            `json:"dc1,omitempty"`
	DC2          []string            `json:"dc2,omitempty"`
	DC3          []string            `json:"dc3,omitempty"`
	SSH          *SSHConfig          `json:"ssh,omitempty"`
	Brokers      []string            `json:"brokers,omitempty"`
	Disks        *DisksConfig        `json:"disks,omitempty"`
	Distribution *DistributionConfig `json:"distribution,omitempty"`
	YDBConfig    *YDBConfig          `json:"ydb_config,omitempty"`
	Database     *DatabaseConfig     `json:"database,omitempty"`
	TLS          *TLSConfig          `json:"tls,omitempty"`
	DynamicNodes []DynamicNode       `json:"dynamic_nodes,omitempty"`
	System       *SystemConfig       `json:"system,omitempty"`
	Operations   *OperationsConfig   `json:"operations,omitempty"`
	Topology     *TopologyConfig     `json:"topology,omitempty"`
}

func (d *InventoryData) AllHosts() []string {
	if len(d.DC1) > 0 || len(d.DC2) > 0 || len(d.DC3) > 0 {
		seen := make(map[string]bool)
		var result []string
		for _, h := range d.DC1 {
			if h != "" && !seen[h] {
				seen[h] = true
				result = append(result, h)
			}
		}
		for _, h := range d.DC2 {
			if h != "" && !seen[h] {
				seen[h] = true
				result = append(result, h)
			}
		}
		for _, h := range d.DC3 {
			if h != "" && !seen[h] {
				seen[h] = true
				result = append(result, h)
			}
		}
		return result
	}
	return d.Hosts
}

func (d *InventoryData) SyncHostsFromDC() {
	d.Hosts = d.AllHosts()
}

type SSHConfig struct {
	Login         string `json:"login"`
	AuthType      string `json:"auth_type,omitempty"`
	KeyPath       string `json:"key_path,omitempty"`
	Password      string `json:"-"` // never serialized to JSON
	UseJumpHost   bool   `json:"use_jump_host"`
	JumpHostFQDN  string `json:"jump_host_fqdn,omitempty"`
	JumpHostUser  string `json:"jump_host_user,omitempty"`
	CustomSSHArgs string `json:"custom_ssh_args,omitempty"`
}

type DiskEntry struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

type DisksConfig struct {
	AllowFormat   bool        `json:"allow_format"`
	SelectedDisks []DiskEntry `json:"selected_disks,omitempty"`
}

type DistributionConfig struct {
	Source      string `json:"source"`
	Version     string `json:"version,omitempty"`
	ArchivePath string `json:"archive_path,omitempty"`
	BinaryPath  string `json:"binary_path,omitempty"`
	CLIBinary   string `json:"cli_binary,omitempty"`
	OpsBinary   string `json:"ops_binary,omitempty"`
	ToolBinary  string `json:"tool_binary,omitempty"`
}

type YDBConfig struct {
	Domain                      string `json:"domain"`
	Password                    string `json:"password,omitempty"`
	User                        string `json:"user,omitempty"`
	CoresStatic                 int    `json:"cores_static"`
	CoresDynamic                int    `json:"cores_dynamic"`
	EnforceUserTokenRequirement bool   `json:"enforce_user_token_requirement"`
	RequestClientCertificate    bool   `json:"request_client_certificate"`
	PoolKind                    string `json:"pool_kind,omitempty"`
}

type DatabaseEntry struct {
	Name     string   `json:"name"`
	Groups   int      `json:"groups"`
	Hosts    []string `json:"hosts,omitempty"`
	Affinity string   `json:"affinity,omitempty"`
}

type DatabaseConfig struct {
	ConfigVariant string          `json:"config_variant"`
	Entries       []DatabaseEntry `json:"entries,omitempty"`
	CoresDynamic  int             `json:"cores_dynamic"`
	Hosts         []string        `json:"hosts,omitempty"`
}

type TLSConfig struct {
	CertPath string `json:"cert_path,omitempty"`
}

type DynamicNode struct {
	Instance string `json:"instance"`
	Offset   int    `json:"offset"`
	Dbname   string `json:"dbname,omitempty"`
}

type SystemConfig struct {
	Timezone   string   `json:"timezone,omitempty"`
	NTPServers []string `json:"ntp_servers,omitempty"`
}

type OperationsConfig struct {
	LocalControl bool `json:"local_control"`
}

type TopologyConfig struct {
	Type string `json:"type"`
}

type EMHost struct {
	FQDN     string `json:"fqdn"`
	Location string `json:"location,omitempty"`
}

// EMConfig is written to 90-inventory.yaml separately from 50-inventory.yaml.
// SSH and ydb_user/ydb_password are inherited from 50-inventory.yaml and not
// duplicated here.
type EMConfig struct {
	Hosts []EMHost `json:"hosts,omitempty"`

	ClusterName string `json:"cluster_name,omitempty"`

	DbConnectionDBRoot string `json:"db_connection_db_root,omitempty"`
	DbConnectionDB     string `json:"db_connection_db,omitempty"`
	// DbConnectionEndpoint is grpcs://host:2135. When empty, writer derives it
	// from the first EM host (ydb_em_frontend).
	DbConnectionEndpoint string `json:"db_connection_endpoint,omitempty"`

	ArchivePath   string `json:"archive_path,omitempty"`
	MetaUIArchive string `json:"meta_ui_archive,omitempty"`

	MetaUIBinary  string `json:"meta_ui_binary,omitempty"`
	CPBinary      string `json:"cp_binary,omitempty"`
	CPAgentBinary string `json:"cp_agent_binary,omitempty"`

	AgentCPU    int `json:"agent_cpu,omitempty"`
	AgentMemory int `json:"agent_memory,omitempty"`

	AZLocations []string `json:"az_locations,omitempty"`

	// NOTE: ydb_em_cp_presets is deliberately absent from this model. It is a
	// file-level default injected via config.EmAlwaysPresentKeys so that user
	// customisations in 90-inventory.yaml are never overwritten.

	UseDynamicConfig bool `json:"use_dynamic_config,omitempty"`

	// EmTLSCertPath is ydb_em_tls_dir — certs-per-host directory used by EM.
	// When empty, playbooks may fall back to the shared YDB TLS dir.
	EmTLSCertPath string `json:"em_tls_cert_path,omitempty"`
}
