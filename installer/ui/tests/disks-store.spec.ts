import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDisksStore } from '@/stores/disks'
import type { DiskInfo, DiskConsistencyResult } from '@/types/disks'

describe('Disks Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes with empty state', () => {
    const store = useDisksStore()

    expect(store.allowFormat).toBe(false)
    expect(store.autoSelectedDisks).toEqual([])
    expect(store.selectedDisks).toEqual([])
    expect(store.entryMode).toBe('auto')
    expect(store.manualDiskInput).toBe('')
    expect(store.manualInputError).toBeNull()
    expect(store.dcDisks).toEqual({
      'DC-1': [],
      'DC-2': [],
      'DC-3': [],
    })
    expect(store.diskResults.size).toBe(0)
    expect(store.consistencyResult).toBeNull()
    expect(store.isDiscovering).toBe(false)
    expect(store.discoveryError).toBeNull()
  })

  it('toggles disk selection in auto mode', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/vda', 'ydb_disk_1')
    expect(store.autoSelectedDisks).toEqual([
      { path: '/dev/vda', label: 'ydb_disk_1' }
    ])
    expect(store.selectedDisks).toEqual([
      { path: '/dev/vda', label: 'ydb_disk_1' }
    ])

    store.toggleDisk('/dev/vda', '')
    expect(store.autoSelectedDisks).toEqual([])
  })

  it('sets disk results for a host', () => {
    const store = useDisksStore()
    const disks: DiskInfo[] = [
      {
        name: 'vda',
        path: '/dev/vda',
        type: 'SSD',
        size: '500G',
        sizeBytes: 500 * 1024 * 1024 * 1024,
        mounted: false,
        mountPoint: '',
        filesystem: '',
        isSystemDisk: false,
      },
    ]

    store.setDiskResults('host1', disks)
    expect(store.getHostDisks('host1')).toEqual(disks)
  })

  it('sets DC-organized disks', () => {
    const store = useDisksStore()
    const disks: DiskInfo[] = [
      {
        name: 'vda',
        path: '/dev/vda',
        type: 'SSD',
        size: '500G',
        sizeBytes: 500 * 1024 * 1024 * 1024,
        mounted: false,
        mountPoint: '',
        filesystem: '',
        isSystemDisk: false,
      },
    ]

    store.setDCDisks('DC-1', disks)
    expect(store.dcDisks['DC-1']).toEqual(disks)
  })

  it('sets consistency result', () => {
    const store = useDisksStore()
    const result: DiskConsistencyResult = {
      isConsistent: true,
      issues: [],
      commonDisks: [],
    }

    store.setConsistencyResult(result)
    expect(store.consistencyResult).toEqual(result)
  })

  it('clears discovery data but preserves autoSelectedDisks', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/vda', 'ydb_disk_1')
    store.setDiskResults('host1', [])
    store.setConsistencyResult({ isConsistent: true, issues: [], commonDisks: [] })
    store.setDiscoveryError('test error')
    expect(store.autoSelectedDisks).toHaveLength(1)

    store.clearDiscovery()

    expect(store.diskResults.size).toBe(0)
    expect(store.consistencyResult).toBeNull()
    expect(store.discoveryError).toBeNull()
    expect(store.autoSelectedDisks).toHaveLength(1)
  })

  it('checks if disk is selected', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/vda', 'ydb_disk_1')

    expect(store.isDiskSelected('/dev/vda')).toBe(true)
    expect(store.isDiskSelected('/dev/vdb')).toBe(false)
  })

  it('setEntryMode switches mode without clearing any state', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/vda', 'ydb_disk_1')
    store.setDiscovering(true)
    expect(store.autoSelectedDisks).toHaveLength(1)

    store.setEntryMode('manual')
    expect(store.entryMode).toBe('manual')
    expect(store.autoSelectedDisks).toHaveLength(1)
    expect(store.isDiscovering).toBe(true)

    store.setManualDiskInput('/dev/vdb\n/dev/vdc')
    expect(store.selectedDisks).toHaveLength(2)
    expect(store.selectedDisks[0].label).toBe('ydb_disk_1')

    store.setEntryMode('auto')
    expect(store.entryMode).toBe('auto')
    expect(store.manualDiskInput).toBe('/dev/vdb\n/dev/vdc')
    expect(store.autoSelectedDisks).toHaveLength(1)
  })

  it('setManualDiskInput validates and state is reflected in selectedDisks getter', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')

    store.setManualDiskInput('/dev/vdb\n/dev/vdc\n/dev/vdd\n/dev/vde')

    expect(store.selectedDisks).toHaveLength(4)
    expect(store.selectedDisks[0].label).toBe('ydb_disk_1')
    expect(store.selectedDisks[3].label).toBe('ydb_disk_4')
    expect(store.manualInputError).toBeNull()
  })

  it('selectedDisks getter returns empty array for empty manual input', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    store.setManualDiskInput('/dev/vdb')
    expect(store.selectedDisks).toHaveLength(1)

    store.setManualDiskInput('')
    expect(store.selectedDisks).toHaveLength(0)
    expect(store.manualInputError).toBeNull()
  })

  it('clearManualInput resets manual state only, preserves auto state', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.setDiscoveryError('should stay')

    store.setEntryMode('manual')
    store.setManualDiskInput('/dev/vdb\n/dev/vdc')

    store.clearManualInput()
    expect(store.manualDiskInput).toBe('')
    expect(store.manualInputError).toBeNull()
    expect(store.selectedDisks).toHaveLength(0)
    expect(store.discoveryError).toBe('should stay')
    expect(store.autoSelectedDisks).toHaveLength(1)

    store.setEntryMode('auto')
    expect(store.selectedDisks).toHaveLength(1)
    expect(store.selectedDisks[0].path).toBe('/dev/sda')
  })

  it('toggleDisk is no-op in manual mode, does not affect autoSelectedDisks', () => {
    const store = useDisksStore()
    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.setEntryMode('manual')

    store.toggleDisk('/dev/vdb', 'ydb_disk_2')
    expect(store.autoSelectedDisks).toHaveLength(1)
    expect(store.autoSelectedDisks[0].path).toBe('/dev/sda')

    store.setEntryMode('auto')
    expect(store.selectedDisks).toHaveLength(1)
    expect(store.selectedDisks[0].path).toBe('/dev/sda')
  })

  it('reconcileSelection removes disks not in discovery results and renumbers labels', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.toggleDisk('/dev/sdb', 'ydb_disk_2')
    store.toggleDisk('/dev/sdc', 'ydb_disk_3')
    expect(store.autoSelectedDisks).toHaveLength(3)

    store.setDiskResults('host1', [
      { name: 'sda', path: '/dev/sda', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
      { name: 'sdb', path: '/dev/sdb', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    store.reconcileSelection()

    expect(store.autoSelectedDisks).toHaveLength(2)
    expect(store.autoSelectedDisks[0].path).toBe('/dev/sda')
    expect(store.autoSelectedDisks[0].label).toBe('ydb_disk_1')
    expect(store.autoSelectedDisks[1].path).toBe('/dev/sdb')
    expect(store.autoSelectedDisks[1].label).toBe('ydb_disk_2')
  })

  it('reconcileSelection preserves all disks when all match discovery', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.toggleDisk('/dev/sdb', 'ydb_disk_2')

    store.setDiskResults('host1', [
      { name: 'sda', path: '/dev/sda', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
      { name: 'sdb', path: '/dev/sdb', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    store.reconcileSelection()

    expect(store.autoSelectedDisks).toHaveLength(2)
    expect(store.autoSelectedDisks[0].path).toBe('/dev/sda')
    expect(store.autoSelectedDisks[1].path).toBe('/dev/sdb')
  })

  it('reconcileSelection is no-op when diskResults is empty', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.toggleDisk('/dev/sdb', 'ydb_disk_2')
    expect(store.autoSelectedDisks).toHaveLength(2)

    store.reconcileSelection()

    expect(store.autoSelectedDisks).toHaveLength(2)
  })

  it('reconcileSelection collects paths from all hosts', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.toggleDisk('/dev/sdb', 'ydb_disk_2')
    store.toggleDisk('/dev/sdc', 'ydb_disk_3')

    store.setDiskResults('host1', [
      { name: 'sda', path: '/dev/sda', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])
    store.setDiskResults('host2', [
      { name: 'sdb', path: '/dev/sdb', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    store.reconcileSelection()

    expect(store.autoSelectedDisks).toHaveLength(2)
    const paths = store.autoSelectedDisks.map(d => d.path)
    expect(paths).toContain('/dev/sda')
    expect(paths).toContain('/dev/sdb')
  })

  it('reconcileSelection removes all when nothing matches', () => {
    const store = useDisksStore()

    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.toggleDisk('/dev/sdb', 'ydb_disk_2')

    store.setDiskResults('host1', [
      { name: 'nvme0n1', path: '/dev/nvme0n1', type: 'NVMe', size: '1T', sizeBytes: 1024 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    store.reconcileSelection()

    expect(store.autoSelectedDisks).toHaveLength(0)
  })
})
