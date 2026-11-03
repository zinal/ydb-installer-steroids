import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import enMessages from '@/locales/en.json'

vi.mock('@/api/client', () => ({
  api: {
    getInstallStatus: vi.fn(),
    cancelInstall: vi.fn(),
    getSession: vi.fn(),
    updateSession: vi.fn(),
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

vi.mock('@/stores/topology', () => ({
  useTopologyStore: vi.fn(() => ({
    selected: 'mirror-3-dc',
    currentTopology: { id: 'mirror-3-dc', nameKey: 'topology.mirror3dc.name' },
  })),
}))

vi.mock('@/stores/hosts', () => ({
  useHostsStore: vi.fn(() => ({
    dcHosts: {
      'DC-1': ['node1.dc1.example.com'],
      'DC-2': ['node1.dc2.example.com'],
      'DC-3': ['node1.dc3.example.com'],
    },
  })),
}))

vi.mock('@/stores/distribution', () => ({
  useDistributionStore: vi.fn(() => ({
    version: '24.3.1',
    archivePath: '',
  })),
}))

vi.mock('@/stores/ydbConfig', () => ({
  useYdbConfigStore: vi.fn(() => ({
    domain: 'Root',
  })),
}))

vi.mock('@/stores/databases', () => ({
  useDatabasesStore: vi.fn(() => ({
    configVariant: 'single',
    entries: [{ id: 1, name: 'testdb', groups: 8, hosts: [], hostMode: 'all', affinity: '' }],
    coresDynamic: 4,
  })),
}))

import { useInstall } from '@/composables/useInstall'
import ResultScreen from '@/views/ResultScreen.vue'

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
    logStatus: ref('none'),
    logsRecovered: ref(false),
    recoveryInProgress: ref(false),
    completedTasks: ref(0),
    start: vi.fn(),
    cancel: vi.fn(),
    recoverLogs: vi.fn(),
    waitForRecovery: vi.fn().mockResolvedValue(undefined),
  }
}

function mountScreen(status: string = 'idle') {
  const state = makeInstallState(status)
  vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
  return { wrapper: mount(ResultScreen, { global: { plugins: [i18n] } }), state }
}

describe('ResultScreen (Screen 10)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the screen title', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('Installation Result')
  })

  it('shows success card when status is success', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.find('.success-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('YDB cluster successfully deployed!')
  })

  it('shows cluster information when success', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('Cluster Information')
    expect(wrapper.text()).toContain('Topology')
    expect(wrapper.text()).toContain('YDB Version')
  })

  it('shows database name when success', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('Database')
    expect(wrapper.text()).toContain('/Root/testdb')
  })

  it('shows created databases section when success', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('Created Databases')
    expect(wrapper.text()).toContain('testdb')
    expect(wrapper.text()).toContain('8 groups')
  })

  it('shows hosts grouped by DC when success', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('Hosts')
    expect(wrapper.text()).toContain('DC-1:')
    expect(wrapper.text()).toContain('node1.dc1.example.com')
    expect(wrapper.text()).toContain('DC-2:')
    expect(wrapper.text()).toContain('node1.dc2.example.com')
    expect(wrapper.text()).toContain('DC-3:')
    expect(wrapper.text()).toContain('node1.dc3.example.com')
  })

  it('shows Healthy status for each host when success', () => {
    const { wrapper } = mountScreen('success')
    const healthyChips = wrapper.findAll('.v-chip')
    expect(healthyChips.length).toBeGreaterThanOrEqual(3)
  })

  it('shows endpoint when success', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('Endpoint')
    expect(wrapper.text()).toContain('grpcs://')
  })

  it('shows YDB UI link when success', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.text()).toContain('YDB UI')
  })

  it('shows logs section in success state', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.find('.logs-header').exists()).toBe(true)
    expect(wrapper.text()).toContain('Installation Logs')
  })

  it('console is collapsed by default in success state', async () => {
    const { wrapper } = mountScreen('success')
    const consoleOutput = wrapper.findComponent({ name: 'ConsoleOutput' })
    expect(consoleOutput.exists()).toBe(true)
    expect(consoleOutput.isVisible()).toBe(false)
  })

  it('console can be expanded by clicking the header in success state', async () => {
    const state = {
      status: ref('success'),
      progress: ref(0),
      tasks: ref([]),
      error: ref(null),
      logs: ref(['line1', 'line2', 'line3']),
      searchQuery: ref(''),
      filteredLogs: ref(['line1', 'line2', 'line3']),
      logStatus: ref('streaming'),
      logsRecovered: ref(false),
      recoveryInProgress: ref(false),
      completedTasks: ref(0),
      start: vi.fn(),
      cancel: vi.fn(),
      recoverLogs: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(ResultScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Show logs')
    expect(wrapper.text()).not.toContain('Hide logs')

    const logsHeader = wrapper.find('.logs-header')
    await logsHeader.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Hide logs')
    expect(wrapper.text()).not.toContain('Show logs')
  })

  it('shows log line count in success state', () => {
    const state = {
      status: ref('success'),
      progress: ref(0),
      tasks: ref([]),
      error: ref(null),
      logs: ref(['line1', 'line2', 'line3']),
      searchQuery: ref(''),
      filteredLogs: ref(['line1', 'line2', 'line3']),
      logStatus: ref('streaming'),
      logsRecovered: ref(false),
      recoveryInProgress: ref(false),
      completedTasks: ref(0),
      start: vi.fn(),
      cancel: vi.fn(),
      recoverLogs: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(ResultScreen, { global: { plugins: [i18n] } })
    expect(wrapper.text()).toContain('3')
  })

  it('shows error card when status is error', () => {
    const { wrapper } = mountScreen('error')
    expect(wrapper.find('.error-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('Installation failed')
  })

  it('shows error information when error', async () => {
    const state = {
      status: ref('error'),
      progress: ref(0),
      tasks: ref([]),
      error: ref<{ task: string; reason: string; host?: string } | null>({ task: 'test-task', reason: 'test-reason', host: 'test-host' }),
      logs: ref([]),
      searchQuery: ref(''),
      filteredLogs: ref([]),
      logStatus: ref('none'),
      logsRecovered: ref(false),
      recoveryInProgress: ref(false),
      completedTasks: ref(0),
      start: vi.fn(),
      cancel: vi.fn(),
      recoverLogs: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(ResultScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Error Information')
    expect(wrapper.text()).toContain('test-task')
    expect(wrapper.text()).toContain('test-reason')
    expect(wrapper.text()).toContain('test-host')
  })

  it('shows recommendations when error with error data', async () => {
    const state = {
      status: ref('error'),
      progress: ref(0),
      tasks: ref([]),
      error: ref<{ task: string; reason: string; host?: string } | null>({ task: 'test-task', reason: 'test-reason', host: 'test-host' }),
      logs: ref([]),
      searchQuery: ref(''),
      filteredLogs: ref([]),
      logStatus: ref('none'),
      logsRecovered: ref(false),
      recoveryInProgress: ref(false),
      completedTasks: ref(0),
      start: vi.fn(),
      cancel: vi.fn(),
      recoverLogs: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(ResultScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Recommendations')
    expect(wrapper.text()).toContain('test-task')
    expect(wrapper.text()).toContain('Check the error details')
  })

  it('shows logs section when error', () => {
    const { wrapper } = mountScreen('error')
    expect(wrapper.text()).toContain('Installation Logs')
  })

  it('shows log path with copy button when error', () => {
    const { wrapper } = mountScreen('error')
    expect(wrapper.text()).toContain('Copy path')
    expect(wrapper.text()).toContain('/tmp/ydb-installer/logs/install-')
  })

  it('shows Go to Error button when error', () => {
    const { wrapper } = mountScreen('error')
    expect(wrapper.text()).toContain('Go to Error')
  })

  it('shows no result message when idle', () => {
    const { wrapper } = mountScreen('idle')
    expect(wrapper.text()).toContain('No installation result available.')
  })

  it('does NOT show success card when idle', () => {
    const { wrapper } = mountScreen('idle')
    expect(wrapper.find('.success-card').exists()).toBe(false)
  })

  it('does NOT show error card when idle', () => {
    const { wrapper } = mountScreen('idle')
    expect(wrapper.find('.error-card').exists()).toBe(false)
  })

  it('waits for recovery on mount', async () => {
    const { state } = mountScreen('idle')
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(state.waitForRecovery).toHaveBeenCalled()
  })

  it('shows success card after recovery (not idle fallback)', async () => {
    const { wrapper } = mountScreen('success')
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(wrapper.find('.success-card').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('No installation result available.')
  })

  it('shows error card after recovery (not idle fallback)', async () => {
    const { wrapper } = mountScreen('error')
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(wrapper.find('.error-card').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('No installation result available.')
  })
})
