<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import { useSessionStore } from '@/stores/session'
import { useInstall } from '@/composables/useInstall'
import { OPERATIONS } from '@/config/operations'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const { t } = useI18n()
const router = useRouter()
const session = useSessionStore()
const install = useInstall()

interface EmInfo {
  hosts: string[]
  endpoint?: string
  database?: string
}

const emInfo = ref<EmInfo>({ hosts: [] })
const configLoading = ref(true)

onMounted(async () => {
  session.setBeforeNext(startDeletionApiOnly)
  try {
    const r = await api.getEMInventory()
    if (r.data) {
      const hosts = (r.data.hosts ?? [])
        .map((h) => (typeof h === 'string' ? h : h.fqdn))
        .map((h) => (h ?? '').trim())
        .filter(Boolean)
      emInfo.value = {
        hosts,
        endpoint: r.data.db_connection_endpoint || undefined,
        database: r.data.db_connection_db || undefined,
      }
    }
  } finally {
    configLoading.value = false
  }
})

onUnmounted(() => {
  session.setBeforeNext(null)
})

const confirmText = ref('')
const CONFIRM_WORD = 'DELETE'
const starting = ref(false)
let startIssued = false

const canStartDeletion = computed(() => confirmText.value === CONFIRM_WORD)

watch(
  confirmText,
  (v) => {
    session.setCanGoNext(v === CONFIRM_WORD)
  },
  { immediate: true },
)

async function startDeletionApiOnly(): Promise<boolean> {
  if (!canStartDeletion.value) return false
  if (startIssued) return true
  if (starting.value) return false
  starting.value = true
  try {
    await install.start(OPERATIONS['delete-em-ydb'].startEndpoint)
    if (install.status.value === 'error') {
      return false
    }
    startIssued = true
    return true
  } finally {
    starting.value = false
  }
}

async function startDeletion() {
  const ok = await startDeletionApiOnly()
  if (ok) {
    router.push({ name: OPERATIONS['delete-em-ydb'].progressStep })
  }
}
</script>

<template>
  <div class="screen-shell">
    <header class="screen-head">
      <ScreenEyebrow variant="danger" />
      <h1 class="screen-title">{{ t('delete.titleEmYdb') }}</h1>
    </header>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label">{{ t('delete.componentsInfo') }}</div>

      <div v-if="configLoading" class="info-loading">
        <v-progress-circular indeterminate size="18" width="2" color="primary" />
      </div>

      <div v-else class="info-grid">
        <div v-if="emInfo.hosts.length > 0" class="info-row">
          <span class="info-key">{{ t('delete.emHosts') }}</span>
          <div class="info-hosts">
            <span v-for="h in emInfo.hosts" :key="h" class="info-host">{{ h }}</span>
          </div>
        </div>
        <div v-else class="info-row">
          <span class="info-key">{{ t('delete.emHosts') }}</span>
          <span class="info-val">—</span>
        </div>
        <div v-if="emInfo.endpoint" class="info-row">
          <span class="info-key">{{ t('delete.connectedEndpoint') }}</span>
          <span class="info-val mono">{{ emInfo.endpoint }}</span>
        </div>
        <div v-if="emInfo.database" class="info-row">
          <span class="info-key">{{ t('delete.connectedDatabase') }}</span>
          <span class="info-val mono">{{ emInfo.database }}</span>
        </div>
      </div>
    </v-card>

    <v-card class="section-card warning-card pa-6 mb-4">
      <div class="warning-header">
        <v-icon size="20" color="error" class="mr-2">mdi-alert</v-icon>
        <span class="warning-title">{{ t('delete.warningTitle') }}</span>
      </div>
      <div class="warning-body">
        <p class="warning-will-delete">{{ t('delete.willDelete') }}</p>
        <ul class="warning-list">
          <li>{{ t('delete.emServices') }}</li>
          <li>{{ t('delete.emAgents') }}</li>
          <li>{{ t('delete.emConfigFiles') }}</li>
          <li>{{ t('delete.emBinaries') }}</li>
        </ul>
        <p class="warning-note">{{ t('delete.emYdbUntouched') }}</p>
      </div>
    </v-card>

    <div class="confirm-block">
      <label class="confirm-label">{{ t('delete.confirmPrompt') }}</label>
      <input
        v-model="confirmText"
        type="text"
        class="confirm-input"
        :class="{ 'is-valid': canStartDeletion }"
        :placeholder="t('delete.confirmPlaceholder')"
        autocomplete="off"
        spellcheck="false"
      />
    </div>

    <div class="start-row">
      <v-btn
        color="error"
        size="large"
        :disabled="!canStartDeletion"
        :loading="starting"
        class="start-btn"
        @click="startDeletion"
      >
        <v-icon start size="18">mdi-delete-forever</v-icon>
        {{ t('delete.startDeletionBtn') }}
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
  margin-bottom: 14px;
}

.info-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #94a3b8;
  font-size: 13px;
}

.info-grid { display: flex; flex-direction: column; gap: 10px; }
.info-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.info-key {
  font-size: 12.5px;
  color: #64748b;
  min-width: 120px;
}
.info-val {
  font-size: 13.5px;
  color: #0f172a;
  font-weight: 500;
}
.info-val.mono,
.info-host {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.info-hosts { display: flex; flex-direction: column; gap: 2px; }
.info-host {
  font-size: 13px;
  color: #334155;
}

.warning-card {
  border: 1.5px solid #fca5a5 !important;
  background: #fff5f5 !important;
}
.warning-header {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
.warning-title {
  font-size: 14px;
  font-weight: 700;
  color: #b91c1c;
}
.warning-body { padding-left: 28px; }
.warning-will-delete {
  font-size: 13px;
  color: #7f1d1d;
  margin: 0 0 8px;
  font-weight: 500;
}
.warning-list {
  margin: 0 0 10px;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.warning-list li {
  font-size: 13px;
  color: #991b1b;
}
.warning-note {
  font-size: 12.5px;
  color: #7f1d1d;
  margin: 0;
  font-style: italic;
}

.confirm-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 20px;
}
.confirm-label {
  font-size: 13.5px;
  font-weight: 500;
  color: #334155;
}
.confirm-input {
  padding: 10px 14px;
  border: 2px solid #e2e8f0;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 600;
  color: #0f172a;
  background: #fff;
  outline: none;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  letter-spacing: 0.08em;
  transition: border-color .15s ease;
  max-width: 240px;
}
.confirm-input:focus { border-color: #94a3b8; }
.confirm-input.is-valid { border-color: #16a34a; color: #15803d; }

.start-row {
  display: flex;
  justify-content: flex-start;
  padding: 8px 0 4px;
}
.start-btn {
  font-weight: 600 !important;
  letter-spacing: -0.005em;
}
</style>
