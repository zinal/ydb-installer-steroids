import type { Session, SessionUpdate } from '@/types/session'
import type { InstallState, InstructionResponse } from '@/types/install'
import type { InventoryResponse, InventoryData, InventoryUpdateResponse, SSHConfig } from '@/types/inventory'
import type { CheckHostsResponse } from '@/types/hosts'
import type { GetDisksResponse } from '@/types/disks'

export interface YdbVersion {
  version: string
  recommended: boolean
}

export interface YdbVersionsResponse {
  versions: YdbVersion[]
}

export interface ValidateFileResponse {
  valid: boolean
  size?: string
  error?: string
}

export interface ConnectivityResponse {
  online: boolean
  checked_at: string
  error?: string
}

async function request<T>(input: string, init?: RequestInit): Promise<T> {
  const r = await fetch(input, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  })
  if (!r.ok) {
    const text = await r.text().catch(() => '')
    throw new Error(`HTTP ${r.status}: ${text || r.statusText}`)
  }
  return (await r.json()) as T
}

export const api = {
  getSession: () => request<Session>('/api/session'),
  updateSession: (u: SessionUpdate) =>
    request<Session>('/api/session', { method: 'PUT', body: JSON.stringify(u) }),

  getDeps: () => request<{ dependencies: Array<{ name: string; ok: boolean; version?: string; required?: string; installed: boolean; source?: string }>; bundled?: boolean }>('/api/system/deps'),
  getDepsInstruction: (dep: string, os: string) =>
    request<InstructionResponse>(`/api/system/deps/instruction/${os}?dep=${encodeURIComponent(dep)}`),
  getConfigs: () => request<{ inventory: { found: boolean; path: string }; config: { found: boolean; path: string } }>('/api/system/configs'),
  getRuntimeInfo: () => request<RuntimeInfo>('/api/system/runtime'),
  getConnectivity: () => request<ConnectivityResponse>('/api/system/connectivity'),

  getInventory: () => request<InventoryResponse>('/api/inventory'),
  updateInventory: (data: InventoryData) =>
    request<InventoryUpdateResponse>('/api/inventory', { method: 'PUT', body: JSON.stringify(data) }),

  checkHosts: (hosts: string[], ssh: any, ports?: number[]) =>
    request<CheckHostsResponse>('/api/hosts/check', { method: 'POST', body: JSON.stringify({ hosts, ssh, ports }) }),

  checkDisks: (hosts: string[], ssh: any) =>
    request<GetDisksResponse>('/api/disks/check', { method: 'POST', body: JSON.stringify({ hosts, ssh }) }),
  validateDisks: (hosts: string[], ssh: any, paths: string[]) =>
    request<import('@/types/disks').ValidateDisksResponse>('/api/disks/validate', {
      method: 'POST',
      body: JSON.stringify({ hosts, ssh, paths }),
    }),

  deleteYdbStart: () =>
    request<{ status: string }>('/api/install/delete-ydb/start', { method: 'POST' }),
  /** Optional op overrides the session flow. */
  getInstallStatus: (op?: string) =>
    request<InstallState>(op ? `/api/install/status?op=${encodeURIComponent(op)}` : '/api/install/status'),
  cancelInstall: (op?: string) =>
    request<{ status: string }>(op ? `/api/install/cancel?op=${encodeURIComponent(op)}` : '/api/install/cancel', { method: 'POST' }),
  getInstallLogs: (op?: string) =>
    request<{ logs: string[]; source?: string; count?: number }>(
      op ? `/api/install/logs?op=${encodeURIComponent(op)}` : '/api/install/logs',
    ),

  getConfigStatus: () =>
    request<ConfigStatusResponse>('/api/install/ydb/config-status'),
  generateConfig: (topology?: string, skipBackup?: boolean) =>
    request<GenerateConfigResponse>('/api/install/ydb/generate-config', {
      method: 'POST',
      body: JSON.stringify({ ...(topology ? { topology } : {}), ...(skipBackup ? { skip_backup: true } : {}) }),
    }),
  startYdbInstall: () =>
    request<{ status: string }>('/api/install/ydb/start', { method: 'POST' }),
  createDatabase: (db: { name: string; groups: number; hosts?: string[]; affinity?: string }) =>
    request<{ status: string }>('/api/install/ydb/create-database', {
      method: 'POST',
      body: JSON.stringify(db),
    }),

  getVersions: () => request<YdbVersionsResponse>('/api/versions'),
  validateFile: (path: string) =>
    request<ValidateFileResponse>('/api/versions/validate-file', {
      method: 'POST',
      body: JSON.stringify({ path }),
    }),

  tlsGenerate: (params: { hosts: string[]; days?: number; force?: boolean; output_dir?: string }) =>
    request<TLSGenerateResponse>('/api/tls/generate', {
      method: 'POST',
      body: JSON.stringify(params),
    }),
  tlsValidate: (params: { cert_path: string; hosts: string[] }) =>
    request<TLSValidateResponse>('/api/tls/validate', {
      method: 'POST',
      body: JSON.stringify(params),
    }),
  tlsExists: (params: { hosts: string[]; cert_path?: string }) =>
    request<TLSExistsResponse>('/api/tls/exists', {
      method: 'POST',
      body: JSON.stringify(params),
    }),

}

export interface TLSNodeFileInfo {
  hostname: string
  short_name: string
  cert_file: string
  key_file: string
  web_pem: string
}

export interface TLSNodeValidation {
  index: number
  hostname: string
  short_name: string
  key_found: boolean
  cert_found: boolean
  cert_valid: boolean
  key_valid: boolean
  pair_match: boolean
  match: boolean
  error?: string
}

export interface TLSGenerateResponse {
  success: boolean
  output_dir?: string
  ca_file?: string
  node_files?: TLSNodeFileInfo[]
  error?: string
  error_code?: string
  existing?: string[]
}

export interface TLSExistsResponse {
  exists: boolean
  path?: string
  reasons?: string[]
  ca_present?: boolean
  host_conflicts?: string[]
  block_generate?: boolean
}

export interface TLSValidateResponse {
  valid: boolean
  ca_valid: boolean
  node_files?: TLSNodeValidation[]
  missing?: string[]
  errors?: string[]
  total_nodes: number
  found_nodes: number
}

/** Three-way match status returned by GET /api/install/ydb/config-status. */
export type ConfigMatchStatus = 'missing' | 'matches' | 'outdated'

export interface ConfigStatusResponse {
  success: boolean
  /** Dry-run comparison result: missing | matches | outdated. */
  status?: ConfigMatchStatus
  path?: string
  content?: string
  error?: string
}

export interface GenerateConfigResponse {
  success: boolean
  path?: string
  content?: string
  error?: string
}

/**
 * Creates a WebSocket connection to the log streaming endpoint.
 * Returns the native WebSocket instance so the caller can attach handlers.
 *
 * Usage:
 *   const ws = createLogsWebSocket()
 *   ws.onmessage = (e) => logs.value.push(e.data)
 *   ws.onclose = () => { ... }
 */
export function createLogsWebSocket(op?: string): WebSocket {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const qs = op ? `?op=${encodeURIComponent(op)}` : ''
  const url = `${protocol}//${window.location.host}/ws/logs${qs}`
  return new WebSocket(url)
}


export interface VaultUpdateResponse {
  status: string
  error?: string
}

export interface RuntimeInfo {
  /** True when the binary ships a bundled Python+Ansible runtime. */
  embedded: boolean
  /** True when the bundled runtime has been extracted and is ready on disk. */
  extracted: boolean
  python?: string
  ansible?: string
  ansible_core?: string
  collection?: string
  path?: string
  error?: string
}
