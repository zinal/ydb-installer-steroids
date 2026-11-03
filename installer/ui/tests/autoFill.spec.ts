import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { InventoryData } from '@/types/inventory'

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn(),
    updateInventory: vi.fn(),
  },
}))

describe('Inventory Stores Integration', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('applies inventory data to all stores', async () => {
    const { useHostsStore } = await import('@/stores/hosts')
    const { useDisksStore } = await import('@/stores/disks')
    const { useYdbConfigStore } = await import('@/stores/ydbConfig')
    const { useDistributionStore } = await import('@/stores/distribution')
    const { useTlsStore } = await import('@/stores/tls')
    const { useDatabasesStore } = await import('@/stores/databases')

    const inv: InventoryData = {
      hosts: ['host1.example.com'],
      ssh: { login: 'admin', use_jump_host: false },
      disks: {
        allow_format: true,
        selected_disks: [{ path: '/dev/sda', label: 'disk1' }],
      },
      ydb_config: {
        domain: 'TestDomain',
        cores_static: 4,
        cores_dynamic: 2,
        enforce_user_token_requirement: true,
        request_client_certificate: false,
        pool_kind: 'ssd',
      },
      distribution: {
        source: 'official',
        version: '25.3.1',
      },
      tls: {
        cert_path: '',
      },
      database: {
        config_variant: 'single',
        cores_dynamic: 4,
        entries: [
          { name: 'mydb', groups: 8 },
        ],
      },
    }

    const hostsStore = useHostsStore()
    const disksStore = useDisksStore()
    const ydbConfigStore = useYdbConfigStore()
    const distributionStore = useDistributionStore()
    const tlsStore = useTlsStore()
    const databasesStore = useDatabasesStore()

    hostsStore.applyFromInventory(inv)
    disksStore.applyFromInventory(inv)
    ydbConfigStore.applyFromInventory(inv)
    distributionStore.applyFromInventory(inv)
    tlsStore.applyFromInventory(inv)
    databasesStore.applyFromInventory(inv)

    expect(hostsStore.dcHosts['DC-1']).toEqual(['host1.example.com'])
    expect(hostsStore.ssh?.login).toBe('admin')

    expect(disksStore.allowFormat).toBe(true)
    expect(disksStore.selectedDisks).toHaveLength(1)

    expect(ydbConfigStore.domain).toBe('TestDomain')
    expect(ydbConfigStore.coresStatic).toBe(4)
    expect(ydbConfigStore.enforceUserTokenRequirement).toBe(true)

    expect(distributionStore.version).toBe('25.3.1')

    expect(tlsStore.mode).toBe('generate')
    expect(tlsStore.days).toBe(365)

    expect(databasesStore.entries[0].name).toBe('mydb')
    expect(databasesStore.entries[0].groups).toBe(8)
    expect(databasesStore.coresDynamic).toBe(4)
    expect(databasesStore.configVariant).toBe('single')
  })
})
