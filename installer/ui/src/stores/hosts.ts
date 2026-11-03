import { defineStore } from 'pinia'
import type { SSHConfig, InventoryData } from '@/types/inventory'

export interface DCHosts {
  'DC-1': string[]
  'DC-2': string[]
  'DC-3': string[]
}

interface HostsState {
  dcHosts: DCHosts
  ssh: SSHConfig | null
  checkResults: Record<string, { ssh: boolean; timeSync: boolean; python: boolean; cpuCores: number }>
}

export const useHostsStore = defineStore('hosts', {
  state: (): HostsState => ({
    dcHosts: { 'DC-1': [], 'DC-2': [], 'DC-3': [] },
    ssh: null,
    checkResults: {},
  }),

  getters: {
    duplicates(state): string[] {
      const seen = new Map<string, number[]>()
      const dcs: (keyof DCHosts)[] = ['DC-1', 'DC-2', 'DC-3']
      for (const dc of dcs) {
        for (const host of state.dcHosts[dc]) {
          const trimmed = host.trim()
          if (trimmed === '') continue
          if (!seen.has(trimmed)) seen.set(trimmed, [])
          seen.get(trimmed)!.push(0)
        }
      }
      return Array.from(seen.entries())
        .filter(([_, counts]) => counts.length > 1)
        .map(([host]) => host)
    },

    total(state): number {
      return state.dcHosts['DC-1'].length + state.dcHosts['DC-2'].length + state.dcHosts['DC-3'].length
    },
  },

  actions: {
    applyFromInventory(inv: InventoryData) {
      if (inv.dc1 || inv.dc2 || inv.dc3) {
        const hasExistingHosts =
          this.dcHosts['DC-1'].length > 0 ||
          this.dcHosts['DC-2'].length > 0 ||
          this.dcHosts['DC-3'].length > 0
        
        if (!hasExistingHosts) {
          this.dcHosts = {
            'DC-1': inv.dc1 ?? [],
            'DC-2': inv.dc2 ?? [],
            'DC-3': inv.dc3 ?? [],
          }
        }
      } else if (inv.hosts) {
        const hasExistingHosts =
          this.dcHosts['DC-1'].length > 0 ||
          this.dcHosts['DC-2'].length > 0 ||
          this.dcHosts['DC-3'].length > 0
        
        if (!hasExistingHosts) {
          this.dcHosts = {
            'DC-1': [],
            'DC-2': [],
            'DC-3': []
          }
          const uniqueHosts = [...new Set(inv.hosts)]
          uniqueHosts.forEach((host, index) => {
            const dc = `DC-${(index % 3) + 1}` as keyof DCHosts
            this.dcHosts[dc].push(host)
          })
        }
      }
      if (inv.ssh) {
        this.ssh = { ...inv.ssh }
      }
    },

    addDCHost(dc: keyof DCHosts, host: string) {
      if (host === '') {
        this.dcHosts[dc].push(host)
        return
      }
      const trimmed = host.trim()
      if (trimmed !== '' && !this.dcHosts[dc].includes(trimmed)) {
        this.dcHosts[dc].push(trimmed)
      }
    },

    addHostsBatch(dc: keyof DCHosts, hosts: string[]): string[] {
      const existing = new Set(this.dcHosts[dc])
      const added: string[] = []
      for (const host of hosts) {
        if (host.trim() !== '' && !existing.has(host)) {
          this.dcHosts[dc].push(host)
          existing.add(host)
          added.push(host)
        }
      }
      return added
    },

    removeDCHost(dc: keyof DCHosts, index: number) {
      if (index >= 0 && index < this.dcHosts[dc].length) {
        this.dcHosts[dc] = [
          ...this.dcHosts[dc].slice(0, index),
          ...this.dcHosts[dc].slice(index + 1)
        ]
      }
    },

    updateDCHost(dc: keyof DCHosts, index: number, host: string) {
      if (index >= 0 && index < this.dcHosts[dc].length) {
        const trimmed = host.trim()
        // Edits always land — even when they collide with an existing host —
        // so the input and store never diverge; duplicates are surfaced by the
        // `duplicates` getter and the UI warning.
        this.dcHosts[dc][index] = trimmed === '' ? host : trimmed
      }
    },

    setSsh(ssh: SSHConfig) {
      this.ssh = ssh
    },

    setCheckResult(host: string, result: { ssh: boolean; timeSync: boolean; python: boolean; cpuCores: number }) {
      this.checkResults[host] = result
    },

    clearCheckResults() {
      this.checkResults = {}
    },
  },
})
