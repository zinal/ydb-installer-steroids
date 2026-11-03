<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ConsoleOutput from '@/components/common/ConsoleOutput.vue'

type LogStatus = 'streaming' | 'disconnected' | 'reconnecting' | 'recovering' | 'recovered' | 'sse' | 'none'

const props = withDefaults(defineProps<{
  /** 'success' | 'error' | anything else → neutral "no result" placeholder. */
  state: 'success' | 'error' | 'idle'
  successTitle: string
  successMessage: string
  errorTitle: string
  errorMessage: string
  /** Shown in place of both cards when state is neither success nor error. */
  noResultMessage: string
  logs: string[]
  searchQuery: string
  logStatus?: LogStatus
  /** Absolute path of the log file; the copy row is hidden when empty. */
  logPath?: string
}>(), {
  logStatus: 'none',
  logPath: '',
})

const emit = defineEmits<{
  'update:searchQuery': [value: string]
}>()

const { t } = useI18n()

const logsExpanded = ref(false)
const copied = ref(false)

const isSuccess = computed(() => props.state === 'success')
const isError = computed(() => props.state === 'error')
const hasResult = computed(() => isSuccess.value || isError.value)

const searchProxy = computed({
  get: () => props.searchQuery,
  set: (v: string) => emit('update:searchQuery', v),
})

function toggleLogs() {
  logsExpanded.value = !logsExpanded.value
}

async function copyLogPath() {
  if (!props.logPath) return
  try {
    await navigator.clipboard.writeText(props.logPath)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    // Clipboard API unavailable (non-secure context) — silently ignore.
  }
}
</script>

<template>
  <v-card v-if="isSuccess" class="section-card success-card pa-8 mb-4">
    <div class="success-icon">
      <v-icon size="48" color="success">mdi-check-circle-outline</v-icon>
    </div>
    <h2 class="success-title">{{ successTitle }}</h2>
    <p class="success-message">{{ successMessage }}</p>
  </v-card>

  <v-card v-else-if="isError" class="section-card error-card pa-8 mb-4">
    <div class="error-icon">
      <v-icon size="48" color="error">mdi-alert-circle-outline</v-icon>
    </div>
    <h2 class="error-title">{{ errorTitle }}</h2>
    <p class="error-message">{{ errorMessage }}</p>
  </v-card>

  <v-card v-else class="section-card pa-8 mb-4 text-center">
    <v-icon size="48" color="grey" class="mb-4">mdi-information-outline</v-icon>
    <p class="text-body-1 text-medium-emphasis">{{ noResultMessage }}</p>
  </v-card>

  <slot name="detail" />

  <v-card v-if="hasResult" class="section-card pa-0 mb-4 overflow-hidden">
    <div class="logs-header pa-4 d-flex align-center cursor-pointer" @click="toggleLogs">
      <v-icon size="18" class="mr-2">
        {{ logsExpanded ? 'mdi-chevron-up' : 'mdi-chevron-down' }}
      </v-icon>
      <span class="section-label">{{ t('result.logs') }}</span>
      <v-spacer />
      <span class="text-caption text-medium-emphasis mr-2">
        {{ logsExpanded ? t('result.hideLogs') : t('result.showLogs') }}
      </span>
      <v-chip size="x-small" variant="tonal" color="default" class="ml-1">
        {{ logs.length }}
      </v-chip>
    </div>
    <v-expand-transition>
      <div v-show="logsExpanded">
        <ConsoleOutput
          :logs="logs"
          :log-status="logStatus"
          v-model:search-query="searchProxy"
        />
      </div>
    </v-expand-transition>
  </v-card>

  <v-card v-if="isError && logPath" class="section-card pa-4 mb-4">
    <div class="log-path-row d-flex align-center">
      <v-icon size="18" color="grey" class="mr-2">mdi-file-document-outline</v-icon>
      <code class="log-path">{{ logPath }}</code>
      <v-spacer />
      <v-btn variant="text" size="small" color="primary" @click="copyLogPath">
        <v-icon start size="14">{{ copied ? 'mdi-check' : 'mdi-content-copy' }}</v-icon>
        {{ copied ? t('common.copied') : t('result.copyPath') }}
      </v-btn>
    </div>
  </v-card>
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
  text-align: center;
  border: 1.5px solid #fca5a5 !important;
  background: #fff5f5 !important;
}
.error-icon { margin-bottom: 16px; }
.error-title {
  font-size: 22px;
  font-weight: 700;
  color: #b91c1c;
  margin: 0 0 10px;
  letter-spacing: -0.02em;
}
.error-message {
  font-size: 14px;
  color: #991b1b;
  margin: 0;
}

.logs-header {
  border-bottom: 1px solid #f1f5f9;
}
.cursor-pointer { cursor: pointer; }

.log-path-row { flex-wrap: wrap; gap: 8px; }
.log-path {
  font-size: 12px;
  color: #64748b;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  background: #f8fafc;
  padding: 4px 8px;
  border-radius: 4px;
  word-break: break-all;
}
</style>
