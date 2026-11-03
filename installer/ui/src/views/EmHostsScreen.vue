<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useHostsStore } from '@/stores/hosts'
import { useEmStore } from '@/stores/em'
import { useHostChecks } from '@/composables/useHostChecks'
import SshSettings from '@/components/forms/SshSettings.vue'
import EmHostInput from '@/components/hosts/EmHostInput.vue'
import CheckResultsList from '@/components/checks/CheckResultsList.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'
import type { SSHConfig } from '@/types/inventory'
import type { SshConfig as FormSshConfig } from '@/types/install'
import { defaultSshConfig } from '@/types/install'
import { parseJumpHost, updateJumpHost } from '@/utils/sshArgs'

const session = useSessionStore()
const hostsStore = useHostsStore()
const emStore = useEmStore()
const { t } = useI18n()

const { checkResults, isChecking, checkError, checkHosts, clearResults, timeSyncSummary } = useHostChecks()

const sshConfig = ref<FormSshConfig>(defaultSshConfig())
let isUpdatingJumpHost = false

// Reactive hydration rule: see HostsScreen.vue for the full explanation.
// hostsStore.ssh is populated asynchronously; a one-shot onMounted would
// always see null on fresh page load. Guarded watcher with immediate: true.
let sshHydrated = false

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
    setTimeout(() => { isUpdatingJumpHost = false }, 0)
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
    setTimeout(() => { isUpdatingJumpHost = false }, 0)
  }
})

const emHostFqdns = computed(() => emStore.hosts.map(h => h.fqdn).filter(h => h.trim() !== ''))

const canCheck = computed(() => emHostFqdns.value.length > 0)

const checksHaveBeenRun = computed(() => checkResults.value.length > 0)

const allHostsPassed = computed(() => {
  if (checkResults.value.length === 0) return false
  if (emHostFqdns.value.length === 0) return false
  return checkResults.value.every(result => {
    if (!emHostFqdns.value.includes(result.host.trim())) return false
    return result.ssh && result.python
  })
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

const hasDuplicates = computed(() => emStore.hasDuplicates.length > 0)

const hasFilledHost = computed(() => emHostFqdns.value.length > 0)

const canProceed = computed(() => {
  return hasFilledHost.value && !hasDuplicates.value && allHostsPassed.value
})

watch(canProceed, (canGo) => {
  session.setCanGoNext(canGo)
}, { immediate: true })

watch([() => emStore.hosts, sshConfig], () => {
  clearResults()
}, { deep: true })

watch(
  () => emStore.hosts.length,
  (len) => {
    if (len === 0) {
      emStore.addEmptyHost()
    }
  },
  { immediate: true },
)

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
  const hosts = emHostFqdns.value
  const sshConfigData: SSHConfig = {
    login: sshConfig.value.login,
    auth_type: 'key',
    key_path: sshConfig.value.keyPath ?? '',
    use_jump_host: sshConfig.value.useJumpHost,
    jump_host_fqdn: sshConfig.value.useJumpHost ? sshConfig.value.jumpHost : undefined,
    custom_ssh_args: sshConfig.value.customSshArgs,
  }
  await checkHosts(hosts, sshConfigData, [8780, 8789, 8787])
}
</script>

<template>
  <div class="em-hosts-shell">
    <header class="em-hosts-head">
      <ScreenEyebrow />
      <h1 class="em-hosts-title">{{ t('emHosts.heading') }}</h1>
      <p class="em-hosts-sub">{{ t('emHosts.subtitle') }}</p>
    </header>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('emHosts.emHostsLabel') }}</div>
      <EmHostInput
        :hosts="emStore.hosts"
        :max-hosts="1"
        @add="onAddHost"
        @update="onHostUpdate"
        @remove="onRemoveHost"
      />
    </v-card>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('emHosts.sshSettings') }}</div>
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
        {{ t('emHosts.checkHosts') }}
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
      <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
      <span>{{ t('emHosts.duplicateWarning') }}</span>
    </div>

    <div v-if="!hasFilledHost" class="validation-message">
      <v-icon size="18" color="#f59e0b">mdi-alert-circle</v-icon>
      <span>{{ t('emHosts.minHostsRequired') }}</span>
    </div>

    <div v-else-if="!checksHaveBeenRun" class="validation-message validation-message--info">
      <v-icon size="18" color="#3b82f6">mdi-information</v-icon>
      <span>{{ t('emHosts.checkRequired') }}</span>
    </div>

    <div v-else-if="!allHostsPassed" class="validation-message validation-message--error">
      <v-icon size="18" color="#dc2626">mdi-close-circle</v-icon>
      <span>{{ t('emHosts.checksFailed') }}</span>
    </div>
  </div>
</template>

<style scoped>
.em-hosts-head { margin-bottom: 28px; }
.em-hosts-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.em-hosts-sub {
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
  margin-bottom: 16px;
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
.validation-message--warning {
  background: #fffbeb;
  border-color: #fcd34d;
  color: #92400e;
}
</style>
