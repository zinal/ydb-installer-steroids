import { defineStore } from 'pinia'
import { api } from '@/api/client'

export type ConnectivityStatus = 'checking' | 'online' | 'offline'

interface ConnectivityState {
  status: ConnectivityStatus
  /** Manual override of the detected result; null = use detection. */
  override: boolean | null
  error: string
}

export const useConnectivityStore = defineStore('connectivity', {
  state: (): ConnectivityState => ({
    status: 'checking',
    override: null,
    error: '',
  }),

  getters: {
    /**
     * Single source of truth for screens: the manual override wins when set,
     * otherwise the auto-detected result is used. Tri-state — `null` while the
     * probe is still running, so consumers never mistake "checking" for
     * "offline". Act only on a definitive boolean.
     */
    effectiveOnline(state): boolean | null {
      if (state.override !== null) return state.override
      if (state.status === 'checking') return null
      return state.status === 'online'
    },
    hasOverride(state): boolean {
      return state.override !== null
    },
    /**
     * True when the installer should behave as if there is no internet.
     * `null` while the probe is still running (indeterminate, not offline).
     */
    offlineMode(state): boolean | null {
      const online = this.effectiveOnline
      return online === null ? null : !online
    },
  },

  actions: {
    async check() {
      this.status = 'checking'
      this.error = ''
      try {
        const r = await api.getConnectivity()
        this.status = r.online ? 'online' : 'offline'
        if (r.error) this.error = r.error
      } catch (e) {
        this.status = 'offline'
        this.error = e instanceof Error ? e.message : String(e)
      }
    },

    setOverride(v: boolean | null) {
      this.override = v
    },

    /** Toggle offline mode; on ⇒ force offline, off ⇒ force online. */
    setOfflineMode(v: boolean | null) {
      this.override = v === true ? false : true
    },
  },
})