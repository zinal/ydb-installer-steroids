import { computed, ref } from 'vue'
import { api, createLogsWebSocket } from '@/api/client'
import { OPERATIONS } from '@/config/operations'
import { useSessionStore } from '@/stores/session'
import type { InstallError, InstallStatus, Task } from '@/types/install'
import type { ConfigMatchStatus } from '@/api/client'

const POLL_INTERVAL_MS = 1000
const MAX_RECONNECT_DELAY = 30000
const MAX_RETRY_ATTEMPTS = 3

const status = ref<InstallStatus>('idle')
const progress = ref(0)
const tasks = ref<Task[]>([])
const error = ref<InstallError | null>(null)
const logs = ref<string[]>([])
const searchQuery = ref('')
const isStarted = ref(false)

const configExists = ref(false)
const configFoundOnDisk = ref(false)
const configPath = ref('')
const configContent = ref('')
const configError = ref<string | null>(null)
const configStatus = ref<ConfigMatchStatus | null>(null)

const logStatus = ref<'streaming' | 'disconnected' | 'reconnecting' | 'recovering' | 'recovered' | 'sse' | 'none'>('none')
const logsRecovered = ref(false)
const recoveryInProgress = ref(false)

let ws: WebSocket | null = null
let es: EventSource | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempt = 0
let cancelling = false
let recoveryAttempted = false
let recoveryPromise: Promise<void> | null = null

const filteredLogs = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return logs.value
  return logs.value.filter((l) => l.toLowerCase().includes(q))
})

const completedTasks = computed(() =>
  tasks.value.filter((t) => t.status === 'done').length,
)

function disconnectWebSocket() {
  if (ws) {
    ws.onclose = null
    ws.onerror = null
    ws.onmessage = null
    ws.close()
    ws = null
  }
}

function disconnectSSE() {
  if (es) {
    es.close()
    es = null
  }
}

function disconnectAll() {
  disconnectWebSocket()
  disconnectSSE()
  if (reconnectTimer !== null) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

function stopPolling() {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function connectSSE() {
  disconnectSSE()
  logStatus.value = 'reconnecting'

  try {
    const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:'
    const url = `${protocol}//${window.location.host}/ws/logs`
    es = new EventSource(url)

    es.onopen = () => {
      logStatus.value = 'sse'
    }

    es.onmessage = (e: MessageEvent<string>) => {
      logs.value.push(String(e.data))
    }

    es.addEventListener('done', () => {
      logStatus.value = 'recovered'
      es?.close()
      es = null
    })

    es.onerror = () => {
      logStatus.value = 'disconnected'
      if (status.value === 'running') {
        recoverLogs()
      }
    }
  } catch {
    logStatus.value = 'disconnected'
    if (status.value === 'running') {
      recoverLogs()
    }
  }
}

function connectWithBackoff() {
  disconnectAll()

  if (cancelling || status.value === 'success' || status.value === 'error') {
    return
  }

  if (reconnectAttempt >= MAX_RETRY_ATTEMPTS) {
    connectSSE()
    return
  }

  logStatus.value = 'reconnecting'

  try {
    ws = createLogsWebSocket()
  } catch {
    scheduleReconnect()
    return
  }

  ws.onopen = () => {
    reconnectAttempt = 0
    logStatus.value = 'streaming'
  }

  ws.onmessage = (e: MessageEvent<string>) => {
    logs.value.push(String(e.data))
  }

  ws.onclose = () => {
    ws = null
    if (status.value === 'running' || status.value === 'idle') {
      scheduleReconnect()
    } else {
      logStatus.value = 'disconnected'
    }
  }

  ws.onerror = () => {
  }
}

function scheduleReconnect() {
  const delay = Math.min(1000 * Math.pow(2, reconnectAttempt), MAX_RECONNECT_DELAY)
  reconnectAttempt++
  logStatus.value = 'reconnecting'
  reconnectTimer = setTimeout(connectWithBackoff, delay)
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    if (cancelling) return
    try {
      const state = await api.getInstallStatus()
      status.value = state.status
      progress.value = state.progress
      tasks.value = state.tasks ?? []
      error.value = state.error ?? null

      if (state.status === 'success' || state.status === 'cancelled' || state.status === 'error') {
        setTimeout(() => {
          stopPolling()
          disconnectAll()
          if (logStatus.value !== 'streaming' && logStatus.value !== 'sse') {
            logStatus.value = 'disconnected'
          }
        }, 100)
      }
    } catch {
    }
  }, POLL_INTERVAL_MS)
}

export function resetInstallState() {
  status.value = 'idle'
  progress.value = 0
  tasks.value = []
  error.value = null
  logs.value = []
  searchQuery.value = ''
  isStarted.value = false
  configExists.value = false
  configFoundOnDisk.value = false
  configPath.value = ''
  configContent.value = ''
  configError.value = null
  configStatus.value = null
  logStatus.value = 'none'
  logsRecovered.value = false
  recoveryInProgress.value = false
  recoveryAttempted = false
  recoveryPromise = null
  stopPolling()
  disconnectAll()
}

async function recoverState(): Promise<void> {
  try {
    const state = await api.getInstallStatus()

    if (state.status === 'idle') {
      return
    }

    status.value = state.status
    progress.value = state.progress
    tasks.value = state.tasks ?? []
    error.value = state.error ?? null

    if (state.status === 'running') {
      isStarted.value = true
    }

    await recoverLogs()

    if (state.status === 'running') {
      reconnectAttempt = 0
      cancelling = false
      connectWithBackoff()
      startPolling()
    }
  } catch {
  }
}

function waitForRecovery(): Promise<void> {
  return recoveryPromise ?? Promise.resolve()
}

async function start(endpoint: string) {
  cancelling = false
  reconnectAttempt = 0
  progress.value = 0
  tasks.value = []
  error.value = null
  logs.value = []
  isStarted.value = true
  status.value = 'running'

  connectWithBackoff()
  startPolling()

  try {
    const res = await fetch(endpoint, { method: 'POST' })
    if (!res.ok && res.status !== 409) {
      let body = ''
      try {
        body = await res.text()
      } catch {
        body = ''
      }
      status.value = 'error'
      error.value = {
        task: 'start',
        host: '',
        reason: body || ('HTTP ' + String(res.status)),
      }
      stopPolling()
      disconnectAll()
      return
    }
  } catch (e) {
    status.value = 'error'
    error.value = { task: 'start', host: '', reason: String(e) }
    stopPolling()
    disconnectAll()
    return
  }

  status.value = 'running'
}

async function cancel() {
  cancelling = true
  status.value = 'cancelled'
  error.value = { task: 'cancelled', host: '', reason: 'Operation cancelled by user' }
  if (tasks.value.length > 0) {
    tasks.value = tasks.value.map(t =>
      t.status === 'running' ? { ...t, status: 'cancelled' as const } : t
    )
  }
  stopPolling()
  disconnectAll()

  try {
    await api.cancelInstall()
  } catch {
  }
}

async function recoverLogs() {
  if (recoveryInProgress.value || logsRecovered.value) {
    return
  }

  recoveryInProgress.value = true
  logStatus.value = 'recovering'
  try {
    const data = await api.getInstallLogs()
    if (data.logs && data.logs.length > 0) {
      logs.value = [...data.logs]
      logsRecovered.value = true
      logStatus.value = 'recovered'
    }
  } catch (e) {
    console.error('Failed to recover logs:', e)
    logStatus.value = 'disconnected'
  } finally {
    recoveryInProgress.value = false
  }
}

async function fetchConfigStatus() {
  try {
    const res = await api.getConfigStatus()
    const session = useSessionStore()

    if (!res.success) {
      configError.value = res.error ?? 'Failed to read config status'
      configStatus.value = null
      configExists.value = false
      configFoundOnDisk.value = false
      return
    }

    const status = res.status ?? null
    configStatus.value = status
    configPath.value = res.path ?? ''
    configError.value = null

    if (status === 'matches') {
      configExists.value = true
      configFoundOnDisk.value = false
      configContent.value = res.content ?? ''
      session.setConfigGenerated(true)
    } else if (status === 'outdated') {
      configExists.value = false
      configFoundOnDisk.value = true
      configContent.value = res.content ?? ''
      session.setConfigGenerated(false)
    } else {
      configExists.value = false
      configFoundOnDisk.value = false
      configContent.value = ''
      session.setConfigGenerated(false)
    }
  } catch (e) {
    configError.value = String(e)
    configStatus.value = null
  }
}

async function generateConfig(skipBackup: boolean = false) {
  status.value = 'idle'
  progress.value = 0
  tasks.value = []
  error.value = null
  logs.value = []
  searchQuery.value = ''
  isStarted.value = false
  logStatus.value = 'none'

  configError.value = null
  configContent.value = ''
  try {
    const { useTopologyStore } = await import('@/stores/topology')
    const topologyStore = useTopologyStore()
    const topology = topologyStore.selected ?? undefined

    const res = await api.generateConfig(topology, skipBackup)
    if (res.success) {
      configExists.value = true
      configFoundOnDisk.value = false
      configPath.value = res.path ?? ''
      configContent.value = res.content ?? ''
      configStatus.value = 'matches'
      const session = useSessionStore()
      session.setConfigGenerated(true)
    } else {
      configError.value = res.error ?? 'Failed to generate config'
    }
  } catch (e) {
    configError.value = String(e)
  }
}

async function startYdbInstall() {
  await start(OPERATIONS.ydb.startEndpoint)
}

async function installWithExistingConfig() {
  configExists.value = true
  configFoundOnDisk.value = false
  try {
    const res = await api.getConfigStatus()
    if (res.success && res.content) {
      configContent.value = res.content
      configPath.value = res.path ?? ''
    }
  } catch {
  }
  const session = useSessionStore()
  session.setConfigGenerated(true)
}

async function restartInstallation(endpoint: string = OPERATIONS.ydb.startEndpoint) {
  const savedConfigExists = configExists.value
  const savedConfigFoundOnDisk = configFoundOnDisk.value
  const savedConfigPath = configPath.value
  const savedConfigContent = configContent.value
  const savedConfigError = configError.value

  progress.value = 0
  tasks.value = []
  error.value = null
  logs.value = []
  searchQuery.value = ''

  configExists.value = savedConfigExists
  configFoundOnDisk.value = savedConfigFoundOnDisk
  configPath.value = savedConfigPath
  configContent.value = savedConfigContent
  configError.value = savedConfigError

  await start(endpoint)
}

export function useInstall() {
  if (!recoveryAttempted) {
    recoveryAttempted = true
    recoveryPromise = recoverState()
  }

  return {
    status,
    progress,
    tasks,
    error,
    logs,
    searchQuery,
    filteredLogs,
    completedTasks,
    isStarted,
    start,
    cancel,
    logStatus,
    logsRecovered,
    recoveryInProgress,
    recoverLogs,
    recoverState,
    waitForRecovery,
    configExists,
    configFoundOnDisk,
    configPath,
    configContent,
    configError,
    configStatus,
    fetchConfigStatus,
    generateConfig,
    startYdbInstall,
    installWithExistingConfig,
    restartInstallation,
    connectWithBackoff,
    disconnectAll,
  }
}
