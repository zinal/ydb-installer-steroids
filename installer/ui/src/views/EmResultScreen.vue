<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useEmStore } from '@/stores/em'
import { useHostsStore } from '@/stores/hosts'
import { useInstall } from '@/composables/useInstall'
import { OPERATIONS } from '@/config/operations'
import ResultSummary from '@/components/common/ResultSummary.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

/** Port the YDB EM web UI listens on. */
const EM_UI_PORT = 8789
/** URL path prefix for the YDB EM web UI. */
const EM_UI_PATH = '/ui/home'

const { t } = useI18n()
const session = useSessionStore()
const em = useEmStore()
const hosts = useHostsStore()
const install = useInstall()

onMounted(async () => {
  await install.waitForRecovery()
})

const isSuccess = computed(() => install.status.value === 'success')
const isError = computed(() => install.status.value === 'error')

/** Narrow the raw status to the tri-state ResultSummary expects. */
const summaryState = computed<'success' | 'error' | 'idle'>(() => {
  if (isSuccess.value) return 'success'
  if (isError.value) return 'error'
  return 'idle'
})

const logStatus = computed(() => install.logStatus?.value ?? 'none')

/** Non-empty EM host FQDNs (ydb_em group in 90-inventory.yaml). */
const emHosts = computed(() => em.emHostsList)

/**
 * DB slot hosts — the YDB dynamic nodes EM manages. They come from
 * 50-inventory.yaml via useInventorySync, so the list is legitimately empty
 * when EM is installed standalone against a cluster whose inventory is absent.
 */
const slotHosts = computed(() => {
  const dc = hosts.dcHosts
  return [...(dc['DC-1'] ?? []), ...(dc['DC-2'] ?? []), ...(dc['DC-3'] ?? [])].filter(Boolean)
})

/** Endpoint EM was pointed at (explicit value wins over the derived one). */
const ydbEndpoint = computed(() => em.dbConnectionEndpoint.trim() || em.derivedEndpoint)

const emUiUrl = computed(() => {
  const host = emHosts.value[0]
  return host ? `https://${host}:${EM_UI_PORT}${EM_UI_PATH}` : ''
})

const logPath = computed(() => {
  const today = new Date().toISOString().split('T')[0]
  return `/tmp/ydb-installer/logs/install-em-${today}.log`
})

/** Recommendations derived from the failure, mirroring Screen 10. */
const errorRecommendations = computed(() => {
  const err = install.error.value
  if (!err) return []
  const recs: string[] = []
  if (err.task) recs.push(t('emResult.recTask', { task: err.task }))
  if (err.host) recs.push(t('emResult.recHost', { host: err.host }))
  recs.push(t('emResult.rec1'))
  recs.push(t('emResult.rec2'))
  return recs
})

function goBack() {
  session.setCurrent(OPERATIONS['ydb-em'].progressStep)
}
</script>

<template>
  <div class="screen-shell">
    <header class="screen-head">
      <ScreenEyebrow />
      <h1 class="screen-title">{{ t('emResult.heading') }}</h1>
    </header>

    <ResultSummary
      :state="summaryState"
      :success-title="t('emResult.successTitle')"
      :success-message="t('emResult.successMessage')"
      :error-title="t('emResult.errorTitle')"
      :error-message="t('emResult.errorMessage')"
      :no-result-message="t('emResult.noResult')"
      :logs="install.logs.value"
      :log-status="logStatus"
      v-model:search-query="install.searchQuery.value"
      :log-path="logPath"
    >
      <template #detail>
        <v-card v-if="isSuccess" class="section-card pa-6 mb-4">
          <div class="section-label mb-4">{{ t('emResult.emInfo') }}</div>

          <div class="section-sublabel mb-3">{{ t('emResult.emHosts') }}</div>
          <div class="hosts-list mb-4">
            <div v-for="(host, i) in emHosts" :key="i" class="host-item">
              <v-icon size="16" color="success" class="mr-2">mdi-check-circle</v-icon>
              <span class="host-name">{{ host }}</span>
            </div>
            <p v-if="emHosts.length === 0" class="empty-note">
              {{ t('emResult.noEmHosts') }}
            </p>
          </div>

          <v-divider class="my-4" />

          <div class="section-sublabel mb-3">{{ t('emResult.slotHosts') }}</div>
          <div class="hosts-list">
            <div v-for="(host, i) in slotHosts" :key="i" class="host-item">
              <v-icon size="16" color="success" class="mr-2">mdi-check-circle</v-icon>
              <span class="host-name">{{ host }}</span>
            </div>
            <p v-if="slotHosts.length === 0" class="empty-note">
              {{ t('emResult.noSlotHosts') }}
            </p>
          </div>

          <v-divider class="my-4" />

          <div class="endpoint-block">
            <v-icon size="16" class="mr-2 text-medium-emphasis">mdi-console-line</v-icon>
            <span class="info-key">{{ t('emResult.connectedTo') }}</span>
            <code class="endpoint-val">{{ ydbEndpoint || t('emResult.unknown') }}</code>
          </div>

          <div v-if="emUiUrl" class="em-ui-row">
            <v-icon size="16" class="mr-2 text-primary">mdi-open-in-new</v-icon>
            <span class="info-key">{{ t('emResult.emUi') }}:</span>
            <a :href="emUiUrl" target="_blank" class="ui-link" rel="noopener noreferrer">
              {{ emUiUrl }}
              <v-icon size="14" class="ml-1">mdi-open-in-new</v-icon>
            </a>
          </div>
        </v-card>

        <v-card
          v-else-if="isError && install.error.value"
          class="section-card pa-6 mb-4 error-detail-card"
        >
          <div class="section-label mb-4">{{ t('emResult.errorInfo') }}</div>

          <div class="error-grid">
            <div class="error-row">
              <span class="error-key">{{ t('install.errorTask') }}</span>
              <code class="error-val">{{ install.error.value.task }}</code>
            </div>
            <div v-if="install.error.value.host" class="error-row">
              <span class="error-key">{{ t('install.errorHost') }}</span>
              <code class="error-val">{{ install.error.value.host }}</code>
            </div>
            <div class="error-row">
              <span class="error-key">{{ t('install.errorReason') }}</span>
              <span class="error-val-reason">{{ install.error.value.reason }}</span>
            </div>
          </div>

          <v-divider class="my-4" />

          <div class="section-sublabel mb-3">{{ t('emResult.recommendations') }}</div>
          <ul class="recommendations-list">
            <li v-for="(rec, idx) in errorRecommendations" :key="idx">{{ rec }}</li>
          </ul>
        </v-card>
      </template>
    </ResultSummary>

    <div v-if="isError" class="result-actions">
      <v-btn variant="outlined" size="large" @click="goBack">
        <v-icon start>mdi-arrow-left</v-icon>
        {{ t('emResult.goBack') }}
      </v-btn>
    </div>
  </div>
</template>

<style scoped>
.screen-head { margin-bottom: 28px; }
.screen-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0;
}

.section-card { background: rgb(var(--v-theme-surface)); }
.section-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #94a3b8;
}
.section-sublabel {
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
}

.hosts-list { display: flex; flex-direction: column; gap: 6px; }
.host-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 6px;
}
.host-name {
  font-size: 13px;
  color: #0f172a;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.empty-note {
  font-size: 12px;
  color: #94a3b8;
  margin: 0;
  font-style: italic;
}

.info-key {
  font-size: 13px;
  color: #64748b;
  flex-shrink: 0;
}
.endpoint-block {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.endpoint-val {
  font-size: 13px;
  color: #0f172a;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  background: #f1f5f9;
  padding: 4px 8px;
  border-radius: 4px;
}

.em-ui-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 8px 0;
  flex-wrap: wrap;
}
.ui-link {
  font-size: 13px;
  color: #2563eb;
  text-decoration: none;
  display: flex;
  align-items: center;
}
.ui-link:hover { text-decoration: underline; }

.error-detail-card {
  border: 1px solid #fecaca !important;
}
.error-grid { display: flex; flex-direction: column; gap: 12px; }
.error-row { display: flex; gap: 16px; align-items: baseline; }
.error-key {
  font-size: 12px;
  color: #64748b;
  min-width: 80px;
  flex-shrink: 0;
}
.error-val {
  font-size: 13px;
  color: #0f172a;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
}
.error-val-reason {
  font-size: 13px;
  color: #b91c1c;
  font-weight: 500;
}

.recommendations-list {
  margin: 0;
  padding-left: 20px;
  color: #475569;
  font-size: 13px;
}
.recommendations-list li { margin-bottom: 6px; }

.result-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-top: 8px;
}
</style>
