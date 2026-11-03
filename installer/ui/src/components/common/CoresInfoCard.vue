<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CpuStats, CoresBudget, UtilizationStatus } from '@/composables/useCoresInfo'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  /** CPU stats from host checks. */
  cpuStats: CpuStats
  /** Core budget from useCoresInfo. */
  budget: CoresBudget
  /** When true, renders compact variant (used inside cards with dense layout). */
  compact?: boolean
}>(), {
  compact: false,
})

const statusIcon = computed(() => {
  const map: Record<UtilizationStatus, string> = {
    low: 'mdi-check-circle',
    medium: 'mdi-alert-circle-outline',
    high: 'mdi-alert',
    critical: 'mdi-alert-circle',
  }
  return map[props.budget.utilizationStatus]
})

const statusColor = computed(() => {
  const map: Record<UtilizationStatus, string> = {
    low: '#059669',
    medium: '#ca8a04',
    high: '#ea580c',
    critical: '#dc2626',
  }
  return map[props.budget.utilizationStatus]
})

const statusText = computed(() => {
  const fallback: Record<UtilizationStatus, string> = {
    low: t('coresInfo.statusLow'),
    medium: t('coresInfo.statusMedium'),
    high: t('coresInfo.statusHigh'),
    critical: t('coresInfo.statusCritical'),
  }
  return fallback[props.budget.utilizationStatus]
})

const staticWidth = computed(() => {
  if (!props.cpuStats.available || props.cpuStats.min === 0) return 0
  return Math.min((props.budget.staticPerHost / props.cpuStats.min) * 100, 100)
})

const dynamicWidth = computed(() => {
  if (!props.cpuStats.available || props.cpuStats.min === 0) return 0
  return Math.min((props.budget.dynamicPerHost / props.cpuStats.min) * 100, 100 - staticWidth.value)
})

const freeWidth = computed(() => {
  if (!props.cpuStats.available || props.cpuStats.min === 0) return 0
  return Math.max(0, 100 - staticWidth.value - dynamicWidth.value)
})

const freePerHost = computed(() =>
  Math.max(props.cpuStats.min - props.budget.staticPerHost - props.budget.dynamicPerHost, 0),
)

const remainingForDynamic = computed(() =>
  Math.max(props.cpuStats.min - props.budget.staticPerHost, 0),
)
</script>

<template>
  <div class="cores-card" :class="{ 'is-compact': compact }">
    <div class="cores-card-title">
      <v-icon icon="mdi-memory" size="16" color="info" class="title-icon" />
      <span>{{ t('coresInfo.title') }}</span>
      <span class="title-unit">{{ t('coresInfo.perHostUnit') }}</span>
    </div>

    <div v-if="cpuStats.available" class="budget-row">
      <span
        class="budget-value budget-value--static"
        :class="{ 'is-unset': budget.staticPerHost === 0 }"
      >
        {{ budget.staticPerHost }}
      </span>
      <span
        class="budget-label-chip static-chip"
        :class="{ 'is-unset': budget.staticPerHost === 0 }"
      >
        {{ t('coresInfo.static') }}
      </span>

      <v-icon icon="mdi-plus" size="14" color="#94a3b8" />

      <span
        class="budget-value budget-value--dynamic"
        :class="{ 'is-unset': budget.dynamicPerHost === 0 }"
      >
        {{ budget.dynamicPerHost }}
      </span>
      <span
        class="budget-label-chip dynamic-chip"
        :class="{ 'is-unset': budget.dynamicPerHost === 0 }"
      >
        {{ t('coresInfo.dynamic') }}
      </span>

      <v-icon icon="mdi-equal" size="14" color="#94a3b8" />

      <span class="budget-value budget-value--total">
        {{ budget.totalAllocatedPerHost }}
      </span>
      <span class="budget-unit">{{ t('coresInfo.of') }}</span>
      <span class="budget-total">{{ cpuStats.min }}</span>
    </div>
    <div v-else class="budget-row is-muted">
      <span class="budget-na">{{ t('coresInfo.notAvailable') }}</span>
    </div>

    <div v-if="cpuStats.available" class="progress-track">
      <div
        v-if="staticWidth > 0"
        class="progress-seg progress-seg--static"
        :style="{ width: staticWidth + '%' }"
      />
      <div
        v-if="dynamicWidth > 0"
        class="progress-seg progress-seg--dynamic"
        :style="{ width: dynamicWidth + '%' }"
      />
      <div
        v-if="freeWidth > 0"
        class="progress-seg progress-seg--free"
        :style="{ width: freeWidth + '%' }"
      />
    </div>

    <div v-if="cpuStats.available" class="legend-row">
      <span class="legend-item" :class="{ 'is-unset': budget.staticPerHost === 0 }">
        <span class="legend-dot static-dot" />
        <span class="legend-label">{{ t('coresInfo.static') }}</span>
        <span class="legend-value">{{ budget.staticPerHost }}</span>
      </span>
      <span class="legend-item" :class="{ 'is-unset': budget.dynamicPerHost === 0 }">
        <span class="legend-dot dynamic-dot" />
        <span class="legend-label">{{ t('coresInfo.dynamic') }}</span>
        <span class="legend-value">{{ budget.dynamicPerHost }}</span>
      </span>
      <span class="legend-item">
        <span class="legend-dot free-dot" />
        <span class="legend-label">{{ t('coresInfo.free') }}</span>
        <span class="legend-value">{{ freePerHost }}</span>
      </span>
    </div>

    <div v-if="cpuStats.available" class="breakdown-rows">
      <div class="breakdown-item">
        <span class="breakdown-dot static-dot" />
        <span class="breakdown-label">{{ t('coresInfo.staticReserved') }}</span>
        <span class="breakdown-value">{{ budget.staticPerHost }} {{ t('coresInfo.cores') }}</span>
      </div>
      <div class="breakdown-item">
        <span class="breakdown-dot dynamic-dot" />
        <span class="breakdown-label">{{ t('coresInfo.remainingForDynamic') }}</span>
        <span class="breakdown-value">
          {{ remainingForDynamic }} {{ t('coresInfo.cores') }}
        </span>
      </div>
    </div>

    <div v-if="cpuStats.available" class="meta-row">
      <span class="meta-label">{{ t('coresInfo.clusterTotal') }}</span>
      <span class="meta-value">{{ budget.totalAllocatedCluster }} / {{ cpuStats.sum }}</span>
    </div>

    <div class="status-row" :style="{ color: statusColor }">
      <v-icon :icon="statusIcon" size="14" class="mr-1" />
      <span>{{ statusText }}</span>
    </div>
  </div>
</template>

<style scoped>
.cores-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0;
}
.cores-card.is-compact {
  gap: 6px;
}

.cores-card-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #374151;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.title-icon {
  flex-shrink: 0;
}
.title-unit {
  font-weight: 400;
  text-transform: none;
  color: #94a3b8;
  font-size: 11.5px;
}

.budget-row {
  display: flex;
  align-items: center;
  gap: 4px;
  font-variant-numeric: tabular-nums;
  flex-wrap: wrap;
}
.budget-row.is-muted {
  opacity: 0.7;
}
.budget-value {
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1;
}
.budget-value--static { color: #1f6dd9; }
.budget-value--dynamic { color: #7c3aed; }
.budget-value--total { color: #0f172a; }
.budget-value.is-unset {
  color: #94a3b8;
  font-weight: 600;
}
.budget-of {
  font-size: 13px;
  color: #94a3b8;
  margin: 0 2px;
}
.budget-total {
  font-size: 20px;
  font-weight: 600;
  color: #475569;
}
.budget-unit {
  font-size: 13px;
  color: #94a3b8;
}
.budget-na {
  font-size: 13px;
  color: #94a3b8;
  font-style: italic;
}
.budget-label-chip {
  display: inline-flex;
  align-items: center;
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  line-height: 1.2;
}
.static-chip {
  background: #eef4ff;
  color: #1f6dd9;
}
.dynamic-chip {
  background: #f5f3ff;
  color: #7c3aed;
}
.budget-label-chip.is-unset {
  background: #f1f5f9;
  color: #94a3b8;
}

.progress-track {
  height: 8px;
  background: #f1f5f9;
  border-radius: 999px;
  overflow: hidden;
  display: flex;
}
.progress-seg {
  height: 100%;
  transition: width 0.3s ease;
}
.progress-seg:first-child {
  border-radius: 999px 0 0 999px;
}
.progress-seg:last-child {
  border-radius: 0 999px 999px 0;
}
.progress-seg--static {
  background: #1f6dd9;
}
.progress-seg--dynamic {
  background: #7c3aed;
}
.progress-seg--free {
  background: #e2e8f0;
}

.legend-row {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11.5px;
  color: #64748b;
}
.legend-item.is-unset {
  opacity: 0.65;
}
.legend-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex-shrink: 0;
}
.free-dot { background: #e2e8f0; }
.legend-label {
  font-weight: 500;
}
.legend-value {
  font-weight: 600;
  color: #374151;
  font-variant-numeric: tabular-nums;
}

.breakdown-rows {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.breakdown-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
}
.breakdown-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  flex-shrink: 0;
}
.static-dot { background: #1f6dd9; }
.dynamic-dot { background: #7c3aed; }
.breakdown-label {
  color: #64748b;
  flex: 1;
}
.breakdown-value {
  font-weight: 600;
  color: #374151;
  font-variant-numeric: tabular-nums;
}

.meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  padding-top: 4px;
  border-top: 1px solid #f1f5f9;
}
.meta-label {
  color: #94a3b8;
}
.meta-value {
  font-weight: 600;
  color: #374151;
  font-variant-numeric: tabular-nums;
}

.status-row {
  display: flex;
  align-items: center;
  font-size: 12px;
  font-weight: 500;
}
</style>
