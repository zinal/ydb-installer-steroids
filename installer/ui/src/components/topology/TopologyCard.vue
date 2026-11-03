<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Topology } from '@/types/topology'
import TopologyDiagram from './TopologyDiagram.vue'

interface Props {
  topology: Topology
  selected: boolean
  selectable: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  select: [id: string]
}>()

const { t } = useI18n()

const minHostsText = computed(() => {
  const count = props.topology.minHosts
  return t('topology.minHosts', { count })
})

const minDisksText = computed(() => {
  if (props.topology.minDisks === 0) {
    return t('topology.noDisks')
  }
  const count = props.topology.minDisks
  return t('topology.minDisks', { count })
})

function handleClick() {
  if (props.selectable) {
    emit('select', props.topology.id)
  }
}
</script>

<template>
  <div
    class="topology-card"
    :class="{
      'is-selected': selected,
      'is-disabled': !selectable,
    }"
    @click="handleClick"
  >
    <div class="topology-diagram">
      <TopologyDiagram :type="topology.diagramType" />
    </div>

    <div class="topology-content">
      <div class="topology-header">
        <h3 class="topology-name">{{ t(topology.nameKey) }}</h3>
        <div v-if="selected" class="topology-badge topology-badge--selected">
          <v-icon size="16">mdi-check</v-icon>
          {{ t('topology.selected') }}
        </div>
      </div>

      <p class="topology-description">{{ t(topology.descriptionKey) }}</p>

      <div class="topology-requirements">
        <div v-if="topology.id !== 'bridge-2-dc'" class="requirement-item">
          <v-icon size="16" color="#64748b">mdi-server</v-icon>
          <span>{{ minHostsText }}</span>
        </div>
        <div v-if="topology.id !== 'bridge-2-dc'" class="requirement-item">
          <v-icon size="16" color="#64748b">mdi-harddisk</v-icon>
          <span>{{ minDisksText }}</span>
        </div>
        <div v-if="topology.id === 'bridge-2-dc'" class="requirement-item">
          <v-icon size="16" color="#64748b">mdi-link-variant</v-icon>
          <span>{{ t('topology.relatedTopologies') }}</span>
        </div>
      </div>
    </div>

    <div v-if="!selectable" class="topology-lock-badge">
      <v-icon size="16" color="#f59e0b">mdi-lock</v-icon>
      <span>{{ t('topology.comingSoon') }}</span>
    </div>
  </div>
</template>

<style scoped>
.topology-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
  background: #fff;
  border: 2px solid #e2e8f0;
  border-radius: 16px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.topology-card:hover:not(.is-disabled) {
  border-color: #cbd5e1;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08);
}

.topology-card.is-selected {
  border-color: #1f6dd9;
  background: linear-gradient(180deg, #f5f9ff 0%, #ffffff 100%);
  box-shadow: 0 0 0 4px rgba(31, 109, 217, 0.1), 0 4px 12px rgba(15, 23, 42, 0.08);
}

.topology-card.is-disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.topology-card.is-disabled:hover {
  border-color: #e2e8f0;
  box-shadow: none;
}

.topology-diagram {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 140px;
  background: #f8fafc;
  border-radius: 12px;
  padding: 16px;
}

.topology-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.topology-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.topology-name {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: -0.01em;
}

.topology-description {
  margin: 0;
  font-size: 13.5px;
  color: #64748b;
  line-height: 1.5;
}

.topology-requirements {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.requirement-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: #64748b;
}

.topology-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
  text-transform: uppercase;
}

.topology-badge--selected {
  background: #dbeafe;
  color: #1e40af;
}

.topology-lock-badge {
  position: absolute;
  top: 12px;
  right: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: #fef3c7;
  color: #92400e;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
  text-transform: uppercase;
}
</style>
