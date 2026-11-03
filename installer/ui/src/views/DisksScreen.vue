<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useHostsStore } from '@/stores/hosts'
import { useDisksStore } from '@/stores/disks'
import { useTopologyStore } from '@/stores/topology'
import { minDisksToSelect } from '@/config/topologies'
import { systemRequirementsUrl } from '@/config/helpLinks'
import { useDiskChecks } from '@/composables/useDiskChecks'
import DiskCard from '@/components/disks/DiskCard.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'
import type { DiskInfo, DiskEntryMode } from '@/types/disks'
import type { SSHConfig } from '@/types/inventory'

const session = useSessionStore()
const hostsStore = useHostsStore()
const disksStore = useDisksStore()
const topologyStore = useTopologyStore()
const { t, locale } = useI18n()

// Docs URL follows the active UI locale so the link stays in the user's language.
const systemRequirementsHref = computed(() =>
  systemRequirementsUrl(locale.value === 'en' ? 'en' : 'ru'),
)

const { isChecking, checkError, checkDisks, clearResults, isValidating, validationError, validationResults, validateDisks, clearValidation } = useDiskChecks()

const selectedDisksSet = ref<Set<string>>(new Set())

const isAutoMode = computed({
  get: () => disksStore.entryMode === 'auto',
  set: (v) => { handleEntryModeChange(v ? 'auto' : 'manual') },
})

const dcNames = computed(() => ['DC-1', 'DC-2', 'DC-3'] as const)

const topologyLabel = computed(() => {
  const labels: Record<string, string> = {
    'mirror-3-dc': 'mirror-3-dc',
    'block-4-2': 'block-4-2',
    'bridge-2-dc': 'bridge-2-dc',
    'none': 'none',
  }
  return labels[topologyStore.selected || 'none']
})

const allHosts = computed(() => {
  const hosts: string[] = []
  for (const dc of dcNames.value) {
    hosts.push(...hostsStore.dcHosts[dc].filter(h => h.trim() !== ''))
  }
  return hosts
})

const sshConfig = computed((): SSHConfig | null => {
  if (!hostsStore.ssh) return null
  return hostsStore.ssh
})

const canCheck = computed(() => {
  return allHosts.value.length > 0 && sshConfig.value !== null
})

const referenceHost = computed(() => {
  return allHosts.value.length > 0 ? allHosts.value[0] : null
})

const referenceDisks = computed((): DiskInfo[] => {
  if (!referenceHost.value) return []
  return disksStore.getHostDisks(referenceHost.value)
})

const isDiskSelected = (disk: DiskInfo) => {
  return selectedDisksSet.value.has(disk.path)
}

const requiredDisks = computed(() =>
  minDisksToSelect(topologyStore.selected, allHosts.value.length),
)

const minDisksHintKey = computed(() => {
  if (requiredDisks.value <= 1) return 'disks.selectAtLeastOne'
  return disksStore.entryMode === 'manual'
    ? 'disks.selectAtLeastThreeManual'
    : 'disks.selectAtLeastThree'
})

const isValid = computed(() => {
  if (disksStore.entryMode === 'manual') {
    return disksStore.selectedDisks.length >= requiredDisks.value && !disksStore.manualInputError
  }
  return selectedDisksSet.value.size >= requiredDisks.value
})

const consistencyResult = computed(() => {
  return disksStore.consistencyResult
})

const allDisksValid = computed(() => {
  if (validationResults.value.length === 0) return false
  return validationResults.value.every(r => r.valid)
})

const availableDisks = computed(() => {
  return referenceDisks.value.filter(d => !d.isSystemDisk)
})

const systemDisks = computed(() => {
  return referenceDisks.value.filter(d => d.isSystemDisk)
})

async function handleCheckDisks() {
  if (!canCheck.value || !sshConfig.value) return

  await checkDisks(allHosts.value, sshConfig.value)

  selectedDisksSet.value = new Set(
    disksStore.selectedDisks.map(d => d.path)
  )
}

function handleToggleDisk(disk: DiskInfo) {
  if (disk.isSystemDisk) return

  if (selectedDisksSet.value.has(disk.path)) {
    selectedDisksSet.value.delete(disk.path)
    disksStore.toggleDisk(disk.path, '')

    renumberDisks()
  } else {
    selectedDisksSet.value.add(disk.path)

    renumberDisks()
  }
}

function renumberDisks() {
  const diskPaths = Array.from(selectedDisksSet.value)

  const currentSelections = [...disksStore.selectedDisks]

  currentSelections.forEach(disk => {
    disksStore.toggleDisk(disk.path, '')
  })

  diskPaths.forEach((path, index) => {
    const label = `ydb_disk_${index + 1}`
    disksStore.toggleDisk(path, label)
  })
}

function handleEntryModeChange(mode: DiskEntryMode) {
  disksStore.setEntryMode(mode)
  selectedDisksSet.value = new Set(
    disksStore.selectedDisks.map(d => d.path)
  )
}

watch(() => disksStore.entryMode, async (mode) => {
  if (mode === 'auto' && referenceDisks.value.length === 0 && canCheck.value && sshConfig.value) {
    await handleCheckDisks()
  }
})

function handleManualInputChange(text: string) {
  disksStore.setManualDiskInput(text)
  clearValidation()
}

async function handleValidateDisks() {
  if (!canCheck.value || !sshConfig.value) return
  const paths = disksStore.selectedDisks.map(d => d.path)
  if (paths.length === 0) return
  await validateDisks(allHosts.value, sshConfig.value, paths)
}

watch(allHosts, (newHosts, oldHosts) => {
  if (JSON.stringify(newHosts) !== JSON.stringify(oldHosts)) {
    clearResults()
    clearValidation()
    selectedDisksSet.value.clear()
  }
}, { deep: true })

watch(isValid, (valid) => {
  session.setCanGoNext(valid)
}, { immediate: true })

// Reactive hydration rule: never use one-shot onMounted here.
// disksStore.selectedDisks and hostsStore.ssh/dcHosts are populated
// asynchronously by useInventorySync() (wired in App.vue's onMounted).
// Child onMounted fires BEFORE parent's, so a one-shot read would always
// see empty stores on fresh page load — selection mirror stays empty and
// disk auto-discovery never fires. Guarded watchers with immediate: true
// react the moment data becomes available.

let disksSelectionHydrated = false
watch(
  () => disksStore.selectedDisks,
  (selected) => {
    if (disksSelectionHydrated || selected.length === 0) return
    disksSelectionHydrated = true
    selectedDisksSet.value = new Set(selected.map(d => d.path))
  },
  { immediate: true, deep: true },
)

let diskAutoDiscoveryAttempted = false
watch(
  canCheck,
  async (canCheckNow) => {
    if (diskAutoDiscoveryAttempted || !canCheckNow) return
    if (disksStore.entryMode !== 'auto' || referenceDisks.value.length > 0) return
    diskAutoDiscoveryAttempted = true
    await handleCheckDisks()
  },
  { immediate: true },
)
</script>

<template>
  <div class="disks-shell">
    <header class="disks-head">
      <ScreenEyebrow />
      <h1 class="disks-title">{{ t('disks.heading') }}</h1>
      <p class="disks-sub">{{ t('disks.topologyLabel', { topology: topologyLabel }) }}</p>
      <a
        :href="systemRequirementsHref"
        target="_blank"
        rel="noopener"
        class="disks-doc-link"
      >
        <v-icon size="14" class="disks-doc-link__icon">mdi-open-in-new</v-icon>
        {{ t('disks.systemRequirementsLink') }}
      </a>
    </header>

    <section class="intro-section">
      <div class="section-head">
        <span class="section-eyebrow eyebrow">{{ t('disks.entryMode') }}</span>
      </div>
      <div class="source-toggle">
        <label class="source-option" :class="{ 'is-active': isAutoMode }">
          <input type="radio" v-model="isAutoMode" :value="true" />
          <v-icon size="16" class="source-icon">mdi-magnify</v-icon>
          <span class="source-label">{{ t('disks.entryModeAuto') }}</span>
        </label>
        <label class="source-option" :class="{ 'is-active': !isAutoMode }">
          <input type="radio" v-model="isAutoMode" :value="false" />
          <v-icon size="16" class="source-icon">mdi-pencil</v-icon>
          <span class="source-label">{{ t('disks.entryModeManual') }}</span>
        </label>
      </div>
    </section>

    <section v-if="disksStore.entryMode === 'auto'" class="intro-section">
      <div class="section-head">
        <span class="section-eyebrow eyebrow">{{ t('disks.discovery') }}</span>
      </div>

      <v-alert
        v-if="consistencyResult && !consistencyResult.isConsistent"
        type="warning"
        class="mb-4"
      >
        <div class="section-label mb-2">{{ t('disks.consistencyWarning') }}</div>
        <div
          v-for="(issue, index) in consistencyResult.issues"
          :key="index"
          class="mb-2"
        >
          <div class="issue-message">
            <strong>{{ issue.type }}:</strong> {{ issue.message }}
          </div>
          <div class="issue-hosts">
            <v-icon size="14">mdi-server</v-icon>
            <span>{{ issue.hosts.join(', ') }}</span>
          </div>
        </div>
      </v-alert>

      <v-alert
        v-if="checkError"
        type="error"
        class="mb-4"
      >
        {{ checkError }}
      </v-alert>

      <v-card v-if="isChecking" class="pa-5">
        <div class="loading-state">
          <v-progress-circular indeterminate color="primary" size="48" />
          <div class="loading-text">{{ t('disks.discovering') }}</div>
        </div>
      </v-card>

      <v-card v-else-if="referenceHost && referenceDisks.length > 0" class="pa-5">
        <div v-if="availableDisks.length > 0" class="disk-table-section">
          <div class="disk-section-head">
            <span class="disk-table-section-title">{{ t('disks.free') }} ({{ availableDisks.length }})</span>
            <v-btn
              v-if="!isChecking"
              variant="outlined"
              size="small"
              prepend-icon="mdi-refresh"
              :disabled="!canCheck"
              @click="handleCheckDisks"
              class="recheck-btn"
            >
              {{ t('disks.recheckDisks') }}
            </v-btn>
          </div>
          <div class="disk-table-list">
            <DiskCard
              v-for="disk in availableDisks"
              :key="disk.path"
              :disk="disk"
              :selected="isDiskSelected(disk)"
              :disabled="isChecking"
              @toggle="handleToggleDisk"
            />
          </div>
        </div>

        <div v-if="systemDisks.length > 0" class="disk-table-section disk-table-section--system">
          <div class="disk-table-section-title">
            {{ t('disks.hasFS') }} ({{ systemDisks.length }})
          </div>
          <div class="disk-table-list">
            <DiskCard
              v-for="disk in systemDisks"
              :key="disk.path"
              :disk="disk"
              :selected="false"
              :disabled="true"
              @toggle="handleToggleDisk"
            />
          </div>
        </div>
      </v-card>

      <v-card v-else-if="allHosts.length === 0" class="pa-5">
        <div class="no-hosts-message">
          <v-icon size="48" color="#cbd5e1">mdi-server-off</v-icon>
          <div class="no-hosts-text">
            <div class="no-hosts-title">{{ t('disks.noHosts') }}</div>
            <div class="no-hosts-sub">{{ t('disks.addHostsFirst') }}</div>
          </div>
        </div>
      </v-card>

      <v-card v-else class="pa-5">
        <div class="no-disks-message">
          <v-icon size="48" color="#cbd5e1">mdi-harddisk-remove</v-icon>
          <div class="no-disks-text">
            <div class="no-disks-title">{{ t('disks.noDisks') }}</div>
            <div class="no-disks-sub">{{ t('disks.autoDiscovering') }}</div>
          </div>
        </div>
      </v-card>

      <div v-if="!isValid" class="validation-message">
        <v-icon size="18" color="#f59e0b">mdi-alert-circle</v-icon>
        <span>{{ t(minDisksHintKey) }}</span>
      </div>
    </section>

    <section v-else class="intro-section">
      <div class="section-head">
        <span class="section-eyebrow eyebrow">{{ t('disks.entryModeManual') }}</span>
      </div>

      <v-card class="pa-5">
        <p class="manual-desc">{{ t('disks.manualEntryDesc') }}</p>

        <v-textarea
          :model-value="disksStore.manualDiskInput"
          :placeholder="t('disks.manualEntryPlaceholder')"
          :error="!!disksStore.manualInputError"
          :error-messages="disksStore.manualInputError ? [disksStore.manualInputError] : []"
          variant="outlined"
          rows="6"
          class="manual-textarea"
          hide-details="auto"
          @update:model-value="handleManualInputChange"
        />

        <div class="manual-hint">
          <v-icon size="14" color="#94a3b8">mdi-information-outline</v-icon>
          <span>{{ t('disks.manualEntryHint') }}</span>
        </div>

        <div v-if="disksStore.selectedDisks.length > 0" class="manual-preview">
          <div class="manual-preview-title">
            {{ t('disks.manualPreview', { count: disksStore.selectedDisks.length }) }}
          </div>
          <div class="manual-preview-list">
            <div
              v-for="disk in disksStore.selectedDisks"
              :key="disk.path"
              class="manual-preview-item"
            >
              <v-icon size="16" color="#3b82f6">mdi-harddisk</v-icon>
              <span class="manual-preview-path">{{ disk.path }}</span>
              <span class="manual-preview-label">{{ disk.label }}</span>
            </div>
          </div>
        </div>

        <div v-if="disksStore.selectedDisks.length > 0 && !disksStore.manualInputError" class="manual-validate">
          <v-btn
            variant="flat"
            size="small"
            color="primary"
            prepend-icon="mdi-check-circle"
            :loading="isValidating"
            :disabled="!canCheck"
            @click="handleValidateDisks"
            class="validate-btn"
          >
            {{ t('disks.validateDisks') }}
          </v-btn>
        </div>
      </v-card>

      <div v-if="validationResults.length > 0 || validationError || isValidating" class="validation-panel">
        <div v-if="isValidating" class="validation-panel__loading">
          <v-progress-circular indeterminate color="primary" size="20" width="2" />
          <span>{{ t('disks.validating') }}</span>
        </div>

        <div v-else-if="validationError" class="validation-panel__error">
          <v-icon size="16" color="#ef4444">mdi-alert-circle</v-icon>
          <span>{{ validationError }}</span>
        </div>

        <template v-else>
          <div
            class="validation-summary"
            :class="allDisksValid ? 'validation-summary--ok' : 'validation-summary--fail'"
          >
            <v-icon size="16" :color="allDisksValid ? '#10b981' : '#ef4444'">
              {{ allDisksValid ? 'mdi-check-circle' : 'mdi-alert-circle' }}
            </v-icon>
            <span>{{ allDisksValid ? t('disks.validationOk') : t('disks.validationErrors') }}</span>
          </div>

          <div class="validation-list">
            <div
              v-for="result in validationResults"
              :key="result.host"
              class="validation-row"
            >
              <div class="validation-row__host">
                <v-icon size="14" color="#94a3b8">mdi-server</v-icon>
                <span>{{ result.host }}</span>
                <v-icon
                  v-if="!result.error"
                  size="14"
                  :color="result.valid ? '#10b981' : '#ef4444'"
                  class="validation-row__badge"
                >
                  {{ result.valid ? 'mdi-check' : 'mdi-close' }}
                </v-icon>
              </div>
              <div class="validation-row__disks">
                <span
                  v-for="disk in result.disks"
                  :key="disk.path"
                  class="validation-chip"
                  :class="disk.found ? 'validation-chip--ok' : 'validation-chip--fail'"
                >
                  <v-icon size="12" :color="disk.found ? '#10b981' : '#ef4444'">
                    {{ disk.found ? 'mdi-check' : 'mdi-close' }}
                  </v-icon>
                  {{ disk.path }}
                </span>
              </div>
            </div>
          </div>
        </template>
      </div>

      <div v-if="!isValid && disksStore.manualDiskInput.length > 0 && !disksStore.manualInputError" class="validation-message">
        <v-icon size="18" color="#f59e0b">mdi-alert-circle</v-icon>
        <span>{{ t(minDisksHintKey) }}</span>
      </div>
    </section>
  </div>
</template>

<style scoped>
.disks-head { margin-bottom: 28px; }
.disks-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.disks-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
}
.disks-doc-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  font-size: 13px;
  font-weight: 500;
  color: #1f6dd9;
  text-decoration: none;
  transition: color 0.15s ease;
}
.disks-doc-link:hover {
  color: #1a5cb3;
  text-decoration: underline;
}
.disks-doc-link__icon {
  color: #1f6dd9;
}

.intro-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;
}

.section-head {
  padding-left: 4px;
}

.section-eyebrow.eyebrow {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #94a3b8;
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
  gap: 6px;
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

.source-icon {
  color: #94a3b8;
  transition: color .18s ease;
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

.source-option.is-active .source-icon {
  color: #1f6dd9;
}

.source-option.is-active .source-label {
  color: #0f172a;
  font-weight: 600;
}

.disk-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  gap: 12px;
}

.recheck-btn {
  flex-shrink: 0;
  height: 28px !important;
  min-height: 28px !important;
}

.section-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #94a3b8;
  margin-bottom: 14px;
}

.issue-message {
  font-size: 14px;
  margin-bottom: 4px;
}
.issue-hosts {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #64748b;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  gap: 16px;
}

.loading-text {
  font-size: 14px;
  color: #64748b;
}

.disk-table-section {
  margin-bottom: 20px;
}

.disk-table-section:last-child {
  margin-bottom: 0;
}

.disk-table-section--system {
  opacity: 0.7;
}

.disk-table-section-title {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: #64748b;
  margin-bottom: 12px;
}

.disk-table-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.no-hosts-message {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
}
.no-hosts-text {
  flex: 1;
}
.no-hosts-title {
  font-weight: 600;
  font-size: 16px;
  color: #0f172a;
  margin-bottom: 4px;
}
.no-hosts-sub {
  font-size: 14px;
  color: #64748b;
}

.no-disks-message {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
}
.no-disks-text {
  flex: 1;
}
.no-disks-title {
  font-weight: 600;
  font-size: 16px;
  color: #0f172a;
  margin-bottom: 4px;
}
.no-disks-sub {
  font-size: 14px;
  color: #64748b;
}

.validation-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #fffbeb;
  border: 1.5px solid #fcd34d;
  border-radius: 8px;
  color: #92400e;
  font-size: 14px;
}

.manual-desc {
  font-size: 14px;
  color: #64748b;
  line-height: 1.5;
  margin: 0 0 16px;
}

.manual-textarea {
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
  font-size: 13px;
}

.manual-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: 13px;
  color: #94a3b8;
}

.manual-preview {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f1f5f9;
}

.manual-preview-title {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: #64748b;
  margin-bottom: 10px;
}

.manual-preview-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.manual-preview-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
}

.manual-preview-path {
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
  color: #0f172a;
  font-weight: 500;
}

.manual-preview-label {
  margin-left: auto;
  color: #64748b;
  font-size: 12px;
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
}

.manual-validate {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f1f5f9;
}

.validate-btn {
  flex-shrink: 0;
  height: 28px !important;
  min-height: 28px !important;
}

.validation-panel {
  margin-top: 12px;
  padding: 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
}

.validation-panel__loading {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: #64748b;
}

.validation-panel__error {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #ef4444;
}

.validation-summary {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  padding: 8px 12px;
  border-radius: 8px;
  margin-bottom: 12px;
}

.validation-summary--ok {
  background: #ecfdf5;
  color: #047857;
}

.validation-summary--fail {
  background: #fef2f2;
  color: #b91c1c;
}

.validation-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.validation-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.validation-row__host {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #0f172a;
}

.validation-row__badge {
  margin-left: auto;
}

.validation-row__disks {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding-left: 20px;
}

.validation-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
  font-weight: 500;
}

.validation-chip--ok {
  background: #ecfdf5;
  color: #047857;
  border: 1px solid #a7f3d0;
}

.validation-chip--fail {
  background: #fef2f2;
  color: #b91c1c;
  border: 1px solid #fecaca;
}
</style>
