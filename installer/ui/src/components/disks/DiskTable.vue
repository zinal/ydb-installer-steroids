<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DiskInfo } from '@/types/disks'
import DiskCard from './DiskCard.vue'

interface Props {
  host: string
  disks: DiskInfo[]
  selectedDisks: Set<string>
  disabled?: boolean
}

interface Emits {
  (e: 'toggle', disk: DiskInfo): void
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
})

const emit = defineEmits<Emits>()
const { t } = useI18n()

const availableDisks = computed(() => {
  return props.disks.filter(d => !d.isSystemDisk)
})

const systemDisks = computed(() => {
  return props.disks.filter(d => d.isSystemDisk)
})

const isDiskSelected = (disk: DiskInfo) => {
  return props.selectedDisks.has(disk.path)
}

function handleToggle(disk: DiskInfo) {
  emit('toggle', disk)
}
</script>

<template>
  <div class="disk-table">
    <div class="disk-table__header">
      <div class="disk-table__host">
        <v-icon size="16" color="#64748b">mdi-server</v-icon>
        <span>{{ host }}</span>
      </div>
      <div class="disk-table__count">
        <span class="disk-table__count-label">{{ t('disks.total') }}:</span>
        <span class="disk-table__count-value">{{ disks.length }}</span>
      </div>
    </div>

    <div v-if="availableDisks.length > 0" class="disk-table__section">
      <div class="disk-table__section-title">
        {{ t('disks.free') }} ({{ availableDisks.length }})
      </div>
      <div class="disk-table__list">
        <DiskCard
          v-for="disk in availableDisks"
          :key="disk.path"
          :disk="disk"
          :selected="isDiskSelected(disk)"
          :disabled="disabled"
          @toggle="handleToggle"
        />
      </div>
    </div>

    <div v-if="systemDisks.length > 0" class="disk-table__section disk-table__section--system">
      <div class="disk-table__section-title">
        {{ t('disks.hasFS') }} ({{ systemDisks.length }})
      </div>
      <div class="disk-table__list">
        <DiskCard
          v-for="disk in systemDisks"
          :key="disk.path"
          :disk="disk"
          :selected="false"
          :disabled="true"
          @toggle="handleToggle"
        />
      </div>
    </div>

    <div v-if="disks.length === 0" class="disk-table__empty">
      <v-icon size="32" color="#cbd5e1">mdi-harddisk-remove</v-icon>
      <span>No disks found</span>
    </div>
  </div>
</template>

<style scoped>
.disk-table {
  background: #fff;
  border: 1.5px solid #e2e8f0;
  border-radius: 12px;
  overflow: hidden;
}

.disk-table__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
}

.disk-table__host {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 14px;
  color: #0f172a;
}

.disk-table__count {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #64748b;
}

.disk-table__count-label {
  font-weight: 500;
}

.disk-table__count-value {
  font-weight: 600;
  color: #0f172a;
}

.disk-table__section {
  padding: 12px;
  border-bottom: 1px solid #f1f5f9;
}

.disk-table__section:last-child {
  border-bottom: none;
}

.disk-table__section--system {
  background: #fefce8;
}

.disk-table__section-title {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #94a3b8;
  margin-bottom: 8px;
}

.disk-table__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.disk-table__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px;
  gap: 12px;
  color: #94a3b8;
  font-size: 14px;
}
</style>
