import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  api: {
    getDeps: vi.fn().mockResolvedValue({ dependencies: [], bundled: true }),
    getDepsInstruction: vi.fn().mockResolvedValue({ commands: '' }),
    getSession: vi.fn().mockResolvedValue({ flow: '', current_step: '1', steps: [], default_lang: 'en' }),
    updateSession: vi.fn().mockResolvedValue({}),
    getConnectivity: vi.fn().mockResolvedValue({ online: true, checked_at: 'x' }),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { useConnectivityStore } from '@/stores/connectivity'
import { useSessionStore } from '@/stores/session'
import IntroScreen from '@/views/IntroScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      app: { title: 'YDB Installer', beta: 'BETA' },
      intro: {
        heading: 'Welcome',
        description: 'Installer',
        ydbWizard: 'Install YDB',
        ydbWizardDesc: 'desc',
        emWizard: 'Install YDB EM',
        emWizardDesc: 'desc',
        deleteYdb: 'Delete YDB',
        deleteYdbDesc: 'desc',
        deleteEmYdb: 'Delete YDB EM',
        deleteEmYdbDesc: 'desc',
        depsTitle: 'Dependency check',
        ydbGroup: 'Installation',
        deleteGroup: 'Removal',
      },
      connectivity: {
        title: 'Internet access',
        checking: 'Checking...',
        online: 'Online',
        offline: 'Offline',
        offlineMode: 'Offline mode',
        offlineHint: 'Use offline mode in closed/isolated environments without internet access.',
        retry: 'Re-check',
      },
      deps: { retry: 'Retry' },
    },
  },
})

describe('IntroScreen — connectivity row', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function mountScreen() {
    return mount(IntroScreen, {
      global: {
        plugins: [i18n],
        stubs: {
          'v-card': { template: '<div class="v-card"><slot /></div>' },
          'v-icon': true,
          'v-progress-circular': true,
          'v-btn': { template: '<button class="v-btn"><slot /></button>' },
          'HelpTooltip': { props: ['textKey'], template: '<span class="help-tooltip" :title="textKey" />' },
        },
      },
    })
  }

  it('renders beta immediately after the installer version', async () => {
    useSessionStore().version = 'v0.4.3'
    const wrapper = mountScreen()
    await flushPromises()
    const version = wrapper.find('.intro-eyebrow .app-version__number')
    const beta = wrapper.find('.intro-eyebrow .app-version__badge')

    expect(version.text()).toBe('v0.4.3')
    expect(beta.text()).toBe('BETA')
    expect(version.element.nextElementSibling).toBe(beta.element)
  })

  it('renders the compact row with the offline-mode checkbox and help tooltip', async () => {
    const wrapper = mountScreen()
    await flushPromises()
    expect(wrapper.text()).toContain('Offline mode')
    expect(wrapper.find('.conn-checkbox').exists()).toBe(true)
    expect(wrapper.find('.help-tooltip').exists()).toBe(true)
    expect(wrapper.find('.help-tooltip').attributes('title')).toBe('connectivity.offlineHint')
  })

  it('shows online status after a successful check', async () => {
    const conn = useConnectivityStore()
    conn.status = 'online'
    conn.override = null

    const wrapper = mountScreen()
    await flushPromises()
    expect(wrapper.text()).toContain('Online')
  })

  it('shows offline status when the probe reports offline', async () => {
    const conn = useConnectivityStore()
    conn.status = 'offline'
    conn.override = null

    const wrapper = mountScreen()
    await flushPromises()
    expect(wrapper.text()).toContain('Offline')
  })

  it('the checkbox enables offline mode and sets the store override', async () => {
    const conn = useConnectivityStore()
    conn.status = 'online'
    conn.override = null

    const wrapper = mountScreen()
    await flushPromises()

    const box = wrapper.find('.conn-checkbox input')
    expect(box.exists()).toBe(true)

    await box.setValue()
    expect(conn.offlineMode).toBe(true)
    expect(conn.effectiveOnline).toBe(false)
  })

  it('disables the offline-mode checkbox while the probe is checking', async () => {
    const conn = useConnectivityStore()
    conn.status = 'checking'
    conn.override = null

    const wrapper = mountScreen()
    await flushPromises()

    const box = wrapper.find('.conn-checkbox input')
    expect((box.element as HTMLInputElement).disabled).toBe(true)
  })
})
