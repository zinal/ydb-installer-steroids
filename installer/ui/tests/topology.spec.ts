import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTopologyStore } from '@/stores/topology'
import { TOPOLOGIES, minDisksToSelect } from '@/config/topologies'

describe('useTopologyStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('starts with no topology selected', () => {
    const store = useTopologyStore()
    expect(store.selected).toBe(null)
    expect(store.hostsCount).toBe(0)
  })

  it('can select an available topology', () => {
    const store = useTopologyStore()
    store.select('mirror-3-dc')
    expect(store.selected).toBe('mirror-3-dc')
  })

  it('cannot select an unavailable topology', () => {
    const store = useTopologyStore()
    // bridge-2-dc and none are still marked unavailable (TODO) in the config.
    store.select('bridge-2-dc')
    expect(store.selected).toBe(null)
  })

  it('canSelect returns true for available topologies', () => {
    const store = useTopologyStore()
    // mirror-3-dc and block-4-2 are implemented and available.
    expect(store.canSelect('mirror-3-dc')).toBe(true)
    expect(store.canSelect('block-4-2')).toBe(true)
    // bridge-2-dc and none are not yet available.
    expect(store.canSelect('bridge-2-dc')).toBe(false)
    expect(store.canSelect('none')).toBe(false)
  })

  it('currentTopology returns the selected topology object', () => {
    const store = useTopologyStore()
    store.select('mirror-3-dc')
    expect(store.currentTopology).toEqual(TOPOLOGIES['mirror-3-dc'])
  })

  it('currentTopology returns null when nothing selected', () => {
    const store = useTopologyStore()
    expect(store.currentTopology).toBe(null)
  })

  it('setHostsCount updates the hosts count', () => {
    const store = useTopologyStore()
    store.setHostsCount(5)
    expect(store.hostsCount).toBe(5)
  })

  it('applyFromInventory loads topology from inventory data', () => {
    const store = useTopologyStore()
    store.applyFromInventory({ topology: { type: 'mirror-3-dc' } })
    expect(store.selected).toBe('mirror-3-dc')
  })

  it('applyFromInventory ignores invalid topology', () => {
    const store = useTopologyStore()
    store.applyFromInventory({ topology: { type: 'invalid-topology' } })
    expect(store.selected).toBe(null)
  })

  it('applyFromInventory handles missing topology', () => {
    const store = useTopologyStore()
    store.applyFromInventory({})
    expect(store.selected).toBe(null)
  })

  it('toInventory returns empty object when nothing selected', () => {
    const store = useTopologyStore()
    expect(store.toInventory()).toEqual({})
  })

  it('toInventory returns topology object when selected', () => {
    const store = useTopologyStore()
    store.select('mirror-3-dc')
    expect(store.toInventory()).toEqual({ topology: 'mirror-3-dc' })
  })
})

describe('minDisksToSelect', () => {
  it('requires 3 disks for mirror-3-dc below 9 hosts', () => {
    expect(minDisksToSelect('mirror-3-dc', 3)).toBe(3)
    expect(minDisksToSelect('mirror-3-dc', 8)).toBe(3)
  })

  it('requires 1 disk for mirror-3-dc at 9+ hosts', () => {
    expect(minDisksToSelect('mirror-3-dc', 9)).toBe(1)
    expect(minDisksToSelect('mirror-3-dc', 12)).toBe(1)
  })

  it('requires 1 disk for other topologies regardless of host count', () => {
    expect(minDisksToSelect('block-4-2', 3)).toBe(1)
    expect(minDisksToSelect(null, 3)).toBe(1)
  })
})
