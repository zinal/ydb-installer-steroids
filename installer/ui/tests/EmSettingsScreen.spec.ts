import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

import EmSettingsScreen from '@/views/EmSettingsScreen.vue'
import { useEmStore } from '@/stores/em'
import { useSessionStore } from '@/stores/session'

const getEMInventoryMock = vi.fn()
const updateEMInventoryMock = vi.fn()
const checkHostsMock = vi.fn()
const validateFileMock = vi.fn()

vi.mock('@/api/client', () => ({
  api: {
    getEMInventory: (...args: unknown[]) => getEMInventoryMock(...args),
    updateEMInventory: (...args: unknown[]) => updateEMInventoryMock(...args),
    checkHosts: (...args: unknown[]) => checkHostsMock(...args),
    validateFile: (...args: unknown[]) => validateFileMock(...args),
  },
}))

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
}

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      common: { step: 'Step' },
      screens: { '7.1': 'YDB EM settings' },
      emSettings: {
        heading: 'YDB EM Settings',
        subtitle: 'Compact EM configuration',
        connectionLabel: 'Connection',
        endpoint: 'Endpoint',
        database: 'Database',
        connectionHint: 'Bridged from the databases step',
        emHostsLabel: 'EM host',
        checkHosts: 'Check hosts',
        duplicateHosts: 'Duplicate hosts',
        checkRequired: 'Run the host check before continuing',
        distributionLabel: 'Distribution',
      },
      hosts: {
        driftOk: 'Time drift {drift}ms — OK',
        driftWarning: 'Time drift {drift}ms exceeds {threshold}ms',
        driftError: 'Time drift {drift}ms exceeds {threshold}ms',
        driftNtpMissing: 'NTP is not configured',
      },
    },
  },
})

function mountScreen() {
  return mount(EmSettingsScreen, {
    global: { plugins: [i18n] },
  })
}

describe('EmSettingsScreen', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getEMInventoryMock.mockReset().mockResolvedValue({ em_inventory_found: false, data: null })
    updateEMInventoryMock.mockReset().mockResolvedValue({ status: 'ok' })
    checkHostsMock.mockReset()
    validateFileMock.mockReset().mockResolvedValue({ valid: false })
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders the bridged connection summary from the EM store', async () => {
    const em = useEmStore()
    em.setDbConnectionEndpoint('grpcs://ydb-1.example.com:2135')
    em.setDbConnectionDBRoot('/Root')
    em.setDbConnectionDB('ydb_em')

    const wrapper = mountScreen()
    await flushPromises()

    const summary = wrapper.findAll('.summary-val').map(el => el.text())
    expect(summary).toContain('grpcs://ydb-1.example.com:2135')
    expect(summary).toContain('/Root/ydb_em')
  })

  it('keeps at least one EM host row', async () => {
    const wrapper = mountScreen()
    await flushPromises()
    expect(useEmStore().hosts.length).toBeGreaterThanOrEqual(1)
    wrapper.unmount()
  })

  it('disables the check button until a host is filled', async () => {
    const wrapper = mountScreen()
    await flushPromises()

    const checkBtn = wrapper.find('.check-button')
    expect(checkBtn.attributes('disabled')).toBeDefined()

    useEmStore().updateHost(0, 'em-1.example.com')
    await nextTick()
    expect(checkBtn.attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('checks hosts with EM ports and ssh config defaults', async () => {
    checkHostsMock.mockResolvedValueOnce({
      results: [{ host: 'em-1.example.com', ssh: true, python: true }],
      timeSync: null,
    })

    const wrapper = mountScreen()
    await flushPromises()
    const em = useEmStore()
    em.updateHost(0, 'em-1.example.com')
    await nextTick()

    await wrapper.find('.check-button').trigger('click')
    await flushPromises()

    expect(checkHostsMock).toHaveBeenCalledTimes(1)
    const [hosts, ssh, ports] = checkHostsMock.mock.calls[0]
    expect(hosts).toEqual(['em-1.example.com'])
    expect(ssh.login).toBe('root')
    expect(ssh.auth_type).toBe('key')
    expect(ports).toEqual([8780, 8789, 8787])
    wrapper.unmount()
  })

  it('grants canGoNext only with host + passed checks + valid dist + params', async () => {
    checkHostsMock.mockResolvedValue({
      results: [{ host: 'em-1.example.com', ssh: true, python: true }],
      timeSync: null,
    })

    const wrapper = mountScreen()
    await flushPromises()
    const em = useEmStore()
    const session = useSessionStore()

    expect(session.canGoNext).toBe(false)

    em.updateHost(0, 'em-1.example.com')
    em.setDbConnectionDBRoot('/Root')
    em.setDbConnectionDB('ydb_em')
    em.archiveValid = true
    em.metaUIArchiveValid = true
    await nextTick()

    expect(session.canGoNext).toBe(false)
    expect(wrapper.find('.validation-message--warning').exists()).toBe(true)

    await wrapper.find('.check-button').trigger('click')
    await flushPromises()

    expect(session.canGoNext).toBe(true)
    wrapper.unmount()
  })

  it('warns about duplicate EM hosts loaded from inventory', async () => {
    const wrapper = mountScreen()
    await flushPromises()
    const em = useEmStore()

    em.applyFromEMConfig({
      hosts: [{ fqdn: 'em-1.example.com' }, { fqdn: 'em-1.example.com' }],
    })
    await nextTick()

    expect(em.hasDuplicates.length).toBeGreaterThan(0)
    expect(wrapper.text()).toContain('Duplicate hosts')
    wrapper.unmount()
  })
})
