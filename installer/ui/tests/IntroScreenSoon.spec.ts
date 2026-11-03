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
    getRuntimeInfo: vi.fn().mockResolvedValue({ embedded: true, extracted: true }),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { useSessionStore } from '@/stores/session'
import { useRuntimeStore } from '@/stores/runtime'
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

describe('IntroScreen — EM scenario visibility', () => {
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

  it('hides EM cards when the flag is off', async () => {
    const { api } = await import('@/api/client')
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({ embedded: true, extracted: true, show_em_scenarios: false })
    await useRuntimeStore().load()

    const wrapper = mountScreen()
    await flushPromises()

    expect(wrapper.findAll('.mode-card')).toHaveLength(2)
    expect(wrapper.findAll('.mode-grid')).toHaveLength(2)
    expect(wrapper.findAll('.mode-grid').every((grid) => grid.classes('mode-grid--single'))).toBe(true)
    expect(wrapper.text()).not.toContain('Install YDB EM')
    expect(wrapper.text()).not.toContain('Delete YDB EM')
  })

  it('does not show an EM inventory hint while EM delete is hidden', async () => {
    const { api } = await import('@/api/client')
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({ embedded: true, extracted: true, show_em_scenarios: false })
    await useRuntimeStore().load()

    const session = useSessionStore()
    session.inventoryFound = true
    session.configFound = true

    const wrapper = mountScreen()
    await flushPromises()

    expect(wrapper.find('.delete-info').exists()).toBe(false)
  })

  it('renders EM cards as selectable when the flag is on', async () => {
    const { api } = await import('@/api/client')
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({ embedded: true, extracted: true, show_em_scenarios: true })
    await useRuntimeStore().load()

    const wrapper = mountScreen()
    await flushPromises()

    expect(wrapper.findAll('.mode-card')).toHaveLength(4)
    expect(wrapper.findAll('.mode-grid').every((grid) => !grid.classes('mode-grid--single'))).toBe(true)
    const session = useSessionStore()
    const emCard = wrapper.findAll('.mode-card').find((c) => c.text().includes('Install YDB EM'))
    const deleteEmCard = wrapper.findAll('.mode-card').find((c) => c.text().includes('Delete YDB EM'))
    expect(emCard).toBeDefined()
    expect(deleteEmCard).toBeDefined()
    await emCard!.trigger('click')
    expect(session.flow).toBe('ydb-em')
  })

  it('keeps the runtime store flag in sync with the API', async () => {
    const { api } = await import('@/api/client')
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({ embedded: true, extracted: true, show_em_scenarios: true })
    await useRuntimeStore().load()

    const wrapper = mountScreen()
    await flushPromises()

    const runtime = useRuntimeStore()
    expect(runtime.showEmScenarios).toBe(true)
  })
})
