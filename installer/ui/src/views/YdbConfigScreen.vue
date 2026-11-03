<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useHostsStore } from '@/stores/hosts'
import { useTopologyStore } from '@/stores/topology'
import { useCoresInfo } from '@/composables/useCoresInfo'
import CoresInfoCard from '@/components/common/CoresInfoCard.vue'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'
import { HELP_LINKS } from '@/config/helpLinks'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

const session = useSessionStore()
const ydb = useYdbConfigStore()
const hosts = useHostsStore()
const { t } = useI18n()
const topologyStore = useTopologyStore()

const coresInfo = useCoresInfo()
const { cpuStats, budget, suggestedMax } = coresInfo

const hostCount = computed(() => {
  const allHosts = [
    ...hosts.dcHosts['DC-1'],
    ...hosts.dcHosts['DC-2'],
    ...hosts.dcHosts['DC-3'],
  ]
  return Math.max(allHosts.filter((h: string) => h.trim() !== '').length, 1)
})

const topologyLabel = computed(() => {
  const labels: Record<string, string> = {
    'mirror-3-dc': 'mirror-3-dc',
    'block-4-2': 'block-4-2',
    'bridge-2-dc': 'bridge-2-dc',
    'none': 'none',
  }
  return labels[topologyStore.selected || 'none']
})

const domain = computed({
  get: () => ydb.domain,
  set: (v) => { ydb.domain = v },
})

const password = computed({
  get: () => ydb.password,
  set: (v) => { ydb.password = v },
})

const showPassword = ref(false)

function handleGeneratePassword() {
  ydb.generatePassword(16)
}

const coresStatic = computed({
  get: () => ydb.coresStatic,
  set: (v: number | string | null | undefined) => {
    const n = typeof v === 'number' ? v : parseInt(String(v ?? ''), 10)
    ydb.coresStatic = isNaN(n) ? 0 : Math.max(0, n)
  },
})

function setEnforceUserToken(val: boolean) {
  ydb.enforceUserTokenRequirement = val
}

function setRequestClientCert(val: boolean) {
  ydb.requestClientCertificate = val
}

const showAdvanced = ref(false)

const affinity = computed({
  get: () => ydb.affinity,
  set: (v) => { ydb.affinity = v },
})

watch(() => ydb.canProceed, (can) => {
  session.setCanGoNext(can)
}, { immediate: true })
</script>

<template>
  <div class="config-shell">
    <header class="config-head">
      <ScreenEyebrow />
      <h1 class="config-title">{{ t('ydbConfig.heading') }}</h1>
      <p class="config-sub">{{ t('ydbConfig.topologyLabel', { topology: topologyLabel }) }}</p>
    </header>

    <section class="config-section">
      <v-card class="config-card">
        <div class="card-header">
          <span class="section-eyebrow">{{ t('ydbConfig.clusterParams') }}</span>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('ydbConfig.domain') }}
            <HelpTooltip text-key="help.ydbDomain" :doc-link="HELP_LINKS.ydbDomain" />
            <span class="required-mark">*</span>
          </label>
          <v-text-field
            v-model="domain"
            :placeholder="t('ydbConfig.domainPlaceholder')"
            density="compact"
            variant="outlined"
            hide-details="auto"
            class="text-input"
          />
          <div v-if="!domain.trim()" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('ydbConfig.domainRequired') }}
          </div>
        </div>

        <div class="field-group">
          <label class="field-label">{{ t('ydbConfig.password') }}</label>
          <div class="password-row">
            <v-text-field
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              :placeholder="t('ydbConfig.passwordPlaceholder')"
              density="compact"
              variant="outlined"
              hide-details="auto"
              class="text-input password-input"
            >
              <template #append-inner>
                <v-btn
                  variant="text"
                  size="small"
                  density="compact"
                  :icon="showPassword ? 'mdi-eye-off' : 'mdi-eye'"
                  @click="showPassword = !showPassword"
                />
              </template>
            </v-text-field>
            <v-btn
              variant="outlined"
              size="small"
              prepend-icon="mdi-dice-5"
              @click="handleGeneratePassword"
              class="generate-btn"
            >
              {{ t('ydbConfig.generatePassword') }}
            </v-btn>
          </div>
        </div>

        <div class="field-group">
          <label class="field-label">
            {{ t('ydbConfig.coresStatic') }}
            <HelpTooltip text-key="help.coresStatic" :doc-link="HELP_LINKS.coresStatic" />
            <span class="required-mark">*</span>
          </label>
          <div class="slider-row">
            <v-slider
              v-model="coresStatic"
              :min="0"
              :max="suggestedMax"
              :step="1"
              thumb-label
              density="compact"
              class="cores-slider"
            />
            <v-text-field
              v-model="coresStatic"
              density="compact"
              variant="outlined"
              hide-details="auto"
              type="number"
              :min="0"
              :max="suggestedMax"
              class="cores-input"
            />
          </div>
          <div v-if="coresStatic <= 0" class="validation-hint is-error">
            <v-icon icon="mdi-alert-circle" size="14" color="error" class="mr-1" />
            {{ t('ydbConfig.coresRequired') }}
          </div>
        </div>

        <div class="advanced-toggle">
          <button
            type="button"
            class="advanced-toggle-btn"
            @click="showAdvanced = !showAdvanced"
          >
            <v-icon size="14" class="mr-1">
              {{ showAdvanced ? 'mdi-chevron-down' : 'mdi-chevron-right' }}
            </v-icon>
            {{ t('ydbConfig.advancedSettings') }}
          </button>
        </div>

        <v-expand-transition>
          <div v-if="showAdvanced">
            <div class="field-group">
              <label class="field-label">
                {{ t('ydbConfig.enforceUserToken') }}
                <HelpTooltip text-key="help.enforceUserToken" :doc-link="HELP_LINKS.enforceUserToken" />
              </label>
              <div class="radio-row">
                <button
                  type="button"
                  class="radio-option"
                  :class="{ 'is-active': ydb.enforceUserTokenRequirement === true }"
                  @click="setEnforceUserToken(true)"
                >
                  <span class="radio-label">{{ t('ydbConfig.enabled') }}</span>
                </button>
                <button
                  type="button"
                  class="radio-option"
                  :class="{ 'is-active': ydb.enforceUserTokenRequirement === false }"
                  @click="setEnforceUserToken(false)"
                >
                  <span class="radio-label">{{ t('ydbConfig.disabled') }}</span>
                </button>
              </div>
              <div class="field-hint inventory-flag-hint">{{ t('ydbConfig.enforceUserTokenFlag') }}</div>
            </div>

            <div class="field-group">
              <label class="field-label">
                {{ t('ydbConfig.requestClientCert') }}
                <HelpTooltip text-key="help.requestClientCert" :doc-link="HELP_LINKS.requestClientCert" />
              </label>
              <div class="radio-row">
                <button
                  type="button"
                  class="radio-option"
                  :class="{ 'is-active': ydb.requestClientCertificate === true }"
                  @click="setRequestClientCert(true)"
                >
                  <span class="radio-label">{{ t('ydbConfig.enabled') }}</span>
                </button>
                <button
                  type="button"
                  class="radio-option"
                  :class="{ 'is-active': ydb.requestClientCertificate === false }"
                  @click="setRequestClientCert(false)"
                >
                  <span class="radio-label">{{ t('ydbConfig.disabled') }}</span>
                </button>
              </div>
              <div class="field-hint inventory-flag-hint">{{ t('ydbConfig.requestClientCertFlag') }}</div>
            </div>

            <div class="field-group field-group--last">
              <label class="field-label">
                {{ t('ydbConfig.affinity') }}
                <HelpTooltip text-key="help.affinityStatic" :doc-link="HELP_LINKS.affinityStatic" />
              </label>
              <v-text-field
                v-model="affinity"
                :placeholder="t('ydbConfig.affinityPlaceholder')"
                density="compact"
                disabled
                variant="outlined"
                hide-details="auto"
                class="text-input"
              />
              <div class="field-hint">{{ t('ydbConfig.affinityDesc') }}</div>
            </div>
          </div>
        </v-expand-transition>
      </v-card>
    </section>

    <section class="config-section">
      <v-card class="config-card">
        <CoresInfoCard
          :cpu-stats="cpuStats"
          :budget="budget"
          compact
        />

        <div class="validation-summary">
          <div v-if="ydb.canProceed" class="validation-item is-ok">
            <v-icon icon="mdi-check-circle" size="16" color="success" class="mr-1" />
            {{ t('ydbConfig.validationOk') }}
          </div>
          <div v-else class="validation-item is-warn">
            <v-icon icon="mdi-alert" size="16" color="warning" class="mr-1" />
            {{ t('ydbConfig.validationRequired') }}
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

.card-header { margin-bottom: 16px; }
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

.inventory-flag-hint {
  font-size: 11px;
  color: #b0b8c4;
  font-style: italic;
  margin-top: 0;
}

.required-mark {
  color: #dc2626;
  margin-left: 2px;
}

.text-input {
  max-width: 520px;
}

.password-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.password-input {
  max-width: 400px;
}

.generate-btn {
  white-space: nowrap;
}

.slider-row {
  display: flex;
  align-items: center;
  gap: 12px;
  max-width: 520px;
}

.cores-slider {
  flex: 1;
}

.cores-input {
  width: 80px;
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

.validation-hint {
  display: flex;
  align-items: center;
  font-size: 12px;
  padding: 4px 0;
}

.validation-hint.is-error {
  color: #dc2626;
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

.resource-value--recommended {
  color: #ca8a04;
}

.resource-value--info {
  color: #1f6dd9;
}

.resource-sep {
  width: 1px;
  height: 24px;
  background: #e2e8f0;
}

.resource-hint {
  display: flex;
  align-items: center;
  font-size: 12px;
  color: #94a3b8;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f1f5f9;
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