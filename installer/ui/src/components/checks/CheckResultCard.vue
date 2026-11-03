<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CheckHostResult } from '@/types/hosts'

interface Props {
  result: CheckHostResult
}

const props = defineProps<Props>()
const { t } = useI18n()

const showDetails = ref(false)

const overallStatus = computed(() => {
  if (!props.result.ssh && !props.result.python) return 'error'
  if (!props.result.ssh || !props.result.python || !props.result.ports) return 'warning'
  // NTP daemon completely missing is an error
  if (props.result.timeSync && !props.result.timeSync.ntpActive) return 'error'
  // NTP running but not synced is a warning
  if (props.result.timeSync && props.result.timeSync.ntpActive && !props.result.timeSync.ntpSynced) return 'warning'
  return 'success'
})

const timeSyncStatus = computed(() => {
  const ts = props.result.timeSync
  if (!ts) return null
  if (!ts.ntpActive) return { state: 'error' as const }
  if (!ts.ntpSynced) return { state: 'warning' as const }
  return { state: 'success' as const, daemon: ts.ntpDaemon, offset: ts.clockOffsetMs }
})

const hasExpandableErrors = computed(() => {
  return !!(props.result.sshError || props.result.pythonError || !props.result.ports ||
    (props.result.timeSync && props.result.timeSync.error))
})

const portStatus = computed(() => {
  if (!props.result.portDetails || props.result.ports !== false) return null
  const occupied = props.result.portDetails.filter(p => !p.available).length
  const total = props.result.portDetails.length
  return { occupied, total }
})

const statusIcon = computed(() => {
  switch (overallStatus.value) {
    case 'success':
      return 'mdi-check-circle'
    case 'warning':
      return 'mdi-alert-circle'
    case 'error':
      return 'mdi-close-circle'
    default:
      return 'mdi-help-circle'
  }
})

const statusColor = computed(() => {
  switch (overallStatus.value) {
    case 'success':
      return '#10b981'
    case 'warning':
      return '#f59e0b'
    case 'error':
      return '#ef4444'
    default:
      return '#6b7280'
  }
})

function toggleDetails() {
  showDetails.value = !showDetails.value
}

function copyError() {
  const portsText = props.result.portsError ? `\nPorts: ${props.result.portsError}` : ''
  const timeSyncText = props.result.timeSync?.error ? `\nTime: ${props.result.timeSync.error}` : ''
  const errorText = `${props.result.host}\nSSH: ${props.result.sshError || 'OK'}\nPython: ${props.result.pythonError || props.result.pythonVersion || 'Not found'}${portsText}${timeSyncText}`
  navigator.clipboard.writeText(errorText)
}
</script>

<template>
  <div class="check-result-card">
    <div class="check-result-header">
      <v-icon :color="statusColor" size="20">
        {{ statusIcon }}
      </v-icon>
      <div class="check-result-host">
        {{ result.host }}
      </div>
      <v-btn
        v-if="hasExpandableErrors"
        icon
        size="x-small"
        variant="text"
        @click="toggleDetails"
      >
        <v-icon size="20">
          {{ showDetails ? 'mdi-chevron-up' : 'mdi-chevron-down' }}
        </v-icon>
      </v-btn>
    </div>

    <div class="check-result-summary">
      <div class="check-result-item">
        <span class="check-result-label">{{ t('hosts.ssh') }}:</span>
        <span :class="['check-result-value', result.ssh ? 'success' : 'error']">
          {{ result.ssh ? t('hosts.sshOk') : t('hosts.sshFailed') }}
        </span>
      </div>
      <div class="check-result-item">
        <span class="check-result-label">{{ t('hosts.python') }}:</span>
        <span :class="['check-result-value', result.python ? 'success' : 'error']">
          {{ result.python ? t('hosts.pythonOk', { version: result.pythonVersion }) : t('hosts.pythonFailed') }}
        </span>
      </div>
      <div v-if="timeSyncStatus" class="check-result-item">
        <span class="check-result-label">{{ t('hosts.timeSyncLabel') }}:</span>
        <span :class="['check-result-value', timeSyncStatus.state]">
          <template v-if="timeSyncStatus.state === 'success'">
            {{ t('hosts.timeSyncOk', { daemon: timeSyncStatus.daemon, offset: timeSyncStatus.offset }) }}
          </template>
          <template v-else-if="timeSyncStatus.state === 'warning'">
            {{ t('hosts.timeSyncNotSynced', { daemon: result.timeSync.ntpDaemon }) }}
          </template>
          <template v-else>
            {{ result.timeSync?.error || t('hosts.timeSyncNoDaemon') }}
          </template>
        </span>
      </div>
      <div v-if="portStatus" class="check-result-item">
        <span class="check-result-label">{{ t('hosts.ports') }}:</span>
        <span class="check-result-value error">
          {{ t('hosts.portOccupied', { occupied: portStatus.occupied, total: portStatus.total }) }}
        </span>
      </div>
    </div>

    <div v-if="showDetails && hasExpandableErrors" class="check-result-details">
      <div v-if="result.sshError" class="check-result-error">
        <div class="check-result-error-text">{{ result.sshError }}</div>
      </div>
      <div v-if="result.pythonError" class="check-result-error">
        <div class="check-result-error-text">{{ result.pythonError }}</div>
      </div>
      <div v-if="result.timeSync?.error" class="check-result-error">
        <div class="check-result-error-text">{{ result.timeSync.error }}</div>
      </div>
      <template v-if="!result.ports && result.portDetails">
        <div
          v-for="port in result.portDetails.filter(p => !p.available)"
          :key="port.port"
          class="check-result-error"
        >
          <div class="check-result-error-text">{{ port.error }}</div>
        </div>
      </template>
      <v-btn
        size="x-small"
        variant="outlined"
        prepend-icon="mdi-content-copy"
        @click="copyError"
      >
        {{ t('hosts.copyError') }}
      </v-btn>
    </div>
  </div>
</template>

<style scoped>
.check-result-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 8px;
}

.check-result-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.check-result-host {
  flex: 1;
  font-weight: 500;
  font-size: 0.875rem;
  color: #1e293b;
}

.check-result-summary {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.check-result-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.check-result-label {
  font-weight: 500;
  color: #64748b;
  min-width: 40px;
  font-size: 0.8125rem;
}

.check-result-value {
  font-weight: 500;
  font-size: 0.8125rem;
}

.check-result-value.success {
  color: #10b981;
}

.check-result-value.warning {
  color: #f59e0b;
}

.check-result-value.error {
  color: #ef4444;
}

.check-result-details {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #e2e8f0;
}

.check-result-error {
  margin-bottom: 8px;
}

.check-result-error:last-child {
  margin-bottom: 8px;
}

.check-result-error-text {
  font-family: 'Courier New', monospace;
  font-size: 0.75rem;
  color: #ef4444;
  background: #fef2f2;
  padding: 6px;
  border-radius: 4px;
  word-break: break-all;
}
</style>
