import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import enMessages from '@/locales/en.json'

vi.mock('@/api/client', () => ({
  api: {
    getInstallStatus: vi.fn(),
    cancelInstall: vi.fn(),
    getSession: vi.fn(),
    updateSession: vi.fn(),
    validateFile: vi.fn().mockResolvedValue({ valid: true, size: '1MB' }),
  },
  createLogsWebSocket: vi.fn(() => ({
    onmessage: null,
    onerror: null,
    onclose: null,
    close: vi.fn(),
  })),
}))

vi.mock('@/composables/useInstall', () => ({
  useInstall: vi.fn(),
}))

vi.mock('@/stores/hosts', () => ({
  useHostsStore: vi.fn(() => ({
    dcHosts: {
      'DC-1': ['node1.dc1.example.com'],
      'DC-2': ['node1.dc2.example.com'],
      'DC-3': [],
    },
  })),
}))

import { useInstall } from '@/composables/useInstall'
import { useEmStore } from '@/stores/em'
import EmResultScreen from '@/views/EmResultScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: enMessages },
})

function makeInstallState(status: string = 'idle') {
  return {
    status: ref(status),
    progress: ref(0),
    tasks: ref([]),
    error: ref(null),
    logs: ref([]),
    searchQuery: ref(''),
    filteredLogs: ref([]),
    completedTasks: ref(0),
    isStarted: ref(false),
    logStatus: ref('none'),
    logsRecovered: ref(false),
    recoveryInProgress: ref(false),
    recoverLogs: vi.fn(),
    waitForRecovery: vi.fn().mockResolvedValue(undefined),
    start: vi.fn(),
    cancel: vi.fn(),
  }
}

function seedEmStore() {
  const em = useEmStore()
  em.hosts = [{ fqdn: 'em.example.com' }]
  em.dbConnectionDBRoot = '/Root'
  em.dbConnectionDB = 'db'
  return em
}

function mountScreen(status: string = 'idle') {
  const state = makeInstallState(status)
  vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
  const wrapper = mount(EmResultScreen, { global: { plugins: [i18n] } })
  return { wrapper, state }
}

describe('EmResultScreen (Screen 25)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the screen title', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('YDB EM Installation Result')
  })

  it('shows the success card when status is success', () => {
    seedEmStore()
    const { wrapper } = mountScreen('success')
    expect(wrapper.find('.success-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('YDB EM successfully installed!')
  })

  it('lists the EM hosts on success', () => {
    seedEmStore()
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('YDB EM hosts')
    expect(wrapper.text()).toContain('em.example.com')
  })

  it('lists the database slot hosts from the YDB inventory', () => {
    seedEmStore()
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('Database slot hosts')
    expect(wrapper.text()).toContain('node1.dc1.example.com')
    expect(wrapper.text()).toContain('node1.dc2.example.com')
  })

  it('shows the connected YDB endpoint derived from the EM host', () => {
    seedEmStore()
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('Connected to YDB')
    expect(wrapper.text()).toContain('grpcs://em.example.com:2135')
  })

  it('prefers an explicitly configured endpoint over the derived one', () => {
    const em = seedEmStore()
    em.dbConnectionEndpoint = 'grpcs://balancer.example.com:2135'
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('grpcs://balancer.example.com:2135')
  })

  it('renders the EM UI link as an anchor, not a button', () => {
    seedEmStore()
    const { wrapper } = mountScreen('success')
    const link = wrapper.find('a.ui-link')
    expect(link.exists()).toBe(true)
    expect(link.attributes('href')).toBe('https://em.example.com:8789/ui/home')
  })

  it('opens the EM UI link safely in a new tab', () => {
    seedEmStore()
    const { wrapper } = mountScreen('success')
    const link = wrapper.find('a.ui-link')
    expect(link.attributes('target')).toBe('_blank')
    expect(link.attributes('rel')).toContain('noopener')
  })

  it('shows a placeholder when no EM host is configured', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('No YDB EM hosts configured')
  })

  it('omits the EM UI link when there is no EM host to link to', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.find('a.ui-link').exists()).toBe(false)
  })

  it('shows the collapsible logs section on success', () => {
    seedEmStore()
    const { wrapper } = mountScreen('success')
    expect(wrapper.find('.logs-header').exists()).toBe(true)
  })

  it('shows the error card when status is error', () => {
    const { wrapper } = mountScreen('error')
    expect(wrapper.find('.error-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('YDB EM installation failed')
  })

  it('shows task, host and reason on error', async () => {
    const state = makeInstallState('error')
    state.error = ref({ task: 'register_agents', reason: 'agent timeout', host: 'em.example.com' }) as never
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(EmResultScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Error Information')
    expect(wrapper.text()).toContain('register_agents')
    expect(wrapper.text()).toContain('agent timeout')
    expect(wrapper.text()).toContain('em.example.com')
  })

  it('shows recommendations naming the failed task on error', async () => {
    const state = makeInstallState('error')
    state.error = ref({ task: 'install_agent', reason: 'boom', host: '' }) as never
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(EmResultScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Recommendations')
    expect(wrapper.text()).toContain('install_agent')
  })

  it('shows an EM-specific log path with a copy button on error', () => {
    const { wrapper } = mountScreen('error')
    expect(wrapper.text()).toContain('Copy path')
    expect(wrapper.text()).toContain('/tmp/ydb-installer/logs/install-em-')
  })

  it('offers a back link on error', () => {
    const { wrapper } = mountScreen('error')
    expect(wrapper.text()).toContain('Go to Error')
  })

  it('shows the no-result placeholder when idle', () => {
    const { wrapper } = mountScreen('idle')
    expect(wrapper.text()).toContain('No YDB EM installation result available.')
  })

  it('shows neither success nor error card when idle', () => {
    const { wrapper } = mountScreen('idle')
    expect(wrapper.find('.success-card').exists()).toBe(false)
    expect(wrapper.find('.error-card').exists()).toBe(false)
  })

  it('waits for recovery on mount', async () => {
    const { state } = mountScreen('idle')
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(state.waitForRecovery).toHaveBeenCalled()
  })

  it('renders the success card after recovery instead of the idle fallback', async () => {
    seedEmStore()
    const { wrapper } = mountScreen('success')
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(wrapper.find('.success-card').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('No YDB EM installation result available.')
  })
})
