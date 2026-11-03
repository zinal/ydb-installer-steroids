<script setup lang="ts">
// Load order (fixes the URL/backend-session race that clobbered navigation
// history on refresh):
//   1. Router `beforeEach` (registered at router-creation time) positions
//      `session.currentStep`/`visited` from the URL hash.
//   2. `usePersistedState(...)` calls below rehydrate UI ephemerals from
//      sessionStorage synchronously, before first render.
//   3. `session.load()` fetches the backend's remembered ui_state and *merges*
//      it with whatever the router already set (see `session.applyServer()` —
//      it unions visited/errored sets and only adopts the backend's
//      currentStep if the router left it at the '1' default).
//   4. `useSessionSync()` / `usePersistedState()` subscriptions start watching
//      for further local mutations and mirror them back (debounced) to the
//      backend / sessionStorage respectively.
import { onMounted, onUnmounted } from 'vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppFooter from '@/components/layout/AppFooter.vue'
import { useSessionStore } from '@/stores/session'
import { setLocale } from '@/plugins/i18n'
import { useInventorySync } from '@/composables/useInventorySync'
import { useEMInventorySync } from '@/composables/useEMInventorySync'
import { useSessionSync } from '@/composables/useSessionSync'
import { usePersistedState } from '@/composables/usePersistedState'
import { useDisksStore } from '@/stores/disks'
import { useTlsStore } from '@/stores/tls'
import { useDistributionStore } from '@/stores/distribution'
import { useDatabasesStore } from '@/stores/databases'
import { useHostsStore } from '@/stores/hosts'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useEmStore } from '@/stores/em'
import { useConnectivityStore } from '@/stores/connectivity'
import { useUiStore } from '@/stores/ui'
import { useRuntimeStore } from '@/stores/runtime'
import { deriveEmAzLocations } from '@/utils/emAzLocations'
import { api } from '@/api/client'

const session = useSessionStore()
const runtime = useRuntimeStore()
const connectivity = useConnectivityStore()

// EM AZ bridge: when YDB DC hosts change and EM inventory already exists,
// refresh ydb_em_az_locations in 90-inventory.yaml automatically.
// Screen-level useEMInventorySync handles the EM-flow writes (Screens 20/21);
// this subscription covers host edits on Screen 3 while EM inventory is present.
const emStore = useEmStore()
const hostsStore = useHostsStore()
let hostsDebounce: ReturnType<typeof setTimeout> | null = null
const unsubHosts = hostsStore.$subscribe(() => {
  if (!emStore.emInventoryFound) return
  if (hostsDebounce) clearTimeout(hostsDebounce)
  hostsDebounce = setTimeout(async () => {
    try {
      await api.updateEMInventory({
        ...emStore.toEMConfig(),
        az_locations: deriveEmAzLocations(hostsStore.dcHosts),
      })
    } catch {
    }
  }, 500)
})
api.getEMInventory().then((resp) => {
  emStore.setEMInventoryFound(resp.em_inventory_found)
  if (resp.data) emStore.applyFromEMConfig(resp.data)
}).catch(() => {})
onUnmounted(() => {
  unsubHosts()
  if (hostsDebounce) clearTimeout(hostsDebounce)
})

useInventorySync()
useEMInventorySync()
useSessionSync()

usePersistedState(useDisksStore(), {
  name: 'disks',
  paths: ['entryMode', 'manualDiskInput', 'manualInputError', 'dcDisks', 'diskResults', 'consistencyResult'],
})
usePersistedState(useTlsStore(), {
  name: 'tls',
  paths: ['mode', 'days'],
})
usePersistedState(useDistributionStore(), {
  name: 'distribution',
  paths: ['archiveValid', 'archiveSize'],
})
usePersistedState(useEmStore(), {
  name: 'em',
  paths: [
    'distMode',
    'archivePath', 'metaUIArchivePath',
    'metaUIBinary', 'cpBinary', 'cpAgentBinary',
    'archiveValid', 'archiveSize',
    'metaUIArchiveValid', 'metaUIArchiveSize',
    'metaUIBinaryValid', 'metaUIBinarySize',
    'cpBinaryValid', 'cpBinarySize',
    'cpAgentBinaryValid', 'cpAgentBinarySize',
  ],
})
usePersistedState(useDatabasesStore(), {
  name: 'databases',
  paths: ['savedPerVariant'],
})
usePersistedState(useHostsStore(), {
  name: 'hosts',
  paths: ['checkResults'],
})
usePersistedState(useYdbConfigStore(), {
  name: 'ydbConfig',
  paths: ['affinity'],
})
usePersistedState(useConnectivityStore(), {
  name: 'connectivity',
  paths: ['override'],
})
usePersistedState(useUiStore(), {
  name: 'ui',
  paths: ['consoleExpanded'],
})

onMounted(async () => {
  await session.load()
  runtime.load()
  // Connectivity is app-level state, not screen-level: probe once at startup so
  // the result is ready regardless of which screen the user lands on (a direct
  // refresh on screen 5 must not leave the status stuck in "checking").
  connectivity.check()

  if (typeof localStorage !== 'undefined' && !localStorage.getItem('ydb-installer.locale')) {
    if (session.defaultLang === 'ru' || session.defaultLang === 'en') {
      setLocale(session.defaultLang)
    }
  }
})
</script>

<template>
  <v-app>
    <AppHeader />
    <AppSidebar />
    <v-main class="content-area">
      <div class="content-canvas layout-column">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </v-main>
    <AppFooter />
  </v-app>
</template>

<style>
.content-area { background: rgb(var(--v-theme-background)); }
.content-canvas {
  padding: 40px var(--content-gutter) 56px;
  min-height: 100%;
}

.fade-enter-active, .fade-leave-active { transition: opacity .14s ease, transform .14s ease; }
.fade-enter-from { opacity: 0; transform: translateY(4px); }
.fade-leave-to   { opacity: 0; transform: translateY(-4px); }
</style>
