import { defineStore } from 'pinia'
import { api, type RuntimeInfo } from '@/api/client'

interface RuntimeState {
  embedded: boolean
  extracted: boolean
  python: string
  ansible: string
  ansibleCore: string
  collection: string
  emCollection: string
  path: string
  error: string
  loading: boolean
  loaded: boolean
  /**
   * Enables the YDB EM scenarios (ydb-em wizard, EM delete) in the UI.
   * Set from the --show-em-scenarios CLI flag; off by default.
   */
  showEmScenarios: boolean
}

export const useRuntimeStore = defineStore('runtime', {
  state: (): RuntimeState => ({
    embedded: false,
    extracted: false,
    python: '',
    ansible: '',
    ansibleCore: '',
    collection: '',
    emCollection: '',
    path: '',
    error: '',
    loading: false,
    loaded: false,
    showEmScenarios: false,
  }),

  getters: {
    /** True when the bundled runtime is ready, making dependency checks implicit. */
    bundleActive(state): boolean {
      return state.embedded && state.extracted
    },
  },

  actions: {
    async load() {
      this.loading = true
      this.error = ''
      try {
        const info: RuntimeInfo = await api.getRuntimeInfo()
        this.embedded = info.embedded
        this.extracted = info.extracted
        this.python = info.python ?? ''
        this.ansible = info.ansible ?? ''
        this.ansibleCore = info.ansible_core ?? ''
        this.collection = info.collection ?? ''
        this.emCollection = info.em_collection ?? ''
        this.path = info.path ?? ''
        this.showEmScenarios = info.show_em_scenarios ?? false
        if (info.error) this.error = info.error
        this.loaded = true
      } catch (e) {
        this.error = e instanceof Error ? e.message : String(e)
        this.embedded = false
        this.extracted = false
        this.loaded = true
      } finally {
        this.loading = false
      }
    },
  },
})
