import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import IntroScreen from '@/views/IntroScreen.vue'
import TopologyScreen from '@/views/TopologyScreen.vue'
import HostsScreen from '@/views/HostsScreen.vue'
import DisksScreen from '@/views/DisksScreen.vue'
import DistributionScreen from '@/views/DistributionScreen.vue'
import YdbConfigScreen from '@/views/YdbConfigScreen.vue'
import DatabasesScreen from '@/views/DatabasesScreen.vue'
import TLSScreen from '@/views/TLSScreen.vue'
import InstallationScreen from '@/views/InstallationScreen.vue'
import ResultScreen from '@/views/ResultScreen.vue'
import PlaceholderScreen from '@/views/PlaceholderScreen.vue'
import DeleteYdbConfirmScreen from '@/views/DeleteYdbConfirmScreen.vue'
import DeleteYdbProgressScreen from '@/views/DeleteYdbProgressScreen.vue'
import { ALL_STEP_IDS } from '@/config/flows'
import { useSessionStore } from '@/stores/session'

const REAL_VIEWS: Record<string, RouteRecordRaw['component']> = {
  '1': IntroScreen,
  '2': TopologyScreen,
  '3': HostsScreen,
  '4': DisksScreen,
  '5': DistributionScreen,
  '6': YdbConfigScreen,
  '7': DatabasesScreen,
  '8': TLSScreen,
  '9': InstallationScreen,
  '10': ResultScreen,
  '30': DeleteYdbConfirmScreen,
  '31': DeleteYdbProgressScreen,
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
    if (typeof to.name === 'string') {
      session.setCurrent(to.name)
    }
    const isPlaceholder = !(to.name && to.name in REAL_VIEWS)
    if (isPlaceholder) {
    session.setCanGoNext(true)
  }
    next()
})

export { ALL_STEP_IDS }
