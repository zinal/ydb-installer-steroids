import { defineStore } from 'pinia'
import type { YDBConfig, InventoryData } from '@/types/inventory'

interface YdbConfigState {
  domain: string
  password: string
  user: string
  coresStatic: number
  coresDynamic: number
  enforceUserTokenRequirement: boolean
  requestClientCertificate: boolean
  poolKind: string
  affinity: string
}

export const useYdbConfigStore = defineStore('ydbConfig', {
  state: (): YdbConfigState => ({
    domain: 'Root',
    password: '',
    user: '',
    coresStatic: 0,
    coresDynamic: 0,
    enforceUserTokenRequirement: false,
    requestClientCertificate: true,
    poolKind: 'ssd',
    affinity: '',
  }),

  getters: {
    canProceed: (state) => {
      return state.domain.trim() !== '' && state.coresStatic > 0
    },
  },

  actions: {
    applyFromInventory(inv: InventoryData) {
      if (inv.ydb_config) {
        const cfg = inv.ydb_config
        this.domain = cfg.domain || ''
        this.password = cfg.password || ''
        this.user = cfg.user || ''
        this.coresStatic = cfg.cores_static ?? 0
        this.coresDynamic = cfg.cores_dynamic ?? 0
        this.enforceUserTokenRequirement = cfg.enforce_user_token_requirement ?? false
        this.requestClientCertificate = cfg.request_client_certificate ?? true
        this.poolKind = cfg.pool_kind || 'ssd'
      }
    },

    generatePassword(length = 16) {
      const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
      let pwd = ''
      for (let i = 0; i < length; i++) {
        pwd += chars.charAt(Math.floor(Math.random() * chars.length))
      }
      this.password = pwd
    },

    toYdbConfig(): YDBConfig {
      const cfg: YDBConfig = {
        domain: this.domain,
        user: this.user,
        cores_static: this.coresStatic,
        cores_dynamic: this.coresDynamic,
        enforce_user_token_requirement: this.enforceUserTokenRequirement,
        request_client_certificate: this.requestClientCertificate,
        pool_kind: this.poolKind,
        password: this.password || '',
      }
      return cfg
    },
  },
})