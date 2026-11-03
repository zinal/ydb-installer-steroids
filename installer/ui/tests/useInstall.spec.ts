import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useInstall, resetInstallState } from '@/composables/useInstall'
import { useSessionStore } from '@/stores/session'

const getConfigStatusMock = vi.fn()
const getInstallStatusMock = vi.fn()
const cancelInstallMock = vi.fn()
const getInstallLogsMock = vi.fn()
const generateConfigMock = vi.fn()
const createLogsWebSocketMock = vi.fn()

vi.mock('@/api/client', () => ({
  api: {
    getConfigStatus: (...args: unknown[]) => getConfigStatusMock(...args),
    getInstallStatus: (...args: unknown[]) => getInstallStatusMock(...args),
    cancelInstall: (...args: unknown[]) => cancelInstallMock(...args),
    getInstallLogs: (...args: unknown[]) => getInstallLogsMock(...args),
    generateConfig: (...args: unknown[]) => generateConfigMock(...args),
  },
  createLogsWebSocket: (...args: unknown[]) => createLogsWebSocketMock(...args),
}))

function makeFakeWs() {
  return {
    onopen: null as (() => void) | null,
    onmessage: null as ((e: { data: string }) => void) | null,
    onclose: null as (() => void) | null,
    onerror: null as (() => void) | null,
    close: vi.fn(),
  }
}

describe('useInstall', () => {
  const fetchMock = vi.fn()

  beforeEach(() => {
    setActivePinia(createPinia())
    resetInstallState()
    getConfigStatusMock.mockReset()
    getInstallStatusMock.mockReset().mockResolvedValue({ status: 'idle', progress: 0, tasks: [] })
    cancelInstallMock.mockReset().mockResolvedValue({ status: 'ok' })
    getInstallLogsMock.mockReset().mockResolvedValue({ logs: [] })
    generateConfigMock.mockReset()
    createLogsWebSocketMock.mockReset().mockImplementation(makeFakeWs)
    fetchMock.mockReset().mockResolvedValue({ ok: true, status: 200, text: async () => '' })
    vi.stubGlobal('fetch', fetchMock)
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  /** Create the composable and let the auto-recovery settle. */
  async function freshInstall() {
    const install = useInstall()
    await install.waitForRecovery()
    getInstallStatusMock.mockClear()
    return install
  }

  it('fetchConfigStatus always calls the API and maps matches/outdated/missing', async () => {
    const install = await freshInstall()
    const session = useSessionStore()

    getConfigStatusMock.mockResolvedValueOnce({
      success: true, status: 'matches', path: '/config.yaml', content: 'cfg',
    })
    await install.fetchConfigStatus()
    expect(getConfigStatusMock).toHaveBeenCalledTimes(1)
    expect(install.configStatus.value).toBe('matches')
    expect(install.configExists.value).toBe(true)
    expect(install.configFoundOnDisk.value).toBe(false)
    expect(session.configGenerated).toBe(true)

    getConfigStatusMock.mockResolvedValueOnce({
      success: true, status: 'outdated', path: '/config.yaml', content: 'old',
    })
    await install.fetchConfigStatus()
    expect(getConfigStatusMock).toHaveBeenCalledTimes(2)
    expect(install.configStatus.value).toBe('outdated')
    expect(install.configExists.value).toBe(false)
    expect(install.configFoundOnDisk.value).toBe(true)
    expect(session.configGenerated).toBe(false)

    getConfigStatusMock.mockResolvedValueOnce({ success: true, status: 'missing' })
    await install.fetchConfigStatus()
    expect(getConfigStatusMock).toHaveBeenCalledTimes(3)
    expect(install.configStatus.value).toBe('missing')
    expect(install.configExists.value).toBe(false)
    expect(install.configFoundOnDisk.value).toBe(false)
  })

  it('fetchConfigStatus surfaces backend failure via configError', async () => {
    const install = await freshInstall()
    getConfigStatusMock.mockResolvedValueOnce({ success: false, error: 'boom' })
    await install.fetchConfigStatus()
    expect(install.configError.value).toBe('boom')
    expect(install.configStatus.value).toBeNull()
  })

  it('start() goes straight to running, POSTs the endpoint, and polls to success', async () => {
    const install = await freshInstall()

    const tasks = [
      { id: 1, name: 'task-1', status: 'pending' },
      { id: 2, name: 'task-2', status: 'pending' },
    ]
    getInstallStatusMock
      .mockResolvedValueOnce({ status: 'running', progress: 50, tasks })
      .mockResolvedValueOnce({ status: 'success', progress: 100, tasks })

    await install.start('/api/install/ydb/start')

    expect(install.status.value).toBe('running')
    expect(install.isStarted.value).toBe(true)
    expect(fetchMock).toHaveBeenCalledWith('/api/install/ydb/start', { method: 'POST' })
    expect(createLogsWebSocketMock).toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1000)
    expect(install.status.value).toBe('running')
    expect(install.progress.value).toBe(50)
    expect(install.tasks.value).toHaveLength(2)

    await vi.advanceTimersByTimeAsync(1000)
    expect(install.status.value).toBe('success')
    expect(install.progress.value).toBe(100)
    await vi.advanceTimersByTimeAsync(200)

    const callsAfterFinish = getInstallStatusMock.mock.calls.length
    await vi.advanceTimersByTimeAsync(5000)
    expect(getInstallStatusMock.mock.calls.length).toBe(callsAfterFinish)
  })

  it('start() reports an error when the start endpoint fails', async () => {
    const install = await freshInstall()
    fetchMock.mockResolvedValueOnce({ ok: false, status: 500, text: async () => 'internal' })

    await install.start('/api/install/ydb/start')

    expect(install.status.value).toBe('error')
    expect(install.error.value?.task).toBe('start')
    expect(install.error.value?.reason).toContain('internal')
  })

  it('start() treats HTTP 409 (already running) as success', async () => {
    const install = await freshInstall()
    fetchMock.mockResolvedValueOnce({ ok: false, status: 409, text: async () => '' })

    await install.start('/api/install/ydb/start')

    expect(install.status.value).toBe('running')
    expect(install.error.value).toBeNull()
  })

  it('cancel() marks the operation cancelled and notifies the backend', async () => {
    const install = await freshInstall()
    await install.start('/api/install/ydb/start')

    await install.cancel()

    expect(install.status.value).toBe('cancelled')
    expect(install.error.value?.task).toBe('cancelled')
    expect(cancelInstallMock).toHaveBeenCalled()
  })

  it('filteredLogs filters by searchQuery case-insensitively', async () => {
    const install = await freshInstall()
    install.logs.value = ['PLAY [all]', 'TASK [setup]', 'fatal: node-1 FAILED']

    expect(install.filteredLogs.value).toHaveLength(3)

    install.searchQuery.value = 'fatal'
    expect(install.filteredLogs.value).toEqual(['fatal: node-1 FAILED'])

    install.searchQuery.value = 'TASK'
    expect(install.filteredLogs.value).toEqual(['TASK [setup]'])

    install.searchQuery.value = '   '
    expect(install.filteredLogs.value).toHaveLength(3)
  })

  it('resetInstallState clears operation and config state', async () => {
    const install = await freshInstall()
    getConfigStatusMock.mockResolvedValueOnce({ success: true, status: 'matches', content: 'x' })
    await install.fetchConfigStatus()
    install.logs.value = ['line']

    resetInstallState()

    expect(install.status.value).toBe('idle')
    expect(install.progress.value).toBe(0)
    expect(install.logs.value).toEqual([])
    expect(install.isStarted.value).toBe(false)
    expect(install.configStatus.value).toBeNull()
    expect(install.configExists.value).toBe(false)
  })
})
