<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useTopologyStore } from '@/stores/topology'
import { TOPOLOGIES, TOPOLOGY_ORDER } from '@/config/topologies'
import TopologyCard from '@/components/topology/TopologyCard.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const session = useSessionStore()
const topologyStore = useTopologyStore()
const { t } = useI18n()

const topologies = computed(() => TOPOLOGY_ORDER)

function handleSelect(id: string) {
  topologyStore.select(id as any)
}

watch(() => topologyStore.selected, (selected) => {
  session.setCanGoNext(selected !== null)
}, { immediate: true })

onMounted(() => {
})
</script>

<template>
  <div class="topology-shell">
    <header class="topology-head">
      <ScreenEyebrow />
      <h1 class="topology-title">{{ t('topology.heading') }}</h1>
      <p class="topology-sub">{{ t('topology.description') }}</p>
    </header>

    <div class="topology-grid">
      <TopologyCard
        v-for="id in TOPOLOGY_ORDER"
        :key="id"
        :topology="TOPOLOGIES[id]"
        :selected="topologyStore.selected === id"
        :selectable="topologyStore.canSelect(id)"
        @select="handleSelect"
      />
    </div>
  </div>
</template>

<style scoped>
.topology-head { margin-bottom: 28px; }
.topology-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.topology-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
}

.topology-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

@media (max-width: 768px) {
  .topology-grid {
    grid-template-columns: 1fr;
  }
}
</style>
