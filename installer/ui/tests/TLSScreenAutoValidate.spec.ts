// Regression test for the "TLS validation fires by itself, without clicking
// the button" bug.
//
// Root cause: TLSScreen used to auto-validate on screen load (originally via
// a one-shot onMounted, later via a guarded watcher) whenever a cert path
// was present. The user reported validation firing by itself at startup —
// any automatic validation is unwanted; validation must only run when the
// user explicitly clicks the "Validate" button.
//
// Fix: the auto-validate-on-load watcher was removed entirely. This test
// asserts that neither typing a cert path by hand NOR loading one from
// inventory.yaml triggers api.tlsValidate() on its own.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import enMessages from '@/locales/en.json'

const tlsValidateMock = vi.fn().mockResolvedValue({
  valid: true,
  ca_valid: true,
  node_files: [],
  missing: [],
  errors: [],
  total_nodes: 1,
  found_nodes: 1,
})
const tlsGenerateMock = vi.fn().mockResolvedValue({
  success: true,
  output_dir: '/opt/ydb/TLS/CA/certs/certs-per-host',
  ca_file: '/opt/ydb/TLS/CA/certs/ca.crt',
  node_files: [],
})

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn(() => new Promise(() => { /* never resolves */ })),
    updateInventory: vi.fn().mockResolvedValue({ status: 'ok', data: null }),
    tlsValidate: (...args: unknown[]) => tlsValidateMock(...args),
    tlsGenerate: (...args: unknown[]) => tlsGenerateMock(...args),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { useHostsStore } from '@/stores/hosts'
import { useTlsStore } from '@/stores/tls'
import { useTopologyStore } from '@/stores/topology'
import TLSScreen from '@/views/TLSScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: enMessages },
})

function mountTls() {
  return mount(TLSScreen, { global: { plugins: [i18n] } })
}

describe('TLSScreen — auto-validate only fires for inventory-loaded cert path', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    tlsValidateMock.mockClear()
    tlsGenerateMock.mockClear()
  })

  it('does NOT auto-validate when the user types a cert path by hand', async () => {
    const hostsStore = useHostsStore()
    hostsStore.addDCHost('DC-1', 'node1.example.com')

    const tlsStore = useTlsStore()
    tlsStore.setMode('existing')

    const wrapper = mountTls()
    await wrapper.vm.$nextTick()

    const pathInput = wrapper.find('.path-input input')
    expect(pathInput.exists()).toBe(true)
    await pathInput.setValue('/etc/ydb/certs')
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(tlsValidateMock).not.toHaveBeenCalled()
  })

  it('does NOT auto-validate when a cert path is loaded from inventory either', async () => {
    const hostsStore = useHostsStore()
    hostsStore.addDCHost('DC-1', 'node1.example.com')

    const tlsStore = useTlsStore()
    tlsStore.applyFromInventory({ tls: { cert_path: '/etc/ydb/certs' } })

    const wrapper = mountTls()
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(tlsValidateMock).not.toHaveBeenCalled()
  })
})
