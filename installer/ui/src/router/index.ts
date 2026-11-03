import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import IntroScreen from '@/views/IntroScreen.vue'
import TopologyScreen from '@/views/TopologyScreen.vue'
import HostsScreen from '@/views/HostsScreen.vue'
import DisksScreen from '@/views/DisksScreen.vue'
import DistributionScreen from '@/views/DistributionScreen.vue'
import YdbConfigScreen from '@/views/YdbConfigScreen.vue'
import DatabasesScreen from '@/views/DatabasesScreen.vue'
import EmSettingsScreen from '@/views/EmSettingsScreen.vue'
import TLSScreen from '@/views/TLSScreen.vue'
import InstallationScreen from '@/views/InstallationScreen.vue'
import ResultScreen from '@/views/ResultScreen.vue'
import EmHostsScreen from '@/views/EmHostsScreen.vue'
import EmParamsScreen from '@/views/EmParamsScreen.vue'
import EmDistributionScreen from '@/views/EmDistributionScreen.vue'
import EmTlsScreen from '@/views/EmTlsScreen.vue'
import EmInstallScreen from '@/views/EmInstallScreen.vue'
import EmResultScreen from '@/views/EmResultScreen.vue'
import PlaceholderScreen from '@/views/PlaceholderScreen.vue'
import DeleteYdbConfirmScreen from '@/views/DeleteYdbConfirmScreen.vue'
import DeleteYdbProgressScreen from '@/views/DeleteYdbProgressScreen.vue'
import DeleteEmConfirmScreen from '@/views/DeleteEmConfirmScreen.vue'
import DeleteEmProgressScreen from '@/views/DeleteEmProgressScreen.vue'
import { ALL_STEP_IDS } from '@/config/flows'
import { useSessionStore } from '@/stores/session'
import { useRuntimeStore } from '@/stores/runtime'

const REAL_VIEWS: Record<string, RouteRecordRaw['component']> = {
  '1': IntroScreen,
  '2': TopologyScreen,
  '3': HostsScreen,
  '4': DisksScreen,
  '5': DistributionScreen,
  '6': YdbConfigScreen,
  '7': DatabasesScreen,
  '7.1': EmSettingsScreen,
  '8': TLSScreen,
  '9': InstallationScreen,
  '10': ResultScreen,
  '20': EmHostsScreen,
  '21': EmParamsScreen,
  '22': EmDistributionScreen,
  '23': EmTlsScreen,
  '24': EmInstallScreen,
  '25': EmResultScreen,
  '30': DeleteYdbConfirmScreen,
  '31': DeleteYdbProgressScreen,
  '32': DeleteEmConfirmScreen,
  '33': DeleteEmProgressScreen,
}

/**
 * EM-only step IDs (standalone ydb-em wizard 20–25, EM delete 32–33, and the
 * single-plus-em settings step 7.1). Gated behind --show-em-scenarios.
 */
const EM_STEP_IDS = new Set(['7.1', '20', '21', '22', '23', '24', '25', '32', '33'])

/**
 * Pure guard: an EM step is blocked only once the runtime flag has loaded and
 * EM scenarios are disabled. The `runtimeLoaded` condition lets a direct
 * refresh on an EM step (with the flag on) still resolve before the async
 * flag arrives.
 */
export function isEMStepBlocked(stepId: string, runtimeLoaded: boolean, showEmScenarios: boolean): boolean {
  return EM_STEP_IDS.has(stepId) && runtimeLoaded && !showEmScenarios
}

const routes: RouteRecordRaw[] = [
    { path: '/', redirect: { name: '1' } },
    ...ALL_STEP_IDS.map<RouteRecordRaw>((id) => ({
        path: `/screen/${id}`,
        name: id,
      component: REAL_VIEWS[id] ?? PlaceholderScreen,
  })),
    { path: '/:pathMatch(.*)*', redirect: { name: '1' } },
]

export const router = createRouter({
    // Hash history avoids needing a server-side rewrite for sub-routes when the
    // SPA is served from embed.FS — every URL still resolves to index.html.
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  const session = useSessionStore()
  const runtime = useRuntimeStore()
    if (typeof to.name === 'string') {
      if (to.name === '7.1' && !session.showEmStep) {
        next({ name: '7' })
        return
      }
      if (isEMStepBlocked(to.name, runtime.loaded, runtime.showEmScenarios)) {
        // 7.1 is part of the YDB flow — land on screen 7; standalone EM steps
        // (20–25, 32–33) go back to the intro.
        next({ name: to.name === '7.1' ? '7' : '1' })
        return
      }
      session.setCurrent(to.name)
    }
    const isPlaceholder = !(to.name && to.name in REAL_VIEWS)
    if (isPlaceholder) {
    session.setCanGoNext(true)
  }
    next()
})

export { ALL_STEP_IDS }
