<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CheckResultCard from './CheckResultCard.vue'
import type { CheckHostResult } from '@/types/hosts'

interface Props {
  results: CheckHostResult[]
  isChecking: boolean
}

const props = defineProps<Props>()
const { t } = useI18n()

const summary = computed(() => {
  const total = props.results.length
  const sshSuccess = props.results.filter(r => r.ssh).length
  const pythonSuccess = props.results.filter(r => r.python).length
  const ntpOk = props.results.filter(r => r.timeSync && r.timeSync.ntpActive && r.timeSync.ntpSynced).length
  const ntpFail = props.results.filter(r => !r.timeSync || !r.timeSync.ntpActive).length
  
  return {
    total,
    sshSuccess,
    pythonSuccess,
    sshFailed: total - sshSuccess,
    pythonFailed: total - pythonSuccess,
    ntpOk,
    ntpFail,
    ntpWarn: total - ntpOk - ntpFail
  }
})
</script>

<template>
  <div v-if="isChecking || results.length > 0" class="check-results-list">
    <div v-if="results.length > 0" class="check-results-summary">
      <div class="check-results-summary-item">
        <span class="check-results-summary-label">{{ t('hosts.total') }}:</span>
        <span class="check-results-summary-value">{{ summary.total }}</span>
      </div>
      <div class="check-results-summary-item">
        <span class="check-results-summary-label">{{ t('hosts.ssh') }}:</span>
        <span class="check-results-summary-value success">{{ summary.sshSuccess }}</span>
        <span class="check-results-summary-value error">{{ summary.sshFailed }}</span>
      </div>
      <div class="check-results-summary-item">
        <span class="check-results-summary-label">{{ t('hosts.python') }}:</span>
        <span class="check-results-summary-value success">{{ summary.pythonSuccess }}</span>
        <span class="check-results-summary-value error">{{ summary.pythonFailed }}</span>
      </div>
      <div class="check-results-summary-item">
        <span class="check-results-summary-label">{{ t('hosts.timeSyncLabel') }}:</span>
        <span class="check-results-summary-value success">{{ summary.ntpOk }}</span>
        <span v-if="summary.ntpWarn > 0" class="check-results-summary-value warning">{{ summary.ntpWarn }}</span>
        <span class="check-results-summary-value error">{{ summary.ntpFail }}</span>
      </div>
    </div>

    <div class="check-results-cards">
      <CheckResultCard
        v-for="result in results"
        :key="result.host"
        :result="result"
      />
    </div>
  </div>
</template>

<style scoped>
.check-results-list {
  margin-top: 24px;
}

.check-results-summary {
  display: flex;
  gap: 24px;
  padding: 12px 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  margin-bottom: 16px;
}

.check-results-summary-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.check-results-summary-label {
  font-weight: 500;
  color: #64748b;
}

.check-results-summary-value {
  font-weight: 600;
  color: #1e293b;
}

.check-results-summary-value.success {
  color: #10b981;
}

.check-results-summary-value.warning {
  color: #f59e0b;
}

.check-results-summary-value.error {
  color: #ef4444;
}

.check-results-cards {
  display: flex;
  flex-direction: column;
}
</style>
