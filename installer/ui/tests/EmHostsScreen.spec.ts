import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn().mockResolvedValue({ inventory_found: false, data: null }),
    updateInventory: vi.fn().mockResolvedValue({ status: 'ok' }),
    getEMInventory: vi.fn().mockResolvedValue({ em_inventory_found: false, data: null }),
    updateEMInventory: vi.fn().mockResolvedValue({ status: 'ok' }),
    checkHosts: vi.fn().mockResolvedValue({ results: [] }),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { useEmStore } from '@/stores/em'
import { useSessionStore } from '@/stores/session'
import EmHostsScreen from '@/views/EmHostsScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      emHosts: {
        heading: 'YDB EM Hosts',
        subtitle: 'Select hosts for YDB EM.',
        emHostsLabel: 'EM Hosts',
        hostnameLabel: 'Hostname / FQDN',
        hostnamePlaceholder: 'hostname.example.com',
        addHost: 'Add host',
        removeHost: 'Remove host',
        minHostsRequired: 'Enter the EM host FQDN to continue',
        duplicateWarning: 'Duplicate hostnames detected.',
        sshSettings: 'SSH Settings',
        checkHosts: 'Check Hosts',
        checkRequired: 'Please run host connectivity checks',
        checksFailed: 'All hosts must pass checks',
      },
      ssh: {
        login: 'Login',
        specifyKey: 'Specify key',
        keyPath: 'Key path',
        useJumpHost: 'Use jump host',
        jumpHostFqdn: 'Jump host',
        advancedSettings: 'Advanced',
        customSshArgs: 'Custom SSH args',
      },
      hosts: {
        driftOk: 'OK {drift}',
        driftWarning: 'Warn {drift} {threshold}',
        driftError: 'Err {drift} {threshold}',
        driftNtpMissing: 'NTP missing',
      },
      help: {
        docsLink: 'Docs',
        jumpHost: 'Jump host help',
      },
    },
  },
})

describe('EmHostsScreen', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function mountScreen() {
    return mount(EmHostsScreen, {
      global: {
        plugins: [i18n],
        stubs: {
          CheckResultsList: true,
          HelpTooltip: true,
        },
      },
    })
  }

  it('renders EM hosts section and SSH settings', async () => {
    const wrapper = mountScreen()
    await nextTick()
    expect(wrapper.text()).toContain('YDB EM Hosts')
    expect(wrapper.text()).toContain('EM Hosts')
    expect(wrapper.text()).toContain('SSH Settings')
  })

  it('seeds one empty host row on mount', async () => {
    const wrapper = mountScreen()
    const store = useEmStore()
    await nextTick()

    expect(store.hosts).toEqual([{ fqdn: '' }])
    expect(wrapper.find('.em-host-input').exists()).toBe(true)
    expect(wrapper.find('.em-host-add').exists()).toBe(false)
    expect(wrapper.find('.em-host-remove').exists()).toBe(false)
  })

  it('typing in the row updates em store FQDN', async () => {
    const wrapper = mountScreen()
    await nextTick()

    const input = wrapper.find('.em-host-input')
    await input.setValue('em.example.com')
    await nextTick()

    expect(useEmStore().hosts[0].fqdn).toBe('em.example.com')
  })

  it('re-seeds an empty row if hosts are cleared', async () => {
    mountScreen()
    await nextTick()
    const store = useEmStore()

    store.hosts = []
    await nextTick()

    expect(store.hosts).toEqual([{ fqdn: '' }])
  })

  it('keeps canGoNext false until a filled host passes checks', async () => {
    mountScreen()
    const session = useSessionStore()
    const store = useEmStore()

    await flushPromises()
    expect(session.canGoNext).toBe(false)

    store.updateHost(0, 'em.example.com')
    await nextTick()
    expect(session.canGoNext).toBe(false)
  })
})
