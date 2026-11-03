<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { EmPreflightRow } from '@/stores/em'

const props = defineProps<{
  rows: EmPreflightRow[]
  /** True when every blocking row is satisfied. */
  ready: boolean
}>()

const { t } = useI18n()

/** Unsatisfied blocking rows — drive the summary banner at the foot of the card. */
const blockingRows = computed(() =>
  props.rows.filter((r) => r.severity === 'error' && !r.ok),
)

function rowIcon(row: EmPreflightRow): string {
  if (row.ok) return 'mdi-check-circle'
  return row.severity === 'warning' ? 'mdi-alert-outline' : 'mdi-close-circle'
}

function rowColor(row: EmPreflightRow): string {
  if (row.ok) return 'success'
  return row.severity === 'warning' ? 'warning' : 'error'
}
</script>

<template>
  <v-card class="section-card pa-6 mb-4">
    <div class="section-label mb-4">{{ t('emInstall.preflight.title') }}</div>

    <div class="preflight-list">
      <div
        v-for="row in rows"
        :key="row.id"
        class="preflight-row"
        :class="{ 'preflight-row--bad': !row.ok }"
      >
        <v-icon :color="rowColor(row)" size="18" class="preflight-icon">
          {{ rowIcon(row) }}
        </v-icon>

        <div class="preflight-body">
          <div class="preflight-label">{{ t(row.labelKey) }}</div>
          <code v-if="row.value" class="preflight-value">{{ row.value }}</code>
          <div v-else-if="row.hintKey" class="preflight-hint">{{ t(row.hintKey) }}</div>
        </div>
      </div>
    </div>

    <v-alert
      v-if="!ready && blockingRows.length > 0"
      type="error"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      {{ t('emInstall.preflight.blocked', { count: blockingRows.length }) }}
    </v-alert>
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

.preflight-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.preflight-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  background: #f8fafc;
  border-radius: 6px;
}
.preflight-row--bad {
  background: #fffbeb;
}

.preflight-icon {
  flex-shrink: 0;
  margin-top: 1px;
}

.preflight-body {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.preflight-label {
  font-size: 13px;
  color: #0f172a;
  font-weight: 500;
}

.preflight-value {
  font-size: 12px;
  color: #475569;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  word-break: break-all;
}

.preflight-hint {
  font-size: 12px;
  color: #92400e;
}
</style>
