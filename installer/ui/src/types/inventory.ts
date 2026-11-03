export interface SSHConfig {
  login: string
  auth_type?: 'key' | 'password'
  key_path?: string
  use_jump_host: boolean
  jump_host_fqdn?: string
  jump_host_user?: string
  custom_ssh_args?: string
}

export interface DiskEntry {
  path: string
  label: string
}

export interface DisksConfig {
  allow_format: boolean
  selected_disks: DiskEntry[]
}

export interface DistributionConfig {
  source: string
  version?: string
  archive_path?: string
  binary_path?: string
  cli_binary?: string
  ops_binary?: string
  tool_binary?: string
}

export interface YDBConfig {
  domain: string
  password?: string
  user?: string
  cores_static: number
  cores_dynamic: number
  enforce_user_token_requirement: boolean
  request_client_certificate: boolean
  pool_kind?: string
}

export interface DatabaseEntry {
  name: string
  groups: number
  hosts?: string[]
  affinity?: string
}

export interface DatabaseConfig {
  config_variant: string
  entries: DatabaseEntry[]
  cores_dynamic: number
}

export interface TLSConfig {
  cert_path?: string
}

export interface DynamicNode {
  instance: string
  offset: number
}

export interface SystemConfig {
  timezone?: string
  ntp_servers?: string[]
}

export interface OperationsConfig {
  local_control: boolean
}

/** Full parsed inventory data from the backend. */
export interface InventoryData {
  hosts?: string[]
  ssh?: SSHConfig
  brokers?: string[]
  disks?: DisksConfig
  distribution?: DistributionConfig
  ydb_config?: YDBConfig
  database?: DatabaseConfig
  tls?: TLSConfig
  dynamic_nodes?: DynamicNode[]
  system?: SystemConfig
  operations?: OperationsConfig
  topology?: TopologyConfig
  dc1?: string[]
  dc2?: string[]
  dc3?: string[]
}

export interface TopologyConfig {
  type: string
}

/** Response from GET /api/inventory. */
export interface InventoryResponse {
  inventory_found: boolean
  config_found: boolean
  data: InventoryData | null
}

/** Response from PUT /api/inventory. */
export interface InventoryUpdateResponse {
  status: string
  data: InventoryData | null
}

export interface EMHost {
  fqdn: string
  location?: string
}

export interface EMConfig {
  hosts?: EMHost[]
  cluster_name?: string
  db_connection_db_root?: string
  db_connection_db?: string
  db_connection_endpoint?: string
  /** Local archive mode: main EM archive path (ydb_em_archive). */
  archive_path?: string
  /** Local archive mode: meta UI archive path (ydb_em_meta_ui_archive). */
  meta_ui_archive?: string
  meta_ui_binary?: string
  cp_binary?: string
  cp_agent_binary?: string
  agent_cpu?: number
  agent_memory?: number
  az_locations?: string[]
  use_dynamic_config?: boolean
  /** EM TLS certs-per-host directory (ydb_em_tls_dir). */
  em_tls_cert_path?: string
}

/** Response from GET /api/em-inventory. */
export interface EMInventoryResponse {
  em_inventory_found: boolean
  data: EMConfig | null
}

/** Response from PUT /api/em-inventory. */
export interface EMInventoryUpdateResponse {
  status: string
  data: EMConfig | null
}
