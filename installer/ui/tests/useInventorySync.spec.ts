import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h, nextTick } from 'vue'
import { useInventorySync } from '@/composables/useInventorySync'
import { useHostsStore } from '@/stores/hosts'
import { useTopologyStore } from '@/stores/topology'

const getInventoryMock = vi.fn()
const updateInventoryMock = vi.fn()

vi.mock('@/api/client', () => ({
  api: {
    getInventory: (...args: unknown[]) => getInventoryMock(...args),
    updateInventory: (...args: unknown[]) => updateInventoryMock(...args),
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
      useInventorySync()
      return () => h('div')
    },
  })
  return mount(Comp)
}

describe('useInventorySync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getInventoryMock.mockReset().mockResolvedValue({
      inventory_found: false,
      config_found: false,
      data: null,
    })
    updateInventoryMock.mockReset().mockResolvedValue({ status: 'ok' })
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('applies initial inventory from the API to the stores', async () => {
    getInventoryMock.mockResolvedValueOnce({
      inventory_found: true,
      config_found: false,
      data: {
        dc1: ['node-a.example.com'],
        dc2: [],
        dc3: [],
        ssh: { login: 'root', auth_type: 'key' },
        topology: { type: 'mirror-3-dc' },
      },
    })

    mountWithSync()
    await flushPromises()

    const hosts = useHostsStore()
    expect(hosts.dcHosts['DC-1']).toEqual(['node-a.example.com'])
    expect(hosts.ssh?.login).toBe('root')
    expect(useTopologyStore().selected).toBe('mirror-3-dc')
  })

  it('debounces store mutations into a single PUT with collected data', async () => {
    const wrapper = mountWithSync()
    await flushPromises()

    const hosts = useHostsStore()
    hosts.addDCHost('DC-1', 'node-a.example.com')
    hosts.addDCHost('DC-2', 'node-b.example.com')
    await nextTick()

    expect(updateInventoryMock).not.toHaveBeenCalled()
    vi.advanceTimersByTime(500)
    await flushPromises()

    expect(updateInventoryMock).toHaveBeenCalledTimes(1)
    const [payload] = updateInventoryMock.mock.calls[0]
    expect(payload.dc1).toEqual(['node-a.example.com'])
    expect(payload.dc2).toEqual(['node-b.example.com'])
    expect(payload.hosts).toEqual(['node-a.example.com', 'node-b.example.com'])
    wrapper.unmount()
  })

  it('filters invalid/placeholder hostnames out of the payload', async () => {
    const wrapper = mountWithSync()
    await flushPromises()

    useHostsStore().addDCHost('DC-1', '')
    await nextTick()
    vi.advanceTimersByTime(500)
    await flushPromises()

    if (updateInventoryMock.mock.calls.length > 0) {
      const [payload] = updateInventoryMock.mock.calls[0]
      expect(payload.dc1).toBeUndefined()
      expect(payload.hosts).toBeUndefined()
    }
    wrapper.unmount()
  })

  it('reloads inventory on WS notification without echoing a write back', async () => {
    const wrapper = mountWithSync()
    await flushPromises()

    const ws = FakeWebSocket.instances[0]
    expect(ws).toBeTruthy()
    expect(ws.url).toContain('/ws/inventory')

    getInventoryMock.mockResolvedValueOnce({
      inventory_found: true,
      config_found: false,
      data: { dc1: ['ws-host.example.com'], dc2: [], dc3: [] },
    })
    ws.emitMessage()
    await flushPromises()

    expect(useHostsStore().dcHosts['DC-1']).toEqual(['ws-host.example.com'])

    vi.advanceTimersByTime(1000)
    await flushPromises()
    expect(updateInventoryMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('does not lose a write that happens while a PUT is in flight', async () => {
    const wrapper = mountWithSync()
    await flushPromises()

    let resolveFirstPut: (v: unknown) => void
    updateInventoryMock
      .mockImplementationOnce(() => new Promise((res) => { resolveFirstPut = res }))
      .mockResolvedValue({ status: 'ok' })

    const hosts = useHostsStore()
    hosts.addDCHost('DC-1', 'node-a.example.com')
    await nextTick()
    vi.advanceTimersByTime(500)
    await nextTick()
    expect(updateInventoryMock).toHaveBeenCalledTimes(1)

    // Keystroke lands while the first PUT is still pending — it must be
    // persisted by a trailing flush instead of being dropped.
    hosts.addDCHost('DC-2', 'node-b.example.com')
    await nextTick()

    resolveFirstPut!({ status: 'ok' })
    await flushPromises()

    vi.advanceTimersByTime(500)
    await flushPromises()

    expect(updateInventoryMock).toHaveBeenCalledTimes(2)
    const [, secondPayload] = updateInventoryMock.mock.calls
    expect(secondPayload[0].dc1).toEqual(['node-a.example.com'])
    expect(secondPayload[0].dc2).toEqual(['node-b.example.com'])
    expect(secondPayload[0].hosts).toEqual(['node-a.example.com', 'node-b.example.com'])
    wrapper.unmount()
  })

  it('flushes a keystroke that lands during a WS reload', async () => {
    const wrapper = mountWithSync()
    await flushPromises()

    const ws = FakeWebSocket.instances[0]
    let resolveGet: (v: unknown) => void
    getInventoryMock.mockImplementationOnce(() => new Promise((res) => { resolveGet = res }))

    ws.emitMessage()
    await nextTick()

    // Keystroke arrives while the reload fetch is still pending.
    const hosts = useHostsStore()
    hosts.addDCHost('DC-1', 'typed-during-reload.example.com')
    await nextTick()

    resolveGet!({
      inventory_found: true,
      config_found: false,
      data: { dc1: ['server.example.com'], dc2: [], dc3: [] },
    })
    await flushPromises()

    vi.advanceTimersByTime(500)
    await flushPromises()

    expect(updateInventoryMock).toHaveBeenCalledTimes(1)
    const [payload] = updateInventoryMock.mock.calls[0]
    expect(payload.dc1).toEqual(['typed-during-reload.example.com'])
    wrapper.unmount()
  })

  it('cancels pending writes on unmount', async () => {
    const wrapper = mountWithSync()
    await flushPromises()

    useHostsStore().addDCHost('DC-1', 'node-a.example.com')
    await nextTick()
    wrapper.unmount()

    vi.advanceTimersByTime(1000)
    await flushPromises()
    expect(updateInventoryMock).not.toHaveBeenCalled()
  })
})
