<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useEmStore } from '@/stores/em'
import { useInstall } from '@/composables/useInstall'
import { OPERATIONS } from '@/config/operations'
import EmPreflightCard from '@/components/em/EmPreflightCard.vue'
import InstallProgressPanel from '@/components/common/InstallProgressPanel.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

/** Delay before auto-navigating so the success card is briefly visible. */
const SUCCESS_NAV_DELAY_MS = 300

const { t } = useI18n()
const router = useRouter()
const session = useSessionStore()
const em = useEmStore()
const install = useInstall()

onMounted(async () => {
  await install.waitForRecovery()
})

const isIdle = computed(() => install.status.value === 'idle')
const isRunning = computed(() => install.status.value === 'running')
const isSuccess = computed(() => install.status.value === 'success')

const isStarted = computed(() => install.isStarted.value)

const logStatus = computed(() => install.logStatus?.value ?? 'none')
const logsRecovered = computed(() => install.logsRecovered?.value ?? false)
const recoveryInProgress = computed(() => install.recoveryInProgress?.value ?? false)

/** Pre-flight is only actionable before the first start. */
const showPreflight = computed(() => !isStarted.value)
const showStartButton = computed(() => !isStarted.value)

watch(
  [install.status, install.isStarted],
  ([s, started]) => {
    session.setCanGoNext(started && s === 'success')
  },
  { immediate: true },
)

watch(
  [install.status, install.isStarted],
  ([s, started]) => {
    if (started && s === 'success') {
      setTimeout(() => {
        router.push({ name: OPERATIONS['ydb-em'].resultStep })
      }, SUCCESS_NAV_DELAY_MS)
    }
  },
)

function startInstallation() {
  install.startEmInstall()
}

function handleRestart() {
  install.restartInstallation(OPERATIONS['ydb-em'].startEndpoint)
}

async function confirmCancel() {
  await install.cancel()
}
</script>

<template>
  <div class="screen-shell">
    <header class="screen-head">
      <ScreenEyebrow />
      <h1 class="screen-title">{{ t('emInstall.heading') }}</h1>
    </header>

    <template v-if="showPreflight">
      <EmPreflightCard :rows="em.emPreflightRows" :ready="em.emCanInstall" />

      <div v-if="showStartButton" class="start-row">
        <v-btn
          color="primary"
          variant="elevated"
          size="large"
          :disabled="!em.emCanInstall"
          :loading="isRunning"
          @click="startInstallation"
        >
          <v-icon start size="20">mdi-rocket-launch-outline</v-icon>
          {{ t('emInstall.startInstall') }}
        </v-btn>
      </div>
    </template>

    <v-card v-if="isSuccess && isStarted" class="section-card success-card pa-8 mb-4">
      <div class="success-icon">
        <v-icon size="48" color="success">mdi-check-circle-outline</v-icon>
      </div>
      <h2 class="success-title">{{ t('emInstall.successTitle') }}</h2>
      <p class="success-message">{{ t('emInstall.successMessage') }}</p>
    </v-card>

    <InstallProgressPanel
      v-else-if="!isIdle"
      :status="install.status.value"
      :progress="install.progress.value"
      :tasks="install.tasks.value"
      :completed-tasks="install.completedTasks.value"
      :error="install.error.value"
      :logs="install.logs.value"
      :log-status="logStatus"
      v-model:search-query="install.searchQuery.value"
      :restart-label="t('emInstall.restartInstall')"
      :cancelled-title="t('emInstall.cancelledTitle')"
      :cancelled-message="t('emInstall.cancelledMessage')"
      :cancel-title="t('emInstall.cancelTitle')"
      :cancel-message="t('emInstall.cancelMessage')"
      :cancel-continue-label="t('emInstall.continueBtn')"
      :cancel-confirm-label="t('emInstall.stopConfirmBtn')"
      :logs-recovered="logsRecovered"
      :recovery-in-progress="recoveryInProgress"
      @cancel="confirmCancel"
      @restart="handleRestart"
      @recover-logs="install.recoverLogs?.()"
    />
  </div>
</template>

<style scoped>
.screen-head { margin-bottom: 28px; }
.screen-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0;
}

.section-card { background: rgb(var(--v-theme-surface)); }

.start-row {
  display: flex;
  padding: 4px 0 8px;
}

.success-card {
  text-align: center;
  border: 1.5px solid #bbf7d0 !important;
  background: #f0fdf4 !important;
}
.success-icon { margin-bottom: 16px; }
.success-title {
  font-size: 22px;
  font-weight: 700;
  color: #15803d;
  margin: 0 0 10px;
  letter-spacing: -0.02em;
}
.success-message {
  font-size: 14px;
  color: #166534;
  margin: 0;
}
</style>
