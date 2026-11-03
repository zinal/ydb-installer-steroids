import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useHostsStore } from '@/stores/hosts'
import type { InventoryData } from '@/types/inventory'

describe('useHostsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('has initial empty state', () => {
    const store = useHostsStore()
    expect(store.dcHosts).toEqual({ 'DC-1': [], 'DC-2': [], 'DC-3': [] })
    expect(store.ssh).toBeNull()
  })

  it('applies inventory data and distributes hosts to DCs', () => {
    const store = useHostsStore()
    const inv: InventoryData = {
      hosts: ['host1', 'host2', 'host3'],
      ssh: { login: 'root', use_jump_host: false },
    }
    store.applyFromInventory(inv)
    expect(store.dcHosts['DC-1']).toEqual(['host1'])
    expect(store.dcHosts['DC-2']).toEqual(['host2'])
    expect(store.dcHosts['DC-3']).toEqual(['host3'])
    expect(store.ssh?.login).toBe('root')
  })

  it('adds and removes DC hosts', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'host1')
    store.addDCHost('DC-1', 'host2')
    expect(store.dcHosts['DC-1']).toHaveLength(2)

    store.removeDCHost('DC-1', 0)
    expect(store.dcHosts['DC-1']).toEqual(['host2'])
  })

  it('does not add duplicate DC hosts', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'host1')
    store.addDCHost('DC-1', 'host1')
    expect(store.dcHosts['DC-1']).toHaveLength(1)
  })

  it('allows multiple empty DC hosts', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', '')
    store.addDCHost('DC-1', '')
    expect(store.dcHosts['DC-1']).toHaveLength(2)
  })

  it('addDCHost trims and rejects whitespace-only hosts', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', '  host1  ')
    expect(store.dcHosts['DC-1'][0]).toBe('host1')
    store.addDCHost('DC-1', 'host1')
    expect(store.dcHosts['DC-1']).toHaveLength(1)
    store.addDCHost('DC-1', '   ')
    expect(store.dcHosts['DC-1']).toHaveLength(1)
    store.addDCHost('DC-1', '')
    expect(store.dcHosts['DC-1']).toHaveLength(2)
  })

  it('updates DC host with trimming', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'host1')
    store.updateDCHost('DC-1', 0, '  host2  ')
    expect(store.dcHosts['DC-1'][0]).toBe('host2')
  })

  it('updateDCHost allows edits that collide with an existing host', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'host1')
    store.addDCHost('DC-1', 'host2')
    store.updateDCHost('DC-1', 1, 'host1')
    expect(store.dcHosts['DC-1']).toEqual(['host1', 'host1'])
    store.updateDCHost('DC-1', 1, 'host3')
    expect(store.dcHosts['DC-1'][1]).toBe('host3')
  })

  it('sets SSH config', () => {
    const store = useHostsStore()
    store.setSsh({ login: 'admin', use_jump_host: true, jump_host_fqdn: 'jump.example.com' })
    expect(store.ssh?.login).toBe('admin')
    expect(store.ssh?.use_jump_host).toBe(true)
  })

  it('sets check result with CPU cores', () => {
    const store = useHostsStore()
    store.setCheckResult('host1', { ssh: true, timeSync: true, python: true, cpuCores: 16 })
    expect(store.checkResults['host1']).toEqual({ ssh: true, timeSync: true, python: true, cpuCores: 16 })
  })

  it('clears check results', () => {
    const store = useHostsStore()
    store.setCheckResult('host1', { ssh: true, timeSync: true, python: true, cpuCores: 8 })
    store.clearCheckResults()
    expect(store.checkResults).toEqual({})
  })

  it('adds hosts in batch skipping duplicates', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'existing')
    const added1 = store.addHostsBatch('DC-1', ['host2', 'host3'])
    expect(added1).toHaveLength(2)
    expect(added1).toEqual(['host2', 'host3'])
    const added2 = store.addHostsBatch('DC-1', ['existing', 'host4'])
    expect(added2).toEqual(['host4'])
    expect(store.dcHosts['DC-1']).toEqual(['existing', 'host2', 'host3', 'host4'])
  })

  it('addHostsBatch skips empty strings', () => {
    const store = useHostsStore()
    const added = store.addHostsBatch('DC-1', ['host1', '', '  ', 'host2'])
    expect(added).toHaveLength(2)
    expect(added).toEqual(['host1', 'host2'])
  })

  it('duplicates returns empty array when all hosts are unique', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'host1')
    store.addDCHost('DC-2', 'host2')
    store.addDCHost('DC-3', 'host3')
    expect(store.duplicates).toEqual([])
  })

  it('duplicates detects host appearing in 2 different DCs', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'host1')
    store.addDCHost('DC-2', 'host1')
    store.addDCHost('DC-3', 'host2')
    expect(store.duplicates).toEqual(['host1'])
  })

  it('duplicates detects host appearing in all 3 DCs', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'duplicate-host')
    store.addDCHost('DC-2', 'duplicate-host')
    store.addDCHost('DC-3', 'duplicate-host')
    expect(store.duplicates).toEqual(['duplicate-host'])
  })

  it('duplicates ignores empty strings', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', '')
    store.addDCHost('DC-2', '')
    store.addDCHost('DC-3', '')
    expect(store.duplicates).toEqual([])
  })

  it('duplicates returns multiple duplicate hosts', () => {
    const store = useHostsStore()
    store.addDCHost('DC-1', 'host-a')
    store.addDCHost('DC-2', 'host-a')
    store.addDCHost('DC-1', 'host-b')
    store.addDCHost('DC-3', 'host-b')
    store.addDCHost('DC-2', 'unique-host')
    expect(store.duplicates).toEqual(['host-a', 'host-b'])
  })

  it('duplicates flags host appearing twice within same DC array', () => {
    const store = useHostsStore()
    store.dcHosts['DC-1'] = ['host1', 'host1']
    expect(store.duplicates).toEqual(['host1'])
  })
})

describe('useHostsStore time sync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('sets check result with timeSync false when NTP not active', () => {
    const store = useHostsStore()
    store.setCheckResult('host1', { ssh: true, timeSync: false, python: true, cpuCores: 8 })
    expect(store.checkResults['host1'].timeSync).toBe(false)
  })

  it('sets check result with timeSync true when NTP synced', () => {
    const store = useHostsStore()
    store.setCheckResult('host1', { ssh: true, timeSync: true, python: true, cpuCores: 8 })
    expect(store.checkResults['host1'].timeSync).toBe(true)
  })
})

describe('TimeSync types', () => {
  it('TimeSyncResult with synced chrony has correct fields', () => {
    const ts = {
      ntpActive: true,
      ntpDaemon: 'chrony',
      ntpSynced: true,
      clockOffsetMs: 2,
    }
    expect(ts.ntpActive).toBe(true)
    expect(ts.ntpDaemon).toBe('chrony')
    expect(ts.ntpSynced).toBe(true)
    expect(ts.clockOffsetMs).toBe(2)
  })

  it('TimeSyncResult with no daemon has correct fields', () => {
    const ts = {
      ntpActive: false,
      ntpDaemon: '',
      ntpSynced: false,
      clockOffsetMs: 0,
      error: 'NTP daemon not found',
    }
    expect(ts.ntpActive).toBe(false)
    expect(ts.error).toBeDefined()
  })

  it('TimeSyncSummary classifies ok status correctly', () => {
    const summary = {
      maxDriftMs: 3,
      status: 'ok' as const,
      threshold: { warningMs: 5, errorMs: 25 },
    }
    expect(summary.status).toBe('ok')
    expect(summary.maxDriftMs).toBeLessThan(summary.threshold.warningMs)
  })

  it('TimeSyncSummary classifies warning status correctly', () => {
    const summary = {
      maxDriftMs: 12,
      status: 'warning' as const,
      threshold: { warningMs: 5, errorMs: 25 },
    }
    expect(summary.status).toBe('warning')
    expect(summary.maxDriftMs).toBeGreaterThanOrEqual(summary.threshold.warningMs)
    expect(summary.maxDriftMs).toBeLessThan(summary.threshold.errorMs)
  })

  it('TimeSyncSummary classifies error status correctly', () => {
    const summary = {
      maxDriftMs: 30,
      status: 'error' as const,
      threshold: { warningMs: 5, errorMs: 25 },
    }
    expect(summary.status).toBe('error')
    expect(summary.maxDriftMs).toBeGreaterThanOrEqual(summary.threshold.errorMs)
  })
})
