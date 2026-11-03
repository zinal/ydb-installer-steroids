<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { EMHost } from '@/types/inventory'

const props = withDefaults(
  defineProps<{
    hosts: EMHost[]
    /** Max rows. Default 1. Pass 0 or negative for unlimited (future multi-host). */
    maxHosts?: number
  }>(),
  {
    maxHosts: 1,
  },
)

const emit = defineEmits<{
  (e: 'add'): void
  (e: 'update', index: number, fqdn: string): void
  (e: 'remove', index: number): void
}>()

const { t } = useI18n()

const unlimited = computed(() => props.maxHosts <= 0)

const canAdd = computed(
  () => unlimited.value || props.hosts.length < props.maxHosts,
)

/** Keep at least one row — delete only when multiple hosts exist. */
const canRemove = computed(() => props.hosts.length > 1)

function onInput(index: number, event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update', index, value)
}
</script>

<template>
  <div class="em-host-form">
    <label class="em-host-label" for="em-host-fqdn-0">
      {{ t('emHosts.hostnameLabel') }}
    </label>

    <div class="em-host-list">
      <div
        v-for="(host, index) in hosts"
        :key="index"
        class="em-host-row"
      >
        <span class="em-host-index" aria-hidden="true">{{ index + 1 }}</span>
        <input
          :id="`em-host-fqdn-${index}`"
          type="text"
          class="em-host-input"
          :value="host.fqdn"
          :placeholder="t('emHosts.hostnamePlaceholder')"
          autocomplete="off"
          spellcheck="false"
          :aria-label="t('emHosts.hostnameLabel')"
          @input="onInput(index, $event)"
        />
        <button
          v-if="canRemove"
          type="button"
          class="em-host-remove"
          :title="t('emHosts.removeHost')"
          :aria-label="t('emHosts.removeHost')"
          @click="emit('remove', index)"
        >
          <v-icon size="16">mdi-close</v-icon>
        </button>
      </div>
    </div>

    <button
      v-if="canAdd"
      type="button"
      class="em-host-add"
      @click="emit('add')"
    >
      <v-icon size="16" class="em-host-add-icon">mdi-plus</v-icon>
      {{ t('emHosts.addHost') }}
    </button>
  </div>
</template>

<style scoped>
.em-host-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.em-host-label {
  font-size: 12.5px;
  font-weight: 500;
  color: #334155;
}

.em-host-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.em-host-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.em-host-index {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: #f1f5f9;
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.em-host-input {
  flex: 1;
  min-width: 0;
  padding: 8px 12px;
  border: 1.5px solid #e2e8f0;
  border-radius: 8px;
  font-size: 13.5px;
  color: #0f172a;
  background: #fff;
  outline: none;
  transition: border-color 0.15s ease;
  font-family: inherit;
  box-sizing: border-box;
}

.em-host-input::placeholder {
  color: #94a3b8;
}

.em-host-input:focus {
  border-color: #1f6dd9;
}

.em-host-remove {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  color: #dc2626;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;
}

.em-host-remove:hover {
  background: #fee2e2;
  border-color: #fca5a5;
}

.em-host-add {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  gap: 4px;
  padding: 6px 2px;
  background: none;
  border: none;
  font-size: 12.5px;
  font-weight: 500;
  color: #1f6dd9;
  cursor: pointer;
  font-family: inherit;
  transition: color 0.15s ease, opacity 0.15s ease;
}

.em-host-add:hover {
  color: #1557b0;
}

.em-host-add-icon {
  margin-right: 2px;
}
</style>
