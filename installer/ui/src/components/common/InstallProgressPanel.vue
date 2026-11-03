<script setup lang="ts">
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import type { InstallError, InstallStatus, Task } from '@/types/install'
import ProgressBar from '@/components/common/ProgressBar.vue'
import TaskList from '@/components/common/TaskList.vue'
import ConsoleOutput from '@/components/common/ConsoleOutput.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import RestartButton from '@/components/common/RestartButton.vue'
import { useUiStore } from '@/stores/ui'

type LogStatus = 'streaming' | 'disconnected' | 'reconnecting' | 'recovering' | 'recovered' | 'sse' | 'none'

const props = withDefaults(defineProps<{
  status: InstallStatus
  progress: number
  tasks: Task[]
  completedTasks: number
  error: InstallError | null
  logs: string[]
  searchQuery: string
  logStatus?: LogStatus
  /** Overrides the restart button label (delete flows pass their own). */
  restartLabel?: string
  /** Cancelled-state card wording. */
  cancelledTitle: string
  cancelledMessage: string
  /** Cancel confirmation dialog wording. */
  cancelTitle: string
  cancelMessage: string
  cancelContinueLabel: string
  cancelConfirmLabel: string
  /** Recover-logs affordance (shown only when the stream dropped). */
  logsRecovered?: boolean
  recoveryInProgress?: boolean
}>(), {
  logStatus: 'none',
  restartLabel: undefined,
  logsRecovered: false,
  recoveryInProgress: false,
})

const emit = defineEmits<{
  cancel: []
  restart: []
  'recover-logs': []
  'update:searchQuery': [value: string]
}>()

const { t } = useI18n()
const cancelDialogOpen = ref(false)

const ui = useUiStore()
const { consoleExpanded } = storeToRefs(ui)

const isRunning = computed(() => props.status === 'running')
const isError = computed(() => props.status === 'error')
const isCancelled = computed(() => props.status === 'cancelled')

/** Panel is only meaningful for in-flight and terminal-failure states. */
const isVisible = computed(() => isRunning.value || isError.value || isCancelled.value)

/** Restart is offered only after a failure or an explicit cancellation. */
const showRestart = computed(() => isError.value || isCancelled.value)

/** Offer log recovery only when the stream died and nothing was recovered yet. */
const showRecoverLogs = computed(
  () => props.logStatus === 'disconnected' && !props.logsRecovered,
)

const searchProxy = computed({
  get: () => props.searchQuery,
  set: (v: string) => emit('update:searchQuery', v),
})

function openCancelDialog() {
  cancelDialogOpen.value = true
}
</script>

<template>
  <template v-if="isVisible">
    <v-card class="section-card pa-6 mb-4">
      <ProgressBar
        :progress="progress"
        :completed-tasks="completedTasks"
        :total-tasks="tasks.length"
        :status="status"
      />
    </v-card>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-3">{{ t('install.tasks') }}</div>
      <TaskList :tasks="tasks" />
    </v-card>

    <v-card v-if="isError && error" class="section-card error-card pa-6 mb-4">
      <div class="error-header">
        <v-icon size="18" color="error" class="mr-2">mdi-alert-circle-outline</v-icon>
        <span class="error-title">{{ t('install.errorTitle') }}</span>
      </div>
      <div class="error-grid">
        <div class="error-row">
          <span class="error-key">{{ t('install.errorTask') }}</span>
          <span class="error-val">{{ error.task }}</span>
        </div>
        <div v-if="error.host" class="error-row">
          <span class="error-key">{{ t('install.errorHost') }}</span>
          <span class="error-val">{{ error.host }}</span>
        </div>
        <div class="error-row">
          <span class="error-key">{{ t('install.errorReason') }}</span>
          <span class="error-val">{{ error.reason }}</span>
        </div>
      </div>
      <RestartButton
        v-if="showRestart"
        :label="restartLabel"
        @restart="$emit('restart')"
      />
    </v-card>

    <v-card v-if="isCancelled" class="section-card cancelled-card pa-6 mb-4">
      <div class="cancelled-header">
        <v-icon size="18" color="warning" class="mr-2">mdi-alert-outline</v-icon>
        <span class="cancelled-title">{{ cancelledTitle }}</span>
      </div>
      <p class="cancelled-message">{{ cancelledMessage }}</p>
      <RestartButton
        v-if="showRestart"
        :label="restartLabel"
        @restart="$emit('restart')"
      />
    </v-card>

    <v-card class="section-card pa-0 mb-4 overflow-hidden">
      <div class="console-header pa-4 d-flex align-center cursor-pointer" @click="ui.toggleConsole()">
        <v-icon size="18" class="mr-2">
          {{ consoleExpanded ? 'mdi-chevron-up' : 'mdi-chevron-down' }}
        </v-icon>
        <span class="section-label">{{ t('install.console') }}</span>
        <v-spacer />
        <v-btn
          v-if="showRecoverLogs"
          size="small"
          variant="text"
          :loading="recoveryInProgress"
          @click.stop="$emit('recover-logs')"
        >
          <v-icon start size="14">mdi-refresh</v-icon>
          {{ t('install.recoverLogs') }}
        </v-btn>
      </div>
      <v-expand-transition>
        <div v-show="consoleExpanded">
          <ConsoleOutput
            :logs="logs"
            :log-status="logStatus"
            v-model:search-query="searchProxy"
          />
        </div>
      </v-expand-transition>
    </v-card>

    <div v-if="isRunning" class="cancel-row">
      <v-btn
        variant="outlined"
        color="error"
        class="cancel-btn"
        @click="openCancelDialog"
      >
        <v-icon start size="16">mdi-stop-circle-outline</v-icon>
        {{ t('install.stopBtn') }}
      </v-btn>
    </div>

    <ConfirmDialog
      v-model="cancelDialogOpen"
      :title="cancelTitle"
      :message="cancelMessage"
      :cancel-label="cancelContinueLabel"
      :confirm-label="cancelConfirmLabel"
      @confirm="$emit('cancel')"
    />
  </template>
</template>

<style scoped>
.section-card { background: rgb(var(--v-theme-surface)); }
.section-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #94a3b8;
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

.console-header {
  border-bottom: 1px solid #f1f5f9;
}

.cancel-row {
  display: flex;
  justify-content: center;
  padding: 8px 0 4px;
}
.cancel-btn {
  font-weight: 600 !important;
  letter-spacing: -0.005em;
}
</style>
