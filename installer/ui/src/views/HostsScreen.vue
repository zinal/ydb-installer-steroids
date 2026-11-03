<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useHostsStore, type DCHosts } from '@/stores/hosts'
import { useTopologyStore } from '@/stores/topology'
import { useHostChecks } from '@/composables/useHostChecks'
import { useHostFileImport } from '@/composables/useHostFileImport'
import SshSettings from '@/components/forms/SshSettings.vue'
import DCHostInput from '@/components/hosts/DCHostInput.vue'
import CheckResultsList from '@/components/checks/CheckResultsList.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'
import type { SSHConfig } from '@/types/inventory'
import type { SshConfig as FormSshConfig } from '@/types/install'
import { defaultSshConfig } from '@/types/install'
import { parseJumpHost, updateJumpHost } from '@/utils/sshArgs'

const session = useSessionStore()
const hostsStore = useHostsStore()
const topologyStore = useTopologyStore()
const { t } = useI18n()

const { checkResults, isChecking, checkError, checkHosts, clearResults, timeSyncSummary } = useHostChecks()

const sshConfig = ref<FormSshConfig>(defaultSshConfig())

let isUpdatingJumpHost = false

let sshHydrated = false

// Reactive hydration rule: never use a one-shot onMounted to read hostsStore.ssh.
// hostsStore.ssh is populated asynchronously by useInventorySync() (wired in
// App.vue's onMounted). Child onMounted fires BEFORE parent's, so a one-shot
// read would always see null on fresh page load — the form silently stays at
// hardcoded defaults forever. A guarded watcher with immediate: true fires
// immediately AND reacts when hostsStore.ssh transitions from null to a value.
watch(
  () => hostsStore.ssh,
  (invSsh) => {
    if (sshHydrated || !invSsh) return
    sshHydrated = true

    const useJumpHost = invSsh.use_jump_host || !!invSsh.jump_host_fqdn

    isUpdatingJumpHost = true

    sshConfig.value = {
      login: invSsh.login ?? 'root',
      keyPath: invSsh.key_path ?? undefined,
      useJumpHost,
      jumpHost: invSsh.jump_host_fqdn,
      customSshArgs: invSsh.custom_ssh_args,
    }

    setTimeout(() => {
      isUpdatingJumpHost = false
    }, 0)
  },
  { immediate: true },
)

watch(sshConfig, (newConfig) => {
  const sshData: SSHConfig = {
    login: newConfig.login,
    auth_type: 'key',
    key_path: newConfig.keyPath ?? '',
    use_jump_host: newConfig.useJumpHost,
    jump_host_fqdn: newConfig.useJumpHost ? newConfig.jumpHost : undefined,
    custom_ssh_args: newConfig.customSshArgs,
  }
  hostsStore.setSsh(sshData)
}, { deep: true })

watch(() => sshConfig.value.jumpHost, (newJumpHost) => {
  if (!sshConfig.value.useJumpHost || isUpdatingJumpHost) return
  
  const currentArgs = sshConfig.value.customSshArgs ?? ''
  const newArgs = updateJumpHost(currentArgs, newJumpHost)
  
  if (sshConfig.value.customSshArgs !== newArgs) {
    sshConfig.value.customSshArgs = newArgs || undefined
  }
})

watch(() => sshConfig.value.customSshArgs, (newArgs) => {
  if (isUpdatingJumpHost) return
  
  const parsed = parseJumpHost(newArgs ?? '')
  
  if (parsed && parsed.host !== sshConfig.value.jumpHost) {
    isUpdatingJumpHost = true
    sshConfig.value.jumpHost = parsed.host
    sshConfig.value.useJumpHost = true
    setTimeout(() => {
      isUpdatingJumpHost = false
    }, 0)
  }
})

const dcNames = computed(() => ['DC-1', 'DC-2', 'DC-3'] as const)

const topologyLabel = computed(() => {
  const labels: Record<string, string> = {
    'mirror-3-dc': 'mirror-3-dc',
    'block-4-2': 'block-4-2',
    'bridge-2-dc': 'bridge-2-dc',
    'none': 'none',
  }
  return labels[topologyStore.selected || 'none']
})

const isValid = computed(() => {
  if (topologyStore.selected == 'mirror-3-dc') return dcNames.value.every(dc => hostsStore.dcHosts[dc].length > 0)
  if (topologyStore.selected == 'block-4-2') {
    const totalHosts = Object.values(hostsStore.dcHosts).reduce((sum, hosts) => sum + hosts.length, 0)
    return totalHosts >= 8
  }
  return false
})

const isDCCount = computed(() => {
  return topologyStore.selected == 'mirror-3-dc'
})

const canCheck = computed(() => {
  const allHosts = [
    ...hostsStore.dcHosts['DC-1'],
    ...hostsStore.dcHosts['DC-2'],
    ...hostsStore.dcHosts['DC-3']
  ].filter(h => h.trim() !== '')
  return allHosts.length > 0
})

const allHostsPassed = computed(() => {
  if (checkResults.value.length === 0) return false
  const allHosts = getHostsToCheck()
  if (allHosts.length === 0) return false

  return checkResults.value.every(result => {
    const host = result.host.trim()
    if (!allHosts.includes(host) || !result.ssh || !result.python) return false
    if (result.timeSync && !result.timeSync.ntpActive) return false
    return true
  })
})

const checksHaveBeenRun = computed(() => {
  return checkResults.value.length > 0
})

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

const hasDuplicates = computed(() => hostsStore.duplicates.length > 0)

const hostsTotal = computed(() => hostsStore.total)

const canProceed = computed(() => {
  return isValid.value && allHostsPassed.value && !hasDuplicates.value
})

const getHostsToCheck = () => {
  return [
    ...hostsStore.dcHosts['DC-1'],
    ...hostsStore.dcHosts['DC-2'],
    ...hostsStore.dcHosts['DC-3']
  ].filter(h => h.trim() !== '')
}

async function handleCheckHosts() {
  const hosts = getHostsToCheck()
  const sshConfigData: SSHConfig = {
    login: sshConfig.value.login,
    auth_type: 'key',
    key_path: sshConfig.value.keyPath ?? '',
    use_jump_host: sshConfig.value.useJumpHost,
    jump_host_fqdn: sshConfig.value.useJumpHost ? sshConfig.value.jumpHost : undefined,
    custom_ssh_args: sshConfig.value.customSshArgs,
  }
  await checkHosts(hosts, sshConfigData)

  hostsStore.clearCheckResults()
  for (const result of checkResults.value) {
    hostsStore.setCheckResult(result.host, {
      ssh: result.ssh,
      timeSync: result.timeSync ? (result.timeSync.ntpActive && result.timeSync.ntpSynced) : false,
      python: result.python,
      cpuCores: result.cpuCores,
    })
  }
}

watch([() => hostsStore.dcHosts, sshConfig], () => {
  clearResults()
}, { deep: true })

watch(canProceed, (canGo) => {
  session.setCanGoNext(canGo)
}, { immediate: true })

function handleSshUpdate(formSsh: FormSshConfig) {
  const sshConfig: SSHConfig = {
    login: formSsh.login,
    auth_type: 'key',
    key_path: formSsh.keyPath ?? '',
    use_jump_host: formSsh.useJumpHost,
    jump_host_fqdn: formSsh.jumpHost,
    jump_host_user: formSsh.jumpHost ? 'root' : undefined,
    custom_ssh_args: formSsh.customSshArgs,
  }
  hostsStore.setSsh(sshConfig)
}

function handleHostUpdate(dc: keyof DCHosts, index: number, value: string) {
  hostsStore.updateDCHost(dc, index, value)
}

function handleHostRemove(dc: keyof DCHosts, index: number) {
  hostsStore.removeDCHost(dc, index)
}

function addHostToDC(dc: keyof DCHosts) {
  hostsStore.addDCHost(dc, '')
}

const { importing, importError, readHostFile } = useHostFileImport()

const fileInputs = ref<Record<string, HTMLInputElement | null>>({
  'DC-1': null,
  'DC-2': null,
  'DC-3': null,
})

const snackbarMessage = ref('')
const showSnackbar = ref(false)

function triggerFileInput(dc: keyof DCHosts) {
  fileInputs.value[dc]?.click()
}

async function handleFileSelect(dc: keyof DCHosts, event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  const hosts = await readHostFile(file)
  if (hosts.length > 0) {
    const added = hostsStore.addHostsBatch(dc, hosts)
    snackbarMessage.value = t('hosts.hostsLoaded', { count: added.length, dc })
    showSnackbar.value = true
  } else if (importError.value) {
    snackbarMessage.value = t(importError.value)
    showSnackbar.value = true
  }

  input.value = ''
}

</script>

<template>
  <div class="hosts-shell">
    <header class="hosts-head">
      <ScreenEyebrow />
      <h1 class="hosts-title">{{ t('hosts.heading') }}</h1>
      <p class="hosts-sub">{{ t('hosts.topologyLabel', { topology: topologyLabel }) }}</p>
    </header>

    <div class="dc-grid">
      <div v-for="dc in dcNames" :key="dc" class="dc-section">
        <div class="dc-header-row">
          <div class="dc-badge-large">
            <v-icon size="16">mdi-server-network</v-icon>
            <span>{{ dc }}</span>
          </div>
          <div class="dc-actions">
            <button
              type="button"
              class="add-host-btn"
              @click="addHostToDC(dc)"
            >
              <v-icon size="14">mdi-plus</v-icon>
              <span>{{ t('hosts.addHost') }}</span>
            </button>
            <v-menu
              open-on-hover
              :close-on-content-click="false"
              :open-delay="200"
              :close-delay="200"
              location="top"
            >
              <template #activator="{ props: menuProps }">
                <button
                  type="button"
                  class="load-file-btn"
                  v-bind="menuProps"
                  :disabled="importing"
                  @click="triggerFileInput(dc)"
                >
                  <v-icon size="16">mdi-file-upload-outline</v-icon>
                </button>
              </template>
              <v-card
                max-width="360"
                class="file-import-help-card"
                elevation="4"
                rounded="lg"
              >
                <v-card-text class="file-import-help-body">
                  <p class="file-import-help-text">{{ t('hosts.loadFromFileTooltip') }}</p>
                </v-card-text>
              </v-card>
            </v-menu>
          </div>
        </div>
        <input
          :ref="el => { fileInputs[dc] = el as HTMLInputElement | null }"
          type="file"
          accept=".txt"
          style="display: none"
          @change="handleFileSelect(dc, $event)"
        />
        
        <div class="dc-hosts-list">
          <DCHostInput
            v-for="(host, index) in hostsStore.dcHosts[dc]"
            :key="`${dc}-${index}`"
            :dc-name="dc"
            :hosts="hostsStore.dcHosts[dc]"
            :index="index"
            @update="handleHostUpdate"
            @remove="handleHostRemove"
          />
        </div>
      </div>
    </div>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('hosts.sshSettings') }}</div>
      <SshSettings v-model="sshConfig" />
    </v-card>

    <div class="check-button-container">
      <v-btn
        color="primary"
        :loading="isChecking"
        :disabled="!canCheck"
        class="check-button"
        @click="handleCheckHosts"
      >
        {{ t('hosts.checkHosts') }}
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
      <v-icon size="18" :color="driftIconColor">
        {{ driftIcon }}
      </v-icon>
      <span class="drift-banner-text">
        <template v-if="timeSyncSummary.status === 'ok'">
          {{ t('hosts.driftOk', { drift: timeSyncSummary.maxDriftMs }) }}
        </template>
        <template v-else-if="timeSyncSummary.ntpNotSynced">
          {{ t('hosts.driftNtpNotSynced') }}
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

    <v-alert
      v-if="checkError"
      type="error"
      class="mb-4"
    >
      {{ checkError }}
    </v-alert>

    <div
      v-if="hasDuplicates"
      class="validation-message validation-message--warning"
    >
      <div class="validation-message-header">
        <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
        <span>{{ t('hosts.duplicateTitle') }}</span>
      </div>
      <p class="validation-message-desc">{{ t('hosts.duplicateDescription') }}</p>
      <ul class="duplicate-list">
        <li v-for="host in hostsStore.duplicates" :key="host">
          <v-icon size="14" color="#d97706">mdi-close-circle-outline</v-icon>
          <strong>{{ host }}</strong>
        </li>
      </ul>
    </div>

    <div v-if="!isValid && isDCCount" class="validation-message">
      <v-icon size="18" color="#f59e0b">mdi-alert-circle</v-icon>
      <span>{{ t('hosts.minDCHostsRequired', { count: 1 }) }}</span>
    </div>

    <div v-else-if="!isValid && !isDCCount" class="validation-message">
      <v-icon size="18" color="#f59e0b">mdi-alert-circle</v-icon>
      <span>{{ t('hosts.minHostsRequired', { count: 8 }) }}</span>
    </div>

    <div v-else-if="isValid && !checksHaveBeenRun" class="validation-message validation-message--info">
      <v-icon size="18" color="#3b82f6">mdi-information</v-icon>
      <span>{{ t('hosts.checkRequired') }}</span>
    </div>

    <div v-else-if="isValid && checksHaveBeenRun && !allHostsPassed" class="validation-message validation-message--error">
      <v-icon size="18" color="#dc2626">mdi-close-circle</v-icon>
      <span>{{ t('hosts.checksFailed') }}</span>
    </div>

    <v-snackbar v-model="showSnackbar" :timeout="3000" color="primary">
      {{ snackbarMessage }}
    </v-snackbar>
  </div>
</template>

<style scoped>
.hosts-head { margin-bottom: 28px; }
.hosts-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.hosts-sub {
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
  margin-bottom: 14px;
}

.card {
  background: #fff;
  border: 1.5px solid #e2e8f0;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f1f5f9;
}

.card-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.card-title {
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
}

.ssh-card {
  background: linear-gradient(135deg, #fff 0%, #f8fafc 100%);
}

.dc-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

@media (max-width: 768px) {
  .dc-grid {
    grid-template-columns: 1fr;
  }
}

.dc-section {
  background: #fff;
  border: 1.5px solid #e2e8f0;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}

.dc-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.dc-badge-large {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  color: #0369a1;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.add-host-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 6px;
  color: #16a34a;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.add-host-btn:hover {
  background: #dcfce7;
  border-color: #86efac;
  transform: translateY(-1px);
}

.add-host-btn:active {
  transform: translateY(0);
}

.dc-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.load-file-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
  color: #1d4ed8;
  cursor: pointer;
  transition: all 0.15s ease;
}

.load-file-btn:hover {
  background: #dbeafe;
  border-color: #93c5fd;
  transform: translateY(-1px);
}

.load-file-btn:active {
  transform: translateY(0);
}

.load-file-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.dc-hosts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.validation-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #fffbeb;
  border: 1.5px solid #fcd34d;
  border-radius: 8px;
  color: #92400e;
  font-size: 14px;
}

.validation-message--info {
  background: #eff6ff;
  border-color: #93c5fd;
  color: #1e40af;
}

.validation-message--error {
  background: #fef2f2;
  border-color: #fca5a5;
  color: #991b1b;
}

.drift-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 8px;
  margin-bottom: 12px;
  font-size: 0.8125rem;
  font-weight: 500;
}

.drift-banner--ok {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #065f46;
}

.drift-banner--warning {
  background: #fffbeb;
  border: 1px solid #fcd34d;
  color: #92400e;
}

.drift-banner--error {
  background: #fef2f2;
  border: 1px solid #fca5a5;
  color: #991b1b;
}

.drift-banner-text {
  flex: 1;
}

.validation-message--warning {
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 16px;
  background: #fffbeb;
  border-color: #fcd34d;
  color: #92400e;
}

.validation-message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
}

.validation-message-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.45;
  color: #92400e;
}

.check-results {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 20px;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #fff;
  border: 1.5px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  transition: all 0.15s ease;
}

.result-item.result-ok {
  border-color: #86efac;
  background: #f0fdf4;
}

.result-item.result-error {
  border-color: #fca5a5;
  background: #fef2f2;
}

.host-name {
  font-weight: 600;
  color: #0f172a;
  flex-shrink: 0;
}

.status-ok {
  color: #16a34a;
  font-weight: 500;
}

.status-error {
  color: #dc2626;
  font-weight: 500;
}

.check-button-container {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 24px 0;
}

.check-button {
  min-width: 200px;
  justify-content: center;
}

.file-import-help-card {
  background: #f1f5f9 !important;
  border: 1px solid #e2e8f0;
}

.file-import-help-body {
  padding: 10px 14px !important;
}

.file-import-help-text {
  margin: 0;
  font-size: 13px;
  font-weight: 400;
  line-height: 1.5;
  color: #334155;
  white-space: pre-line;
}

.duplicate-list {
  list-style: none;
  padding: 0;
  margin: 0;
  width: 100%;
}

.duplicate-list li {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 0 3px 26px;
  font-size: 13px;
  color: #92400e;
  line-height: 1.4;
}

</style>
