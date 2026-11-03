import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import enMessages from '@/locales/en.json'

const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useRoute: () => ({}),
}))

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

import { useInstall } from '@/composables/useInstall'
import { useEmStore } from '@/stores/em'
import EmInstallScreen from '@/views/EmInstallScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: enMessages },
})

function makeInstallState(status: string = 'idle', started: boolean = false) {
  return {
    status: ref(status),
    progress: ref(0),
    tasks: ref([]),
    error: ref(null),
    logs: ref([]),
    searchQuery: ref(''),
    filteredLogs: ref([]),
    completedTasks: ref(0),
    isStarted: ref(started),
    logStatus: ref('none'),
    logsRecovered: ref(false),
    recoveryInProgress: ref(false),
    recoverLogs: vi.fn(),
    recoverState: vi.fn(),
    waitForRecovery: vi.fn().mockResolvedValue(undefined),
    start: vi.fn(),
    cancel: vi.fn(),
    startEmInstall: vi.fn(),
    restartInstallation: vi.fn(),
  }
}

/** Populate the em store so the pre-flight passes. */
function makeStoreReady() {
  const em = useEmStore()
  em.hosts = [{ fqdn: 'em.example.com' }]
  em.dbConnectionDBRoot = '/Root'
  em.dbConnectionDB = 'db'
  em.distMode = 'archive'
  em.archivePath = '/opt/dist/ydb-em.tar.gz'
  em.metaUIArchivePath = '/opt/dist/ui.tar.gz'
  em.archiveValid = true
  em.metaUIArchiveValid = true
  em.tlsCertPath = 'TLS/CA/certs/certs-per-host'
  em.useDynamicConfig = true
  return em
}

function mountScreen(status: string = 'idle', started: boolean = false) {
  const state = makeInstallState(status, started)
  vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
  const wrapper = mount(EmInstallScreen, { global: { plugins: [i18n] } })
  return { wrapper, state }
}

function findStartButton(wrapper: ReturnType<typeof mountScreen>['wrapper']) {
  return wrapper.findAll('button').find(b => b.text().includes('Start Installation'))
}

describe('EmInstallScreen (Screen 24)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the screen title', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('YDB EM Installation')
  })

  it('does NOT offer config.yaml generation', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).not.toContain('Generate config.yaml')
  })

  it('shows the pre-flight card before the first start', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('Pre-flight check')
  })

  it('disables Start when the pre-flight is not satisfied', () => {
    const { wrapper } = mountScreen()
    const startBtn = findStartButton(wrapper)
    expect(startBtn).toBeDefined()
    expect(startBtn!.attributes('disabled')).toBeDefined()
  })

  it('shows a blocking summary when requirements are unmet', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('must be resolved')
  })

  it('enables Start once the pre-flight is satisfied', async () => {
    makeStoreReady()
    const { wrapper } = mountScreen()
    await wrapper.vm.$nextTick()
    const startBtn = findStartButton(wrapper)
    expect(startBtn).toBeDefined()
    expect(startBtn!.attributes('disabled')).toBeUndefined()
  })

  it('lists the EM host in the pre-flight summary', async () => {
    makeStoreReady()
    const { wrapper } = mountScreen()
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('em.example.com')
  })

  it('surfaces the dynamic-config advisory without blocking Start', async () => {
    const em = makeStoreReady()
    em.useDynamicConfig = false
    const { wrapper } = mountScreen()
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('YDB dynamic configuration')
    expect(findStartButton(wrapper)!.attributes('disabled')).toBeUndefined()
  })

  it('does NOT auto-start the installation on mount', async () => {
    makeStoreReady()
    const { state } = mountScreen()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(state.startEmInstall).not.toHaveBeenCalled()
  })

  it('calls startEmInstall when Start is clicked', async () => {
    makeStoreReady()
    const { wrapper, state } = mountScreen()
    await wrapper.vm.$nextTick()
    await findStartButton(wrapper)!.trigger('click')
    expect(state.startEmInstall).toHaveBeenCalled()
  })

  it('waits for recovery on mount', async () => {
    const { state } = mountScreen()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(state.waitForRecovery).toHaveBeenCalled()
  })

  it('hides the pre-flight and Start button once started', () => {
    makeStoreReady()
    const { wrapper } = mountScreen('running', true)
    expect(wrapper.text()).not.toContain('Pre-flight check')
    expect(findStartButton(wrapper)).toBeUndefined()
  })

  it('shows progress, tasks and console when running', () => {
    const { wrapper } = mountScreen('running', true)
    expect(wrapper.find('.progress-shell').exists()).toBe(true)
    expect(wrapper.text()).toContain('Tasks')
    expect(wrapper.text()).toContain('Console')
  })

  it('shows the stop button when running', () => {
    const { wrapper } = mountScreen('running', true)
    expect(wrapper.find('.cancel-row').exists()).toBe(true)
  })

  it('cancel button opens the confirmation dialog', async () => {
    const { wrapper } = mountScreen('running', true)
    await wrapper.find('.cancel-btn').trigger('click')
    expect(wrapper.findComponent({ name: 'ConfirmDialog' }).exists()).toBe(true)
  })

  it('calls cancel when the dialog is confirmed', async () => {
    const { wrapper, state } = mountScreen('running', true)
    await wrapper.find('.cancel-btn').trigger('click')
    await wrapper.findComponent({ name: 'ConfirmDialog' }).vm.$emit('confirm')
    expect(state.cancel).toHaveBeenCalled()
  })

  it('shows the error card and restart button on error', async () => {
    const state = makeInstallState('error', true)
    state.error = ref({ task: 'register_agents', reason: 'boom', host: 'em.example.com' }) as never
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(EmInstallScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.error-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('Restart Installation')
  })

  it('restarts against the EM endpoint (not the YDB one)', async () => {
    const state = makeInstallState('cancelled', true)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(EmInstallScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    const restartBtn = wrapper.findAll('button').find(b => b.text().includes('Restart Installation'))
    expect(restartBtn).toBeDefined()
    await restartBtn!.trigger('click')
    expect(state.restartInstallation).toHaveBeenCalledWith('/api/install/ydb-em/start')
  })

  it('shows the cancelled card when cancelled', async () => {
    const { wrapper } = mountScreen('cancelled', true)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.cancelled-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('Installation Cancelled')
  })

  it('shows the success card when the sequence completes', async () => {
    const { wrapper } = mountScreen('success', true)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.success-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('YDB EM successfully installed!')
  })

  it('auto-navigates to screen 25 on success after start', async () => {
    makeStoreReady()
    const { wrapper, state } = mountScreen()
    await wrapper.vm.$nextTick()
    await findStartButton(wrapper)!.trigger('click')

    state.isStarted.value = true
    state.status.value = 'success'

    await new Promise(resolve => setTimeout(resolve, 400))
    expect(mockPush).toHaveBeenCalledWith({ name: '25' })
  })

  it('does NOT auto-navigate on a stale success (never started)', async () => {
    const { state } = mountScreen()
    state.status.value = 'success'
    await new Promise(resolve => setTimeout(resolve, 400))
    expect(mockPush).not.toHaveBeenCalled()
  })

  it('does NOT auto-navigate on error', async () => {
    const { state } = mountScreen()
    state.isStarted.value = true
    state.status.value = 'error'
    await new Promise(resolve => setTimeout(resolve, 400))
    expect(mockPush).not.toHaveBeenCalled()
  })
})
