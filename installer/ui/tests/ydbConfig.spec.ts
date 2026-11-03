import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useYdbConfigStore } from '@/stores/ydbConfig'

describe('useYdbConfigStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('starts with default values', () => {
    const store = useYdbConfigStore()
    expect(store.domain).toBe('Root')
    expect(store.user).toBe('')
    expect(store.coresStatic).toBe(0)
    expect(store.coresDynamic).toBe(0)
    expect(store.enforceUserTokenRequirement).toBe(false)
    expect(store.requestClientCertificate).toBe(true)
    expect(store.poolKind).toBe('ssd')
    expect(store.password).toBe('')
    expect(store.affinity).toBe('')
  })

  it('canProceed is false when domain is empty', () => {
    const store = useYdbConfigStore()
    store.domain = ''
    store.coresStatic = 4
    expect(store.canProceed).toBe(false)
  })

  it('canProceed is false when coresStatic is 0', () => {
    const store = useYdbConfigStore()
    expect(store.coresStatic).toBe(0)
    expect(store.canProceed).toBe(false)
  })

  it('canProceed is true when domain is filled and coresStatic > 0', () => {
    const store = useYdbConfigStore()
    store.coresStatic = 4
    expect(store.canProceed).toBe(true)
  })

  it('canProceed is true with whitespace-only domain is false', () => {
    const store = useYdbConfigStore()
    store.domain = '   '
    store.coresStatic = 4
    expect(store.canProceed).toBe(false)
  })

  it('generatePassword creates a 16-character password', () => {
    const store = useYdbConfigStore()
    store.generatePassword(16)
    expect(store.password.length).toBe(16)
  })

  it('generatePassword creates different passwords on successive calls', () => {
    const store = useYdbConfigStore()
    store.generatePassword(16)
    const first = store.password
    store.generatePassword(16)
    const second = store.password
    expect(first).not.toBe(second)
  })

  it('applyFromInventory fills store from inventory data including password', () => {
    const store = useYdbConfigStore()
    store.applyFromInventory({
      ydb_config: {
        domain: 'Root',
        password: 'inventoryPassword',
        user: 'admin',
        cores_static: 8,
        cores_dynamic: 4,
        enforce_user_token_requirement: true,
        request_client_certificate: false,
        pool_kind: 'nvme',
      },
    })

    expect(store.domain).toBe('Root')
    expect(store.password).toBe('inventoryPassword')
    expect(store.user).toBe('admin')
    expect(store.coresStatic).toBe(8)
    expect(store.coresDynamic).toBe(4)
    expect(store.enforceUserTokenRequirement).toBe(true)
    expect(store.requestClientCertificate).toBe(false)
    expect(store.poolKind).toBe('nvme')
    // affinity remains UI-only
    expect(store.affinity).toBe('')
  })

  it('applyFromInventory handles missing ydb_config gracefully', () => {
    const store = useYdbConfigStore()
    store.domain = 'Root'
    store.password = 'secret'
    store.affinity = '/Root/db'

    store.applyFromInventory({})

    // Should not reset existing values if ydb_config is absent
    expect(store.domain).toBe('Root')
    expect(store.password).toBe('secret')
    expect(store.affinity).toBe('/Root/db')
  })

  it('applyFromInventory preserves zero cores (does not force defaults)', () => {
    const store = useYdbConfigStore()
    store.coresStatic = 8
    store.coresDynamic = 4

    store.applyFromInventory({
      ydb_config: {
        domain: 'Root',
        cores_static: 0,
        cores_dynamic: 0,
        enforce_user_token_requirement: false,
        request_client_certificate: true,
        pool_kind: 'ssd',
      },
    })

    expect(store.coresStatic).toBe(0)
    expect(store.coresDynamic).toBe(0)
  })

  it('toYdbConfig includes password when set', () => {
    const store = useYdbConfigStore()
    store.domain = 'Root'
    store.user = 'admin'
    store.coresStatic = 8
    store.coresDynamic = 4
    store.enforceUserTokenRequirement = true
    store.requestClientCertificate = false
    store.poolKind = 'nvme'
    store.password = 'secret123'

    const config = store.toYdbConfig()
    expect(config).toEqual({
      domain: 'Root',
      password: 'secret123',
      user: 'admin',
      cores_static: 8,
      cores_dynamic: 4,
      enforce_user_token_requirement: true,
      request_client_certificate: false,
      pool_kind: 'nvme',
    })
  })

  it('toYdbConfig includes password as empty string when not set', () => {
    const store = useYdbConfigStore()
    store.domain = 'Root'
    store.user = 'admin'
    store.coresStatic = 8
    store.password = ''

    const config = store.toYdbConfig()
    expect(config.password).toBe('')
  })

  it('toYdbConfig uses defaults when fields are empty', () => {
    const store = useYdbConfigStore()
    store.domain = 'Root'

    const config = store.toYdbConfig()
    expect(config).toEqual({
      domain: 'Root',
      user: '',
      cores_static: 0,
      cores_dynamic: 0,
      enforce_user_token_requirement: false,
      request_client_certificate: true,
      pool_kind: 'ssd',
      password: '',
    })
  })
})