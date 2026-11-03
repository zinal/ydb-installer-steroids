<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  logs: string[]
  searchQuery?: string
  logStatus?: 'streaming' | 'disconnected' | 'reconnecting' | 'recovering' | 'recovered' | 'sse' | 'none'
}>()

const emit = defineEmits<{
  (e: 'update:searchQuery', value: string): void
}>()

const { t } = useI18n()
const containerRef = ref<HTMLElement | null>(null)
const localSearch = ref(props.searchQuery ?? '')

watch(() => props.searchQuery, (v) => { if (v !== undefined) localSearch.value = v })
watch(localSearch, (v) => emit('update:searchQuery', v))

const filteredLogs = computed(() => {
  const q = localSearch.value.trim().toLowerCase()
  if (!q) return props.logs
  return props.logs.filter((line) => line.toLowerCase().includes(q))
})

watch(() => props.logs.length, async () => {
  await nextTick()
  const el = containerRef.value
  if (!el) return
  el.scrollTop = el.scrollHeight
})

function highlight(line: string): string {
  const q = localSearch.value.trim()
  if (!q) return escapeHtml(line)
  const escaped = escapeHtml(line)
  const escapedQ = escapeHtml(q)
  return escaped.replace(
    new RegExp(escapedQ.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi'),
    (m) => `<mark class="log-highlight">${m}</mark>`,
  )
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&')
    .replace(/</g, '<')
    .replace(/>/g, '>')
    .replace(/"/g, '"')
}

const statusIcon = computed(() => {
  switch (props.logStatus) {
    case 'streaming':
      return 'mdi-wifi'
    case 'disconnected':
      return 'mdi-wifi-off'
    case 'reconnecting':
      return 'mdi-connection'
    case 'recovering':
      return 'mdi-refresh'
    case 'recovered':
      return 'mdi-check-circle'
    case 'sse':
      return 'mdi-access-point'
    case 'none':
      return 'mdi-help-circle'
    default:
      return 'mdi-help-circle'
  }
})

const statusColor = computed(() => {
  switch (props.logStatus) {
    case 'streaming':
      return 'success'
    case 'disconnected':
      return 'error'
    case 'reconnecting':
      return 'warning'
    case 'recovering':
      return 'warning'
    case 'recovered':
      return 'info'
    case 'sse':
      return 'info'
    case 'none':
      return 'grey'
    default:
      return 'grey'
  }
})

const statusText = computed(() => {
  switch (props.logStatus) {
    case 'streaming':
      return 'Live'
    case 'disconnected':
      return 'Disconnected'
    case 'reconnecting':
      return 'Reconnecting...'
    case 'recovering':
      return 'Recovering...'
    case 'recovered':
      return 'Recovered'
    case 'sse':
      return 'SSE Fallback'
    case 'none':
      return ''
    default:
      return ''
  }
})
</script>

<template>
  <div class="console-shell">
    <div class="console-toolbar">
      <v-icon size="14" color="#64748b" class="mr-1">mdi-magnify</v-icon>
      <input
        v-model="localSearch"
        type="text"
        class="console-search"
        :placeholder="t('install.search')"
      />
      <span v-if="localSearch && filteredLogs.length !== logs.length" class="console-match-count">
        {{ filteredLogs.length }} / {{ logs.length }}
      </span>
      
      <div v-if="logStatus" class="log-status-indicator">
        <v-icon 
          size="14" 
          :color="statusColor"
          class="mr-1"
        >
          {{ statusIcon }}
        </v-icon>
        <span class="log-status-text">{{ statusText }}</span>
      </div>
    </div>

    <div ref="containerRef" class="console-body">
      <div
        v-for="(line, idx) in filteredLogs"
        :key="idx"
        class="log-line"
        v-html="highlight(line)"
      />
      <div v-if="filteredLogs.length === 0" class="log-empty">
        {{ localSearch ? 'No matches' : '…' }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.console-shell {
  display: flex;
  flex-direction: column;
  border-radius: 10px;
  overflow: hidden;
  background: #0f172a;
  border: 1px solid #1e293b;
}

.console-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  gap: 6px;
}

.console-search {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  font-size: 12px;
  color: #e2e8f0;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.console-search::placeholder { color: #475569; }

.console-match-count {
  font-size: 11px;
  color: #64748b;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.log-status-indicator {
  display: flex;
  align-items: center;
  font-size: 11px;
  color: #64748b;
  margin-left: auto;
}

.log-status-text {
  font-variant-numeric: tabular-nums;
}

.console-body {
  height: 330px;
  overflow-y: auto;
  padding: 12px 14px;
  scroll-behavior: smooth;
}

.log-line {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12px;
  line-height: 1.65;
  color: #94a3b8;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-empty {
  font-size: 12px;
  color: #334155;
  font-style: italic;
  padding: 8px 0;
}

.console-body::-webkit-scrollbar { width: 6px; }
.console-body::-webkit-scrollbar-track { background: transparent; }
.console-body::-webkit-scrollbar-thumb { background: #334155; border-radius: 3px; }
</style>

<style>
.log-highlight {
  background: #fbbf24;
  color: #0f172a;
  border-radius: 2px;
  padding: 0 1px;
}
</style>
