<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

type DiagramType = 'three-dc' | 'block' | 'bridge' | 'single'

interface Props {
  type: DiagramType
}

const props = defineProps<Props>()
const { t } = useI18n()

const diagramContent = computed(() => {
  switch (props.type) {
    case 'three-dc':
      return renderThreeDC()
    case 'block':
      return renderBlock()
    case 'bridge':
      return renderBridge()
    case 'single':
      return renderSingle()
    default:
      return ''
  }
})

function renderThreeDC() {
  return `
    <!-- DC Zones -->
    <rect x="10" y="10" width="90" height="140" rx="8" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="55" y="30" text-anchor="middle" fill="#64748b" font-size="10" font-weight="600">DC 1</text>
    
    <rect x="110" y="10" width="90" height="140" rx="8" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="155" y="30" text-anchor="middle" fill="#64748b" font-size="10" font-weight="600">DC 2</text>
    
    <rect x="210" y="10" width="90" height="140" rx="8" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="255" y="30" text-anchor="middle" fill="#64748b" font-size="10" font-weight="600">DC 3</text>
    
    <!-- Nodes DC 1 -->
    <rect x="30" y="45" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="55" y="63" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
    <rect x="30" y="80" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="55" y="98" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
    <rect x="30" y="115" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="55" y="133" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
    
    <!-- Nodes DC 2 -->
    <rect x="130" y="45" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="155" y="63" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
    <rect x="130" y="80" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="155" y="98" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
    <rect x="130" y="115" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="155" y="133" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
    
    <!-- Nodes DC 3 -->
    <rect x="230" y="45" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="255" y="63" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
    <rect x="230" y="80" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="255" y="98" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
    <rect x="230" y="115" width="50" height="28" rx="6" fill="#1f6dd9"/>
    <text x="255" y="133" text-anchor="middle" fill="#fff" font-size="8" font-weight="600">node</text>
  `
}

function renderBlock() {
  return `
    <!-- Row 1 -->
    <rect x="10" y="50" width="70" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="45" y="65" text-anchor="middle" fill="#64748b" font-size="8" font-weight="600">Node 1</text>
    <rect x="20" y="70" width="50" height="15" rx="4" fill="#1f6dd9"/>
    
    <rect x="90" y="50" width="70" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="125" y="65" text-anchor="middle" fill="#64748b" font-size="8" font-weight="600">Node 2</text>
    <rect x="100" y="70" width="50" height="15" rx="4" fill="#1f6dd9"/>
    
    <rect x="170" y="50" width="70" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="205" y="65" text-anchor="middle" fill="#64748b" font-size="8" font-weight="600">Node 3</text>
    <rect x="180" y="70" width="50" height="15" rx="4" fill="#1f6dd9"/>
    
    <rect x="250" y="50" width="70" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="285" y="65" text-anchor="middle" fill="#64748b" font-size="8" font-weight="600">Node 4</text>
    <rect x="260" y="70" width="50" height="15" rx="4" fill="#1f6dd9"/>
    
    <!-- Row 2 -->
    <rect x="10" y="100" width="70" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="45" y="115" text-anchor="middle" fill="#64748b" font-size="8" font-weight="600">Node 5</text>
    <rect x="20" y="120" width="50" height="15" rx="4" fill="#1f6dd9"/>
    
    <rect x="90" y="100" width="70" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="125" y="115" text-anchor="middle" fill="#64748b" font-size="8" font-weight="600">Node 6</text>
    <rect x="100" y="120" width="50" height="15" rx="4" fill="#1f6dd9"/>
    
    <rect x="170" y="100" width="70" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="205" y="115" text-anchor="middle" fill="#64748b" font-size="8" font-weight="600">Node 7</text>
    <rect x="180" y="120" width="50" height="15" rx="4" fill="#1f6dd9"/>
    
    <rect x="250" y="100" width="70" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="285" y="115" text-anchor="middle" fill="#64748b" font-size="8" font-weight="600">Node 8</text>
    <rect x="260" y="120" width="50" height="15" rx="4" fill="#1f6dd9"/>
  `
}

function renderBridge() {
  return `
    <!-- DC 1 -->
    <rect x="10" y="50" width="130" height="80" rx="8" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="75" y="70" text-anchor="middle" fill="#64748b" font-size="10" font-weight="600">DC 1</text>
    <rect x="25" y="80" width="100" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1"/>
    <text x="75" y="105" text-anchor="middle" fill="#64748b" font-size="9">Internal topology</text>
    
    <!-- DC 2 -->
    <rect x="170" y="50" width="130" height="80" rx="8" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <text x="235" y="70" text-anchor="middle" fill="#64748b" font-size="10" font-weight="600">DC 2</text>
    <rect x="185" y="80" width="100" height="40" rx="6" fill="#ffffff" stroke="#e2e8f0" stroke-width="1"/>
    <text x="235" y="105" text-anchor="middle" fill="#64748b" font-size="9">Internal topology</text>
    
    <!-- Bridge link -->
    <line x1="140" y1="90" x2="170" y2="90" stroke="#1f6dd9" stroke-width="3"/>
  `
}

function renderSingle() {
  return `
    <!-- Single node -->
    <rect x="80" y="60" width="150" height="70" rx="8" fill="#ffffff" stroke="#e2e8f0" stroke-width="1.5"/>
    <rect x="110" y="80" width="90" height="30" rx="6" fill="#1f6dd9"/>
    <text x="155" y="100" text-anchor="middle" fill="#fff" font-size="10" font-weight="600">localhost</text>
    <text x="155" y="120" text-anchor="middle" fill="#64748b" font-size="9">(no disks)</text>
  `
}
</script>

<template>
  <svg viewBox="0 0 310 170" class="topology-diagram-svg">
    <g v-html="diagramContent" />
  </svg>
</template>

<style scoped>
.topology-diagram-svg {
  width: 100%;
  height: auto;
  max-width: 310px;
}
</style>
