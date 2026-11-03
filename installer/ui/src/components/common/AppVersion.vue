<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'

const { t } = useI18n()
const session = useSessionStore()
</script>

<template>
  <span class="app-version">
    <span class="app-version__number">{{ session.version || '—' }}</span>
    <span class="app-version__badge">{{ t('app.beta') }}</span>
  </span>
</template>

<style scoped>
.app-version {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.app-version__number {
  font-variant-numeric: tabular-nums;
}

.app-version__badge {
  display: inline-flex;
  align-items: center;
  min-height: 15px;
  padding: 1px 6px;
  border: 1px solid #fed7aa;
  border-radius: 999px;
  background: #fff7ed;
  color: #c2410c;
  font-size: 9px;
  font-weight: 700;
  line-height: 1;
  letter-spacing: 0.08em;
}
</style>
