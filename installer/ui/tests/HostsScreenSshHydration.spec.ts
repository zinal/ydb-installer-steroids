// Regression test for the "SSH Settings reset to defaults on refresh" bug.
//
// Root cause: HostsScreen.vue used to hydrate its local `sshConfig` ref from
// hostsStore.ssh inside a one-shot `onMounted`. hostsStore.ssh is populated
// ASYNCHRONOUSLY (useInventorySync's loadInitial() awaits api.getInventory()
// before calling applyFromInventory on the hosts store). The one-shot
// onMounted callback runs synchronously and therefore always observed
// hostsStore.ssh as still null — the SSH form silently stayed at hardcoded
// defaults ('root', no jump host) forever, even after the real inventory
// data arrived a moment later, because nothing ever re-triggered hydration.
//
// The fix replaces the one-shot onMounted with a guarded
// `watch(() => hostsStore.ssh, ..., { immediate: true })`, which reacts
// whenever hostsStore.ssh actually transitions from null to a populated
// value — regardless of whether that happens synchronously or after an
// async gap. This test reproduces the async gap explicitly (populating the
// store on a microtask AFTER the component has already mounted) and asserts
// the form catches up.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn(() => new Promise(() => { /* never resolves */ })),
    updateInventory: vi.fn().mockResolvedValue({ status: 'ok', data: null }),
    checkHosts: vi.fn(),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { useHostsStore } from '@/stores/hosts'
import { useTopologyStore } from '@/stores/topology'
import HostsScreen from '@/views/HostsScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      hosts: {
        heading: 'Hosts & SSH',
        topologyLabel: 'Topology: {topology}',
        addHost: 'Add host',
        loadFromFileTooltip: 'Load from file',
        sshSettings: 'SSH Settings',
        checkHosts: 'Check hosts',
        driftNtpNotSynced: 'NTP not synced',
        driftNtpMissing: 'NTP missing',
        duplicateTitle: 'Duplicate hosts',
        duplicateDescription: 'Some hosts appear in more than one DC',
        checkRequired: 'Run the check',
        checksFailed: 'Checks failed',
        minDCHostsRequired: 'At least {count} host(s) required per DC',
        minHostsRequired: 'At least {count} hosts required',
        hostsLoaded: '{count} hosts loaded into {dc}',
      },
      ssh: {
        login: 'Login',
        specifyKey: 'Specify private key',
        keyPath: 'Key path',
        useJumpHost: 'Use jump host',
        jumpHostFqdn: 'Jump host FQDN',
        advancedSettings: 'Advanced settings',
        customSshArgs: 'Custom SSH args',
      },
      help: {
        jumpHost: 'Jump host help',
      },
    },
  },
})

describe('HostsScreen — SSH settings hydration race (regression)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('SSH form stays at defaults immediately after mount when the store is not yet populated', () => {
    const wrapper = mount(HostsScreen, {
      global: { plugins: [i18n] },
    })
    const loginInput = wrapper.find('input[autocomplete="username"]')
    expect((loginInput.element as HTMLInputElement).value).toBe('root')
  })

  it('SSH form hydrates with the real values once hostsStore.ssh is populated AFTER mount (async race)', async () => {
    const wrapper = mount(HostsScreen, {
      global: { plugins: [i18n] },
    })
    const topologyStore = useTopologyStore()
    topologyStore.select('mirror-3-dc')

    // Sanity check: before the store is populated, the form shows defaults —
    // proving this test actually exercises the async-gap scenario rather
    // than a pre-populated store.
    let loginInput = wrapper.find('input[autocomplete="username"]')
    expect((loginInput.element as HTMLInputElement).value).toBe('root')

    // Simulate the inventory arriving asynchronously, well after mount —
    // this is exactly the scenario that broke with the one-shot onMounted.
    const hostsStore = useHostsStore()
    hostsStore.setSsh({
      login: 'admin',
      auth_type: 'key',
      key_path: '/home/admin/.ssh/id_ed25519',
      use_jump_host: true,
      jump_host_fqdn: 'bastion.example.com',
    })

    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    loginInput = wrapper.find('input[autocomplete="username"]')
    expect((loginInput.element as HTMLInputElement).value).toBe('admin')

    const jumpHostCheckbox = wrapper.findAll('input[type="checkbox"]')[1]
    expect((jumpHostCheckbox.element as HTMLInputElement).checked).toBe(true)
  })

  it('does not re-hydrate (clobber) the form after the user has made their own edits', async () => {
    const wrapper = mount(HostsScreen, {
      global: { plugins: [i18n] },
    })
    const hostsStore = useHostsStore()

    // First hydration from the store.
    hostsStore.setSsh({
      login: 'admin',
      auth_type: 'key',
      use_jump_host: false,
    })
    await wrapper.vm.$nextTick()

    let loginInput = wrapper.find('input[autocomplete="username"]')
    expect((loginInput.element as HTMLInputElement).value).toBe('admin')

    // User edits the login field by hand.
    await loginInput.setValue('custom-user')
    await wrapper.vm.$nextTick()

    // Store round-trips through the watch(sshConfig, ...) → hostsStore.setSsh
    // sync; a subsequent external mutation with a DIFFERENT login must NOT
    // silently overwrite the user's in-progress edit (hydration is one-shot).
    hostsStore.setSsh({
      login: 'someone-else',
      auth_type: 'key',
      use_jump_host: false,
    })
    await wrapper.vm.$nextTick()

    loginInput = wrapper.find('input[autocomplete="username"]')
    expect((loginInput.element as HTMLInputElement).value).toBe('custom-user')
  })
})
