<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useEmStore } from '@/stores/em'
import EmDistributionFields from '@/components/em/EmDistributionFields.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const session = useSessionStore()
const em = useEmStore()
const { t } = useI18n()

watch(() => em.distCanProceed, (can) => {
  session.setCanGoNext(can)
}, { immediate: true })
</script>

<template>
  <div class="em-dist-shell">
    <header class="em-dist-head">
      <ScreenEyebrow />
      <h1 class="em-dist-title">{{ t('emDistribution.heading') }}</h1>
      <p class="em-dist-sub">{{ t('emDistribution.subtitle') }}</p>
    </header>

    <EmDistributionFields />
  </div>
</template>

<style scoped>
.em-dist-head { margin-bottom: 28px; }
.em-dist-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.em-dist-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
}
</style>
