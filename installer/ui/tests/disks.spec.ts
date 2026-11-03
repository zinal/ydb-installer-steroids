import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useDisksStore } from '@/stores/disks'
import { MIN_RECOMMENDED_DISK_SIZE_BYTES } from '@/types/disks'
import type { InventoryData } from '@/types/inventory'

describe('useDisksStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('has initial empty state', () => {
    const store = useDisksStore()
    expect(store.allowFormat).toBe(false)
    expect(store.autoSelectedDisks).toEqual([])
    expect(store.selectedDisks).toEqual([])
    expect(store.entryMode).toBe('auto')
    expect(store.manualDiskInput).toBe('')
    expect(store.manualInputError).toBeNull()
  })

  it('applies inventory data to autoSelectedDisks without changing mode', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    expect(store.entryMode).toBe('manual')

    const inv: InventoryData = {
      disks: {
        allow_format: true,
        selected_disks: [
          { path: '/dev/sda', label: 'disk1' },
          { path: '/dev/sdb', label: 'disk2' },
        ],
      },
    }
    store.applyFromInventory(inv)
    expect(store.allowFormat).toBe(true)
    expect(store.autoSelectedDisks).toHaveLength(2)
    expect(store.autoSelectedDisks[0].path).toBe('/dev/sda')
    expect(store.entryMode).toBe('manual')
  })

  it('selectedDisks getter returns autoSelectedDisks in auto mode', () => {
    const store = useDisksStore()
    store.toggleDisk('/dev/sda', 'disk1')
    store.toggleDisk('/dev/sdb', 'disk2')
    expect(store.selectedDisks).toHaveLength(2)
    expect(store.selectedDisks[0].path).toBe('/dev/sda')
  })

  it('toggles disk selection in auto mode', () => {
    const store = useDisksStore()
    store.toggleDisk('/dev/sda', 'disk1')
    expect(store.autoSelectedDisks).toHaveLength(1)
    expect(store.selectedDisks).toHaveLength(1)

    store.toggleDisk('/dev/sda', 'disk1')
    expect(store.autoSelectedDisks).toHaveLength(0)
    expect(store.selectedDisks).toHaveLength(0)
  })

  it('toggleDisk is a no-op in manual mode', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    store.toggleDisk('/dev/sda', 'disk1')
    expect(store.autoSelectedDisks).toHaveLength(0)
    expect(store.selectedDisks).toHaveLength(0)
  })

  it('sets allow format flag', () => {
    const store = useDisksStore()
    store.setAllowFormat(true)
    expect(store.allowFormat).toBe(true)
  })

  it('switches to manual entry mode without clearing auto state', () => {
    const store = useDisksStore()
    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.setDiscoveryError('test error')
    expect(store.autoSelectedDisks).toHaveLength(1)

    store.setEntryMode('manual')
    expect(store.entryMode).toBe('manual')
    expect(store.autoSelectedDisks).toHaveLength(1)
    expect(store.discoveryError).toBe('test error')
    expect(store.manualDiskInput).toBe('')
    expect(store.manualInputError).toBeNull()
  })

  it('switches back to auto mode without clearing manual state', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    store.setManualDiskInput('/dev/vdb\n/dev/vdc')
    expect(store.selectedDisks).toHaveLength(2)

    store.setEntryMode('auto')
    expect(store.entryMode).toBe('auto')
    expect(store.manualDiskInput).toBe('/dev/vdb\n/dev/vdc')
    expect(store.manualInputError).toBeNull()
  })

  it('selectedDisks getter parses manual input into sequential labels', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    store.setManualDiskInput('/dev/vdb\n/dev/vdc\n/dev/vdd')

    expect(store.selectedDisks).toHaveLength(3)
    expect(store.selectedDisks[0]).toEqual({ path: '/dev/vdb', label: 'ydb_disk_1' })
    expect(store.selectedDisks[1]).toEqual({ path: '/dev/vdc', label: 'ydb_disk_2' })
    expect(store.selectedDisks[2]).toEqual({ path: '/dev/vdd', label: 'ydb_disk_3' })
    expect(store.manualInputError).toBeNull()
  })

  it('trims whitespace and skips empty lines in manual input', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    store.setManualDiskInput('  /dev/vdb  \n\n  /dev/vdc  \n\n')

    expect(store.selectedDisks).toHaveLength(2)
    expect(store.selectedDisks[0].path).toBe('/dev/vdb')
    expect(store.selectedDisks[1].path).toBe('/dev/vdc')
  })

  it('rejects invalid device paths in manual input', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    store.setManualDiskInput('/dev/vdb\ninvalid-path\n/dev/vdd')

    expect(store.manualInputError).toContain('Invalid device path')
    expect(store.selectedDisks).toHaveLength(3)
  })

  it('rejects duplicate device paths in manual input', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    store.setManualDiskInput('/dev/vdb\n/dev/vdc\n/dev/vdb')

    expect(store.manualInputError).toContain('Duplicate device path')
    expect(store.selectedDisks).toHaveLength(3)
  })

  it('clears manual input state without affecting auto state', () => {
    const store = useDisksStore()
    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    expect(store.autoSelectedDisks).toHaveLength(1)

    store.setEntryMode('manual')
    store.setManualDiskInput('/dev/vdb\n/dev/vdc')

    store.clearManualInput()
    expect(store.manualDiskInput).toBe('')
    expect(store.manualInputError).toBeNull()
    expect(store.selectedDisks).toHaveLength(0)
    expect(store.autoSelectedDisks).toHaveLength(1)

    store.setEntryMode('auto')
    expect(store.selectedDisks).toHaveLength(1)
    expect(store.selectedDisks[0].path).toBe('/dev/sda')
  })

  it('getManualDisksCount returns 0 in auto mode', () => {
    const store = useDisksStore()
    expect(store.getManualDisksCount).toBe(0)
  })

  it('getManualDisksCount returns parsed count in manual mode', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    store.setManualDiskInput('/dev/vdb\n/dev/vdc\n/dev/vdd')
    expect(store.getManualDisksCount).toBe(3)
  })

  it('does nothing when switching to the same mode', () => {
    const store = useDisksStore()
    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    store.setEntryMode('auto')
    expect(store.autoSelectedDisks).toHaveLength(1)
  })

  it('isDiskSelected checks selectedDisks getter', () => {
    const store = useDisksStore()
    store.toggleDisk('/dev/sda', 'ydb_disk_1')
    expect(store.isDiskSelected('/dev/sda')).toBe(true)
    expect(store.isDiskSelected('/dev/vdb')).toBe(false)
  })

  it('MIN_RECOMMENDED_DISK_SIZE_BYTES equals 80 GB', () => {
    expect(MIN_RECOMMENDED_DISK_SIZE_BYTES).toBe(80 * 1024 * 1024 * 1024)
  })

  it('applyFromInventory then discovery removes stale disks', () => {
    const store = useDisksStore()

    const inv: InventoryData = {
      disks: {
        allow_format: true,
        selected_disks: [
          { path: '/dev/sda', label: 'ydb_disk_1' },
          { path: '/dev/sdb', label: 'ydb_disk_2' },
          { path: '/dev/sdc', label: 'ydb_disk_3' },
        ],
      },
    }
    store.applyFromInventory(inv)
    expect(store.autoSelectedDisks).toHaveLength(3)

    store.setDiskResults('host1', [
      { name: 'sda', path: '/dev/sda', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
      { name: 'sdb', path: '/dev/sdb', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    store.reconcileSelection()

    expect(store.autoSelectedDisks).toHaveLength(2)
    const paths = store.autoSelectedDisks.map(d => d.path)
    expect(paths).toContain('/dev/sda')
    expect(paths).toContain('/dev/sdb')
    expect(paths).not.toContain('/dev/sdc')
  })

  it('applyFromInventory filters out undiscovered disks and renumbers labels', () => {
    const store = useDisksStore()

    store.setDiskResults('host1', [
      { name: 'sda', path: '/dev/sda', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
      { name: 'sdb', path: '/dev/sdb', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    const inv: InventoryData = {
      disks: {
        allow_format: true,
        selected_disks: [
          { path: '/dev/sda', label: 'ydb_disk_1' },
          { path: '/dev/sdb', label: 'ydb_disk_2' },
          { path: '/dev/sdc', label: 'ydb_disk_3' },
        ],
      },
    }
    store.applyFromInventory(inv)

    expect(store.autoSelectedDisks).toHaveLength(2)
    const paths = store.autoSelectedDisks.map(d => d.path)
    expect(paths).toContain('/dev/sda')
    expect(paths).toContain('/dev/sdb')
    expect(paths).not.toContain('/dev/sdc')
    expect(store.autoSelectedDisks[0].label).toBe('ydb_disk_1')
    expect(store.autoSelectedDisks[1].label).toBe('ydb_disk_2')
  })

  it('applyFromInventory renumbers labels when middle disk is removed', () => {
    const store = useDisksStore()

    store.setDiskResults('host1', [
      { name: 'sda', path: '/dev/sda', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
      { name: 'sdc', path: '/dev/sdc', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    const inv: InventoryData = {
      disks: {
        allow_format: false,
        selected_disks: [
          { path: '/dev/sda', label: 'ydb_disk_1' },
          { path: '/dev/sdb', label: 'ydb_disk_2' },
          { path: '/dev/sdc', label: 'ydb_disk_3' },
        ],
      },
    }
    store.applyFromInventory(inv)

    expect(store.autoSelectedDisks).toHaveLength(2)
    expect(store.autoSelectedDisks[0].path).toBe('/dev/sda')
    expect(store.autoSelectedDisks[0].label).toBe('ydb_disk_1')
    expect(store.autoSelectedDisks[1].path).toBe('/dev/sdc')
    expect(store.autoSelectedDisks[1].label).toBe('ydb_disk_2')
  })

  it('applyFromInventory loads all disks when diskResults is empty (before discovery)', () => {
    const store = useDisksStore()

    expect(store.diskResults.size).toBe(0)

    const inv: InventoryData = {
      disks: {
        allow_format: true,
        selected_disks: [
          { path: '/dev/sda', label: 'ydb_disk_1' },
          { path: '/dev/sdb', label: 'ydb_disk_2' },
          { path: '/dev/sdc', label: 'ydb_disk_3' },
        ],
      },
    }
    store.applyFromInventory(inv)

    expect(store.autoSelectedDisks).toHaveLength(3)
  })

  it('applyFromInventory preserves allowFormat flag', () => {
    const store = useDisksStore()

    store.setDiskResults('host1', [
      { name: 'sda', path: '/dev/sda', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    const inv: InventoryData = {
      disks: {
        allow_format: true,
        selected_disks: [
          { path: '/dev/sda', label: 'ydb_disk_1' },
        ],
      },
    }
    store.applyFromInventory(inv)

    expect(store.allowFormat).toBe(true)
  })

  it('applyFromInventory does not change entryMode', () => {
    const store = useDisksStore()
    store.setEntryMode('manual')
    expect(store.entryMode).toBe('manual')

    store.setDiskResults('host1', [
      { name: 'sda', path: '/dev/sda', type: 'SSD', size: '500G', sizeBytes: 500 * 1024 * 1024 * 1024, mounted: false, mountPoint: '', filesystem: '', isSystemDisk: false },
    ])

    const inv: InventoryData = {
      disks: {
        allow_format: false,
        selected_disks: [{ path: '/dev/sda', label: 'ydb_disk_1' }],
      },
    }
    store.applyFromInventory(inv)

    expect(store.entryMode).toBe('manual')
  })
})
