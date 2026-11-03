<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { InstallStatus } from '@/types/install'

const props = defineProps<{
  progress: number
  completedTasks: number
  totalTasks: number
  status: InstallStatus
}>()

const { t } = useI18n()

const barColor = computed(() => {
  switch (props.status) {
    case 'success': return 'success'
    case 'error':   return 'error'
    case 'running': return 'primary'
    default:        return 'primary'
  }
})

const label = computed(() => {
  if (props.status === 'success') return '✅ ' + t('install.progress', { done: props.completedTasks, total: props.totalTasks })
  if (props.status === 'error')   return '❌ ' + t('install.errorTitle')
  return t('install.progress', { done: props.completedTasks, total: props.totalTasks })
})
</script>

<template>
  <div class="progress-shell">
    <div class="progress-header">
      <span class="progress-pct">{{ progress }}%</span>
      <span class="progress-label">{{ label }}</span>
    </div>
    <v-progress-linear
      :model-value="progress"
      :color="barColor"
      height="8"
      rounded
      bg-color="#e2e8f0"
    />
  </div>
</template>

<style scoped>
.progress-shell {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.progress-header {
  display: flex;
  align-items: baseline;
  gap: 10px;
}

.progress-pct {
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.03em;
}

.progress-label {
  font-size: 13px;
  color: #64748b;
}
</style>
