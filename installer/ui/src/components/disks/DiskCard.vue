<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DiskInfo } from '@/types/disks'
import { MIN_RECOMMENDED_DISK_SIZE_BYTES } from '@/types/disks'

const { t } = useI18n()

interface Props {
  disk: DiskInfo
  selected: boolean
  disabled?: boolean
}

interface Emits {
  (e: 'toggle', disk: DiskInfo): void
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
})

const emit = defineEmits<Emits>()

const diskTypeIcon = computed(() => {
  switch (props.disk.type) {
    case 'SSD':
      return 'mdi-harddisk'
    case 'HDD':
      return 'mdi-harddisk'
    case 'NVMe':
      return 'mdi-sd'
    default:
      return 'mdi-harddisk'
  }
})

const diskTypeColor = computed(() => {
  switch (props.disk.type) {
    case 'SSD':
      return '#3b82f6'
    case 'HDD':
      return '#64748b'
    case 'NVMe':
      return '#8b5cf6'
    default:
      return '#64748b'
  }
})

const statusText = computed(() => {
  if (props.disk.isSystemDisk) {
    return t('disks.statusSystemDisk')
  }
  if (props.disk.mounted) {
    return t('disks.statusMounted')
  }
  return t('disks.statusAvailable')
})

const statusColor = computed(() => {
  if (props.disk.isSystemDisk) {
    return '#f59e0b'
  }
  if (props.disk.mounted) {
    return '#ef4444'
  }
  return '#10b981'
})

const isSmallDisk = computed(() => {
  return props.disk.sizeBytes > 0 && props.disk.sizeBytes < MIN_RECOMMENDED_DISK_SIZE_BYTES
})

function handleToggle() {
  if (!props.disabled && !props.disk.isSystemDisk) {
    emit('toggle', props.disk)
  }
}
</script>

<template>
  <div
    class="disk-card"
    :class="{
      'disk-card--selected': selected,
      'disk-card--disabled': disabled || disk.isSystemDisk
    }"
    @click="handleToggle"
  >
    <div class="disk-card__checkbox">
      <v-checkbox
        :model-value="selected"
        :disabled="disabled || disk.isSystemDisk"
        hide-details
        density="compact"
        color="#1f6dd9"
        @update:model-value="handleToggle"
      />
    </div>
    
    <div class="disk-card__icon">
      <v-icon :color="diskTypeColor" size="24">
        {{ diskTypeIcon }}
      </v-icon>
    </div>
    
    <div class="disk-card__info">
      <div class="disk-card__name">{{ disk.name }}</div>
      <div class="disk-card__details">
        <span class="disk-card__type" :style="{ color: diskTypeColor }">
          {{ disk.type }}
        </span>
        <span class="disk-card__separator">·</span>
        <span class="disk-card__size">
          {{ disk.size }}
          <v-tooltip
            v-if="isSmallDisk"
            :text="t('disks.smallDiskWarning')"
            location="top"
          >
            <template #activator="{ props: tooltipProps }">
              <v-icon
                v-bind="tooltipProps"
                size="14"
                color="#f59e0b"
                class="disk-card__small-warn"
              >mdi-alert</v-icon>
            </template>
          </v-tooltip>
        </span>
      </div>
    </div>

    <div class="disk-card__status" :style="{ color: statusColor }">
      <v-icon size="14" :color="statusColor">
        {{ disk.isSystemDisk ? 'mdi-alert-circle' : 'mdi-check-circle' }}
      </v-icon>
      <span>{{ statusText }}</span>
    </div>
  </div>
</template>

<style scoped>
.disk-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #fff;
  border: 1.5px solid #e2e8f0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.disk-card:hover:not(.disk-card--disabled) {
  border-color: #3b82f6;
  background: #f0f9ff;
}

.disk-card--selected {
  border-color: #3b82f6;
  background: #eff6ff;
}

.disk-card--disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background: #f8fafc;
}

.disk-card__checkbox {
  flex-shrink: 0;
}

.disk-card__icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f1f5f9;
  border-radius: 8px;
}

.disk-card__info {
  flex: 1;
  min-width: 0;
}

.disk-card__name {
  font-weight: 600;
  font-size: 14px;
  color: #0f172a;
  margin-bottom: 4px;
}

.disk-card__details {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #64748b;
}

.disk-card__type {
  font-weight: 500;
  text-transform: uppercase;
  font-size: 11px;
  letter-spacing: 0.05em;
}

.disk-card__separator {
  color: #cbd5e1;
}

.disk-card__size {
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.disk-card__small-warn {
  flex-shrink: 0;
}

.disk-card__status {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 500;
  flex-shrink: 0;
}
</style>
