import { defineStore } from 'pinia'
import type { DatabaseConfig, DatabaseEntry, InventoryData } from '@/types/inventory'

export interface DatabaseEntryUI {
  id: number
  name: string
  groups: number
  hosts: string[]
  hostMode: 'all' | 'select' | 'custom'
  affinity: string
}

export type ConfigVariant = 'single' | 'single-plus-em' | 'multiple'

interface DatabasesState {
  configVariant: ConfigVariant
  entries: DatabaseEntryUI[]
  coresDynamic: number
  nextId: number
  savedPerVariant: Record<string, DatabaseEntryUI[]>
}

export const DEFAULT_EM_DB_NAME = 'ydb_em'

function createEntry(id: number, name = 'database'): DatabaseEntryUI {
  return {
    id,
    name,
    groups: 8,
    hosts: [],
    hostMode: 'all',
    affinity: '',
  }
}

function cloneEntries(entries: DatabaseEntryUI[]): DatabaseEntryUI[] {
  return entries.map(e => ({ ...e, hosts: [...e.hosts] }))
}

function createSinglePlusEmEntries(userSeed?: DatabaseEntryUI): DatabaseEntryUI[] {
  const user = userSeed
    ? { ...userSeed, hosts: [...userSeed.hosts] }
    : createEntry(1, 'database')
  user.id = 1
  const em = createEntry(2, DEFAULT_EM_DB_NAME)
  return [user, em]
}

export const useDatabasesStore = defineStore('databases', {
  state: (): DatabasesState => ({
    configVariant: 'single',
    entries: [createEntry(1)],
    coresDynamic: 0,
    nextId: 2,
    savedPerVariant: {},
  }),

  getters: {
    canProceed: (state) => {
      if (state.configVariant === 'single-plus-em' && state.entries.length !== 2) {
        return false
      }
      if (state.entries.length === 0) return false
      if (state.entries.some((e) => !e.name.trim())) return false
      if (state.coresDynamic <= 0) return false

      const counts: Record<string, number> = {}
      for (const e of state.entries) {
        const name = e.name.trim()
        if (!name) continue
        counts[name] = (counts[name] || 0) + 1
        if (counts[name] > 1) return false
      }
      return true
    },

    hostCount: () => 0,

    /**
     * Names that appear more than once among all entries (trimmed, case-sensitive).
     * Empty for single-DB mode or when all names are unique.
     */
    duplicateNames: (state): string[] => {
      const counts: Record<string, number> = {}
      for (const e of state.entries) {
        const name = e.name.trim()
        if (!name) continue
        counts[name] = (counts[name] || 0) + 1
      }
      return Object.entries(counts)
        .filter(([, count]) => count > 1)
        .map(([name]) => name)
    },

    hasDuplicateNames(): boolean {
      return this.duplicateNames.length > 0
    },

    /**
     * Total dynamic cores per host across ALL database entries.
     * Equals coresDynamic × entries.length because each database entry
     * allocates its own dynamic nodes (including single-plus-em's two DBs).
     */
    totalDynamicPerHost: (state): number => {
      if (state.coresDynamic <= 0) return 0
      return state.coresDynamic * state.entries.length
    },

    emEntry(state): DatabaseEntryUI | null {
      if (state.configVariant !== 'single-plus-em') return null
      return state.entries[1] ?? null
    },

    userEntry(state): DatabaseEntryUI | null {
      return state.entries[0] ?? null
    },

    isSinglePlusEm(state): boolean {
      return state.configVariant === 'single-plus-em'
    },
  },

  actions: {
    applyFromInventory(inv: InventoryData) {
      if (inv.database) {
        const db = inv.database
        this.coresDynamic = db.cores_dynamic ?? 0
        this.configVariant = (db.config_variant as ConfigVariant) || 'single'

        if (db.entries && db.entries.length > 0) {
          this.entries = db.entries.map((entry: DatabaseEntry, idx: number) => ({
            id: idx + 1,
            name: entry.name || '',
            groups: entry.groups || 8,
            hosts: entry.hosts || [],
            hostMode: entry.hosts && entry.hosts.length > 0 ? 'select' : 'all',
            affinity: entry.affinity || '',
          }))
          if (this.configVariant === 'single-plus-em') {
            if (this.entries.length === 1) {
              this.entries.push(createEntry(2, DEFAULT_EM_DB_NAME))
            } else if (this.entries.length >= 2) {
              this.entries[1].name = DEFAULT_EM_DB_NAME
            }
          }
          this.nextId = this.entries.length + 1
        } else if (this.configVariant === 'single-plus-em') {
          this.entries = createSinglePlusEmEntries()
          this.nextId = 3
        } else {
          this.entries = [createEntry(1)]
          this.nextId = 2
        }
      }
    },

    setConfigVariant(variant: ConfigVariant) {
      const oldVariant = this.configVariant
      if (oldVariant === variant) return

      if (oldVariant && this.entries.length > 0) {
        this.savedPerVariant[oldVariant] = cloneEntries(this.entries)
      }

      this.configVariant = variant

      const saved = this.savedPerVariant[variant]
      if (saved && saved.length > 0) {
        this.entries = cloneEntries(saved)
        this.nextId = Math.max(...saved.map(e => e.id)) + 1
        if (variant === 'single-plus-em') {
          if (this.entries.length !== 2) {
            this.entries = createSinglePlusEmEntries(this.entries[0])
            this.nextId = 3
          } else {
            this.entries[1].name = DEFAULT_EM_DB_NAME
          }
        }
        return
      }

      if (variant === 'single') {
        this.entries = [this.entries[0] ? { ...this.entries[0], hosts: [...this.entries[0].hosts], id: 1 } : createEntry(1)]
        this.nextId = 2
      } else if (variant === 'single-plus-em') {
        this.entries = createSinglePlusEmEntries(this.entries[0])
        this.nextId = 3
      } else if (variant === 'multiple') {
        if (this.entries.length < 2) {
          if (this.entries.length === 0) {
            this.entries = [createEntry(1), createEntry(2)]
            this.nextId = 3
          } else {
            this.entries.push(createEntry(this.nextId++))
          }
        }
      }
    },

    addEntry() {
      if (this.configVariant === 'single-plus-em' || this.configVariant === 'single') return
      this.entries.push(createEntry(this.nextId++))
    },

    removeEntry(index: number) {
      if (this.configVariant === 'single-plus-em' || this.configVariant === 'single') return
      if (this.entries.length > 1) {
        this.entries.splice(index, 1)
      }
    },

    updateEntry(index: number, field: keyof DatabaseEntryUI, value: unknown) {
      if (index >= 0 && index < this.entries.length) {
        ; (this.entries[index] as Record<string, unknown>)[field] = value
      }
    },

    toDatabaseConfig(): DatabaseConfig {
      return {
        config_variant: this.configVariant,
        cores_dynamic: this.coresDynamic,
        entries: this.entries.map((e) => ({
          name: e.name,
          groups: e.groups,
          hosts: e.hostMode === 'all' ? [] : e.hosts,
          affinity: e.affinity || undefined,
        })),
      }
    },
  },
})
