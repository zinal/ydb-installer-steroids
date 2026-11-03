import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h, nextTick } from 'vue'
import { useEMInventorySync } from '@/composables/useEMInventorySync'
import { useEmStore } from '@/stores/em'
import { useHostsStore } from '@/stores/hosts'

const getEMInventoryMock = vi.fn()
const updateEMInventoryMock = vi.fn()

vi.mock('@/api/client', () => ({
  api: {
    getEMInventory: (...args: unknown[]) => getEMInventoryMock(...args),
    updateEMInventory: (...args: unknown[]) => updateEMInventoryMock(...args),
  },
}))

// Fake WebSocket: captures instances and lets tests fire onmessage manually.
class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  url: string
  onmessage: ((ev: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  closed = false
  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }
  close() {
    this.closed = true
  }
  emitMessage() {
    this.onmessage?.({ data: '' })
  }
}

function mountWithSync() {
  const Comp = defineComponent({
    setup() {
      useEMInventorySync()
      return () => h('div')
    },
  })
  return mount(Comp)
}

/** Flush the catch-up write that onMounted schedules when the local payload
 *  differs from the server fingerprint. Returns the number of PUTs made. */
async function flushCatchUpWrite(): Promise<number> {
  await nextTick()
  vi.advanceTimersByTime(500)
  await flushPromises()
  const calls = updateEMInventoryMock.mock.calls.length
  updateEMInventoryMock.mockClear()
  return calls
}

describe('useEMInventorySync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getEMInventoryMock.mockReset().mockResolvedValue({
      em_inventory_found: false,
      data: null,
    })
    updateEMInventoryMock.mockReset().mockResolvedValue({ status: 'ok' })
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('applies initial EM inventory and raises the found flag', async () => {
    getEMInventoryMock.mockResolvedValueOnce({
      em_inventory_found: true,
      data: {
        hosts: [{ fqdn: 'em-1.example.com' }],
        cluster_name: 'em-cluster',
      },
    })

    mountWithSync()
    await flushPromises()

    const em = useEmStore()
    expect(em.emInventoryFound).toBe(true)
    expect(em.hosts).toEqual([{ fqdn: 'em-1.example.com' }])
    expect(em.clusterName).toBe('em-cluster')
  })

  it('PUTs EM config with az_locations derived from YDB DC hosts', async () => {
    const hostsStore = useHostsStore()
    hostsStore.dcHosts['DC-1'] = ['node-a.example.com']
    hostsStore.dcHosts['DC-2'] = ['node-b.example.com']

    const wrapper = mountWithSync()
    await flushPromises()
    await flushCatchUpWrite()

    useEmStore().clusterName = 'new-cluster'
    await nextTick()
    vi.advanceTimersByTime(500)
    await flushPromises()

    expect(updateEMInventoryMock).toHaveBeenCalledTimes(1)
    const [payload] = updateEMInventoryMock.mock.calls[0]
    expect(payload.cluster_name).toBe('new-cluster')
    expect(payload.az_locations).toEqual(['dc-1', 'dc-2'])
    wrapper.unmount()
  })

  it('fingerprint skips no-op PUTs (validation ephemerals do not write)', async () => {
    const wrapper = mountWithSync()
    await flushPromises()
    await flushCatchUpWrite()

    const em = useEmStore()
    em.archiveValid = true
    em.archiveSize = '1.2GB'
    await nextTick()
    vi.advanceTimersByTime(500)
    await flushPromises()

    expect(updateEMInventoryMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('hosts changes refresh az_locations only when EM inventory exists', async () => {
    const wrapper = mountWithSync()
    await flushPromises()
    await flushCatchUpWrite()

    useHostsStore().addDCHost('DC-1', 'node-a.example.com')
    await nextTick()
    vi.advanceTimersByTime(500)
    await flushPromises()
    expect(updateEMInventoryMock).not.toHaveBeenCalled()

    useEmStore().setEMInventoryFound(true)
    useHostsStore().addDCHost('DC-2', 'node-b.example.com')
    await nextTick()
    vi.advanceTimersByTime(500)
    await flushPromises()

    expect(updateEMInventoryMock).toHaveBeenCalledTimes(1)
    const [payload] = updateEMInventoryMock.mock.calls[0]
    expect(payload.az_locations).toEqual(['dc-1', 'dc-2'])
    wrapper.unmount()
  })

  it('reloads EM inventory on WS notification and realigns the fingerprint', async () => {
    const wrapper = mountWithSync()
    await flushPromises()
    await flushCatchUpWrite()

    const ws = FakeWebSocket.instances[0]
    expect(ws).toBeTruthy()

    getEMInventoryMock.mockResolvedValueOnce({
      em_inventory_found: true,
      data: {
        hosts: [{ fqdn: 'em-ws.example.com' }],
        cluster_name: 'ws-cluster',
      },
    })
    ws.emitMessage()
    await flushPromises()

    const em = useEmStore()
    expect(em.emInventoryFound).toBe(true)
    expect(em.hosts).toEqual([{ fqdn: 'em-ws.example.com' }])
    expect(em.clusterName).toBe('ws-cluster')

    vi.advanceTimersByTime(1000)
    await flushPromises()
    expect(updateEMInventoryMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
