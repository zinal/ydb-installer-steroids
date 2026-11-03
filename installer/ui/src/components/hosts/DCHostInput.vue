<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DCHosts } from '@/stores/hosts'

const props = defineProps<{
  dcName: keyof DCHosts
  hosts: string[]
  index: number
}>()

const emit = defineEmits<{
  (e: 'update', dc: keyof DCHosts, index: number, value: string): void
  (e: 'remove', dc: keyof DCHosts, index: number): void
}>()

const { t } = useI18n()

const dcLabel = computed(() => {
  const labels: Record<keyof DCHosts, string> = {
    'DC-1': 'DC-1',
    'DC-2': 'DC-2',
    'DC-3': 'DC-3',
  }
  return labels[props.dcName]
})

const dcIcon = computed(() => {
  const icons: Record<keyof DCHosts, string> = {
    'DC-1': 'mdi-server-network',
    'DC-2': 'mdi-server-network',
    'DC-3': 'mdi-server-network',
  }
  return icons[props.dcName]
})

const hostValue = computed({
  get: () => props.hosts[props.index] || '',
  set: (value: string) => emit('update', props.dcName, props.index, value),
})

const canRemove = computed(() => props.hosts.length > 0)
</script>

<template>
  <div class="dc-host-input">
    <div class="dc-header">
      <div class="dc-badge">
        <v-icon size="14">{{ dcIcon }}</v-icon>
        <span>{{ dcLabel }}</span>
      </div>
      <button
        v-if="canRemove"
        type="button"
        class="remove-btn"
        @click="emit('remove', props.dcName, props.index)"
        :title="t('hosts.removeHost')"
      >
        <v-icon size="16">mdi-close</v-icon>
      </button>
    </div>
    <input
      v-model="hostValue"
      type="text"
      class="host-input"
      :placeholder="t('hosts.hostname')"
      autocomplete="off"
    />
  </div>
</template>

<style scoped>
.dc-host-input {
  background: #fff;
  border: 1.5px solid #e2e8f0;
  border-radius: 12px;
  padding: 16px;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.dc-host-input:hover {
  border-color: #cbd5e1;
}

.dc-host-input:focus-within {
  border-color: #1f6dd9;
  box-shadow: 0 0 0 3px rgba(31, 109, 217, 0.1);
}

.dc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.dc-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  color: #0369a1;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.remove-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: #fef2f2;
  border: none;
  border-radius: 6px;
  color: #dc2626;
  cursor: pointer;
  transition: all 0.15s ease;
}

.remove-btn:hover {
  background: #fee2e2;
  transform: scale(1.05);
}

.remove-btn:active {
  transform: scale(0.95);
}

.host-input {
  width: 100%;
  padding: 10px 14px;
  border: 1.5px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #0f172a;
  background: #f8fafc;
  outline: none;
  transition: all 0.15s ease;
  font-family: inherit;
}

.host-input::placeholder {
  color: #94a3b8;
}

.host-input:focus {
  background: #fff;
  border-color: #1f6dd9;
  box-shadow: 0 0 0 3px rgba(31, 109, 217, 0.1);
}
</style>
