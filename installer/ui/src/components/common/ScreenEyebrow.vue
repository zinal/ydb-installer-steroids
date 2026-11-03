<script setup lang="ts">
// Step number is sequential per flow (from visibleFlowSteps), never the
// design-doc screen ID.
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'

const props = withDefaults(defineProps<{
  /** `danger` renders the red pill used by the delete flows. */
  variant?: 'default' | 'danger'
}>(), {
  variant: 'default',
})

const session = useSessionStore()
const { t } = useI18n()

const flowLabel = computed(() => {
  switch (session.flow) {
    case 'ydb':        return 'YDB'
    case 'delete-ydb': return t('intro.deleteYdb')
    default:           return ''
  }
})
</script>

<template>
  <div class="screen-eyebrow">
    <span
      class="eyebrow-pill"
      :class="{ 'eyebrow-pill--danger': props.variant === 'danger' }"
      data-test="step-pill"
    >
      {{ t('common.step') }} {{ session.currentStepNumber }}
    </span>
    <span v-if="flowLabel" class="eyebrow-flow">· {{ flowLabel }}</span>
  </div>
</template>

<style scoped>
.screen-eyebrow {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.eyebrow-pill {
  background: #eef4ff;
  color: #1f6dd9;
  font-weight: 600;
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 4px 10px;
  border-radius: 999px;
  font-variant-numeric: tabular-nums;
}
.eyebrow-pill--danger {
  background: #fee2e2;
  color: #dc2626;
}

.eyebrow-flow {
  font-size: 12px;
  color: #94a3b8;
  font-weight: 500;
}
</style>
