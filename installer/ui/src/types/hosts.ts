export interface PortCheckResult {
  port: number
  available: boolean
  error?: string
}

export interface TimeSyncResult {
  ntpActive: boolean
  ntpDaemon: string
  ntpSynced: boolean
  clockOffsetMs: number
  error?: string
}

export interface TimeSyncSummary {
  maxDriftMs: number
  status: 'ok' | 'warning' | 'error'
  ntpNotSynced: boolean
  threshold: { warningMs: number; errorMs: number }
}

export interface CheckHostResult {
  host: string
  ssh: boolean
  sshError?: string
  python: boolean
  pythonVersion?: string
  pythonError?: string
  cpuCores: number
  ports: boolean
  portsError?: string
  portDetails?: PortCheckResult[]
  timeSync: TimeSyncResult
  clockEpochNs: number
}

export interface CheckHostsResponse {
  results: CheckHostResult[]
  timeSync?: TimeSyncSummary
}
