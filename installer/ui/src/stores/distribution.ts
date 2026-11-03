import { defineStore } from 'pinia'
import { api, type YdbVersion } from '@/api/client'
import type { DistributionConfig, InventoryData } from '@/types/inventory'

interface DistributionState {
  source: string
  version: string
  archivePath: string
  binaryPath: string
  cliBinary: string
  opsBinary: string
  dstoolBinary: string
  versions: YdbVersion[]
  loadingVersions: boolean
  archiveValid: boolean | null
  archiveSize: string
}

export const useDistributionStore = defineStore('distribution', {
  state: (): DistributionState => ({
    source: 'official',
    version: '',
    archivePath: '',
    binaryPath: '',
    cliBinary: '',
    opsBinary: '',
    dstoolBinary: '',
    versions: [],
    loadingVersions: false,
    archiveValid: null,
    archiveSize: '',
  }),

  getters: {
    canProceed: (state) => {
      if (state.source === 'official') {
        return state.version !== ''
      }
      return state.archiveValid === true
    },
  },

  actions: {
    applyFromInventory(inv: InventoryData) {
      if (inv.distribution) {
        const d = inv.distribution
        this.source = d.source || 'official'
        this.version = d.version || ''
        this.archivePath = d.archive_path || ''
        this.binaryPath = d.binary_path || ''
        this.cliBinary = d.cli_binary || ''
        this.opsBinary = d.ops_binary || ''
        this.dstoolBinary = d.tool_binary || ''
      }
    },

    async fetchVersions() {
      this.loadingVersions = true
      try {
        const resp = await api.getVersions()
        this.versions = resp.versions || []
        if (!this.version && this.versions.length > 0) {
          const recommended = this.versions.find((v) => v.recommended)
          if (recommended) {
            this.version = recommended.version
          } else {
            this.version = this.versions[this.versions.length - 1].version
          }
        }
      } catch (e) {
        console.error('Failed to fetch YDB versions:', e)
      } finally {
        this.loadingVersions = false
      }
    },

    async validateArchivePath(path: string) {
      if (!path) {
        this.archiveValid = false
        this.archiveSize = ''
        return
      }
      try {
        const resp = await api.validateFile(path)
        this.archiveValid = resp.valid
        this.archiveSize = resp.size || ''
      } catch (e) {
        this.archiveValid = false
        this.archiveSize = ''
      }
    },

    toDistributionConfig(): DistributionConfig {
      const config: DistributionConfig = {
        source: this.source,
      }
      if (this.source === 'official') {
        config.version = this.version
      } else {
        config.archive_path = this.archivePath
        config.binary_path = this.binaryPath
        config.cli_binary = this.cliBinary
        config.ops_binary = this.opsBinary
        config.tool_binary = this.dstoolBinary
      }
      return config
    },
  },
})
