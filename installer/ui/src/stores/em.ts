import { defineStore } from 'pinia'
import { api, type TLSGenerateResponse, type TLSValidateResponse } from '@/api/client'
import type { EMConfig, EMHost } from '@/types/inventory'

export const TLS_ERR_CERTS_EXIST = 'certs_already_exist'

export type EmDistMode = 'archive' | 'binaries'
export type EmTlsMode = 'generate' | 'existing'

export interface EmPreflightRow {
  id: string
  labelKey: string
  value: string
  ok: boolean
  severity: 'error' | 'warning'
  hintKey?: string
}

export interface EmTlsNodeInfo {
  hostname: string
  shortName: string
  certFile: string
  keyFile: string
  webPem: string
}

export interface EmTlsGenerationResult {
  success: boolean
  outputDir: string
  caFile: string
  nodeFiles: EmTlsNodeInfo[]
  error?: string
}

export interface EmTlsNodeValidation {
  index: number
  hostname: string
  shortName: string
  keyFound: boolean
  certFound: boolean
  certValid: boolean
  keyValid: boolean
  pairMatch: boolean
  match: boolean
  error?: string
}

export interface EmTlsValidationResult {
  valid: boolean
  caValid: boolean
  nodeFiles: EmTlsNodeValidation[]
  missing: string[]
  errors: string[]
  totalNodes: number
  foundNodes: number
}

interface EmState {
  hosts: EMHost[]
  clusterName: string
  dbConnectionDBRoot: string
  dbConnectionDB: string
  dbConnectionEndpoint: string
  distMode: EmDistMode
  archivePath: string
  metaUIArchivePath: string
  metaUIBinary: string
  cpBinary: string
  cpAgentBinary: string
  archiveValid: boolean | null
  archiveSize: string
  metaUIArchiveValid: boolean | null
  metaUIArchiveSize: string
  metaUIBinaryValid: boolean | null
  metaUIBinarySize: string
  cpBinaryValid: boolean | null
  cpBinarySize: string
  cpAgentBinaryValid: boolean | null
  cpAgentBinarySize: string
  agentCPU: number
  agentMemory: number
  useDynamicConfig: boolean
  emInventoryFound: boolean
  tlsMode: EmTlsMode
  tlsCertPath: string
  tlsDays: number
  tlsGenerating: boolean
  tlsValidating: boolean
  tlsGenerationResult: EmTlsGenerationResult | null
  tlsValidationResult: EmTlsValidationResult | null
  tlsError: string
  tlsCertsExist: boolean | null
  tlsExistPath: string
  tlsExistReasons: string[]
  tlsExistChecking: boolean
  tlsCaPresent: boolean
  tlsPathAutoPromoted: boolean
  tlsPathUserCleared: boolean
}

function isJinjaPath(v: string | undefined | null): boolean {
  return !!v && v.includes('{{')
}

function cleanPath(v: string | undefined | null): string {
  if (!v || isJinjaPath(v)) return ''
  return v
}

export const useEmStore = defineStore('em', {
  state: (): EmState => ({
    hosts: [],
    clusterName: '',
    dbConnectionDBRoot: '',
    dbConnectionDB: '',
    dbConnectionEndpoint: '',
    distMode: 'archive',
    archivePath: '',
    metaUIArchivePath: '',
    metaUIBinary: '',
    cpBinary: '',
    cpAgentBinary: '',
    archiveValid: null,
    archiveSize: '',
    metaUIArchiveValid: null,
    metaUIArchiveSize: '',
    metaUIBinaryValid: null,
    metaUIBinarySize: '',
    cpBinaryValid: null,
    cpBinarySize: '',
    cpAgentBinaryValid: null,
    cpAgentBinarySize: '',
    agentCPU: 0,
    agentMemory: 0,
    useDynamicConfig: false,
    emInventoryFound: false,
    tlsMode: 'generate',
    tlsCertPath: '',
    tlsDays: 365,
    tlsGenerating: false,
    tlsValidating: false,
    tlsGenerationResult: null,
    tlsValidationResult: null,
    tlsError: '',
    tlsCertsExist: null,
    tlsExistPath: '',
    tlsExistReasons: [],
    tlsExistChecking: false,
    tlsCaPresent: false,
    tlsPathAutoPromoted: false,
    tlsPathUserCleared: false,
  }),

  getters: {
    total(state): number {
      return state.hosts.length
    },
    hasDuplicates(state): string[] {
      const seen = new Map<string, number>()
      for (const h of state.hosts) {
        const fqdn = h.fqdn.trim()
        if (!fqdn) continue
        seen.set(fqdn, (seen.get(fqdn) ?? 0) + 1)
      }
      return Array.from(seen.entries())
        .filter(([_, count]) => count > 1)
        .map(([fqdn]) => fqdn)
    },
    firstEmHost(state): string {
      for (const h of state.hosts) {
        const fqdn = h.fqdn.trim()
        if (fqdn) return fqdn
      }
      return ''
    },
    derivedEndpoint(): string {
      const host = this.firstEmHost
      return host ? `grpcs://${host}:2135` : ''
    },
    databasePath(state): string {
      const db = state.dbConnectionDB.trim()
      const root = state.dbConnectionDBRoot.trim()
      if (!db) return root
      if (db.startsWith('/')) return db
      const r = root.replace(/\/$/, '')
      return r ? `${r}/${db}` : `/${db}`
    },
    paramsFieldsValid(state): boolean {
      if (!state.dbConnectionDBRoot.trim()) return false
      if (!state.dbConnectionDB.trim()) return false
      return true
    },
    distCanProceed(state): boolean {
      if (state.distMode === 'archive') {
        return state.archiveValid === true && state.metaUIArchiveValid === true
      }
      return (
        state.metaUIBinaryValid === true
        && state.cpBinaryValid === true
        && state.cpAgentBinaryValid === true
      )
    },
    emHostsList(state): string[] {
      return state.hosts.map(h => h.fqdn.trim()).filter(Boolean)
    },
    tlsCanProceed(state): boolean {
      if (state.tlsMode === 'generate') {
        return state.tlsGenerationResult?.success === true
      }
      return state.tlsValidationResult?.valid === true
    },
    tlsCanGenerate(state): boolean {
      const hosts = state.hosts.map(h => h.fqdn.trim()).filter(Boolean)
      return hosts.length > 0
        && !state.tlsGenerating
        && state.tlsCertsExist !== true
        && !state.tlsExistChecking
    },
    tlsCaReuse(state): boolean {
      return state.tlsCaPresent
        && state.tlsCertsExist !== true
        && state.tlsMode === 'generate'
    },

    /**
     * Screen 24 pre-flight summary, derived entirely from 90-inventory.yaml
     * state that screens 20–23 already collected.
     *
     * This is an inventory-level check only: it confirms the configuration is
     * complete, not that the target cluster is healthy. A live dynamic-config
     * check against the cluster is a separate, future concern — the row shape
     * is deliberately generic so such a row can be appended without
     * restructuring the card.
     */
    emPreflightRows(state): EmPreflightRow[] {
      const rows: EmPreflightRow[] = []

      const hosts = state.hosts.map(h => h.fqdn.trim()).filter(Boolean)
      rows.push({
        id: 'hosts',
        labelKey: 'emInstall.preflight.hosts',
        value: hosts.join(', '),
        ok: hosts.length > 0,
        severity: 'error',
        hintKey: 'emInstall.preflight.hostsHint',
      })

      const endpoint = state.dbConnectionEndpoint.trim() || this.derivedEndpoint
      rows.push({
        id: 'endpoint',
        labelKey: 'emInstall.preflight.endpoint',
        value: endpoint,
        ok: !!endpoint,
        severity: 'error',
        hintKey: 'emInstall.preflight.endpointHint',
      })

      rows.push({
        id: 'database',
        labelKey: 'emInstall.preflight.database',
        value: this.databasePath,
        ok: this.paramsFieldsValid,
        severity: 'error',
        hintKey: 'emInstall.preflight.databaseHint',
      })

      const distValue = state.distMode === 'archive'
        ? [state.archivePath, state.metaUIArchivePath].filter(p => p.trim()).join(', ')
        : [state.metaUIBinary, state.cpBinary, state.cpAgentBinary].filter(p => p.trim()).join(', ')
      rows.push({
        id: 'distribution',
        labelKey: state.distMode === 'archive'
          ? 'emInstall.preflight.distArchive'
          : 'emInstall.preflight.distBinaries',
        value: distValue,
        ok: this.distCanProceed,
        severity: 'error',
        hintKey: 'emInstall.preflight.distHint',
      })

      rows.push({
        id: 'tls',
        labelKey: 'emInstall.preflight.tls',
        value: state.tlsCertPath.trim(),
        ok: !!state.tlsCertPath.trim(),
        severity: 'error',
        hintKey: 'emInstall.preflight.tlsHint',
      })

      rows.push({
        id: 'dynamicConfig',
        labelKey: 'emInstall.preflight.dynamicConfig',
        value: '',
        ok: state.useDynamicConfig,
        severity: 'warning',
        hintKey: 'emInstall.preflight.dynamicConfigHint',
      })

      return rows
    },

    emCanInstall(): boolean {
      return this.emPreflightRows
        .filter((r: EmPreflightRow) => r.severity === 'error')
        .every((r: EmPreflightRow) => r.ok)
    },
  },

  actions: {
    applyFromEMConfig(config: EMConfig) {
      this.hosts = config.hosts ?? []
      this.clusterName = config.cluster_name ?? ''
      this.dbConnectionDBRoot = config.db_connection_db_root ?? ''
      this.dbConnectionDB = config.db_connection_db ?? ''
      this.dbConnectionEndpoint = config.db_connection_endpoint ?? ''

      const archivePath = cleanPath(config.archive_path)
      const metaUIArchive = cleanPath(config.meta_ui_archive)
      const metaUIBinary = cleanPath(config.meta_ui_binary)
      const cpBinary = cleanPath(config.cp_binary)
      const cpAgentBinary = cleanPath(config.cp_agent_binary)

      const hasArchive = !!(archivePath || metaUIArchive)
      const hasBinary = !!(metaUIBinary || cpBinary || cpAgentBinary)

      // Mode-based merge: only overwrite active-mode path fields so a
      // round-trip (XOR inventory) does not wipe the inactive mode's UI values.
      if (hasArchive) {
        this.distMode = 'archive'
        this.archivePath = archivePath
        this.metaUIArchivePath = metaUIArchive
      } else if (hasBinary) {
        this.distMode = 'binaries'
        this.metaUIBinary = metaUIBinary
        this.cpBinary = cpBinary
        this.cpAgentBinary = cpAgentBinary
      }
      this.agentCPU = config.agent_cpu ?? 0
      this.agentMemory = config.agent_memory ?? 0
      this.useDynamicConfig = config.use_dynamic_config ?? false

      const tlsPath = config.em_tls_cert_path?.trim() ?? ''
      if (tlsPath && this.tlsCertPath !== tlsPath) {
        this.tlsGenerationResult = null
        this.tlsValidationResult = null
      }
      if (tlsPath) {
        this.tlsCertPath = tlsPath
        if (this.tlsMode !== 'generate') {
          this.tlsMode = 'existing'
        }
      }
    },

    toEMConfig(): EMConfig {
      const endpoint = this.dbConnectionEndpoint.trim() || this.derivedEndpoint
      const base: EMConfig = {
        hosts: this.hosts.filter(h => h.fqdn.trim() !== ''),
        cluster_name: this.clusterName,
        db_connection_db_root: this.dbConnectionDBRoot,
        db_connection_db: this.dbConnectionDB,
        db_connection_endpoint: endpoint || undefined,
        agent_cpu: this.agentCPU,
        agent_memory: this.agentMemory,
        use_dynamic_config: this.useDynamicConfig,
      }

      if (this.distMode === 'archive') {
        if (this.archivePath.trim()) base.archive_path = this.archivePath.trim()
        if (this.metaUIArchivePath.trim()) base.meta_ui_archive = this.metaUIArchivePath.trim()
      } else {
        if (this.metaUIBinary.trim()) base.meta_ui_binary = this.metaUIBinary.trim()
        if (this.cpBinary.trim()) base.cp_binary = this.cpBinary.trim()
        if (this.cpAgentBinary.trim()) base.cp_agent_binary = this.cpAgentBinary.trim()
      }

      if (this.tlsCertPath.trim()) {
        base.em_tls_cert_path = this.tlsCertPath.trim()
      }

      return base
    },

    ensureParamsDefaults() {
      if (!this.dbConnectionDBRoot.trim()) {
        this.dbConnectionDBRoot = '/Root'
      }
      if (!this.dbConnectionDB.trim()) {
        this.dbConnectionDB = 'db'
      }
    },

    setDbConnectionDBRoot(v: string) {
      this.dbConnectionDBRoot = v
    },

    setDbConnectionDB(v: string) {
      this.dbConnectionDB = v
    },

    setDbConnectionEndpoint(v: string) {
      this.dbConnectionEndpoint = v
    },

    setDistMode(mode: EmDistMode) {
      this.distMode = mode
    },

    async validatePath(
      field: 'archive' | 'metaUIArchive' | 'metaUIBinary' | 'cpBinary' | 'cpAgentBinary',
      path: string,
    ) {
      const trimmed = path.trim()
      const setResult = (valid: boolean | null, size: string) => {
        switch (field) {
          case 'archive':
            this.archiveValid = valid
            this.archiveSize = size
            break
          case 'metaUIArchive':
            this.metaUIArchiveValid = valid
            this.metaUIArchiveSize = size
            break
          case 'metaUIBinary':
            this.metaUIBinaryValid = valid
            this.metaUIBinarySize = size
            break
          case 'cpBinary':
            this.cpBinaryValid = valid
            this.cpBinarySize = size
            break
          case 'cpAgentBinary':
            this.cpAgentBinaryValid = valid
            this.cpAgentBinarySize = size
            break
        }
      }

      if (!trimmed) {
        setResult(null, '')
        return
      }
      try {
        const resp = await api.validateFile(trimmed)
        setResult(resp.valid, resp.size || '')
      } catch {
        setResult(false, '')
      }
    },

    addHost(fqdn: string, location?: string) {
      const trimmed = fqdn.trim()
      if (trimmed && !this.hosts.some(h => h.fqdn === trimmed)) {
        this.hosts.push({ fqdn: trimmed, location: location?.trim() || undefined })
      }
    },

    addEmptyHost() {
      this.hosts.push({ fqdn: '' })
    },

    updateHost(index: number, fqdn: string, location?: string) {
      if (index >= 0 && index < this.hosts.length) {
        const trimmed = fqdn.trim()
        if (trimmed === '' || !this.hosts.some((h, i) => i !== index && h.fqdn === trimmed)) {
          this.hosts[index] = {
            fqdn: trimmed,
            location: location !== undefined
              ? (location.trim() || undefined)
              : this.hosts[index].location,
          }
        }
      }
    },

    removeHost(index: number) {
      if (index >= 0 && index < this.hosts.length) {
        this.hosts.splice(index, 1)
      }
    },

    setEMInventoryFound(found: boolean) {
      this.emInventoryFound = found
    },

    setTlsMode(mode: EmTlsMode) {
      this.tlsMode = mode
      this.tlsError = ''
      if (mode === 'existing' && !this.tlsCertPath.trim() && this.tlsExistPath) {
        this.tlsCertPath = this.tlsExistPath
      }
    },

    setTlsCertPath(path: string) {
      if (this.tlsCertPath !== path) {
        this.tlsGenerationResult = null
        this.tlsValidationResult = null
        this.tlsPathAutoPromoted = false
      }
      this.tlsCertPath = path
      this.tlsError = ''
      this.tlsPathUserCleared = path.trim() === ''
    },

    switchToExistingWithDiscoveredPath() {
      this.tlsMode = 'existing'
      this.tlsError = ''
      if (!this.tlsCertPath.trim() && this.tlsExistPath) {
        this.tlsCertPath = this.tlsExistPath
        this.tlsPathUserCleared = false
      }
    },

    setTlsDays(days: number) {
      if (days > 0) this.tlsDays = days
    },

    async checkEmTlsExists() {
      const hosts = this.emHostsList
      this.tlsExistChecking = true
      try {
        const resp = await api.tlsExists({
          hosts,
          cert_path: this.tlsCertPath.trim() || undefined,
        })
        this.tlsCertsExist = resp.block_generate ?? resp.exists
        this.tlsExistPath = resp.path || ''
        this.tlsExistReasons = resp.reasons || []
        this.tlsCaPresent = resp.ca_present ?? false
        if (resp.path && !this.tlsCertPath.trim() && !this.tlsPathUserCleared) {
          this.tlsCertPath = resp.path
          this.tlsPathAutoPromoted = true
        }
      } catch {
        this.tlsCertsExist = null
        this.tlsExistPath = ''
        this.tlsExistReasons = []
        this.tlsCaPresent = false
      } finally {
        this.tlsExistChecking = false
      }
    },

    async generateEmTls(): Promise<EmTlsGenerationResult> {
      const hosts = this.emHostsList
      this.tlsGenerating = true
      this.tlsError = ''
      this.tlsGenerationResult = null

      try {
        if (hosts.length === 0) {
          const empty: EmTlsGenerationResult = {
            success: false,
            outputDir: '',
            caFile: '',
            nodeFiles: [],
            error: 'No EM hosts specified',
          }
          this.tlsGenerationResult = empty
          this.tlsError = empty.error || ''
          return empty
        }

        const result: TLSGenerateResponse = await api.tlsGenerate({
          hosts,
          days: this.tlsDays,
        })

        const mapped: EmTlsGenerationResult = {
          success: result.success,
          outputDir: result.output_dir || '',
          caFile: result.ca_file || '',
          nodeFiles: (result.node_files || []).map(nf => ({
            hostname: nf.hostname,
            shortName: nf.short_name,
            certFile: nf.cert_file,
            keyFile: nf.key_file,
            webPem: nf.web_pem,
          })),
          error: result.error,
        }

        this.tlsGenerationResult = mapped
        if (mapped.success) {
          this.tlsCertPath = mapped.outputDir
          this.tlsCertsExist = true
          this.tlsExistPath = mapped.outputDir
        } else if (result.error_code === TLS_ERR_CERTS_EXIST) {
          this.tlsCertsExist = true
          this.tlsExistPath = result.output_dir || this.tlsExistPath
          this.tlsExistReasons = result.existing || []
          this.tlsError = mapped.error || 'Certificates already exist'
        } else {
          this.tlsError = mapped.error || 'Failed to generate certificates'
        }
        return mapped
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Failed to generate certificates'
        this.tlsError = msg
        this.tlsGenerationResult = {
          success: false,
          outputDir: '',
          caFile: '',
          nodeFiles: [],
          error: msg,
        }
        return this.tlsGenerationResult
      } finally {
        this.tlsGenerating = false
      }
    },

    async validateEmTls(): Promise<EmTlsValidationResult> {
      const hosts = this.emHostsList
      if (!this.tlsCertPath.trim()) {
        this.tlsError = 'Certificate path is required'
        const empty: EmTlsValidationResult = {
          valid: false,
          caValid: false,
          nodeFiles: [],
          missing: [],
          errors: ['Certificate path is required'],
          totalNodes: hosts.length,
          foundNodes: 0,
        }
        this.tlsValidationResult = empty
        return empty
      }

      this.tlsValidating = true
      this.tlsError = ''
      this.tlsValidationResult = null

      try {
        const result: TLSValidateResponse = await api.tlsValidate({
          cert_path: this.tlsCertPath.trim(),
          hosts,
        })

        const mapped: EmTlsValidationResult = {
          valid: result.valid,
          caValid: result.ca_valid,
          nodeFiles: (result.node_files || []).map(nv => ({
            index: nv.index,
            hostname: nv.hostname,
            shortName: nv.short_name,
            keyFound: nv.key_found,
            certFound: nv.cert_found,
            certValid: nv.cert_valid,
            keyValid: nv.key_valid,
            pairMatch: nv.pair_match,
            match: nv.match,
            error: nv.error,
          })),
          missing: result.missing || [],
          errors: result.errors || [],
          totalNodes: result.total_nodes,
          foundNodes: result.found_nodes,
        }

        this.tlsValidationResult = mapped
        if (!mapped.valid) {
          const missingMsg = mapped.missing.length > 0
            ? `Missing: ${mapped.missing.join(', ')}`
            : mapped.errors.join('; ')
          this.tlsError = missingMsg || 'Certificates are invalid'
        }
        return mapped
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Failed to validate certificates'
        this.tlsError = msg
        this.tlsValidationResult = {
          valid: false,
          caValid: false,
          nodeFiles: [],
          missing: [],
          errors: [msg],
          totalNodes: hosts.length,
          foundNodes: 0,
        }
        return this.tlsValidationResult
      } finally {
        this.tlsValidating = false
      }
    },

    resetEmTls() {
      this.tlsMode = 'generate'
      this.tlsCertPath = ''
      this.tlsDays = 365
      this.tlsGenerating = false
      this.tlsValidating = false
      this.tlsGenerationResult = null
      this.tlsValidationResult = null
      this.tlsError = ''
      this.tlsCertsExist = null
      this.tlsExistPath = ''
      this.tlsExistReasons = []
      this.tlsExistChecking = false
      this.tlsCaPresent = false
      this.tlsPathAutoPromoted = false
      this.tlsPathUserCleared = false
    },
  },
})
