<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useCoresInfo } from '@/composables/useCoresInfo'
import CoresInfoCard from '@/components/common/CoresInfoCard.vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useDatabasesStore, type ConfigVariant, type DatabaseEntryUI } from '@/stores/databases'
import { useHostsStore } from '@/stores/hosts'
import { useTopologyStore } from '@/stores/topology'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useEmStore } from '@/stores/em'
import { useRuntimeStore } from '@/stores/runtime'
import { HELP_LINKS } from '@/config/helpLinks'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const session = useSessionStore()
const db = useDatabasesStore()
const hosts = useHostsStore()
const ydbConfig = useYdbConfigStore()
const emStore = useEmStore()
const runtime = useRuntimeStore()
const { t } = useI18n()
const topologyStore = useTopologyStore()

const hostCount = computed(() => {
  const allHosts = [
    ...hosts.dcHosts['DC-1'],
    ...hosts.dcHosts['DC-2'],
    ...hosts.dcHosts['DC-3'],
  ]
  return Math.max(allHosts.filter((h: string) => h.trim() !== '').length, 1)
})

const allHostList = computed(() => {
  const allHosts = [
    ...hosts.dcHosts['DC-1'],
    ...hosts.dcHosts['DC-2'],
    ...hosts.dcHosts['DC-3'],
  ]
  return allHosts.filter((h: string) => h.trim() !== '')
})

const coresInfo = useCoresInfo()
const { cpuStats, budget, suggestedMax } = coresInfo

const topologyLabel = computed(() => {
  const labels: Record<string, string> = {
    'mirror-3-dc': 'mirror-3-dc',
    'block-4-2': 'block-4-2',
    'bridge-2-dc': 'bridge-2-dc',
    'none': 'none',
  }
  return labels[topologyStore.selected || 'none']
})

const coresDynamic = computed({
  get: () => db.coresDynamic,
  set: (v: number | string | null | undefined) => {
    const n = typeof v === 'number' ? v : parseInt(String(v ?? ''), 10)
    db.coresDynamic = isNaN(n) ? 0 : Math.max(0, n)
  },
})

const coresExceeded = computed(() => {
  if (!coresInfo.cpuStats.value.available) return false
  return (
    coresInfo.budget.value.utilizationStatus === 'critical' &&
    coresInfo.budget.value.remainingPerHost <= 0
  )
})

function isDuplicateName(name: string): boolean {
  const trimmed = name.trim()
  if (!trimmed) return false
  return db.duplicateNames.includes(trimmed)
}

function setVariant(v: ConfigVariant) {
  db.setConfigVariant(v)
}

const showAdvanced = ref<Record<number, boolean>>({})

function toggleAdvanced(id: number) {
  showAdvanced.value[id] = !showAdvanced.value[id]
}

function updateEntryName(index: number, val: string) {
  db.updateEntry(index, 'name', val)
}

function updateEntryGroups(index: number, val: number | string) {
  const n = typeof val === 'number' ? val : parseInt(String(val ?? ''), 10)
  db.updateEntry(index, 'groups', isNaN(n) ? 0 : Math.max(1, n))
}

function updateEntryHostMode(index: number, mode: DatabaseEntryUI['hostMode']) {
  db.updateEntry(index, 'hostMode', mode)
  if (mode === 'all') {
    db.updateEntry(index, 'hosts', [])
  }
}

function toggleHost(index: number, host: string) {
  const entry = db.entries[index]
  if (!entry) return
  const hostsArr = [...entry.hosts]
  const idx = hostsArr.indexOf(host)
  if (idx >= 0) {
    hostsArr.splice(idx, 1)
  } else {
    hostsArr.push(host)
  }
  db.updateEntry(index, 'hosts', hostsArr)
}

function updateEntryAffinity(index: number, val: string) {
  db.updateEntry(index, 'affinity', val)
}

function addCustomHost(index: number, val: string) {
  if (!val.trim()) return
  const entry = db.entries[index]
  if (!entry) return
  const hostsArr = [...entry.hosts, val.trim()]
  db.updateEntry(index, 'hosts', hostsArr)
}

const customHostInput = ref<Record<number, string>>({})

function handleAddCustomHost(index: number) {
  const val = customHostInput.value[index] || ''
  addCustomHost(index, val)
  customHostInput.value[index] = ''
}

watch(() => db.configVariant, (variant) => {
  session.setShowEmStep(variant === 'single-plus-em')
}, { immediate: true })

watch(
  () => [runtime.showEmScenarios, db.configVariant] as const,
  ([showEm, variant]) => {
    if (variant === 'single-plus-em' && !showEm) db.setConfigVariant('single')
  },
  { immediate: true },
)

function firstYdbHost(): string {
  return allHostList.value[0] ?? ''
}

function normalizeDomainRoot(domain: string): string {
  const d = domain.trim().replace(/^\/+|\/+$/g, '')
  return d ? `/${d}` : '/Root'
}

watch(
  () => [
    db.configVariant,
    db.entries[1]?.name,
    ydbConfig.domain,
    allHostList.value.join('\0'),
  ] as const,
  () => {
    if (db.configVariant !== 'single-plus-em') return

    const emName = (db.entries[1]?.name ?? '').trim()
    const root = normalizeDomainRoot(ydbConfig.domain || 'Root')
    const host = firstYdbHost()

    emStore.useDynamicConfig = true
    if (emName) emStore.setDbConnectionDB(emName)
    emStore.setDbConnectionDBRoot(root)
    if (host) {
      emStore.setDbConnectionEndpoint(`grpcs://${host}:2135`)
    }
    const hasEmHost = emStore.hosts.some(h => h.fqdn.trim() !== '')
    if (!hasEmHost && host) {
      if (emStore.hosts.length === 0) {
        emStore.addEmptyHost()
      }
      emStore.updateHost(0, host)
    }
  },
  { immediate: true },
)

watch(() => [db.canProceed, coresExceeded.value], () => {
  session.setCanGoNext(db.canProceed && !coresExceeded.value)
}, { immediate: true })
</script>

<template>
  <div class="config-shell">
    <header class="config-head">
      <ScreenEyebrow />
      <h1 class="config-title">{{ t('databases.heading') }}</h1>
      <p class="config-sub">{{ t('databases.topologyLabel', { topology: topologyLabel }) }}</p>
    </header>

    <section class="config-section">
      <v-card class="config-card">
        <div class="card-header">
          <span class="section-eyebrow">{{ t('databases.variantLabel') }}</span>
        </div>

        <div class="radio-row variant-row">
          <button
            type="button"
            class="radio-option variant-option"
            :class="{ 'is-active': db.configVariant === 'single' }"
            @click="setVariant('single')"
          >
            <span class="radio-label">{{ t('databases.variantSingle') }}</span>
          </button>
          <button
            type="button"
            class="radio-option variant-option"
            :class="{ 'is-active': db.configVariant === 'multiple' }"
            @click="setVariant('multiple')"
          >
            <span class="radio-label">{{ t('databases.variantMultiple') }}</span>
          </button>
          <button
            v-if="runtime.showEmScenarios"
            type="button"
            class="radio-option variant-option"
            :class="{ 'is-active': db.configVariant === 'single-plus-em' }"
            @click="setVariant('single-plus-em')"
          >
            <span class="radio-label">{{ t('databases.variantSinglePlusEm') }}</span>
          </button>
        </div>
      </v-card>
    </section>

    <section class="config-section">
      <v-card class="config-card">
        <div class="card-header">
          <span class="section-eyebrow">{{ t('databases.commonParams') }}</span>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.coresDynamic') }}
            <HelpTooltip text-key="help.coresDynamic" :doc-link="HELP_LINKS.coresDynamic" />
            <span class="required-mark">*</span>
          </label>
          <div class="slider-row">
            <v-slider
              v-model="coresDynamic"
              :min="0"
              :max="suggestedMax"
              :step="1"
              thumb-label
              density="compact"
              class="cores-slider"
            />
            <v-text-field
              v-model="coresDynamic"
              density="compact"
              variant="outlined"
              hide-details="auto"
              type="number"
              :min="0"
              :max="suggestedMax"
              class="cores-input"
            />
          </div>
          <div v-if="coresDynamic <= 0" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('databases.coresRequired') }}
          </div>
        </div>
      </v-card>
    </section>

    <section v-if="db.configVariant === 'single'" class="config-section">
      <v-card class="config-card">
        <div class="card-header db-card-header">
          <span class="section-eyebrow">{{ t('databases.databaseLabel') }}</span>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.dbName') }}
            <HelpTooltip text-key="help.dbName" :doc-link="HELP_LINKS.dbName" />
            <span class="required-mark">*</span>
          </label>
          <v-text-field
            :model-value="db.entries[0].name"
            @update:model-value="(v: string) => updateEntryName(0, v)"
            :placeholder="t('databases.dbNamePlaceholder')"
            density="compact"
            variant="outlined"
            hide-details="auto"
            class="text-input"
          />
          <div v-if="!db.entries[0].name.trim()" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('databases.dbNameRequired') }}
          </div>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.groups') }}
            <HelpTooltip text-key="help.dbGroups" :doc-link="HELP_LINKS.dbGroups" />
          </label>
          <v-text-field
            :model-value="db.entries[0].groups"
            @update:model-value="(v: string) => updateEntryGroups(0, v)"
            density="compact"
            variant="outlined"
            hide-details="auto"
            type="number"
            :min="1"
            class="text-input"
          />
        </div>

        <div class="field-group">
          <label class="field-label">{{ t('databases.hosts') }}</label>
          <div class="radio-row">
            <button
              type="button"
              class="radio-option"
              :class="{ 'is-active': db.entries[0].hostMode === 'all' }"
              @click="updateEntryHostMode(0, 'all')"
            >
              <span class="radio-label">{{ t('databases.hostsAll') }}</span>
            </button>
            <button
              type="button"
              class="radio-option is-disabled"
              disabled
              :title="t('common.soon')"
            >
              <span class="radio-label">{{ t('databases.hostsSelect') }} ({{ t('common.soon') }})</span>
            </button>
          </div>
          <div v-if="db.entries[0].hostMode === 'all'" class="field-hint">
            {{ t('databases.allHostsHint', { count: hostCount }) }}
          </div>
        </div>

        <div class="field-group field-group--last">
          <div class="advanced-toggle">
            <button
              type="button"
              class="advanced-toggle-btn"
              @click="toggleAdvanced(db.entries[0].id)"
            >
              <v-icon size="14" class="mr-1">
                {{ showAdvanced[db.entries[0].id] ? 'mdi-chevron-down' : 'mdi-chevron-right' }}
              </v-icon>
              {{ t('databases.advancedSettings') }}
            </button>
          </div>
          <v-expand-transition>
            <div v-if="showAdvanced[db.entries[0].id]">
              <div class="field-group">
                <label class="field-label">
                  {{ t('databases.affinity') }}
                  <HelpTooltip text-key="help.affinityDynamic" :doc-link="HELP_LINKS.affinityDynamic" />
                </label>
                <v-text-field
                  :model-value="db.entries[0].affinity"
                  @update:model-value="(v: string) => updateEntryAffinity(0, v)"
                  :placeholder="t('databases.affinityPlaceholder')"
                  density="compact"
                  disabled
                  variant="outlined"
                  hide-details="auto"
                  class="text-input"
                />
                <div class="field-hint">{{ t('databases.affinityDesc') }}</div>
              </div>
            </div>
          </v-expand-transition>
        </div>
      </v-card>
    </section>

    <section v-else-if="db.configVariant === 'single-plus-em'" class="config-section">
      <div class="multiple-mode-hint">
        <v-icon icon="mdi-information-outline" size="15" class="mr-1" />
        {{ t('databases.singlePlusEmHint') }}
      </div>

      <v-card v-if="db.entries[0]" class="config-card">
        <div class="card-header db-card-header">
          <span class="section-eyebrow">{{ t('databases.userDbLabel') }}</span>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.dbName') }}
            <HelpTooltip text-key="help.dbName" :doc-link="HELP_LINKS.dbName" />
            <span class="required-mark">*</span>
          </label>
          <v-text-field
            :model-value="db.entries[0].name"
            @update:model-value="(v: string) => updateEntryName(0, v)"
            :placeholder="t('databases.dbNamePlaceholder')"
            density="compact"
            variant="outlined"
            hide-details="auto"
            class="text-input"
          />
          <div v-if="!db.entries[0].name.trim()" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('databases.dbNameRequired') }}
          </div>
          <div v-else-if="isDuplicateName(db.entries[0].name)" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('databases.dbNameDuplicate') }}
          </div>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.groups') }}
            <HelpTooltip text-key="help.dbGroups" :doc-link="HELP_LINKS.dbGroups" />
          </label>
          <v-text-field
            :model-value="db.entries[0].groups"
            @update:model-value="(v: string) => updateEntryGroups(0, v)"
            density="compact"
            variant="outlined"
            hide-details="auto"
            type="number"
            :min="1"
            class="text-input"
          />
        </div>

        <div class="field-group field-group--last">
          <label class="field-label">{{ t('databases.hosts') }}</label>
          <div class="radio-row">
            <button
              type="button"
              class="radio-option"
              :class="{ 'is-active': db.entries[0].hostMode === 'all' }"
              @click="updateEntryHostMode(0, 'all')"
            >
              <span class="radio-label">{{ t('databases.hostsAll') }}</span>
            </button>
          </div>
          <div class="field-hint">
            {{ t('databases.allHostsHint', { count: hostCount }) }}
          </div>
        </div>
      </v-card>

      <v-card v-if="db.entries[1]" class="config-card">
        <div class="card-header db-card-header">
          <span class="section-eyebrow">{{ t('databases.emDbLabel') }}</span>
          <span class="em-badge">{{ t('databases.emDbBadge') }}</span>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.dbName') }}
            <HelpTooltip text-key="help.dbName" :doc-link="HELP_LINKS.dbName" />
            <span class="required-mark">*</span>
          </label>
          <v-text-field
            :model-value="db.entries[1].name"
            disabled
            density="compact"
            variant="outlined"
            hide-details="auto"
            class="text-input"
          />
          <div v-if="isDuplicateName(db.entries[1].name)" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('databases.dbNameDuplicate') }}
          </div>
          <div class="field-hint">{{ t('databases.emDbHint') }}</div>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.groups') }}
            <HelpTooltip text-key="help.dbGroups" :doc-link="HELP_LINKS.dbGroups" />
          </label>
          <v-text-field
            :model-value="db.entries[1].groups"
            @update:model-value="(v: string) => updateEntryGroups(1, v)"
            density="compact"
            variant="outlined"
            hide-details="auto"
            type="number"
            :min="1"
            class="text-input"
          />
        </div>

        <div class="field-group field-group--last">
          <label class="field-label">{{ t('databases.hosts') }}</label>
          <div class="radio-row">
            <button
              type="button"
              class="radio-option"
              :class="{ 'is-active': db.entries[1].hostMode === 'all' }"
              @click="updateEntryHostMode(1, 'all')"
            >
              <span class="radio-label">{{ t('databases.hostsAll') }}</span>
            </button>
          </div>
          <div class="field-hint">
            {{ t('databases.allHostsHint', { count: hostCount }) }}
          </div>
        </div>
      </v-card>
    </section>

    <section v-else-if="db.configVariant === 'multiple'" class="config-section">
      <div class="multiple-mode-hint">
        <v-icon icon="mdi-information-outline" size="15" class="mr-1" />
        {{ t('databases.multipleModeHint') }}
      </div>

      <v-card
        v-for="(entry, idx) in db.entries"
        :key="entry.id"
        class="config-card"
      >
        <div class="card-header db-card-header">
          <span class="section-eyebrow">
            {{ t('databases.databaseLabel') }} #{{ idx + 1 }}
          </span>
          <button
            v-if="db.entries.length > 1"
            type="button"
            class="remove-btn"
            @click="db.removeEntry(idx)"
          >
            <v-icon icon="mdi-close" size="16" />
          </button>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.dbName') }}
            <HelpTooltip text-key="help.dbName" :doc-link="HELP_LINKS.dbName" />
            <span class="required-mark">*</span>
          </label>
          <v-text-field
            :model-value="entry.name"
            @update:model-value="(v: string) => updateEntryName(idx, v)"
            :placeholder="t('databases.dbNamePlaceholder')"
            density="compact"
            variant="outlined"
            hide-details="auto"
            class="text-input"
          />
          <div v-if="!entry.name.trim()" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('databases.dbNameRequired') }}
          </div>
          <div v-else-if="isDuplicateName(entry.name)" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('databases.dbNameDuplicate') }}
          </div>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('databases.groups') }}
            <HelpTooltip text-key="help.dbGroups" :doc-link="HELP_LINKS.dbGroups" />
          </label>
          <v-text-field
            :model-value="entry.groups"
            @update:model-value="(v: string) => updateEntryGroups(idx, v)"
            density="compact"
            variant="outlined"
            hide-details="auto"
            type="number"
            :min="1"
            class="text-input"
          />
        </div>

        <div class="field-group">
          <label class="field-label">{{ t('databases.hosts') }}</label>
          <div class="radio-row">
            <button
              type="button"
              class="radio-option"
              :class="{ 'is-active': entry.hostMode === 'all' }"
              @click="updateEntryHostMode(idx, 'all')"
            >
              <span class="radio-label">{{ t('databases.hostsAll') }}</span>
            </button>
            <button
              type="button"
              class="radio-option is-disabled"
              disabled
              :title="t('common.soon')"
            >
              <span class="radio-label">{{ t('databases.hostsSelect') }} ({{ t('common.soon') }})</span>
            </button>
          </div>
          <div v-if="entry.hostMode === 'all'" class="field-hint">
            {{ t('databases.allHostsHint', { count: hostCount }) }}
          </div>
        </div>

        <div class="field-group field-group--last">
          <div class="advanced-toggle">
            <button
              type="button"
              class="advanced-toggle-btn"
              @click="toggleAdvanced(entry.id)"
            >
              <v-icon size="14" class="mr-1">
                {{ showAdvanced[entry.id] ? 'mdi-chevron-down' : 'mdi-chevron-right' }}
              </v-icon>
              {{ t('databases.advancedSettings') }}
            </button>
          </div>
          <v-expand-transition>
            <div v-if="showAdvanced[entry.id]">
              <div class="field-group">
                <label class="field-label">
                  {{ t('databases.affinity') }}
                  <HelpTooltip text-key="help.affinityDynamic" :doc-link="HELP_LINKS.affinityDynamic" />
                </label>
                <v-text-field
                  :model-value="entry.affinity"
                  @update:model-value="(v: string) => updateEntryAffinity(idx, v)"
                  :placeholder="t('databases.affinityPlaceholder')"
                  density="compact"
                  disabled
                  variant="outlined"
                  hide-details="auto"
                  class="text-input"
                />
                <div class="field-hint">{{ t('databases.affinityDesc') }}</div>
              </div>
            </div>
          </v-expand-transition>
        </div>
      </v-card>

      <div class="add-entry-row">
        <v-btn
          variant="outlined"
          prepend-icon="mdi-plus"
          size="small"
          @click="db.addEntry()"
        >
          {{ t('databases.addDatabase') }}
        </v-btn>
      </div>
    </section>

    <section class="config-section">
      <v-card class="config-card">
        <CoresInfoCard
          :cpu-stats="cpuStats"
          :budget="budget"
          compact
        />

        <div class="validation-summary">
          <div v-if="db.canProceed && !coresExceeded" class="validation-item is-ok">
            <v-icon icon="mdi-check-circle" size="16" color="success" class="mr-1" />
            {{ t('databases.validationOk') }}
          </div>
          <div v-else-if="coresExceeded" class="validation-item is-error">
            <v-icon icon="mdi-alert-circle" size="16" color="error" class="mr-1" />
            {{ t('databases.coresExceeded') }}
          </div>
          <div v-else class="validation-item is-warn">
            <v-icon icon="mdi-alert" size="16" color="warning" class="mr-1" />
            {{ t('databases.validationRequired') }}
          </div>
        </div>
      </v-card>
    </section>
  </div>
</template>

<style scoped>
.config-shell {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.config-head { padding: 4px 2px 0; margin-bottom: 16px; }
.config-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.config-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0 0 8px;
  max-width: 620px;
}

.config-section { display: flex; flex-direction: column; gap: 12px; }

.config-card {
  padding: 20px;
  background: rgb(var(--v-theme-surface));
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.db-card-header {
  padding-bottom: 8px;
  border-bottom: 1px solid #f1f5f9;
}
.section-eyebrow {
  display: inline-block;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #1f6dd9;
  background: #eef4ff;
  padding: 4px 10px;
  border-radius: 999px;
}

.remove-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  cursor: pointer;
  color: #94a3b8;
  padding: 4px;
  border-radius: 6px;
  transition: all .15s ease;
}
.remove-btn:hover {
  color: #dc2626;
  background: #fef2f2;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}
.field-group--last {
  margin-bottom: 0;
}
.field-label {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
}
.field-hint {
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.4;
  margin-top: 2px;
}
.required-mark {
  color: #dc2626;
  margin-left: 2px;
}
.text-input {
  max-width: 520px;
}

.em-badge {
  display: inline-flex;
  align-items: center;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #1f6dd9;
  background: #eef4ff;
  border-radius: 999px;
  padding: 3px 10px;
}

.variant-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.variant-option {
  flex: 1;
  width: fit-content;
  min-width: max-content;
  text-align: center;
  justify-content: center;
  white-space: nowrap;
}
.radio-row {
  display: flex;
  gap: 0;
  background: #f1f5f9;
  border-radius: 8px;
  padding: 3px;
  width: fit-content;
}
.radio-option {
  display: flex;
  align-items: center;
  padding: 6px 16px;
  border-radius: 6px;
  cursor: pointer;
  transition: all .18s ease;
  user-select: none;
  background: none;
  border: none;
  font-family: inherit;
}
.radio-option:focus-visible {
  outline: 2px solid #1f6dd9;
  outline-offset: 2px;
}
.radio-label {
  font-size: 13px;
  font-weight: 500;
  color: #64748b;
  transition: color .18s ease;
}
.radio-option.is-active {
  background: #fff;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}
  .radio-option.is-active .radio-label {
    color: #0f172a;
    font-weight: 600;
  }
  .radio-option.is-disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .radio-option.is-disabled:focus-visible {
    outline: none;
  }

.slider-row {
  display: flex;
  align-items: center;
  gap: 12px;
  max-width: 520px;
}
.cores-slider { flex: 1; }
.cores-input { width: 80px; }

.validation-hint {
  display: flex;
  align-items: center;
  font-size: 12px;
  padding: 4px 0;
}
.validation-hint.is-error { color: #dc2626; }

.hosts-checklist {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 4px;
  max-width: 520px;
}
.host-check-item {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background .12s ease;
}
.host-check-item:hover {
  background: #f1f5f9;
}
.host-check-label {
  font-size: 13px;
  color: #374151;
}

.custom-hosts-area {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}
.custom-host-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.host-chip {
  font-size: 12px;
}
.custom-host-input-row {
  display: flex;
  align-items: center;
  gap: 0;
}
.custom-host-input-row .text-input {
  max-width: 320px;
}
.ml-2 { margin-left: 8px; }

.multiple-mode-hint {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: #64748b;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 14px;
  line-height: 1.4;
}

.add-entry-row {
  display: flex;
  justify-content: center;
  padding: 8px 0;
}

.resource-indicator {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.resource-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.resource-label {
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
}
.resource-value {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
  font-variant-numeric: tabular-nums;
}
.resource-value--recommended { color: #ca8a04; }
.resource-value--info { color: #1f6dd9; }
.resource-sep {
  width: 1px;
  height: 24px;
  background: #e2e8f0;
}

.validation-summary {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f1f5f9;
}
.validation-item {
  display: flex;
  align-items: center;
  font-size: 13px;
}
.validation-item.is-ok { color: #059669; }
.validation-item.is-warn { color: #ca8a04; }

.advanced-toggle {
  margin-bottom: 4px;
}
.advanced-toggle-btn {
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
.advanced-toggle-btn:hover { color: #1f6dd9; }
</style>
