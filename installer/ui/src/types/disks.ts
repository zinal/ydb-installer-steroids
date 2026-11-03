/** Minimum recommended disk size for YDB: 80 GB in bytes. */
export const MIN_RECOMMENDED_DISK_SIZE_BYTES = 80 * 1024 * 1024 * 1024

/** Mode for entering disks on Screen 4: auto-discovery or manual text input. */
export type DiskEntryMode = 'auto' | 'manual'

export interface DiskInfo {
  name: string
  path: string
  type: string
  size: string
  sizeBytes: number
  mounted: boolean
  mountPoint: string
  filesystem: string
  isSystemDisk: boolean
}

export interface HostDisksResult {
  host: string
  disks: DiskInfo[]
  error?: string
}

export interface GetDisksResponse {
  results: HostDisksResult[]
}

export interface DiskConsistencyIssue {
  type: string
  hosts: string[]
  message: string
}

export interface DiskConsistencyResult {
  isConsistent: boolean
  issues: DiskConsistencyIssue[]
  commonDisks: DiskInfo[]
}

export interface CheckDisksRequest {
  hosts: string[]
  ssh: any
}

export interface DCDisks {
  'DC-1': DiskInfo[]
  'DC-2': DiskInfo[]
  'DC-3': DiskInfo[]
}

export interface DiskSelection {
  host: string
  disk: DiskInfo
  selected: boolean
}

export interface DiskValidation {
  path: string
  found: boolean
}

export interface DiskValidationResult {
  host: string
  valid: boolean
  disks: DiskValidation[]
  error?: string
}

export interface ValidateDisksRequest {
  hosts: string[]
  ssh: any
  paths: string[]
}

export interface ValidateDisksResponse {
  results: DiskValidationResult[]
}
