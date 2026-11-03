import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDatabasesStore } from '@/stores/databases'

describe('useDatabasesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('starts with default values', () => {
    const store = useDatabasesStore()
    expect(store.configVariant).toBe('single')
    expect(store.entries).toHaveLength(1)
    expect(store.entries[0].id).toBe(1)
    expect(store.entries[0].name).toBe('database')
    expect(store.entries[0].groups).toBe(8)
    expect(store.entries[0].hostMode).toBe('all')
    expect(store.entries[0].hosts).toEqual([])
    expect(store.coresDynamic).toBe(0)
    expect(store.nextId).toBe(2)
  })

  it('canProceed is false when name is empty', () => {
    const store = useDatabasesStore()
    store.coresDynamic = 4
    store.entries[0].name = ''
    expect(store.canProceed).toBe(false)
  })

  it('canProceed is false when coresDynamic is 0', () => {
    const store = useDatabasesStore()
    store.entries[0].name = '/Root/testdb'
    expect(store.coresDynamic).toBe(0)
    expect(store.canProceed).toBe(false)
  })

  it('canProceed is true when name is filled and coresDynamic > 0', () => {
    const store = useDatabasesStore()
    store.coresDynamic = 4
    expect(store.canProceed).toBe(true)
  })

  it('setConfigVariant changes variant and adjusts entries', () => {
    const store = useDatabasesStore()
    store.setConfigVariant('multiple')
    expect(store.configVariant).toBe('multiple')
    expect(store.entries).toHaveLength(2)

    store.setConfigVariant('single')
    expect(store.configVariant).toBe('single')
    expect(store.entries).toHaveLength(1)
  })

  it('addEntry and removeEntry work correctly', () => {
    const store = useDatabasesStore()
    store.setConfigVariant('multiple')
    const initialCount = store.entries.length

    store.addEntry()
    expect(store.entries.length).toBe(initialCount + 1)

    store.addEntry()
    expect(store.entries.length).toBe(initialCount + 2)

    store.removeEntry(0)
    expect(store.entries.length).toBe(initialCount + 1)
  })

  it('removeEntry does not reduce below 1', () => {
    const store = useDatabasesStore()
    expect(store.entries).toHaveLength(1)

    store.removeEntry(0)
    expect(store.entries).toHaveLength(1)
  })

  it('updateEntry modifies entry fields', () => {
    const store = useDatabasesStore()
    store.updateEntry(0, 'name', '/Root/testdb')
    expect(store.entries[0].name).toBe('/Root/testdb')

    store.updateEntry(0, 'groups', 4)
    expect(store.entries[0].groups).toBe(4)

    store.updateEntry(0, 'hostMode', 'select')
    expect(store.entries[0].hostMode).toBe('select')
  })

  it('applyFromInventory fills store from inventory data', () => {
    const store = useDatabasesStore()
    store.applyFromInventory({
      database: {
        config_variant: 'multiple',
        cores_dynamic: 8,
        entries: [
          { name: '/Root/db1', groups: 4 },
          { name: '/Root/db2', groups: 8, hosts: ['node1.dc1', 'node1.dc2'] },
        ],
      },
    })

    expect(store.configVariant).toBe('multiple')
    expect(store.coresDynamic).toBe(8)
    expect(store.entries).toHaveLength(2)
    expect(store.entries[0].name).toBe('/Root/db1')
    expect(store.entries[0].groups).toBe(4)
    expect(store.entries[1].name).toBe('/Root/db2')
    expect(store.entries[1].hosts).toEqual(['node1.dc1', 'node1.dc2'])
    expect(store.entries[1].hostMode).toBe('select')
  })

  it('applyFromInventory handles missing database gracefully', () => {
    const store = useDatabasesStore()
    store.entries[0].name = '/Root/testdb'

    store.applyFromInventory({})

    expect(store.entries[0].name).toBe('/Root/testdb')
  })

  it('applyFromInventory preserves zero cores_dynamic (does not force 4)', () => {
    const store = useDatabasesStore()
    store.coresDynamic = 8

    store.applyFromInventory({
      database: {
        config_variant: 'single',
        cores_dynamic: 0,
        entries: [{ name: 'database', groups: 8 }],
      },
    })

    expect(store.coresDynamic).toBe(0)
  })

  it('applyFromInventory treats missing cores_dynamic as 0', () => {
    const store = useDatabasesStore()
    store.coresDynamic = 8

    store.applyFromInventory({
      database: {
        config_variant: 'single',
        entries: [{ name: 'database', groups: 8 }],
      } as any,
    })

    expect(store.coresDynamic).toBe(0)
  })

  it('toDatabaseConfig builds correct config', () => {
    const store = useDatabasesStore()
    store.configVariant = 'single'
    store.coresDynamic = 4
    store.entries[0].name = '/Root/testdb'
    store.entries[0].groups = 8

    const config = store.toDatabaseConfig()
    expect(config).toEqual({
      config_variant: 'single',
      cores_dynamic: 4,
      entries: [{ name: '/Root/testdb', groups: 8, hosts: [], affinity: undefined }],
    })
  })

  it('toDatabaseConfig handles multiple entries', () => {
    const store = useDatabasesStore()
    store.setConfigVariant('multiple')
    store.coresDynamic = 6
    store.entries[0].name = '/Root/db1'
    store.entries[0].groups = 4
    store.entries[1].name = '/Root/db2'
    store.entries[1].groups = 8
    store.entries[1].hostMode = 'select'
    store.entries[1].hosts = ['node1.example.com']
    store.entries[1].affinity = '/Root/location'

    const config = store.toDatabaseConfig()
    expect(config.config_variant).toBe('multiple')
    expect(config.cores_dynamic).toBe(6)
    expect(config.entries).toHaveLength(2)
    expect(config.entries[0].name).toBe('/Root/db1')
    expect(config.entries[1].hosts).toEqual(['node1.example.com'])
    expect(config.entries[1].affinity).toBe('/Root/location')
  })

  it('duplicateNames returns empty when all names are unique', () => {
    const store = useDatabasesStore()
    store.setConfigVariant('multiple')
    store.updateEntry(0, 'name', '/Root/db1')
    store.updateEntry(1, 'name', '/Root/db2')

    expect(store.duplicateNames).toEqual([])
    expect(store.hasDuplicateNames).toBe(false)
  })

  it('duplicateNames detects duplicates (case-sensitive)', () => {
    const store = useDatabasesStore()
    store.setConfigVariant('multiple')
    store.updateEntry(0, 'name', '/Root/db1')
    store.updateEntry(1, 'name', '/Root/db1')

    expect(store.duplicateNames).toEqual(['/Root/db1'])
    expect(store.hasDuplicateNames).toBe(true)
  })

  it('duplicateNames ignores empty names', () => {
    const store = useDatabasesStore()
    store.setConfigVariant('multiple')
    store.updateEntry(0, 'name', '')
    store.updateEntry(1, 'name', '')

    expect(store.duplicateNames).toEqual([])
    expect(store.hasDuplicateNames).toBe(false)
    expect(store.canProceed).toBe(false)
  })

  it('canProceed is false when duplicate names exist', () => {
    const store = useDatabasesStore()
    store.coresDynamic = 4
    store.setConfigVariant('multiple')
    store.updateEntry(0, 'name', '/Root/mydb')
    store.updateEntry(1, 'name', '/Root/mydb')

    expect(store.canProceed).toBe(false)
  })

  it('totalDynamicPerHost equals coresDynamic in single mode', () => {
    const store = useDatabasesStore()
    store.configVariant = 'single'
    store.coresDynamic = 4
    store.entries = [{ id: 1, name: '/Root/db', groups: 8, hosts: [], hostMode: 'all', affinity: '' }]

    expect(store.totalDynamicPerHost).toBe(4)
  })

  it('totalDynamicPerHost multiplies by entries count in multiple mode', () => {
    const store = useDatabasesStore()
    store.setConfigVariant('multiple')
    store.coresDynamic = 4
    expect(store.totalDynamicPerHost).toBe(8)

    store.addEntry()
    expect(store.totalDynamicPerHost).toBe(12)
  })

  it('totalDynamicPerHost returns 0 when coresDynamic is 0', () => {
    const store = useDatabasesStore()
    store.setConfigVariant('multiple')
    store.coresDynamic = 0

    expect(store.totalDynamicPerHost).toBe(0)
  })

  it('preserves entries when switching between variants', () => {
    const store = useDatabasesStore()

    store.setConfigVariant('multiple')
    expect(store.entries).toHaveLength(2)
    store.updateEntry(0, 'name', 'sales')
    store.updateEntry(0, 'groups', 4)
    store.updateEntry(1, 'name', 'analytics')
    store.updateEntry(1, 'groups', 8)
    store.addEntry()
    store.updateEntry(2, 'name', 'testing')
    expect(store.entries).toHaveLength(3)

    store.setConfigVariant('single')
    expect(store.entries).toHaveLength(1)

    store.setConfigVariant('multiple')
    expect(store.entries).toHaveLength(3)
    expect(store.entries[0].name).toBe('sales')
    expect(store.entries[0].groups).toBe(4)
    expect(store.entries[1].name).toBe('analytics')
    expect(store.entries[1].groups).toBe(8)
    expect(store.entries[2].name).toBe('testing')
  })
})