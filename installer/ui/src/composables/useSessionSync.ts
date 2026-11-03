import { onMounted, onUnmounted, ref } from 'vue'
import { useSessionStore } from '@/stores/session'
import { api } from '@/api/client'

const DEBOUNCE_MS = 300

export function useSessionSync() {
  const isSyncing = ref(false)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let unsubscribe: (() => void) | null = null

  function scheduleWrite() {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(async () => {
      if (isSyncing.value) return
      isSyncing.value = true
      try {
        const session = useSessionStore()
        if (!session.hydrated) return
        await api.updateSession({ ui_state: session.toUIState() })
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn('session ui_state sync failed', e)
      } finally {
        isSyncing.value = false
      }
    }, DEBOUNCE_MS)
  }

  function setup() {
    const session = useSessionStore()
    unsubscribe = session.$subscribe((_mutation, _state) => {
      if (!isSyncing.value) {
        scheduleWrite()
      }
    })
  }

  onMounted(() => {
    setup()
  })

  onUnmounted(() => {
    if (debounceTimer) clearTimeout(debounceTimer)
    if (unsubscribe) unsubscribe()
  })

  return { isSyncing }
}
