<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useInstall } from '@/composables/useInstall'
import { OPERATIONS } from '@/config/operations'
import ConsoleOutput from '@/components/common/ConsoleOutput.vue'
import InstallProgressPanel from '@/components/common/InstallProgressPanel.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const { t } = useI18n()
const session = useSessionStore()

const install = useInstall()
const ui = useUiStore()
const { consoleExpanded } = storeToRefs(ui)

onMounted(async () => {
  await install.waitForRecovery()
})

watch(
  [install.status, install.isStarted],
  ([s, started]) => {
    session.setCanGoNext(started && s === 'success')
  },
  { immediate: true },
)

const isSuccess = computed(() => install.status.value === 'success')

function handleRestartDeletion() {
  install.restartInstallation(OPERATIONS['delete-em-ydb'].startEndpoint)
}

async function confirmCancel() {
  await install.cancel()
}

</script>

<template>
  <div class="screen-shell">
    <header class="screen-head">
      <ScreenEyebrow variant="danger" />
      <h1 class="screen-title">{{ t('delete.titleEmYdb') }}</h1>
    </header>

    <v-card v-if="isSuccess" class="section-card success-card pa-8 mb-4">
      <div class="success-icon">
        <v-icon size="48" color="success">mdi-check-circle-outline</v-icon>
      </div>
      <h2 class="success-title">{{ t('delete.successTitleEmYdb') }}</h2>
      <p class="success-message">{{ t('delete.successMessageEmYdb') }}</p>
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
      :log-status="install.logStatus.value"
      v-model:search-query="install.searchQuery.value"
      :restart-label="t('delete.restartDeletion')"
      :cancelled-title="t('delete.cancelledTitle')"
      :cancelled-message="t('delete.cancelledMessage')"
      :cancel-title="t('delete.cancelTitle')"
      :cancel-message="t('delete.cancelMessage')"
      :cancel-continue-label="t('delete.continueBtn')"
      :cancel-confirm-label="t('delete.stopConfirmBtn')"
      :logs-recovered="install.logsRecovered.value"
      :recovery-in-progress="install.recoveryInProgress.value"
      @cancel="confirmCancel"
      @restart="handleRestartDeletion"
      @recover-logs="install.recoverLogs()"
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

.console-header {
  border-bottom: 1px solid #f1f5f9;
}
.cursor-pointer {
  cursor: pointer;
}
</style>
