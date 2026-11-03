<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import { useSessionStore } from '@/stores/session'
import { useInstall } from '@/composables/useInstall'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const { t } = useI18n()
const router = useRouter()
const session = useSessionStore()
const install = useInstall()

interface ClusterInfo {
  topology?: string
  version?: string
  hosts: string[]
}

const clusterInfo = ref<ClusterInfo>({ hosts: [] })
const configLoading = ref(true)

onMounted(async () => {
  session.setBeforeNext(startDeletionApiOnly)
  try {
    const r = await api.getInventory()
    if (r.data) {
      clusterInfo.value = {
        topology: r.data.topology?.type,
        version: r.data.distribution?.version ?? '—',
        hosts: r.data.hosts ?? [],
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
    await install.start('/api/install/delete-ydb/start')
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
    router.push({ name: '31' })
  }
}
</script>

<template>
  <div class="screen-shell">
    <header class="screen-head">
      <ScreenEyebrow variant="danger" />
      <h1 class="screen-title">{{ t('delete.titleYdb') }}</h1>
    </header>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label">{{ t('delete.clusterInfo') }}</div>

      <div v-if="configLoading" class="info-loading">
        <v-progress-circular indeterminate size="18" width="2" color="primary" />
      </div>

      <div v-else class="info-grid">
        <div class="info-row">
          <span class="info-key">{{ t('delete.topology') }}</span>
          <span class="info-val">{{ clusterInfo.topology ?? '—' }}</span>
        </div>
        <div class="info-row">
          <span class="info-key">{{ t('delete.version') }}</span>
          <span class="info-val">{{ clusterInfo.version ?? '—' }}</span>
        </div>
        <div v-if="clusterInfo.hosts.length > 0" class="info-row">
          <span class="info-key">{{ t('delete.hosts') }}</span>
          <div class="info-hosts">
            <span v-for="h in clusterInfo.hosts" :key="h" class="info-host">{{ h }}</span>
          </div>
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
          <li>{{ t('delete.services') }}</li>
          <li>{{ t('delete.diskData') }}</li>
          <li>{{ t('delete.configFiles') }}</li>
          <li>{{ t('delete.binaries') }}</li>
        </ul>
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
        @click="startDeletion"
        class="start-btn"
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
  min-width: 100px;
}
.info-val {
  font-size: 13.5px;
  color: #0f172a;
  font-weight: 500;
}
.info-hosts { display: flex; flex-direction: column; gap: 2px; }
.info-host {
  font-size: 13px;
  color: #334155;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
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
  margin: 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.warning-list li {
  font-size: 13px;
  color: #991b1b;
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
