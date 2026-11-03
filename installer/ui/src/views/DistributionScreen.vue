<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useDistributionStore } from '@/stores/distribution'
import { useTopologyStore } from '@/stores/topology'
import { useConnectivityStore } from '@/stores/connectivity'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'
import { HELP_LINKS } from '@/config/helpLinks'

const session = useSessionStore()
const dist = useDistributionStore()
const topologyStore = useTopologyStore()
const connectivity = useConnectivityStore()
const { t } = useI18n()

const topologyLabel = computed(() => {
  const labels: Record<string, string> = {
    'mirror-3-dc': 'mirror-3-dc',
    'block-4-2': 'block-4-2',
    'bridge-2-dc': 'bridge-2-dc',
    'none': 'none',
  }
  return labels[topologyStore.selected || 'none']
})

const sourceOfficial = computed({
  get: () => dist.source === 'official',
  set: (v) => { dist.source = v ? 'official' : 'local' },
})

const selectedVersion = computed({
  get: () => dist.version,
  set: (v) => { dist.version = v },
})

const archivePath = computed({
  get: () => dist.archivePath,
  set: (v) => { dist.archivePath = v },
})

const cliBinary = computed({
  get: () => dist.cliBinary,
  set: (v) => { dist.cliBinary = v },
})

const binaryPath = computed({
  get: () => dist.binaryPath,
  set: (v) => { dist.binaryPath = v },
})

const opsBinary = computed({
  get: () => dist.opsBinary,
  set: (v) => { dist.opsBinary = v },
})

const dstoolBinary = computed({
  get: () => dist.dstoolBinary,
  set: (v) => { dist.dstoolBinary = v },
})

const archiveValidationMsg = computed(() => {
  if (dist.archiveValid === null) return ''
  if (dist.archiveValid) {
    return t('distribution.fileValid', { size: dist.archiveSize })
  }
  return t('distribution.fileInvalid')
})

const showBinaries = ref(false)

onMounted(async () => {
  if (dist.source === 'official' && dist.versions.length === 0) {
    await dist.fetchVersions()
  }
  if (dist.archivePath) {
    await dist.validateArchivePath(dist.archivePath)
  }
})

watch(archivePath, async (path) => {
  if (path) {
    await dist.validateArchivePath(path)
  } else {
    dist.archiveValid = null
    dist.archiveSize = ''
  }
})

watch(() => dist.canProceed, (can) => {
  session.setCanGoNext(can)
}, { immediate: true })

// The official source needs the GitHub version list; without internet it cannot
// work, so fall back to local files and explain why. Only act on a definitive
// offline result — while the probe is still running (`null`) leave the current
// source untouched so a fast user is not locked out of the official tab.
watch(() => connectivity.effectiveOnline, (online) => {
  if (online === false && dist.source === 'official') {
    dist.source = 'local'
  }
}, { immediate: true })
</script>

<template>
  <div class="dist-shell">
    <header class="dist-head">
      <ScreenEyebrow />
      <h1 class="dist-title">{{ t('distribution.heading') }}</h1>
      <p class="dist-sub">{{ t('distribution.topologyLabel', { topology: topologyLabel }) }}</p>
    </header>

    <section class="dist-section">
      <div class="source-toggle">
        <label
          class="source-option"
          :class="{ 'is-active': sourceOfficial, 'is-disabled': connectivity.effectiveOnline === false }"
        >
          <input
            type="radio"
            v-model="sourceOfficial"
            :value="true"
            :disabled="connectivity.effectiveOnline === false"
          />
          <span class="source-label">{{ t('distribution.official') }}</span>
        </label>
        <label class="source-option" :class="{ 'is-active': !sourceOfficial }">
          <input type="radio" v-model="sourceOfficial" :value="false" />
          <span class="source-label">{{ t('distribution.local') }}</span>
        </label>
      </div>
      <div
        v-if="connectivity.effectiveOnline === false"
        class="offline-warning"
      >
        <v-icon size="16" color="warning" class="mr-1">mdi-wifi-off</v-icon>
        {{ t('distribution.offlineWarning') }}
      </div>
      <div
        v-else-if="connectivity.effectiveOnline === null"
        class="checking-hint"
      >
        <v-progress-circular indeterminate size="14" width="2" color="primary" class="mr-1" />
        {{ t('distribution.checkingConnectivity') }}
      </div>
    </section>

    <section v-if="sourceOfficial" class="dist-section">
      <v-card class="dist-card">
        <div class="card-header">
          <span class="section-eyebrow eyebrow">{{ t('distribution.official') }}</span>
        </div>

        <div class="field-group field-group--last">
          <label class="field-label">{{ t('distribution.version') }}</label>
          <div class="version-row">
            <v-select
              v-model="selectedVersion"
              :items="dist.versions"
              item-title="version"
              item-value="version"
              :loading="dist.loadingVersions"
              :disabled="dist.loadingVersions"
              :placeholder="dist.loadingVersions ? t('distribution.loadingVersions') : t('distribution.selectVersion')"
              density="compact"
              variant="outlined"
              class="version-select"
            >
              <template #item="{ item, props }">
                <v-list-item v-bind="props">
                  <template #title>
                    <span>{{ item.raw.version }}</span>
                    <span v-if="item.raw.recommended" class="recommended-badge">
                      {{ t('distribution.recommended') }}
                    </span>
                  </template>
                </v-list-item>
              </template>
              <template #selection="{ item }">
                {{ item.title }}
                <span v-if="item.raw.recommended" class="recommended-badge ml-2">
                  {{ t('distribution.recommended') }}
                </span>
              </template>
            </v-select>
            <v-btn
              v-if="dist.versions.length === 0 && !dist.loadingVersions"
              size="small"
              variant="outlined"
              prepend-icon="mdi-refresh"
              @click="dist.fetchVersions()"
              class="deps-retry-btn"
            >
              {{ t('distribution.retryFetch') }}
            </v-btn>
          </div>
          <div v-if="dist.versions.length === 0 && !dist.loadingVersions" class="no-versions">
            <v-icon size="16" color="warning" class="mr-1">mdi-alert-outline</v-icon>
            {{ t('distribution.noVersions') }}
          </div>
        </div>
      </v-card>
    </section>

    <section v-else class="dist-section">
      <v-card class="dist-card">
        <div class="card-header">
          <span class="section-eyebrow eyebrow">{{ t('distribution.local') }}</span>
        </div>

        <div class="field-group field-group--last">
          <label class="field-label">
            {{ t('distribution.package') }}
            <span class="required-mark">*</span>
          </label>
          <v-text-field
            v-model="archivePath"
            :placeholder="t('distribution.packagePlaceholder')"
            density="compact"
            variant="outlined"
            hide-details="auto"
            class="path-input"
          />
          <div
            v-if="archiveValidationMsg"
            class="validation-msg"
            :class="{ 'is-valid': dist.archiveValid, 'is-error': !dist.archiveValid }"
          >
            <v-icon
              :icon="dist.archiveValid ? 'mdi-check-circle' : 'mdi-alert-circle'"
              size="14"
              :color="dist.archiveValid ? 'success' : 'error'"
              class="mr-1"
            />
            {{ archiveValidationMsg }}
          </div>
        </div>
      </v-card>
    </section>

    <section class="dist-section">
      <div class="binaries-toggle">
        <button
          type="button"
          class="binaries-toggle-btn"
          @click="showBinaries = !showBinaries"
        >
          <v-icon size="14" class="mr-1">
            {{ showBinaries ? 'mdi-chevron-down' : 'mdi-chevron-right' }}
          </v-icon>
          {{ t('distribution.optionalBinaries') }}
        </button>
      </div>

      <v-expand-transition>
        <v-card v-if="showBinaries" class="dist-card">
          <div class="field-group">
            <label class="field-label">
              {{ t('distribution.cliBinary') }}
              <HelpTooltip text-key="help.ydbCli" :doc-link="HELP_LINKS.ydbCli" />
            </label>
            <v-text-field
              v-model="cliBinary"
              :placeholder="t('distribution.cliBinaryPlaceholder')"
              density="compact"
              variant="outlined"
              hide-details="auto"
              class="path-input"
            />
          </div>

          <div class="field-group">
            <label class="field-label">
              {{ t('distribution.serverBinary') }}
              <HelpTooltip text-key="help.ydbd" :doc-link="HELP_LINKS.ydbd" />
            </label>
            <v-text-field
              v-model="binaryPath"
              :placeholder="t('distribution.serverBinaryPlaceholder')"
              density="compact"
              variant="outlined"
              hide-details="auto"
              class="path-input"
            />
          </div>

          <div class="field-group">
            <label class="field-label">
              {{ t('distribution.opsBinary') }}
              <HelpTooltip text-key="help.ydbops" :doc-link="HELP_LINKS.ydbops" />
            </label>
            <v-text-field
              v-model="opsBinary"
              :placeholder="t('distribution.opsBinaryPlaceholder')"
              density="compact"
              variant="outlined"
              hide-details="auto"
              class="path-input"
            />
          </div>

          <div class="field-group">
            <label class="field-label">
              {{ t('distribution.toolBinary') }}
              <HelpTooltip text-key="help.dstool" :doc-link="HELP_LINKS.dstool" />
            </label>
            <v-text-field
              v-model="dstoolBinary"
              :placeholder="t('distribution.toolBinaryPlaceholder')"
              density="compact"
              variant="outlined"
              hide-details="auto"
              class="path-input"
            />
          </div>
        </v-card>
      </v-expand-transition>
    </section>
  </div>
</template>

<style scoped>
.dist-shell {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.dist-head { padding: 4px 2px 0; margin-bottom: 16px; }
.dist-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.dist-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
}

.dist-section { display: flex; flex-direction: column; gap: 12px; }

.offline-warning {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: 8px;
  font-size: 12.5px;
  color: #92400e;
}

.checking-hint {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  font-size: 12.5px;
  color: #1e40af;
}

.source-toggle {
  display: flex;
  gap: 0;
  background: #f1f5f9;
  border-radius: 12px;
  padding: 4px;
  width: fit-content;
}

.source-option {
  display: flex;
  align-items: center;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
  transition: all .18s ease;
  user-select: none;
}

.source-option input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.source-label {
  font-size: 13.5px;
  font-weight: 500;
  color: #64748b;
  transition: color .18s ease;
}

.source-option.is-active {
  background: #fff;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}

.source-option.is-active .source-label {
  color: #0f172a;
  font-weight: 600;
}

.source-option.is-disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
.source-option.is-disabled:hover {
  background: transparent;
}

.dist-card {
  padding: 20px;
  background: rgb(var(--v-theme-surface));
}
  
.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.field-group--last {
  margin-bottom: 0;
}

.field-label {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
}

.required-mark {
  color: #dc2626;
  margin-left: 2px;
}

.path-input {
  max-width: 600px;
}

.version-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.version-select {
  max-width: 400px;
}

.recommended-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  margin-left: 8px;
  font-size: 11px;
  font-weight: 500;
  color: #15803d;
  background: #dcfce7;
  border-radius: 999px;
}

.validation-msg {
  display: flex;
  align-items: center;
  font-size: 12.5px;
  padding: 6px 10px;
  border-radius: 6px;
  margin-top: 4px;
}

.validation-msg.is-valid {
  color: #15803d;
  background: #f0fdf4;
}

.validation-msg.is-error {
  color: #dc2626;
  background: #fef2f2;
}

.no-versions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px 12px;
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: 8px;
  font-size: 12.5px;
  color: #92400e;
  margin-top: 8px;
}

.binaries-toggle {
  margin-bottom: 4px;
}

.binaries-toggle-btn {
  display: inline-flex;
  align-items: center;
  background: none;
  border: none;
  padding: 0;
  font-size: 12.5px;
  color: #64748b;
  cursor: pointer;
  font-family: inherit;
  transition: color .15s ease;
}
.binaries-toggle-btn:hover { color: #1f6dd9; }
</style>
