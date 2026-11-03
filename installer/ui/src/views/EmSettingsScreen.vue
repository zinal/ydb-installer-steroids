<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useEmStore } from '@/stores/em'
import { useHostsStore } from '@/stores/hosts'
import { useHostChecks } from '@/composables/useHostChecks'
import EmHostInput from '@/components/hosts/EmHostInput.vue'
import EmDistributionFields from '@/components/em/EmDistributionFields.vue'
import CheckResultsList from '@/components/checks/CheckResultsList.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'
import type { SSHConfig } from '@/types/inventory'

const session = useSessionStore()
const emStore = useEmStore()
const hostsStore = useHostsStore()
const { t } = useI18n()

const { checkResults, isChecking, checkError, checkHosts, clearResults, timeSyncSummary } = useHostChecks()

watch(
  () => emStore.hosts.length,
  (len) => {
    if (len === 0) emStore.addEmptyHost()
  },
  { immediate: true },
)

const emHostFqdns = computed(() =>
  emStore.hosts.map(h => h.fqdn).filter(h => h.trim() !== ''),
)

const hasDuplicates = computed(() => emStore.hasDuplicates.length > 0)
const hasFilledHost = computed(() => emHostFqdns.value.length > 0)

const checksHaveBeenRun = computed(() => checkResults.value.length > 0)

const allHostsPassed = computed(() => {
  if (checkResults.value.length === 0) return false
  if (emHostFqdns.value.length === 0) return false
  return checkResults.value.every(result => {
    if (!emHostFqdns.value.includes(result.host.trim())) return false
    return result.ssh && result.python
  })
})

const connectionSummary = computed(() => ({
  endpoint: emStore.dbConnectionEndpoint.trim() || emStore.derivedEndpoint,
  database: emStore.databasePath,
}))

const canProceed = computed(() =>
  hasFilledHost.value
  && !hasDuplicates.value
  && allHostsPassed.value
  && emStore.distCanProceed
  && emStore.paramsFieldsValid,
)

watch(canProceed, (v) => session.setCanGoNext(v), { immediate: true })

watch(() => emStore.hosts, () => clearResults(), { deep: true })

function onAddHost() {
  if (emStore.hosts.length >= 1) return
  emStore.addEmptyHost()
}

function onHostUpdate(index: number, fqdn: string) {
  emStore.updateHost(index, fqdn)
}

function onRemoveHost(index: number) {
  if (emStore.hosts.length <= 1) return
  emStore.removeHost(index)
}

async function handleCheckHosts() {
  const ssh = hostsStore.ssh
  const sshConfigData: SSHConfig = {
    login: ssh?.login ?? 'root',
    auth_type: 'key',
    key_path: ssh?.key_path ?? '',
    use_jump_host: !!ssh?.use_jump_host,
    jump_host_fqdn: ssh?.jump_host_fqdn,
    custom_ssh_args: ssh?.custom_ssh_args,
  }
  await checkHosts(emHostFqdns.value, sshConfigData, [8780, 8789, 8787])
}

const driftIcon = computed(() => {
  if (!timeSyncSummary.value) return 'mdi-clock-outline'
  switch (timeSyncSummary.value.status) {
    case 'ok': return 'mdi-check-circle'
    case 'warning': return 'mdi-alert-circle'
    case 'error': return 'mdi-close-circle'
    default: return 'mdi-clock-outline'
  }
})

const driftIconColor = computed(() => {
  if (!timeSyncSummary.value) return '#6b7280'
  switch (timeSyncSummary.value.status) {
    case 'ok': return '#10b981'
    case 'warning': return '#f59e0b'
    case 'error': return '#ef4444'
    default: return '#6b7280'
  }
})
</script>

<template>
  <div class="em-settings-shell">
    <header class="em-settings-head">
      <ScreenEyebrow />
      <h1 class="em-settings-title">{{ t('emSettings.heading') }}</h1>
      <p class="em-settings-sub">{{ t('emSettings.subtitle') }}</p>
    </header>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('emSettings.connectionLabel') }}</div>
      <div class="summary-grid">
        <div class="summary-row">
          <span class="summary-key">{{ t('emSettings.endpoint') }}</span>
          <code class="summary-val">{{ connectionSummary.endpoint || '—' }}</code>
        </div>
        <div class="summary-row">
          <span class="summary-key">{{ t('emSettings.database') }}</span>
          <code class="summary-val">{{ connectionSummary.database || '—' }}</code>
        </div>
      </div>
      <p class="summary-hint">{{ t('emSettings.connectionHint') }}</p>
    </v-card>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('emSettings.emHostsLabel') }}</div>
      <EmHostInput
        :hosts="emStore.hosts"
        :max-hosts="1"
        @add="onAddHost"
        @update="onHostUpdate"
        @remove="onRemoveHost"
      />
    </v-card>

    <div class="check-button-container">
      <v-btn
        color="primary"
        :loading="isChecking"
        :disabled="!hasFilledHost"
        class="check-button"
        @click="handleCheckHosts"
      >
        {{ t('emSettings.checkHosts') }}
      </v-btn>
    </div>

    <CheckResultsList
      :results="checkResults"
      :is-checking="isChecking"
    />

    <div
      v-if="timeSyncSummary"
      :class="['drift-banner', `drift-banner--${timeSyncSummary.status}`]"
    >
      <v-icon size="18" :color="driftIconColor">{{ driftIcon }}</v-icon>
      <span class="drift-banner-text">
        <template v-if="timeSyncSummary.status === 'ok'">
          {{ t('hosts.driftOk', { drift: timeSyncSummary.maxDriftMs }) }}
        </template>
        <template v-else-if="timeSyncSummary.status === 'warning'">
          {{ t('hosts.driftWarning', { drift: timeSyncSummary.maxDriftMs, threshold: timeSyncSummary.threshold.warningMs }) }}
        </template>
        <template v-else-if="timeSyncSummary.maxDriftMs === 0">
          {{ t('hosts.driftNtpMissing') }}
        </template>
        <template v-else>
          {{ t('hosts.driftError', { drift: timeSyncSummary.maxDriftMs, threshold: timeSyncSummary.threshold.errorMs }) }}
        </template>
      </span>
    </div>

    <v-alert v-if="checkError" type="error" class="mb-4">{{ checkError }}</v-alert>

    <div v-if="hasDuplicates" class="validation-message validation-message--warning mb-4">
      <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
      <span>{{ t('emSettings.duplicateHosts') }}</span>
    </div>

    <div
      v-if="hasFilledHost && !checksHaveBeenRun && !isChecking"
      class="validation-message validation-message--warning mb-4"
    >
      <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
      <span>{{ t('emSettings.checkRequired') }}</span>
    </div>

    <div class="section-label mb-3 mt-2">{{ t('emSettings.distributionLabel') }}</div>
    <EmDistributionFields />
  </div>
</template>

<style scoped>
.em-settings-head { margin-bottom: 28px; }
.em-settings-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.em-settings-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
}

.section-card { background: rgb(var(--v-theme-surface)); }
.section-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #94a3b8;
}

.summary-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.summary-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}
.summary-key {
  font-size: 13px;
  color: #64748b;
  min-width: 90px;
}
.summary-val {
  font-size: 13px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  color: #0f172a;
  background: #f8fafc;
  padding: 2px 8px;
  border-radius: 6px;
}
.summary-hint {
  margin: 12px 0 0;
  font-size: 12.5px;
  color: #94a3b8;
  line-height: 1.45;
}

.check-button-container {
  display: flex;
  justify-content: flex-start;
  margin: 0 0 16px;
}
.check-button { text-transform: none; font-weight: 600; }

.drift-banner {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 10px;
  margin-bottom: 16px;
  font-size: 13px;
  line-height: 1.45;
}
.drift-banner--ok { background: #ecfdf5; color: #065f46; }
.drift-banner--warning { background: #fffbeb; color: #92400e; }
.drift-banner--error { background: #fef2f2; color: #991b1b; }
.drift-banner-text { flex: 1; }

.validation-message {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  padding: 10px 12px;
  border-radius: 8px;
}
.validation-message--warning {
  background: #fffbeb;
  color: #92400e;
}
</style>
