import { defineStore } from 'pinia'
import type { TLSConfig, InventoryData } from '@/types/inventory'
import { api, type TLSGenerateResponse, type TLSValidateResponse } from '@/api/client'

interface TlsState {
  mode: 'generate' | 'existing'
  certPath: string
  days: number
  generating: boolean
  validating: boolean
  generationResult: GenerationResult | null
  validationResult: ValidationResult | null
  error: string
}

export interface NodeFileInfo {
  hostname: string
  shortName: string
  certFile: string
  keyFile: string
  webPem: string
}

export interface GenerationResult {
  success: boolean
  outputDir: string
  caFile: string
  nodeFiles: NodeFileInfo[]
  error?: string
}

export interface NodeValidation {
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

export interface ValidationResult {
  valid: boolean
  caValid: boolean
  nodeFiles: NodeValidation[]
  missing: string[]
  errors: string[]
  totalNodes: number
  foundNodes: number
}

export const useTlsStore = defineStore('tls', {
  state: (): TlsState => ({
    mode: 'generate',
    certPath: '',
    days: 365,
    generating: false,
    validating: false,
    generationResult: null,
    validationResult: null,
    error: '',
  }),

  getters: {
    canProceed(state): boolean {
      if (state.mode === 'generate') {
        return state.generationResult?.success === true
      }
      return state.validationResult?.valid === true
    },

    isGenerating(state): boolean {
      return state.generating
    },

    isValidating(state): boolean {
      return state.validating
    },

    hasError(state): boolean {
      return !!state.error
    },

    statusMessage(state): string {
      if (state.error) return state.error
      if (state.generating) return 'Generating certificates...'
      if (state.validating) return 'Validating certificates...'
      if (state.mode === 'generate' && state.generationResult?.success) {
        return `Certificates generated in ${state.generationResult.outputDir}`
      }
      if (state.mode === 'existing' && state.validationResult?.valid) {
        return 'Certificates are valid'
      }
      return ''
    },
  },

  actions: {
    applyFromInventory(inv: InventoryData) {
      if (inv.tls && inv.tls.cert_path) {
        if (this.certPath !== inv.tls.cert_path) {
          this.generationResult = null
          this.validationResult = null
        }
        this.certPath = inv.tls.cert_path
        if (this.mode !== 'generate') {
          this.mode = 'existing'
        }
      } else {
        this.certPath = ''
      }
    },

    toTLSConfig(): TLSConfig {
      return {
        cert_path: this.certPath,
      }
    },

    /**
     * Set the certificate mode (generate vs existing).
     *
     * Only the mode-agnostic `error` is cleared — `generationResult` and
     * `validationResult` are deliberately PRESERVED so that switching back to
     * a mode restores its previous result (the user shouldn't have to
     * re-generate or re-validate just because they peeked at the other tab).
     * Each result is rendered only inside its own mode's `v-if`/`v-else`
     * section in TLSScreen.vue, so a hidden result is simply not shown.
     */
    setMode(mode: 'generate' | 'existing') {
      this.mode = mode
      this.error = ''
    },

    /** Set the certificate path for existing certificates.
     *
     * When the path changes, previously cached generation/validation results
     * are invalidated — they belong to a different certificate directory and
     * showing them would be misleading.
     */
    setCertPath(path: string) {
      if (this.certPath !== path) {
        this.generationResult = null
        this.validationResult = null
      }
      this.certPath = path
      this.error = ''
    },

    setDays(days: number) {
      this.days = days
    },

    /** Generate self-signed certificates for the given hosts.
     *
     * When `force` is true, the host-conflict check on the backend is skipped —
     * existing per-host material is overwritten. Used by the ydb flow (Screen 8)
     * where the user is setting up a fresh cluster and needs to regenerate.
     * The em flow (Screen 23) uses the default (false) to protect a running cluster.
     */
    async generateCertificates(hosts: string[], force = false): Promise<GenerationResult> {
      this.generating = true
      this.error = ''
      this.generationResult = null

      try {
        const result: TLSGenerateResponse = await api.tlsGenerate({
          hosts,
          days: this.days,
          force,
        })

        const mapped: GenerationResult = {
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

        this.generationResult = mapped

        if (mapped.success) {
          this.certPath = mapped.outputDir
        } else {
          this.error = mapped.error || 'Failed to generate certificates'
        }

        return mapped
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Failed to generate certificates'
        this.error = msg
        this.generationResult = {
          success: false,
          outputDir: '',
          caFile: '',
          nodeFiles: [],
          error: msg,
        }
        return this.generationResult
      } finally {
        this.generating = false
      }
    },

    async validateCertificates(hosts: string[]): Promise<ValidationResult> {
      if (!this.certPath) {
        this.error = 'Certificate path is required'
        return {
          valid: false,
          caValid: false,
          nodeFiles: [],
          missing: [],
          errors: ['Certificate path is required'],
          totalNodes: hosts.length,
          foundNodes: 0,
        }
      }

      this.validating = true
      this.error = ''
      this.validationResult = null

      try {
        const result: TLSValidateResponse = await api.tlsValidate({
          cert_path: this.certPath,
          hosts,
        })

        const mapped: ValidationResult = {
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

        this.validationResult = mapped

        if (!mapped.valid) {
          const missingMsg = mapped.missing.length > 0
            ? `Missing: ${mapped.missing.join(', ')}`
            : mapped.errors.join('; ')
          this.error = missingMsg || 'Certificates are invalid'
        }

        return mapped
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Failed to validate certificates'
        this.error = msg
        this.validationResult = {
          valid: false,
          caValid: false,
          nodeFiles: [],
          missing: [],
          errors: [msg],
          totalNodes: hosts.length,
          foundNodes: 0,
        }
        return this.validationResult
      } finally {
        this.validating = false
      }
    },

    clearError() {
      this.error = ''
    },

    reset() {
      this.mode = 'generate'
      this.certPath = ''
      this.days = 365
      this.generating = false
      this.validating = false
      this.generationResult = null
      this.validationResult = null
      this.error = ''
    },
  },
})
