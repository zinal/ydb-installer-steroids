import { defineStore } from 'pinia'
import type { DisksConfig, InventoryData } from '@/types/inventory'
import type { DiskInfo, DCDisks, DiskSelection, DiskConsistencyResult, DiskEntryMode } from '@/types/disks'

interface DisksState {
  allowFormat: boolean
  entryMode: DiskEntryMode
  autoSelectedDisks: { path: string; label: string }[]
  manualDiskInput: string
  manualInputError: string | null
  dcDisks: DCDisks
  diskResults: Map<string, DiskInfo[]>
  consistencyResult: DiskConsistencyResult | null
  isDiscovering: boolean
  discoveryError: string | null
}

const DEVICE_PATH_RE = /^\/dev\/[a-zA-Z0-9_\/-]+$/

function splitLines(text: string): string[] {
  return text
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)
}

export const useDisksStore = defineStore('disks', {
  state: (): DisksState => ({
    allowFormat: false,
    entryMode: 'auto',
    autoSelectedDisks: [],
    manualDiskInput: '',
    manualInputError: null,
    dcDisks: {
      'DC-1': [],
      'DC-2': [],
      'DC-3': [],
    },
    diskResults: new Map(),
    consistencyResult: null,
    isDiscovering: false,
    discoveryError: null,
  }),

  getters: {
    selectedDisks(state): { path: string; label: string }[] {
      if (state.entryMode === 'manual') {
        const lines = splitLines(state.manualDiskInput)
        if (lines.length === 0) return []
        return lines.map((path, index) => ({
          path,
          label: `ydb_disk_${index + 1}`,
        }))
      }
      return state.autoSelectedDisks
    },

    getManualDisksCount(state): number {
      if (state.entryMode !== 'manual') return 0
      return splitLines(state.manualDiskInput).length
    },
  },

  actions: {
    /**
     * Apply inventory data to this store (auto-fill from backend).
     *
     * When disk discovery has already been performed (diskResults is non-empty),
     * only disks that were actually found via SSH are kept. This prevents stale
     * inventory disks — from previous runs on different hardware — from being
     * re-introduced on page load or WebSocket reload.
     *
     * Before discovery, all inventory disks are loaded as-is (the user hasn't
     * interacted with disk selection yet, so there's nothing to reconcile).
     */
    applyFromInventory(inv: InventoryData) {
      if (inv.disks) {
        this.allowFormat = inv.disks.allow_format
        const inventoryDisks = inv.disks.selected_disks.map(d => ({ path: d.path, label: d.label }))

        if (this.diskResults.size > 0) {
          const discoveredPaths = new Set<string>()
          for (const disks of this.diskResults.values()) {
            for (const d of disks) {
              discoveredPaths.add(d.path)
            }
          }
          this.autoSelectedDisks = inventoryDisks.filter(d => discoveredPaths.has(d.path))
          this.autoSelectedDisks.forEach((d, i) => {
            d.label = `ydb_disk_${i + 1}`
          })
        } else {
          this.autoSelectedDisks = inventoryDisks
        }
        // Do NOT change entryMode — it is user-controlled, not inventory-driven.
      }
    },

    toggleDisk(path: string, label: string) {
      if (this.entryMode !== 'auto') return
      const idx = this.autoSelectedDisks.findIndex(d => d.path === path)
      if (idx >= 0) {
        this.autoSelectedDisks.splice(idx, 1)
      } else {
        this.autoSelectedDisks.push({ path, label })
      }
    },

    setEntryMode(mode: DiskEntryMode) {
      if (this.entryMode === mode) return
      this.entryMode = mode
    },

    setManualDiskInput(text: string) {
      this.manualDiskInput = text
      this.manualInputError = null

      const lines = splitLines(text)
      if (lines.length === 0) return
      const seen = new Set<string>()
      for (const path of lines) {
        if (!DEVICE_PATH_RE.test(path)) {
          this.manualInputError = `Invalid device path: ${path}`
          return
        }
        if (seen.has(path)) {
          this.manualInputError = `Duplicate device path: ${path}`
          return
        }
        seen.add(path)
      }
    },

    clearManualInput() {
      this.manualDiskInput = ''
      this.manualInputError = null
    },

    setAllowFormat(allow: boolean) {
      this.allowFormat = allow
    },

    setDiskResults(host: string, disks: DiskInfo[]) {
      this.diskResults.set(host, disks)
    },

    setDCDisks(dc: keyof DCDisks, disks: DiskInfo[]) {
      this.dcDisks[dc] = disks
    },

    setConsistencyResult(result: DiskConsistencyResult) {
      this.consistencyResult = result
    },

    setDiscovering(isDiscovering: boolean) {
      this.isDiscovering = isDiscovering
    },

    /**
     * Reconcile autoSelectedDisks with discovered disk results.
     * Removes any selected disks that were NOT found during SSH discovery.
     * This prevents stale inventory disks from persisting forever when
     * they are no longer discoverable on the target hosts.
     *
     * No-op when diskResults is empty (discovery hasn't happened yet).
     */
    reconcileSelection() {
      if (this.diskResults.size === 0) return
      const discoveredPaths = new Set<string>()
      for (const disks of this.diskResults.values()) {
        for (const d of disks) {
          discoveredPaths.add(d.path)
        }
      }
      this.autoSelectedDisks = this.autoSelectedDisks.filter(d =>
        discoveredPaths.has(d.path)
      )
      this.autoSelectedDisks.forEach((d, i) => {
        d.label = `ydb_disk_${i + 1}`
      })
    },

    setDiscoveryError(error: string | null) {
      this.discoveryError = error
    },

    clearDiscovery() {
      this.diskResults.clear()
      this.consistencyResult = null
      this.discoveryError = null
      this.isDiscovering = false
      this.dcDisks = {
        'DC-1': [],
        'DC-2': [],
        'DC-3': [],
      }
    },

    getHostDisks(host: string): DiskInfo[] {
      return this.diskResults.get(host) || []
    },

    isDiskSelected(path: string): boolean {
      return this.selectedDisks.some(d => d.path === path)
    },
  },
})
