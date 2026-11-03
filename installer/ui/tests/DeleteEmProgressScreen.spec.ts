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
import DeleteEmProgressScreen from '@/views/DeleteEmProgressScreen.vue'
import { useSessionStore } from '@/stores/session'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      delete: {
        titleEmYdb: 'Delete YDB EM',
        successTitleEmYdb: 'YDB EM successfully deleted!',
        successMessageEmYdb: 'YDB Enterprise Manager has been removed.',
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
    logs: ref([] as string[]),
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

function mountScreen(status: string = 'idle', isStarted = false) {
  const state = makeInstallState(status)
  state.isStarted.value = isStarted
  vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
  return { wrapper: mount(DeleteEmProgressScreen, { global: { plugins: [i18n] } }), state }
}

describe('DeleteEmProgressScreen (Screen 33)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the screen title', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('Delete YDB EM')
  })

  it('awaits recovery on mount and does not auto-start', async () => {
    const { state } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    expect(state.waitForRecovery).toHaveBeenCalled()
    expect(state.start).not.toHaveBeenCalled()
  })

  it('shows success card when status is success', () => {
    const { wrapper } = mountScreen('success', true)
    expect(wrapper.text()).toContain('YDB EM successfully deleted!')
    expect(wrapper.find('.success-card').exists()).toBe(true)
  })

  it('shows InstallProgressPanel when not success', () => {
    const { wrapper } = mountScreen('running', true)
    expect(wrapper.findComponent({ name: 'InstallProgressPanel' }).exists()).toBe(true)
  })

  it('enables canGoNext only when started and success', async () => {
    const { wrapper, state } = mountScreen('idle', false)
    const session = useSessionStore()
    expect(session.canGoNext).toBe(false)

    state.isStarted.value = true
    state.status.value = 'success'
    await wrapper.vm.$nextTick()
    expect(session.canGoNext).toBe(true)
  })

  it('does not enable canGoNext on stale success without isStarted', async () => {
    const { wrapper, state } = mountScreen('success', false)
    const session = useSessionStore()
    await wrapper.vm.$nextTick()
    expect(session.canGoNext).toBe(false)
    expect(state.status.value).toBe('success')
  })

  it('restart calls delete-em-ydb endpoint', async () => {
    const { wrapper, state } = mountScreen('error', true)
    await wrapper.vm.$nextTick()
    const panel = wrapper.findComponent({ name: 'InstallProgressPanel' })
    expect(panel.exists()).toBe(true)
    await panel.vm.$emit('restart')
    expect(state.restartInstallation).toHaveBeenCalledWith('/api/install/delete-em-ydb/start')
  })

  it('cancel delegates to install.cancel', async () => {
    const { wrapper, state } = mountScreen('running', true)
    await wrapper.vm.$nextTick()
    const panel = wrapper.findComponent({ name: 'InstallProgressPanel' })
    await panel.vm.$emit('cancel')
    expect(state.cancel).toHaveBeenCalled()
  })
})
