<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useInstall } from '@/composables/useInstall'
import { OPERATIONS } from '@/config/operations'
import ConsoleOutput from '@/components/common/ConsoleOutput.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import InstallProgressPanel from '@/components/common/InstallProgressPanel.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const { t } = useI18n()
const router = useRouter()
const session = useSessionStore()

const install = useInstall()
const ui = useUiStore()
const { consoleExpanded } = storeToRefs(ui)
const configExpanded = ref(true)
const isGenerating = ref(false)

onMounted(async () => {
  await install.waitForRecovery()
  await install.fetchConfigStatus()
})

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
        const next = OPERATIONS.ydb.resultStep
        router.push({ name: next })
      }, 300)
    }
  },
)

const isIdle = computed(() => install.status.value === 'idle')
const isRunning = computed(() => install.status.value === 'running')
const isSuccess = computed(() => install.status.value === 'success')
const isError = computed(() => install.status.value === 'error')
const isCancelled = computed(() => install.status.value === 'cancelled')

const logStatus = computed(() => install.logStatus?.value ?? 'none')
const logsRecovered = computed(() => install.logsRecovered?.value ?? false)
const recoveryInProgress = computed(() => install.recoveryInProgress?.value ?? false)
const hasConfig = computed(() => install.configExists.value)
const hasConfigOnDisk = computed(() => install.configFoundOnDisk?.value ?? false)
const isStarted = computed(() => install.isStarted.value)
/** Three-way match status: missing | matches | outdated | null (not yet fetched). */
const configStatus = computed(() => install.configStatus?.value ?? null)
/** True when the on-disk config.yaml is outdated vs current inventory params. */
const isConfigOutdated = computed(() => configStatus.value === 'outdated')

// Button visibility matrix:
//   missing  → Generate (primary)
//   outdated → Regenerate (primary) + Install with existing (tonal override)
//   matches  → Start (primary) only — config is up to date, nothing to regenerate
//   (any) + started → no config buttons
const showGenerateButton = computed(() => configStatus.value === 'missing' && !isStarted.value)
const showRegenerateButton = computed(() => isConfigOutdated.value && !isStarted.value)
const showStartButton = computed(() => hasConfig.value && !isStarted.value)
const showInstallWithExistingButton = computed(() => isConfigOutdated.value && !isStarted.value)
const showRestartButton = computed(() => isCancelled.value || isError.value)

const backupDialogOpen = ref(false)

async function handleGenerateClick() {
  if (hasConfigOnDisk.value || isConfigOutdated.value) {
    backupDialogOpen.value = true
  } else {
    await performGenerate(false)
  }
}

async function confirmBackupAndGenerate() {
  backupDialogOpen.value = false
  await performGenerate(false)
}

async function performGenerate(skipBackup: boolean = false) {
  isGenerating.value = true
  try {
    await install.generateConfig(skipBackup)
  } finally {
    isGenerating.value = false
  }
}

function startInstallation() {
  install.isStarted.value = true
  install.startYdbInstall()
}

async function handleInstallWithExisting() {
  await install.installWithExistingConfig()
  startInstallation()
}

function handleRestartInstallation() {
  install.isStarted.value = true
  install.restartInstallation()
}

async function confirmCancel() {
  await install.cancel()
}

function toggleConfig() {
  configExpanded.value = !configExpanded.value
}
</script>

<template>
  <div class="screen-shell">
    <header class="screen-head">
      <ScreenEyebrow />
      <h1 class="screen-title">{{ t('install.heading') }}</h1>
    </header>

  <v-card class="section-card pa-4 mb-4">
    <div class="config-header d-flex align-center">
      <span class="section-label">{{ t('install.configYaml') }}</span>
      <v-spacer />
      <v-chip
        v-if="configStatus === 'matches'"
        color="success"
        size="small"
        variant="tonal"
      >
        <v-icon start size="14">mdi-check-circle-outline</v-icon>
        {{ t('install.configMatches') }}
      </v-chip>
      <v-chip
        v-else-if="configStatus === 'outdated'"
        color="warning"
        size="small"
        variant="tonal"
      >
        <v-icon start size="14">mdi-alert-outline</v-icon>
        {{ t('install.configNeedsUpdate') }}
      </v-chip>
      <v-chip
        v-else-if="configStatus === 'missing'"
        color="default"
        size="small"
        variant="tonal"
      >
        {{ t('install.configNotFound') }}
      </v-chip>
      <v-btn
        v-if="hasConfig || hasConfigOnDisk"
        variant="text"
        size="small"
        class="toggle-header-btn ml-2"
        @click="toggleConfig"
      >
        <v-icon size="16" class="mr-1">
          {{ configExpanded ? 'mdi-chevron-up' : 'mdi-chevron-down' }}
        </v-icon>
        {{ configExpanded ? t('install.hideConfig') : t('install.showConfig') }}
      </v-btn>
    </div>

    <v-alert
      v-if="isConfigOutdated && !isStarted"
      type="warning"
      variant="tonal"
      density="compact"
      class="mt-3 mb-1"
    >
      {{ t('install.configOutdatedWarning') }}
    </v-alert>

      <v-alert
        v-if="install.configError.value"
        type="error"
        variant="tonal"
        density="compact"
        class="mb-3"
      >
        {{ install.configError.value }}
      </v-alert>

      <v-expand-transition>
        <div v-if="(hasConfig || hasConfigOnDisk) && configExpanded" class="config-preview">
          <pre class="config-code">{{ install.configContent.value }}</pre>
        </div>
      </v-expand-transition>

      <div v-if="!isStarted" class="config-actions d-flex gap-3 mt-3">

        <v-btn
          v-if="showGenerateButton"
          color="primary"
          variant="elevated"
          size="large"
          @click="handleGenerateClick"
          :loading="isGenerating"
        >
          <v-icon start size="20">mdi-file-document-plus-outline</v-icon>
          {{ t('install.generateConfig') }}
        </v-btn>

        <v-btn
          v-if="showStartButton"
          color="primary"
          variant="elevated"
          size="large"
          @click="startInstallation"
          :loading="isRunning"
        >
          <v-icon start size="20">mdi-rocket-launch-outline</v-icon>
          {{ t('install.startInstall') }}
        </v-btn>

        <v-btn
          v-if="showRegenerateButton"
          variant="outlined"
          size="large"
          @click="handleGenerateClick"
          :loading="isGenerating"
        >
          <v-icon start size="20">mdi-refresh</v-icon>
          {{ t('install.regenerateConfig') }}
        </v-btn>

        <v-btn
          v-if="showInstallWithExistingButton"
          color="warning"
          variant="tonal"
          size="large"
          @click="handleInstallWithExisting"
        >
          <v-icon start size="20">mdi-rocket-launch-outline</v-icon>
          {{ t('install.installWithExisting') }}
        </v-btn>
      </div>
    </v-card>

    <v-card v-if="isSuccess" class="section-card success-card pa-8 mb-4">
      <div class="success-icon">
        <v-icon size="48" color="success">mdi-check-circle-outline</v-icon>
      </div>
      <h2 class="success-title">{{ t('install.successTitle') }}</h2>
      <p class="success-message">{{ t('install.successMessage') }}</p>
    </v-card>

    <v-card v-if="isSuccess" class="section-card pa-0 mb-4 overflow-hidden">
      <div class="console-header pa-4 d-flex align-center cursor-pointer" @click="ui.toggleConsole()">
        <v-icon size="18" class="mr-2">
          {{ consoleExpanded ? 'mdi-chevron-up' : 'mdi-chevron-down' }}
        </v-icon>
        <span class="section-label">{{ t('install.console') }}</span>
        <v-spacer />
        <span class="text-caption text-medium-emphasis mr-2">
          {{ install.logs.value.length }} lines
        </span>
      </div>
      <v-expand-transition>
        <div v-show="consoleExpanded">
          <ConsoleOutput
            :logs="install.logs.value"
            :log-status="install.logStatus.value"
            v-model:search-query="install.searchQuery.value"
          />
        </div>
      </v-expand-transition>
    </v-card>

    <InstallProgressPanel
      v-else
      :status="install.status.value"
      :progress="install.progress.value"
      :tasks="install.tasks.value"
      :completed-tasks="install.completedTasks.value"
      :error="install.error.value"
      :logs="install.logs.value"
      :log-status="logStatus"
      v-model:search-query="install.searchQuery.value"
      :cancelled-title="t('install.cancelledTitle')"
      :cancelled-message="t('install.cancelledMessage')"
      :cancel-title="t('install.cancelTitle')"
      :cancel-message="t('install.cancelMessage')"
      :cancel-continue-label="t('install.continueBtn')"
      :cancel-confirm-label="t('install.stopConfirmBtn')"
      :logs-recovered="logsRecovered"
      :recovery-in-progress="recoveryInProgress"
      @cancel="confirmCancel"
      @restart="handleRestartInstallation"
      @recover-logs="install.recoverLogs?.()"
    />

    <ConfirmDialog
      v-model="backupDialogOpen"
      :title="t('install.configBackupTitle')"
      :message="t('install.configBackupMessage')"
      :cancel-label="t('install.configBackupCancel')"
      :confirm-label="t('install.configBackupConfirm')"
      @confirm="confirmBackupAndGenerate"
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
.section-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #94a3b8;
}

.config-header {
}
.config-header .toggle-header-btn {
  font-size: 12px;
  font-weight: 500;
  letter-spacing: -0.01em;
  text-transform: none;
}
.config-actions {
  flex-wrap: wrap;
  padding-top: 8px;
}
.config-preview {
  margin-top: 0;
}
.config-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 24px;
  background: #f8fafc;
  border-radius: 8px;
}
.config-code {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12px;
  line-height: 1.5;
  background: #1e293b;
  color: #e2e8f0;
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 400px;
  overflow-y: auto;
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

.error-card {
  border: 1.5px solid #fca5a5 !important;
  background: #fff5f5 !important;
}
.error-header {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
.error-title {
  font-size: 14px;
  font-weight: 700;
  color: #b91c1c;
}
.error-grid { display: flex; flex-direction: column; gap: 8px; padding-left: 26px; }
.error-row { display: flex; gap: 12px; align-items: baseline; }
.error-key {
  font-size: 12px;
  color: #64748b;
  min-width: 80px;
}
.error-val {
  font-size: 13px;
  color: #0f172a;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}

.cancelled-card {
  border: 1.5px solid #fde68a !important;
  background: #fffbeb !important;
}
.cancelled-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}
.cancelled-title {
  font-size: 14px;
  font-weight: 700;
  color: #92400e;
}
.cancelled-message {
  font-size: 13px;
  color: #78350f;
  margin: 0;
}
/* ── Console header ── */
.console-header {
  border-bottom: 1px solid #f1f5f9;
}
.cursor-pointer {
  cursor: pointer;
}

/* ── Cancel ── */
.cancel-row {
  display: flex;
  justify-content: center;
  padding: 8px 0 4px;
}
.cancel-btn {
  font-weight: 600 !important;
  letter-spacing: -0.005em;
}

/* ── Utility ── */
.gap-3 { gap: 12px; }
</style>
