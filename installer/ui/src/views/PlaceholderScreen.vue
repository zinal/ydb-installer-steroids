<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const route = useRoute()
const { t } = useI18n()

const step = computed(() => String(route.params.step ?? route.name ?? '1'))
const heading = computed(() => t(`screens.${step.value}`))
</script>

<template>
  <div class="screen-shell">
    <header class="screen-head">
      <ScreenEyebrow />
      <h1 class="screen-title">{{ heading }}</h1>
    </header>

    <v-card class="screen-card pa-7">
      <div class="placeholder-block">
        <div class="placeholder-icon">
          <svg viewBox="0 0 24 24" fill="none" width="22" height="22">
            <path d="M12 2v6m0 0l-3-3m3 3l3-3M5 21h14a2 2 0 002-2v-7l-4-4H7a2 2 0 00-2 2v11z"
                  stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <div>
          <div class="placeholder-title">{{ t('common.stub') }}</div>
          <div class="placeholder-text">{{ t('common.placeholder') }}</div>
        </div>
      </div>
    </v-card>
  </div>
</template>

<style scoped>
.screen-head { margin-bottom: 24px; }
.screen-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0;
}

.screen-card { background: rgb(var(--v-theme-surface)); }

.placeholder-block {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}
.placeholder-icon {
  flex: 0 0 auto;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: #eef4ff;
  color: #1f6dd9;
  display: flex;
  align-items: center;
  justify-content: center;
}
.placeholder-title {
  font-weight: 600;
  font-size: 14px;
  color: #0f172a;
  margin-bottom: 4px;
}
.placeholder-text {
  font-size: 13.5px;
  color: #64748b;
  line-height: 1.55;
}
</style>
