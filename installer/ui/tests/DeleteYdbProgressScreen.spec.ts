import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'

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

import { useInstall } from '@/composables/useInstall'
import DeleteYdbProgressScreen from '@/views/DeleteYdbProgressScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      delete: {
        titleYdb: 'Delete YDB',
        successTitleYdb: 'YDB successfully deleted!',
        successMessageYdb: 'All YDB components have been removed.',
        restartDeletion: 'Restart Deletion',
        cancelledTitle: 'Deletion Cancelled',
        cancelledMessage: 'The deletion was stopped.',
        cancelTitle: 'Confirm cancellation',
        cancelMessage: 'Stop the deletion?',
        continueBtn: 'Continue',
        stopConfirmBtn: 'Stop',
      },
      install: {
        tasks: 'Tasks',
        console: 'Console',
        search: 'Search logs...',
        progress: '{done} / {total} tasks',
        stopBtn: 'Stop / Cancel',
        errorTitle: 'Error',
        errorTask: 'Task',
        errorHost: 'Host',
        errorReason: 'Reason',
        restartInstall: 'Restart Installation',
        recoverLogs: 'Recover Logs',
      },
    },
  },
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
    logStatus: ref('streaming'),
    logsRecovered: ref(false),
    recoveryInProgress: ref(false),
    recoverLogs: vi.fn(),
    recoverState: vi.fn(),
    waitForRecovery: vi.fn().mockResolvedValue(undefined),
    start: vi.fn(),
    cancel: vi.fn(),
    restartInstallation: vi.fn(),
  }
}

function mountScreen(status: string = 'idle') {
  const state = makeInstallState(status)
  vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
  return { wrapper: mount(DeleteYdbProgressScreen, { global: { plugins: [i18n] } }), state }
}

describe('DeleteYdbProgressScreen (Screen 31)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the screen title', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('Delete YDB')
  })

  it('does NOT auto-start deletion on mount', async () => {
    const { state } = mountScreen('idle')
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(state.start).not.toHaveBeenCalled()
  })

  it('waits for recovery on mount', async () => {
    const { state } = mountScreen('idle')
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(state.waitForRecovery).toHaveBeenCalled()
  })

  it('shows progress bar section when running', () => {
    const { wrapper } = mountScreen('running')
    expect(wrapper.find('.progress-shell').exists()).toBe(true)
  })

  it('shows task list section when running', () => {
    const { wrapper } = mountScreen('running')
    expect(wrapper.text()).toContain('Tasks')
  })

  it('shows console section when running', () => {
    const { wrapper } = mountScreen('running')
    expect(wrapper.text()).toContain('Console')
  })

  it('shows stop button when status is running', () => {
    const { wrapper } = mountScreen('running')
    expect(wrapper.find('.cancel-row').exists()).toBe(true)
    expect(wrapper.text()).toContain('Stop')
  })

  it('shows success card when status is success', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.find('.success-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('YDB successfully deleted!')
  })

  it('shows progress section when status is cancelled', () => {
    const { wrapper } = mountScreen('cancelled')
    expect(wrapper.find('.progress-shell').exists()).toBe(true)
  })

  it('shows cancelled card and restart deletion button when status is cancelled', async () => {
    const state = makeInstallState('cancelled')
    state.isStarted = ref(true)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(DeleteYdbProgressScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.cancelled-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('Deletion Cancelled')
    expect(wrapper.text()).toContain('Restart Deletion')
    expect(wrapper.find('.error-card').exists()).toBe(false)
  })

  it('calls restartInstallation when Restart button clicked on cancelled state', async () => {
    const state = makeInstallState('cancelled')
    state.isStarted = ref(true)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(DeleteYdbProgressScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    const restartBtn = wrapper.findAll('button').find(b => b.text().includes('Restart Deletion'))
    expect(restartBtn).toBeDefined()
    await restartBtn!.trigger('click')
    expect(state.restartInstallation).toHaveBeenCalledWith('/api/install/delete-ydb/start')
  })

  it('shows error block when status is error', async () => {
    const state = {
      status: ref('error'),
      progress: ref(0),
      tasks: ref([{ id: 1, name: 'Test task', status: 'error' }]),
      error: ref<{ task: string; reason: string; host?: string } | null>({ task: 'test-task', reason: 'test-reason', host: 'test-host' }),
      logs: ref([]),
      searchQuery: ref(''),
      filteredLogs: ref([]),
      completedTasks: ref(0),
      isStarted: ref(false),
      logStatus: ref('streaming'),
      logsRecovered: ref(false),
      recoveryInProgress: ref(false),
      recoverLogs: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
      start: vi.fn(),
      cancel: vi.fn(),
      restartInstallation: vi.fn(),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(DeleteYdbProgressScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.error-card').exists()).toBe(true)
  })

  it('shows restart button with deletion label when status is error', async () => {
    const state = {
      status: ref('error'),
      progress: ref(0),
      tasks: ref([{ id: 1, name: 'Test task', status: 'error' }]),
      error: ref<{ task: string; reason: string; host?: string } | null>({ task: 'test-task', reason: 'test-reason', host: 'test-host' }),
      logs: ref([]),
      searchQuery: ref(''),
      filteredLogs: ref([]),
      completedTasks: ref(0),
      isStarted: ref(false),
      logStatus: ref('streaming'),
      logsRecovered: ref(false),
      recoveryInProgress: ref(false),
      recoverLogs: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
      start: vi.fn(),
      cancel: vi.fn(),
      restartInstallation: vi.fn(),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(DeleteYdbProgressScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Restart Deletion')
    expect(wrapper.text()).not.toContain('Restart Installation')
    expect(wrapper.find('.cancelled-card').exists()).toBe(false)
  })

  it('cancel button opens confirm dialog', async () => {
    const { wrapper } = mountScreen('running')
    const stopBtn = wrapper.find('.cancel-btn')
    expect(stopBtn.exists()).toBe(true)
    await stopBtn.trigger('click')
    expect(wrapper.findComponent({ name: 'ConfirmDialog' }).exists()).toBe(true)
  })

  it('cancel dialog uses delete-specific strings', async () => {
    const { wrapper } = mountScreen('running')
    const stopBtn = wrapper.find('.cancel-btn')
    await stopBtn.trigger('click')
    const dialog = wrapper.findComponent({ name: 'ConfirmDialog' })
    expect(dialog.props('title')).toBe('Confirm cancellation')
    expect(dialog.props('message')).toBe('Stop the deletion?')
  })

  it('does NOT show cancel button when idle', () => {
    const { wrapper } = mountScreen('idle')
    expect(wrapper.find('.cancel-row').exists()).toBe(false)
  })

  it('shows localized Recover Logs button when disconnected', async () => {
    const state = makeInstallState('running')
    state.logStatus = ref('disconnected')
    state.logsRecovered = ref(false)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(DeleteYdbProgressScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Recover Logs')
  })
})
