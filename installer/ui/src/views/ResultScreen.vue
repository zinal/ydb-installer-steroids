<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useInstall } from '@/composables/useInstall'
import { OPERATIONS } from '@/config/operations'
import { useTopologyStore } from '@/stores/topology'
import { useHostsStore } from '@/stores/hosts'
import { useDistributionStore } from '@/stores/distribution'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useDatabasesStore } from '@/stores/databases'
import { useEmStore } from '@/stores/em'
import ResultSummary from '@/components/common/ResultSummary.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

/** Default gRPC and monitoring ports exposed by a YDB storage node. */
const YDB_GRPC_PORT = 2135
const YDB_UI_PORT = 8765

const { t } = useI18n()
const session = useSessionStore()
const install = useInstall()
const topology = useTopologyStore()
const hosts = useHostsStore()
const distribution = useDistributionStore()
const ydbConfig = useYdbConfigStore()
const databases = useDatabasesStore()
const emStore = useEmStore()

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

const clusterInfo = computed(() => ({
  topology: topology.selected || 'Unknown',
  version: distribution.version || distribution.archivePath || 'Unknown',
  domain: ydbConfig.domain || 'Root',
  database: databases.entries[0]?.name
    ? `/${ydbConfig.domain}/${databases.entries[0].name}`
    : 'N/A',
}))

const emHostsList = computed(() =>
  emStore.hosts.map(h => h.fqdn.trim()).filter(Boolean),
)

const emUiUrl = computed(() => {
  const host = emHostsList.value[0]
  return host ? `https://${host}:8080` : ''
})

const showEmSection = computed(() => session.showEmStep && isSuccess.value)

const dcEntries = computed(() => {
  const entries: { label: string; hosts: string[] }[] = []
  const dcKeys: (keyof typeof hosts.dcHosts)[] = ['DC-1', 'DC-2', 'DC-3']
  for (const dc of dcKeys) {
    const hostList = hosts.dcHosts[dc]
    if (hostList && hostList.length > 0) {
      entries.push({ label: dc, hosts: hostList })
    }
  }
  if (entries.length === 0) {
    const allHosts = [...hosts.dcHosts['DC-1'], ...hosts.dcHosts['DC-2'], ...hosts.dcHosts['DC-3']].filter(Boolean)
    if (allHosts.length > 0) {
      entries.push({ label: 'Hosts', hosts: allHosts })
    }
  }
  return entries
})

/** First host across all DCs — the cluster's canonical entry point. */
const firstHost = computed(() => dcEntries.value.flatMap((e) => e.hosts)[0] ?? '')

const endpoint = computed(() =>
  firstHost.value ? `grpcs://${firstHost.value}:${YDB_GRPC_PORT}` : 'N/A',
)

const ydbUiUrl = computed(() =>
  firstHost.value ? `https://${firstHost.value}:${YDB_UI_PORT}` : '#',
)

const logPath = computed(() => {
  const today = new Date().toISOString().split('T')[0]
  return `/tmp/ydb-installer/logs/install-${today}.log`
})

const errorRecommendations = computed(() => {
  const err = install.error.value
  if (!err) return []
  const recs: string[] = []

  if (err.task) {
    recs.push(`Failed at step: ${err.task}`)
  }
  if (err.host) {
    recs.push(`Host involved: ${err.host}`)
  }
  recs.push(t('result.rec1'))
  recs.push(t('result.rec2'))

  return recs
})

function goBack() {
  session.setCurrent(OPERATIONS.ydb.progressStep)
}
</script>

<template>
  <div class="screen-shell">
    <header class="screen-head">
      <ScreenEyebrow />
      <h1 class="screen-title">{{ t('result.heading') }}</h1>
    </header>

    <ResultSummary
      :state="summaryState"
      :success-title="t('result.successTitle')"
      :success-message="t('result.successMessage')"
      :error-title="t('result.errorTitle')"
      :error-message="t('result.errorMessage')"
      :no-result-message="t('result.noResult')"
      :logs="install.logs.value"
      :log-status="logStatus"
      v-model:search-query="install.searchQuery.value"
      :log-path="logPath"
    >
      <template #detail>
        <template v-if="isSuccess">
          <v-card class="section-card pa-6 mb-4">
            <div class="section-label mb-4">{{ t('result.clusterInfo') }}</div>

            <div class="info-grid">
              <div class="info-row">
                <span class="info-key">{{ t('result.topology') }}</span>
                <span class="info-val">{{ clusterInfo.topology }}</span>
              </div>
              <div class="info-row">
                <span class="info-key">{{ t('result.version') }}</span>
                <span class="info-val">{{ clusterInfo.version }}</span>
              </div>
              <div class="info-row">
                <span class="info-key">{{ t('result.domain') }}</span>
                <span class="info-val">{{ clusterInfo.domain }}</span>
              </div>
              <div class="info-row">
                <span class="info-key">{{ t('result.database') }}</span>
                <span class="info-val">{{ clusterInfo.database }}</span>
              </div>
            </div>

            <v-divider class="my-4" />

            <div class="section-sublabel mb-3">{{ t('result.hosts') }}</div>
            <div class="hosts-list">
              <div
                v-for="dc in dcEntries"
                :key="dc.label"
                class="dc-group"
              >
                <div class="dc-group-label">{{ t('result.dcLabel', { dc: dc.label }) }}</div>
                <div
                  v-for="(host, hIndex) in dc.hosts"
                  :key="hIndex"
                  class="host-item"
                >
                  <v-icon size="16" color="success" class="mr-2">mdi-check-circle</v-icon>
                  <span class="host-name">{{ host }}</span>
                  <v-chip size="x-small" color="success" variant="tonal" class="ml-2">
                    {{ t('result.healthy') }}
                  </v-chip>
                </div>
              </div>
            </div>

            <v-divider class="my-4" />

            <div class="endpoint-block">
              <v-icon size="16" class="mr-2 text-medium-emphasis">mdi-console-line</v-icon>
              <span class="info-key">{{ t('result.endpoint') }}</span>
              <code class="endpoint-val">{{ endpoint }}</code>
            </div>

            <div class="ydb-ui-row">
              <v-icon size="16" class="mr-2 text-primary">mdi-open-in-new</v-icon>
              <span class="info-key">{{ t('result.ydbUi') }}:</span>
              <a :href="ydbUiUrl" target="_blank" class="ui-link" rel="noopener noreferrer">
                {{ ydbUiUrl }}
                <v-icon size="14" class="ml-1">mdi-open-in-new</v-icon>
              </a>
            </div>
          </v-card>

          <v-card v-if="databases.entries.length > 0" class="section-card pa-6 mb-4">
            <div class="section-label mb-4">{{ t('result.createdDatabases') }}</div>
            <div class="db-list">
              <div
                v-for="(entry, i) in databases.entries"
                :key="i"
                class="db-item"
              >
                <v-icon size="16" color="success" class="mr-2">mdi-check-circle</v-icon>
                <span class="db-name">{{ entry.name }}</span>
                <v-chip
                  v-if="session.showEmStep && i === 1"
                  size="x-small"
                  color="primary"
                  variant="tonal"
                  class="ml-2"
                >
                  {{ t('result.emDbBadge') }}
                </v-chip>
                <v-spacer />
                <span class="text-caption text-medium-emphasis">
                  {{ t('result.groupsCount', { count: entry.groups }) }}
                </span>
              </div>
            </div>
          </v-card>

          <v-card v-if="showEmSection" class="section-card pa-6 mb-4">
            <div class="section-label mb-4">{{ t('result.emSection') }}</div>
            <template v-if="isSuccess">
              <div class="section-sublabel mb-3">{{ t('result.emHosts') }}</div>
              <div class="hosts-list mb-4">
                <div
                  v-for="(host, idx) in emHostsList"
                  :key="idx"
                  class="host-item"
                >
                  <v-icon size="16" color="success" class="mr-2">mdi-check-circle</v-icon>
                  <span class="host-name">{{ host }}</span>
                </div>
                <div v-if="emHostsList.length === 0" class="text-medium-emphasis">
                  {{ t('result.noEmHosts') }}
                </div>
              </div>
              <div v-if="emUiUrl" class="ydb-ui-row">
                <v-icon size="16" class="mr-2 text-primary">mdi-open-in-new</v-icon>
                <span class="info-key">{{ t('result.emUi') }}:</span>
                <a :href="emUiUrl" target="_blank" class="ui-link" rel="noopener noreferrer">
                  {{ emUiUrl }}
                  <v-icon size="14" class="ml-1">mdi-open-in-new</v-icon>
                </a>
              </div>
            </template>
            <div v-else class="text-medium-emphasis">
              {{ t('result.emNotInstalled') }}
            </div>
          </v-card>
        </template>

        <v-card
          v-else-if="isError && install.error.value"
          class="section-card pa-6 mb-4 error-detail-card"
        >
          <div class="section-label mb-4">{{ t('result.errorInfo') }}</div>

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

          <div class="section-sublabel mb-3">{{ t('result.recommendations') }}</div>
          <ul v-if="errorRecommendations.length > 0" class="recommendations-list">
            <li v-for="(rec, idx) in errorRecommendations" :key="idx">{{ rec }}</li>
          </ul>
          <p v-else class="text-caption text-medium-emphasis">
            {{ t('result.noRecommendations') }}
          </p>
        </v-card>
      </template>
    </ResultSummary>

    <div v-if="isError" class="result-actions">
      <v-btn variant="outlined" size="large" @click="goBack">
        <v-icon start>mdi-arrow-left</v-icon>
        {{ t('result.goBack') }}
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

.info-grid { display: flex; flex-direction: column; gap: 12px; }
.info-row { display: flex; gap: 16px; align-items: baseline; }
.info-key {
  font-size: 13px;
  color: #64748b;
  min-width: 100px;
  flex-shrink: 0;
}
.info-val {
  font-size: 14px;
  color: #0f172a;
  font-weight: 500;
}

.hosts-list { display: flex; flex-direction: column; gap: 16px; }
.dc-group { display: flex; flex-direction: column; gap: 6px; }
.dc-group-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #64748b;
  padding: 0 12px;
}
.host-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 6px;
  margin-left: 0;
}
.host-name {
  font-size: 13px;
  color: #0f172a;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}

.endpoint-block {
  display: flex;
  align-items: center;
  gap: 8px;
}
.endpoint-val {
  font-size: 13px;
  color: #0f172a;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  background: #f1f5f9;
  padding: 4px 8px;
  border-radius: 4px;
}

.ydb-ui-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 8px 0;
}
.ui-link {
  font-size: 13px;
  color: #2563eb;
  text-decoration: none;
  display: flex;
  align-items: center;
}
.ui-link:hover {
  text-decoration: underline;
}

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
.recommendations-list li {
  margin-bottom: 6px;
}

.result-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-top: 8px;
}

.db-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.db-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 6px;
  flex-wrap: wrap;
}
.db-name {
  font-size: 14px;
  color: #0f172a;
  font-weight: 500;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
</style>
