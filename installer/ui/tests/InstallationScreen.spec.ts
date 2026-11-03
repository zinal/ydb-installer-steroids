import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'

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
    getConfigStatus: vi.fn(),
    generateConfig: vi.fn(),
    startYdbInstall: vi.fn(),
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
import InstallationScreen from '@/views/InstallationScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: {
        step: 'Step',
      },
      install: {
        heading: 'Installation Process',
        configYaml: 'config.yaml',
        configReady: 'Ready',
        configFoundOnDisk: 'Found on disk',
        configMatches: 'Matches current configuration',
        configNeedsUpdate: 'Needs update',
        configNotFound: 'Not found',
        configOutdatedWarning: 'Parameters have changed since config.yaml was generated. Regenerate before installing, or use the existing file at your own risk.',
        configBackupTitle: 'config.yaml already exists',
        configBackupMessage: 'The existing config.yaml will be backed up',
        configBackupConfirm: 'Generate Anyway',
        configBackupCancel: 'Cancel',
        generateConfig: 'Generate config.yaml',
        regenerateConfig: 'Regenerate',
        installWithExisting: 'Install with existing config',
        startInstall: 'Start Installation',
        restartInstall: 'Restart Installation',
        cancelledTitle: 'Installation Cancelled',
        cancelledMessage: 'The installation was stopped.',
        showConfig: 'Show',
        hideConfig: 'Hide',
        configPreviewNote: 'config.yaml preview will be shown here after generation',
        successTitle: 'Installation successful!',
        successMessage: 'YDB cluster has been deployed successfully.',
        tasks: 'Tasks',
        console: 'Console',
        search: 'Search logs...',
        progress: '{done} / {total} tasks',
        stopBtn: 'Stop / Cancel',
        cancelTitle: 'Confirm cancellation',
        cancelMessage: 'Are you sure?',
        continueBtn: 'Continue',
        stopConfirmBtn: 'Stop',
        errorTitle: 'Error',
        errorTask: 'Task',
        errorHost: 'Host',
        errorReason: 'Reason',
      },
    },
  },
})

function makeInstallState(
  status: string = 'idle',
  configExists: boolean = false,
  configStatus: 'missing' | 'matches' | 'outdated' | null = null,
) {
  const resolvedStatus = configStatus ?? (configExists ? 'matches' : 'missing')
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
    configExists: ref(configExists),
    configFoundOnDisk: ref(false),
    configPath: ref('/path/to/config.yaml'),
    configContent: ref(configExists ? 'generated: true\nhosts:\n  - host1' : ''),
    configError: ref(null),
    configStatus: ref(resolvedStatus),
    fetchConfigStatus: vi.fn(),
    generateConfig: vi.fn(),
    startYdbInstall: vi.fn(),
    installWithExistingConfig: vi.fn(),
    restartInstallation: vi.fn(),
  }
}

function mountScreen(
  status: string = 'idle',
  configExists: boolean = false,
  configStatus?: 'missing' | 'matches' | 'outdated' | null,
) {
  const state = makeInstallState(status, configExists, configStatus)
  vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
  const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
  return { wrapper, state }
}

describe('InstallationScreen (Screen 9)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the screen title', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('Installation Process')
  })

  it('shows config.yaml block', () => {
    const { wrapper } = mountScreen()
    expect(wrapper.text()).toContain('config.yaml')
  })

  it('shows Generate config.yaml button when no config and idle', () => {
    const { wrapper } = mountScreen('idle', false)
    expect(wrapper.text()).toContain('Generate config.yaml')
  })

  it('shows Start Installation button when config exists and idle', () => {
    const { wrapper } = mountScreen('idle', true)
    expect(wrapper.text()).toContain('Start Installation')
  })

  it('does NOT auto-start installation (unlike Screen 31)', () => {
    const { state } = mountScreen('idle', true)
    expect(state.start).not.toHaveBeenCalled()
  })

  it('calls generateConfig when Generate button clicked', async () => {
    const { wrapper, state } = mountScreen('idle', false)
    const generateBtn = wrapper.findAll('button').find(b => b.text().includes('Generate config.yaml'))
    expect(generateBtn).toBeDefined()
    await generateBtn!.trigger('click')
    expect(state.generateConfig).toHaveBeenCalled()
  })

  it('calls startYdbInstall when Start button clicked', async () => {
    const { wrapper, state } = mountScreen('idle', true)
    const startBtn = wrapper.findAll('button').find(b => b.text().includes('Start Installation'))
    expect(startBtn).toBeDefined()
    await startBtn!.trigger('click')
    expect(state.startYdbInstall).toHaveBeenCalled()
  })

  it('shows progress bar section when running', () => {
    const { wrapper } = mountScreen('running')
    expect(wrapper.find('.section-card').exists()).toBe(true)
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
    expect(wrapper.text()).toContain('Installation successful!')
  })

  it('shows config block when success (config is always visible)', () => {
    const { wrapper } = mountScreen('success')
    expect(wrapper.find('.config-header').exists()).toBe(true)
  })

  it('shows error block when status is error', async () => {
    const state = {
      status: ref('error'),
      progress: ref(0),
      tasks: ref([{ id: 1, name: 'Running ydb_platform.ydb.initial_setup', status: 'error' }]),
      error: ref<{ task: string; reason: string; host?: string } | null>({ task: 'Running ydb_platform.ydb.initial_setup', reason: 'test-reason', host: 'test-host' }),
      logs: ref([]),
      searchQuery: ref(''),
      filteredLogs: ref([]),
      completedTasks: ref(0),
      isStarted: ref(false),
      start: vi.fn(),
      cancel: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
      configExists: ref(true),
      configPath: ref('/path/to/config.yaml'),
      configContent: ref('generated: true\nhosts:\n  - host1'),
      configError: ref(null),
      fetchConfigStatus: vi.fn(),
      generateConfig: vi.fn(),
      startYdbInstall: vi.fn(),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.error-card').exists()).toBe(true)
  })

  it('cancel button opens confirm dialog', async () => {
    const { wrapper } = mountScreen('running')
    const stopBtn = wrapper.find('.cancel-btn')
    expect(stopBtn.exists()).toBe(true)
    await stopBtn.trigger('click')
    expect(wrapper.findComponent({ name: 'ConfirmDialog' }).exists()).toBe(true)
  })

  it('does NOT show cancel button when idle', () => {
    const { wrapper } = mountScreen('idle')
    expect(wrapper.find('.cancel-row').exists()).toBe(false)
  })

  it('shows config preview when config exists', () => {
    const { wrapper } = mountScreen('idle', true)
    expect(wrapper.find('.config-code').exists()).toBe(true)
  })

  it('does NOT show config preview when no config', () => {
    const { wrapper } = mountScreen('idle', false)
    expect(wrapper.find('.config-code').exists()).toBe(false)
  })

  it('shows Start Installation button when config exists and idle', () => {
    const { wrapper } = mountScreen('idle', true)
    expect(wrapper.text()).toContain('Start Installation')
  })

  it('shows Regenerate button when config is outdated and idle', () => {
    const state = makeInstallState('idle', false, 'outdated')
    state.configFoundOnDisk = ref(true)
    state.configContent = ref('old-config')
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    expect(wrapper.text()).toContain('Regenerate')
  })

  it('does NOT show Regenerate button when no config', () => {
    const { wrapper } = mountScreen('idle', false)
    expect(wrapper.text()).not.toContain('Regenerate')
  })

  it('does NOT show Regenerate button when config matches', () => {
    const { wrapper } = mountScreen('idle', true, 'matches')
    expect(wrapper.text()).not.toContain('Regenerate')
    expect(wrapper.text()).toContain('Start Installation')
  })

  it('shows backup dialog when config found on disk from previous session', async () => {
    const state = makeInstallState('idle', false)
    state.configFoundOnDisk = ref(true)
    state.configContent = ref('old-config-from-disk')
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    
    const generateBtn = wrapper.findAll('button').find(b => b.text().includes('Generate config.yaml'))
    expect(generateBtn).toBeDefined()
    await generateBtn!.trigger('click')
    const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
    expect(dialogs.length).toBeGreaterThanOrEqual(1)
    expect(dialogs[0].props('title')).toBe('config.yaml already exists')
  })

  it('calls generateConfig without skipBackup after confirming backup dialog', async () => {
    const state = makeInstallState('idle', false)
    state.configFoundOnDisk = ref(true)
    state.configContent = ref('old-config-from-disk')
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    
    const generateBtn = wrapper.findAll('button').find(b => b.text().includes('Generate config.yaml'))
    expect(generateBtn).toBeDefined()
    await generateBtn!.trigger('click')
    const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
    const backupDialog = dialogs.find(d => d.props('title') === 'config.yaml already exists')
    expect(backupDialog).toBeDefined()
    await backupDialog!.vm.$emit('confirm')
    expect(state.generateConfig).toHaveBeenCalledWith(false)
  })

  it('calls generateConfig without skipBackup when no config exists at all', async () => {
    const { wrapper, state } = mountScreen('idle', false)
    const generateBtn = wrapper.findAll('button').find(b => b.text().includes('Generate config.yaml'))
    expect(generateBtn).toBeDefined()
    await generateBtn!.trigger('click')
    expect(state.generateConfig).toHaveBeenCalledWith(false)
  })

  it('shows Start button but NOT Regenerate when config matches and idle', () => {
    const { wrapper } = mountScreen('idle', true, 'matches')
    expect(wrapper.text()).toContain('Start Installation')
    expect(wrapper.text()).not.toContain('Regenerate')
  })

  it('renders action buttons in the config card (outside config preview)', () => {
    const { wrapper } = mountScreen('idle', true, 'matches')
    const configPreview = wrapper.find('.config-preview')
    expect(configPreview.exists()).toBe(true)
    const configActions = wrapper.find('.config-actions')
    expect(configActions.exists()).toBe(true)
    expect(configActions.text()).toContain('Start Installation')
    expect(configActions.text()).not.toContain('Regenerate')
  })

  it('shows toggle button in config header', () => {
    const { wrapper } = mountScreen('idle', true)
    const configHeader = wrapper.find('.config-header')
    expect(configHeader.text()).toContain('Hide')
  })

  it('toggle button toggles config preview visibility', async () => {
    const { wrapper } = mountScreen('idle', true)
    expect(wrapper.find('.config-code').exists()).toBe(true)

    const toggleBtn = wrapper.find('.toggle-header-btn')
    expect(toggleBtn.exists()).toBe(true)
    await toggleBtn.trigger('click')

    expect(wrapper.find('.config-code').exists()).toBe(false)
  })

  it('config preview stays visible after Start Installation is clicked', async () => {
    const { wrapper, state } = mountScreen('idle', true)
    expect(wrapper.find('.config-code').exists()).toBe(true)

    const startBtn = wrapper.findAll('button').find(b => b.text().includes('Start Installation'))
    expect(startBtn).toBeDefined()
    await startBtn!.trigger('click')

    expect(state.startYdbInstall).toHaveBeenCalled()
    expect(wrapper.find('.config-code').exists()).toBe(true)
  })

  it('action buttons remain visible when config is collapsed', async () => {
    const { wrapper } = mountScreen('idle', true, 'matches')
    const toggleBtn = wrapper.find('.toggle-header-btn')
    await toggleBtn.trigger('click')

    expect(wrapper.find('.config-preview').exists()).toBe(false)
    expect(wrapper.find('.config-actions').exists()).toBe(true)
    expect(wrapper.find('.config-actions').text()).toContain('Start Installation')
    expect(wrapper.find('.config-actions').text()).not.toContain('Regenerate')
  })

  it('shows Install with existing config button when config is outdated', () => {
    const state = makeInstallState('idle', false, 'outdated')
    state.configFoundOnDisk = ref(true)
    state.configContent = ref('existing-config-from-disk')
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    expect(wrapper.text()).toContain('Install with existing config')
  })

  it('does NOT show Install with existing config when config matches', () => {
    const { wrapper } = mountScreen('idle', true)
    expect(wrapper.text()).not.toContain('Install with existing config')
    expect(wrapper.text()).toContain('Start Installation')
  })

  it('does NOT show Install with existing config when installation is running', () => {
    const state = makeInstallState('running', false, 'outdated')
    state.isStarted = ref(true)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    expect(wrapper.text()).not.toContain('Install with existing config')
  })

  it('shows "Matches current configuration" chip when configStatus is matches', () => {
    const { wrapper } = mountScreen('idle', true, 'matches')
    expect(wrapper.text()).toContain('Matches current configuration')
  })

  it('shows "Needs update" chip when configStatus is outdated', () => {
    const state = makeInstallState('idle', false, 'outdated')
    state.configFoundOnDisk = ref(true)
    state.configContent = ref('old-config')
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    expect(wrapper.text()).toContain('Needs update')
  })

  it('shows "Not found" chip when configStatus is missing', () => {
    const { wrapper } = mountScreen('idle', false, 'missing')
    expect(wrapper.text()).toContain('Not found')
  })

  it('shows outdated warning alert when configStatus is outdated and not started', () => {
    const state = makeInstallState('idle', false, 'outdated')
    state.configFoundOnDisk = ref(true)
    state.configContent = ref('old-config')
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    expect(wrapper.text()).toContain('Parameters have changed since config.yaml was generated')
  })

  it('does NOT show outdated warning alert when configStatus is matches', () => {
    const { wrapper } = mountScreen('idle', true, 'matches')
    expect(wrapper.text()).not.toContain('Parameters have changed')
  })

  it('does NOT show outdated warning alert when configStatus is missing', () => {
    const { wrapper } = mountScreen('idle', false, 'missing')
    expect(wrapper.text()).not.toContain('Parameters have changed')
  })

  it('shows Generate button (not Regenerate) when configStatus is missing', () => {
    const { wrapper } = mountScreen('idle', false, 'missing')
    expect(wrapper.text()).toContain('Generate config.yaml')
    expect(wrapper.text()).not.toContain('Regenerate')
  })

  it('shows Start but NOT Regenerate when configStatus is matches', () => {
    const { wrapper } = mountScreen('idle', true, 'matches')
    expect(wrapper.text()).toContain('Start Installation')
    expect(wrapper.text()).not.toContain('Regenerate')
  })

  it('shows Regenerate + Install with existing when configStatus is outdated', () => {
    const state = makeInstallState('idle', false, 'outdated')
    state.configFoundOnDisk = ref(true)
    state.configContent = ref('old-config')
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    expect(wrapper.text()).toContain('Regenerate')
    expect(wrapper.text()).toContain('Install with existing config')
    expect(wrapper.text()).not.toContain('Start Installation')
  })

  it('shows cancelled card and restart button when status is cancelled', () => {
    const state = makeInstallState('cancelled', true)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    expect(wrapper.text()).toContain('Installation Cancelled')
    expect(wrapper.text()).toContain('Restart Installation')
    expect(wrapper.find('.cancelled-card').exists()).toBe(true)
  })

  it('does NOT show restart button when status is idle', () => {
    const { wrapper } = mountScreen('idle')
    expect(wrapper.text()).not.toContain('Restart Installation')
  })

  it('does NOT show cancelled card when status is running', () => {
    const { wrapper } = mountScreen('running')
    expect(wrapper.find('.cancelled-card').exists()).toBe(false)
  })

  it('calls restartInstallation when Restart button clicked', async () => {
    const state = makeInstallState('cancelled', true)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    const restartBtn = wrapper.findAll('button').find(b => b.text().includes('Restart Installation'))
    expect(restartBtn).toBeDefined()
    await restartBtn!.trigger('click')
    expect(state.restartInstallation).toHaveBeenCalled()
  })

  it('shows restart button when status is error', async () => {
    const state = {
      status: ref('error'),
      progress: ref(0),
      tasks: ref([{ id: 1, name: 'Test', status: 'error' }]),
      error: ref<{ task: string; reason: string; host?: string } | null>({ task: 'Test', reason: 'error', host: 'host1' }),
      logs: ref([]),
      searchQuery: ref(''),
      filteredLogs: ref([]),
      completedTasks: ref(0),
      isStarted: ref(false),
      start: vi.fn(),
      cancel: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
      configExists: ref(true),
      configFoundOnDisk: ref(false),
      configPath: ref('/path/to/config.yaml'),
      configContent: ref('config'),
      configError: ref(null),
      fetchConfigStatus: vi.fn(),
      generateConfig: vi.fn(),
      startYdbInstall: vi.fn(),
      installWithExistingConfig: vi.fn(),
      restartInstallation: vi.fn(),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Restart Installation')
  })

  it('calls restartInstallation when Restart button clicked on error state', async () => {
    const state = {
      status: ref('error'),
      progress: ref(0),
      tasks: ref([{ id: 1, name: 'Test', status: 'error' }]),
      error: ref<{ task: string; reason: string; host?: string } | null>({ task: 'Test', reason: 'error', host: 'host1' }),
      logs: ref([]),
      searchQuery: ref(''),
      filteredLogs: ref([]),
      completedTasks: ref(0),
      isStarted: ref(false),
      start: vi.fn(),
      cancel: vi.fn(),
      waitForRecovery: vi.fn().mockResolvedValue(undefined),
      configExists: ref(true),
      configFoundOnDisk: ref(false),
      configPath: ref('/path/to/config.yaml'),
      configContent: ref('config'),
      configError: ref(null),
      fetchConfigStatus: vi.fn(),
      generateConfig: vi.fn(),
      startYdbInstall: vi.fn(),
      installWithExistingConfig: vi.fn(),
      restartInstallation: vi.fn(),
    }
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    const restartBtn = wrapper.findAll('button').find(b => b.text().includes('Restart Installation'))
    expect(restartBtn).toBeDefined()
    await restartBtn!.trigger('click')
    expect(state.restartInstallation).toHaveBeenCalled()
  })

  it('auto-navigates to screen 10 when status becomes success and isStarted is true', async () => {
    const { wrapper, state } = mountScreen('idle', true)

    const startBtn = wrapper.findAll('button').find(b => b.text().includes('Start Installation'))
    expect(startBtn).toBeDefined()
    await startBtn!.trigger('click')

    state.status.value = 'success'

    await new Promise(resolve => setTimeout(resolve, 400))

    expect(mockPush).toHaveBeenCalledTimes(1)
    expect(mockPush).toHaveBeenCalledWith({ name: '10' })
  })

  it('does NOT auto-navigate when status is success but isStarted is false (stale state)', async () => {
    const { state } = mountScreen('idle', false)

    state.status.value = 'success'

    await new Promise(resolve => setTimeout(resolve, 400))

    expect(mockPush).not.toHaveBeenCalled()
  })

  it('does NOT auto-navigate when status is error even if isStarted', async () => {
    const { wrapper, state } = mountScreen('idle', true)

    const startBtn = wrapper.findAll('button').find(b => b.text().includes('Start Installation'))
    expect(startBtn).toBeDefined()
    await startBtn!.trigger('click')

    state.status.value = 'error'

    await new Promise(resolve => setTimeout(resolve, 400))

    expect(mockPush).not.toHaveBeenCalled()
  })

  it('waits for recovery on mount before fetching config status', async () => {
    const { state } = mountScreen('idle', false)
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(state.waitForRecovery).toHaveBeenCalled()
    expect(state.fetchConfigStatus).toHaveBeenCalled()
  })

  it('shows running state when isStarted is true and status is running', async () => {
    const state = makeInstallState('running', true)
    state.isStarted = ref(true)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.cancel-row').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Generate config.yaml')
  })

  it('shows success state when isStarted is true and status is success', async () => {
    const state = makeInstallState('success', true)
    state.isStarted = ref(true)
    vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
    const wrapper = mount(InstallationScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.success-card').exists()).toBe(true)
  })

  it('does NOT auto-start installation on mount (only recovers)', async () => {
    const { state } = mountScreen('idle', true)
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(state.startYdbInstall).not.toHaveBeenCalled()
  })
})
