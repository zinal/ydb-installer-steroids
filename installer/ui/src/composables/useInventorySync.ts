import { onMounted, onUnmounted, ref } from 'vue'
import { useHostsStore } from '@/stores/hosts'
import { useDisksStore } from '@/stores/disks'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useDistributionStore } from '@/stores/distribution'
import { useTlsStore } from '@/stores/tls'
import { useDatabasesStore } from '@/stores/databases'
import { useTopologyStore } from '@/stores/topology'
import { api } from '@/api/client'
import type { InventoryData } from '@/types/inventory'

const DEBOUNCE_MS = 500

export function useInventorySync() {
  const ws = ref<WebSocket | null>(null)
  const isSyncing = ref(false)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let unsubscribeFns: Array<() => void> = []
  // Pending-changes flag: any store mutation marks the state dirty; the flag
  // survives an in-flight PUT so a trailing flush always persists the newest
  // state — fast typing must never lose a host.
  let dirty = false
  // Suppresses write scheduling while a WS reload applies server data, breaking
  // the reload → apply → watchers → write → file change → reload loop.
  let applying = false

  function collectInventoryData(): InventoryData {
    const hostsStore = useHostsStore()
    const disksStore = useDisksStore()
    const ydbConfigStore = useYdbConfigStore()
    const distributionStore = useDistributionStore()
    const tlsStore = useTlsStore()
    const databasesStore = useDatabasesStore()
    const topologyStore = useTopologyStore()

    const data: InventoryData = {}

    const isValidHostname = (host: string): boolean => {
      if (!host || host.trim() === '') return false
      if (/\s/.test(host)) return false
      if (/["']/.test(host)) return false
      if (/GET|POST|PUT|DELETE|websocket|Connection|Sec-WebSocket-Key/.test(host)) return false
      const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$|^(\d{1,3}\.){3}\d{1,3}$/
      return hostnameRegex.test(host)
    }

    const filterHosts = (hosts: string[]): string[] =>
      [...new Set(hosts.filter(h => isValidHostname(h)))]

    const dc1 = filterHosts(hostsStore.dcHosts['DC-1'])
    const dc2 = filterHosts(hostsStore.dcHosts['DC-2'])
    const dc3 = filterHosts(hostsStore.dcHosts['DC-3'])

    if (dc1.length > 0 || dc2.length > 0 || dc3.length > 0) {
      data.dc1 = dc1
      data.dc2 = dc2
      data.dc3 = dc3
      data.hosts = [...new Set([...dc1, ...dc2, ...dc3])]
    }
    if (hostsStore.ssh) {
      data.ssh = { ...hostsStore.ssh }
    }

    if (disksStore.selectedDisks.length > 0 || disksStore.allowFormat) {
      data.disks = {
        allow_format: disksStore.allowFormat,
        selected_disks: disksStore.selectedDisks.map(d => ({ path: d.path, label: d.label })),
      }
    }

    if (ydbConfigStore.domain || ydbConfigStore.coresStatic > 0 || ydbConfigStore.coresDynamic > 0) {
      data.ydb_config = ydbConfigStore.toYdbConfig()
    }

    if (distributionStore.source) {
      data.distribution = {
        source: distributionStore.source,
        version: distributionStore.version,
        archive_path: distributionStore.archivePath,
        binary_path: distributionStore.binaryPath,
        cli_binary: distributionStore.cliBinary,
        ops_binary: distributionStore.opsBinary,
        tool_binary: distributionStore.dstoolBinary,
      }
    }

    if (tlsStore.certPath) {
      data.tls = {
        cert_path: tlsStore.certPath,
      }
    }

    if (databasesStore.entries.length > 0 && databasesStore.entries.some(e => e.name)) {
      data.database = databasesStore.toDatabaseConfig()
    }

    if (topologyStore.selected) {
      data.topology = {
        type: topologyStore.selected,
      }
    }

    return data
  }

  function applyToStores(data: InventoryData) {
    const hostsStore = useHostsStore()
    const disksStore = useDisksStore()
    const ydbConfigStore = useYdbConfigStore()
    const distributionStore = useDistributionStore()
    const tlsStore = useTlsStore()
    const databasesStore = useDatabasesStore()
    const topologyStore = useTopologyStore()

    hostsStore.applyFromInventory(data)
    disksStore.applyFromInventory(data)
    ydbConfigStore.applyFromInventory(data)
    distributionStore.applyFromInventory(data)
    tlsStore.applyFromInventory(data)
    databasesStore.applyFromInventory(data)
    topologyStore.applyFromInventory(data)
  }

  function scheduleWrite() {
    dirty = true
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(flush, DEBOUNCE_MS)
  }

  async function flush() {
    if (isSyncing.value) return
    const data = collectInventoryData()
    isSyncing.value = true
    // The current state is about to be persisted — clear the flag. Any store
    // mutation during the PUT re-marks dirty and triggers the trailing flush.
    dirty = false
    try {
      if (Object.keys(data).length > 0) {
        await api.updateInventory(data)
      }
    } catch (e) {
      console.warn('inventory sync write failed', e)
    } finally {
      isSyncing.value = false
      if (dirty) scheduleWrite()
    }
  }

  function setupStoreWatchers() {
    const stores = [
      useHostsStore(),
      useDisksStore(),
      useYdbConfigStore(),
      useDistributionStore(),
      useTlsStore(),
      useDatabasesStore(),
      useTopologyStore(),
    ]

    for (const store of stores) {
      // flush: 'sync' keeps the applying guard accurate — the default 'pre'
      // flush defers these callbacks past applyToStores, so reload-induced
      // mutations would slip through after applying is already released.
      const unsub = store.$subscribe((_mutation, _state) => {
        if (!applying) {
          scheduleWrite()
        }
      }, { flush: 'sync' })
      unsubscribeFns.push(unsub)
    }
  }

  function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}/ws/inventory`

    try {
      ws.value = new WebSocket(url)

      ws.value.onmessage = async () => {
        try {
          const resp = await api.getInventory()
          if (resp.data) {
            // Watchers stay subscribed; the applying guard only suppresses the
            // reload's own store mutations, so keystrokes during the async
            // fetch still mark dirty and are flushed afterwards.
            applying = true
            try {
              applyToStores(resp.data)
            } finally {
              applying = false
            }
          }
        } catch (e) {
          console.warn('inventory reload failed', e)
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
      const resp = await api.getInventory()
      if (resp.data) {
        applyToStores(resp.data)
      }
    } catch {
    }
  }

  onMounted(async () => {
    await loadInitial()
    setupStoreWatchers()
    connectWebSocket()
  })

  onUnmounted(() => {
    if (debounceTimer) clearTimeout(debounceTimer)
    for (const unsub of unsubscribeFns) unsub()
    ws.value?.close()
  })

  return {
    isSyncing,
    loadInitial,
  }
}
