import { onMounted, onUnmounted, ref } from 'vue'
import { useEmStore } from '@/stores/em'
import { useHostsStore } from '@/stores/hosts'
import { api } from '@/api/client'
import { deriveEmAzLocations } from '@/utils/emAzLocations'
import type { EMConfig } from '@/types/inventory'

const DEBOUNCE_MS = 500

export function useEMInventorySync() {
  const ws = ref<WebSocket | null>(null)
  const isSyncing = ref(false)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let unsubEmFn: (() => void) | null = null
  let unsubHostsFn: (() => void) | null = null
  let lastPayload = ''
  // Same dirty-flag/trailing-flush protocol as useInventorySync: a store
  // mutation during an in-flight PUT must never be lost.
  let dirty = false
  // Suppresses write scheduling while a WS reload applies server data.
  let applying = false

  function buildPayload(): EMConfig {
    return {
      ...useEmStore().toEMConfig(),
      az_locations: deriveEmAzLocations(useHostsStore().dcHosts),
    }
  }

  function fingerprint(config: EMConfig): string {
    return JSON.stringify(config)
  }

  function rememberPayload(config: EMConfig) {
    lastPayload = fingerprint(config)
  }

  function scheduleWrite() {
    dirty = true
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(flush, DEBOUNCE_MS)
  }

  async function flush() {
    if (isSyncing.value) return
    const config = buildPayload()
    const fp = fingerprint(config)
    isSyncing.value = true
    dirty = false
    try {
      if (fp === lastPayload) return
      await api.updateEMInventory(config)
      lastPayload = fp
    } catch (e) {
      console.warn('EM inventory sync write failed', e)
    } finally {
      isSyncing.value = false
      if (dirty) scheduleWrite()
    }
  }

  function setupStoreWatcher() {
    // flush: 'sync' keeps the applying guard accurate — the default 'pre'
    // flush defers these callbacks past applyFromEMConfig, so reload-induced
    // mutations would slip through after applying is already released.
    unsubEmFn = useEmStore().$subscribe((_mutation, _state) => {
      if (!applying) {
        scheduleWrite()
      }
    }, { flush: 'sync' })
    unsubHostsFn = useHostsStore().$subscribe((_mutation, _state) => {
      if (useEmStore().emInventoryFound && !applying) {
        scheduleWrite()
      }
    }, { flush: 'sync' })
  }

  function teardownWatchers() {
    if (unsubEmFn) { unsubEmFn(); unsubEmFn = null }
    if (unsubHostsFn) { unsubHostsFn(); unsubHostsFn = null }
  }

  function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}/ws/inventory`

    try {
      ws.value = new WebSocket(url)

      ws.value.onmessage = async () => {
        try {
          const resp = await api.getEMInventory()
          const store = useEmStore()
          if (resp.data) {
            // Watchers stay subscribed; the applying guard only suppresses the
            // reload's own store mutations, so keystrokes during the async
            // fetch still mark dirty and are flushed afterwards.
            applying = true
            try {
              store.applyFromEMConfig(resp.data)
            } finally {
              applying = false
            }
          }
          store.setEMInventoryFound(resp.em_inventory_found)
          rememberPayload(buildPayload())
        } catch (e) {
          console.warn('EM inventory reload failed', e)
        }
      }

      ws.value.onclose = () => {
        setTimeout(connectWebSocket, 3000)
      }

      ws.value.onerror = () => {
        ws.value?.close()
      }
    } catch {
    }
  }

  async function loadInitial() {
    try {
      const resp = await api.getEMInventory()
      const store = useEmStore()
      const serverPayload = resp.data ?? null
      const serverFp = serverPayload ? fingerprint({
        ...serverPayload,
        az_locations: deriveEmAzLocations(useHostsStore().dcHosts),
      }) : ''
      if (resp.data) {
        store.applyFromEMConfig(resp.data)
      }
      store.setEMInventoryFound(resp.em_inventory_found)
      lastPayload = serverFp
    } catch {
    }
  }

  onMounted(async () => {
    await loadInitial()
    setupStoreWatcher()
    const currentFp = fingerprint(buildPayload())
    if (currentFp !== lastPayload) {
      scheduleWrite()
    }
    connectWebSocket()
  })

  onUnmounted(() => {
    if (debounceTimer) clearTimeout(debounceTimer)
    teardownWatchers()
    ws.value?.close()
  })

  return {
    isSyncing,
    loadInitial,
  }
}
