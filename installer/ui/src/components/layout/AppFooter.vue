<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
const router = useRouter()
const { t } = useI18n()

const nextBusy = ref(false)

const navigation = computed(() => {
  const steps = session.visibleFlowSteps
  const idx = steps.findIndex((s) => s.id === session.currentStep)
  return {
    idx,
    total: steps.length,
    back: idx > 0 ? steps[idx - 1].id : null,
    next: idx >= 0 && idx < steps.length - 1 ? steps[idx + 1].id : null,
  }
})

const isTerminal = computed(() => navigation.value.next === null && navigation.value.idx >= 0)
const progressLabel = computed(() => {
  const { idx, total } = navigation.value
  if (idx < 0 || total === 0) return ''
  return `${idx + 1} / ${total}`
})

function back() {
  if (navigation.value.back) router.push({ name: navigation.value.back })
}

async function next() {
  if (!navigation.value.next || nextBusy.value) return
  if (session.beforeNext) {
    nextBusy.value = true
    try {
      const ok = await session.beforeNext()
      if (!ok) return
    } finally {
      nextBusy.value = false
    }
  }
  session.markVisited(session.currentStep)
  router.push({ name: navigation.value.next })
}
</script>

<template>
  <v-footer app :elevation="0" color="surface" class="app-footer">
    <div class="footer-inner layout-column">
      <v-btn variant="text" :disabled="!navigation.back" @click="back" class="nav-btn">
        <v-icon start size="18">mdi-arrow-left</v-icon>
        {{ t('common.back') }}
      </v-btn>

      <div class="progress-pill" v-if="progressLabel">
        <span class="progress-pill-label">{{ progressLabel }}</span>
      </div>

      <v-btn
        v-if="!isTerminal"
        color="primary"
        :disabled="!session.canGoNext || !navigation.next || nextBusy"
        :loading="nextBusy"
        @click="next"
        class="nav-btn primary-btn"
      >
        {{ t('common.next') }}
        <v-icon end size="18">mdi-arrow-right</v-icon>
      </v-btn>
      <v-btn v-else color="primary" variant="tonal" :to="{ name: '1' }" class="nav-btn">
        {{ t('common.finish') }}
      </v-btn>
    </div>
  </v-footer>
</template>

<style scoped>
.app-footer {
  border-top: 1px solid rgb(var(--v-theme-border-soft));
  min-height: 64px;
  padding: 0;
}
.app-footer :deep(.v-footer__content) { padding: 0; }

/* Width/centering comes from the shared `.layout-column` utility
 * (see styles/app.css) so the nav buttons align with the content column. */
.footer-inner {
  display: flex;
  align-items: center;
  padding: 12px var(--content-gutter);
  gap: 16px;
}

.nav-btn {
  font-weight: 500;
  letter-spacing: -0.005em;
}
.primary-btn { box-shadow: 0 1px 2px rgba(31,109,217,.25), 0 4px 12px -2px rgba(31,109,217,.18) !important; }

.progress-pill {
  margin-left: auto;
  margin-right: auto;
  background: #f1f5f9;
  border-radius: 999px;
  padding: 6px 14px;
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
  font-variant-numeric: tabular-nums;
}
</style>
