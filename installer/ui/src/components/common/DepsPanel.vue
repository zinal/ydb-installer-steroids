<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { DepState } from '@/composables/useDeps'

const props = defineProps<{
  deps: DepState[]
  loading: boolean
  /** When true, show the "all dependencies are bundled" success banner. */
  bundled?: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-instruction', name: string): void
  (e: 'skip', name: string): void
  (e: 'unskip', name: string): void
}>()

const { t } = useI18n()
</script>

<template>
  <div class="deps-panel">
    <div v-if="loading" class="deps-loading">
      <v-progress-circular indeterminate size="18" width="2" color="primary" />
      <span class="deps-loading-text">{{ t('deps.loading') }}</span>
    </div>

    <div v-if="!loading && bundled" class="deps-bundled-banner">
      <v-icon size="16" color="success" class="mr-2">mdi-package-variant-closed-check</v-icon>
      <span>{{ t('deps.bundledBanner') }}</span>
    </div>
    <p v-if="!loading && bundled" class="deps-system-hint">
      {{ t('deps.systemDepsHint') }}
    </p>

    <ul v-if="!loading" class="deps-list">
      <li v-for="dep in deps" :key="dep.name" class="deps-item">
        <div class="deps-row">
          <span
            class="deps-mark"
            :class="{
              'is-ok': dep.ok,
              'is-bad': !dep.ok && !dep.skipped,
              'is-skipped': dep.skipped,
            }"
            aria-hidden="true"
          >
            <svg v-if="dep.ok" viewBox="0 0 16 16" fill="none" width="12" height="12">
              <path d="M3.5 8.5l3 3 6-6.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <svg v-else-if="!dep.skipped" viewBox="0 0 16 16" fill="none" width="10" height="10">
              <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
            <svg v-else viewBox="0 0 16 16" fill="none" width="12" height="12">
              <path d="M8 3v5M8 11v1" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </span>

          <span class="deps-name">
            {{ dep.name }}
            <span
              v-if="dep.source"
              class="deps-source"
              :class="{
                'is-bundled': dep.source === 'bundled',
                'is-system': dep.source === 'system',
                'is-missing': dep.source === 'missing',
              }"
            >{{ t('deps.source.' + dep.source) }}</span>
          </span>

          <span class="deps-ver">
            <template v-if="dep.ok">{{ dep.version }}</template>
            <template v-else-if="dep.installed">{{ dep.version }} ({{ t('deps.invalidVersion', { required: dep.required }) }})</template>
            <template v-else-if="dep.skipped">{{ t('deps.skip') }}</template>
            <template v-else>{{ t('deps.notFound') }}</template>
          </span>
        </div>

        <div v-if="!dep.ok" class="deps-actions">
          <button
            type="button"
            class="deps-btn deps-btn--info"
            :class="{ 'is-active': dep.instructionOpen }"
            @click="emit('toggle-instruction', dep.name)"
          >
            <v-icon size="14" class="mr-1">mdi-clipboard-text-outline</v-icon>
            {{ t('deps.instruction') }}
          </button>

          <button
            v-if="!dep.skipped"
            type="button"
            class="deps-btn deps-btn--skip"
            @click="emit('skip', dep.name)"
          >
            <v-icon size="14" class="mr-1">mdi-skip-next-outline</v-icon>
            {{ t('deps.skip') }}
          </button>

          <button
            v-else
            type="button"
            class="deps-btn deps-btn--unskip"
            @click="emit('unskip', dep.name)"
          >
            <v-icon size="14" class="mr-1">mdi-refresh</v-icon>
            {{ t('deps.instruction') }}
          </button>
        </div>

        <div v-if="dep.skipped" class="deps-skip-warning">
          <v-icon size="14" color="warning" class="mr-1">mdi-alert-outline</v-icon>
          {{ t('deps.skippedWarning') }}
        </div>

        <div v-if="dep.instructionOpen && !dep.ok" class="deps-instruction">
          <div v-if="dep.instructionLoading" class="deps-instruction-loading">
            <v-progress-circular indeterminate size="14" width="2" color="primary" />
          </div>
          <pre v-else-if="dep.instruction" class="deps-code">{{ dep.instruction }}</pre>
          <div v-else class="deps-instruction-empty">—</div>
        </div>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.deps-panel {
  display: flex;
  flex-direction: column;
}

.deps-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: #94a3b8;
  padding: 8px 0;
}
.deps-loading-text { color: #94a3b8; }

.deps-bundled-banner {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  margin-bottom: 8px;
  border-radius: 8px;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #047857;
  font-size: 13px;
  font-weight: 500;
}

.deps-system-hint {
  margin: 0 0 8px;
  padding: 0 12px;
  font-size: 12px;
  color: #64748b;
  line-height: 1.4;
}

.deps-source {
  display: inline-block;
  margin-left: 8px;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 10.5px;
  font-weight: 600;
  font-family: system-ui, sans-serif;
  letter-spacing: 0.02em;
  text-transform: uppercase;
  vertical-align: middle;
}
.deps-source.is-bundled { background: #d1fae5; color: #047857; }
.deps-source.is-system  { background: #e0e7ff; color: #4338ca; }
.deps-source.is-missing { background: #fee2e2; color: #b91c1c; }

.deps-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.deps-item {
  border-top: 1px solid #f1f5f9;
  padding: 6px 0 4px;
}
.deps-item:first-child { border-top: 0; padding-top: 0; }

.deps-row {
  display: grid;
  grid-template-columns: 22px 1fr auto;
  align-items: center;
  gap: 12px;
}

.deps-mark {
  width: 22px;
  height: 22px;
  border-radius: 999px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
}
.deps-mark.is-ok      { background: #dcfce7; color: #16a34a; }
.deps-mark.is-bad     { background: #fee2e2; color: #dc2626; }
.deps-mark.is-skipped { background: #fef9c3; color: #ca8a04; }

.deps-name {
  font-size: 13.5px;
  font-weight: 500;
  color: #0f172a;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}

.deps-ver {
  font-size: 12.5px;
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.deps-actions {
  display: flex;
  gap: 6px;
  margin-top: 6px;
  padding-left: 34px;
}

.deps-btn {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  border: 1.5px solid transparent;
  transition: all .15s ease;
  font-family: inherit;
  line-height: 1.4;
}

.deps-btn--info {
  background: #eef4ff;
  color: #1f6dd9;
  border-color: #c7d9f8;
}
.deps-btn--info:hover,
.deps-btn--info.is-active {
  background: #dbeafe;
  border-color: #93c5fd;
}

.deps-btn--skip {
  background: #fff7ed;
  color: #c2410c;
  border-color: #fed7aa;
}
.deps-btn--skip:hover {
  background: #ffedd5;
  border-color: #fb923c;
}

.deps-btn--unskip {
  background: #f8fafc;
  color: #64748b;
  border-color: #e2e8f0;
}
.deps-btn--unskip:hover {
  background: #f1f5f9;
}

.deps-skip-warning {
  display: flex;
  align-items: center;
  margin-top: 4px;
  padding-left: 34px;
  font-size: 12px;
  color: #b45309;
}

.deps-instruction {
  margin-top: 8px;
  margin-left: 34px;
  border-radius: 8px;
  background: #0f172a;
  overflow: hidden;
}

.deps-instruction-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

.deps-code {
  margin: 0;
  padding: 14px 16px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12px;
  line-height: 1.65;
  color: #e2e8f0;
  white-space: pre-wrap;
  word-break: break-all;
}

.deps-instruction-empty {
  padding: 12px 16px;
  font-size: 12px;
  color: #64748b;
}
</style>
