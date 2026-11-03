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
