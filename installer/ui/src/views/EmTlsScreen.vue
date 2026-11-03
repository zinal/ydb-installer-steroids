<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useEmStore } from '@/stores/em'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const session = useSessionStore()
const em = useEmStore()
const { t } = useI18n()

const isGenerateMode = computed(() => em.tlsMode === 'generate')

const certPath = computed({
  get: () => em.tlsCertPath,
  set: (v: string) => em.setTlsCertPath(v),
})

const validityDays = computed({
  get: () => em.tlsDays,
  set: (v: number) => em.setTlsDays(Number(v) || 365),
})

const emHosts = computed(() => em.emHostsList)
const hostCount = computed(() => emHosts.value.length)
const hasEmHost = computed(() => hostCount.value > 0)

const canGenerate = computed(() => em.tlsCanGenerate)
const canValidate = computed(() => certPath.value.trim() !== '' && hasEmHost.value && !em.tlsValidating)
const showExistWarning = computed(() => isGenerateMode.value && em.tlsCertsExist === true)
const showCaReuseInfo = computed(() => isGenerateMode.value && em.tlsCaReuse && hostCount.value > 0)

/** Switch TLS mode. Uses store action that fills path on existing. */
function setMode(mode: 'generate' | 'existing') {
  em.setTlsMode(mode)
}

/** CTA: switch to existing mode with discovered path filled. */
function switchToExisting() {
  em.switchToExistingWithDiscoveredPath()
}

async function handleGenerate() {
  if (!canGenerate.value) return
  await em.generateEmTls()
}

async function handleValidate() {
  if (!canValidate.value) return
  await em.validateEmTls()
}

watch(() => em.tlsCanProceed, (can) => {
  session.setCanGoNext(can)
}, { immediate: true })

watch(
  () => [em.tlsMode, em.emHostsList.join(',')] as const,
  ([mode]) => {
    if (mode === 'generate') {
      void em.checkEmTlsExists()
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="em-tls-shell">
    <header class="em-tls-head">
      <ScreenEyebrow />
      <h1 class="em-tls-title">{{ t('emTls.heading') }}</h1>
      <p class="em-tls-sub">{{ t('emTls.subtitle') }}</p>
    </header>

    <div class="source-toggle" data-testid="em-tls-mode-row">
      <label
        class="source-option"
        :class="{ 'is-active': isGenerateMode }"
        @click="setMode('generate')"
      >
        <input
          type="radio"
          name="em-tls-mode"
          :checked="isGenerateMode"
          @change="setMode('generate')"
        />
        <span class="source-label">{{ t('emTls.modeGenerate') }}</span>
      </label>
      <label
        class="source-option"
        :class="{ 'is-active': !isGenerateMode }"
        @click="setMode('existing')"
      >
        <input
          type="radio"
          name="em-tls-mode"
          :checked="!isGenerateMode"
          @change="setMode('existing')"
        />
        <span class="source-label">{{ t('emTls.modeExisting') }}</span>
      </label>
    </div>

    <div v-if="!hasEmHost" class="validation-message validation-message--warning">
      <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
      <span>{{ t('emTls.noEmHost') }}</span>
    </div>

    <v-card v-if="isGenerateMode" class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('emTls.generationTitle') }}</div>
      <p class="field-description">{{ t('emTls.generateDesc') }}</p>

      <div v-if="hasEmHost" class="host-line">
        <span class="host-label">{{ t('emTls.emHost') }}:</span>
        <code v-for="h in emHosts" :key="h" class="host-chip">{{ h }}</code>
      </div>

      <div
        v-if="showCaReuseInfo"
        class="exist-warning ca-reuse-info"
        data-testid="em-tls-ca-reuse-info"
      >
        <v-icon size="18" color="#2563eb">mdi-information-outline</v-icon>
        <span>{{ t('emTls.caReuseInfo') }}</span>
      </div>

      <p class="will-generate-label">{{ t('emTls.willGenerate') }}</p>
      <ul class="generate-list">
        <li>
          <v-icon size="16" color="primary" class="mr-1">mdi-certificate</v-icon>
          {{ t('emTls.caCert') }}
        </li>
        <li>
          <v-icon size="16" color="primary" class="mr-1">mdi-file-document-outline</v-icon>
          {{ t('emTls.nodeCerts', { count: hostCount }) }}
        </li>
        <li>
          <v-icon size="16" color="primary" class="mr-1">mdi-key-variant</v-icon>
          {{ t('emTls.privateKeys') }}
        </li>
        <li>
          <v-icon size="16" color="primary" class="mr-1">mdi-file-lock-outline</v-icon>
          {{ t('emTls.webPem') }}
        </li>
      </ul>

      <div
        v-if="showExistWarning"
        class="exist-warning"
        data-testid="em-tls-exist-warning"
      >
        <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
        <div class="exist-warning-body">
          <p class="exist-warning-title">{{ t('emTls.certsExistWarning') }}</p>
          <p class="exist-warning-hint">
            {{ t('emTls.certsExistHint', { path: em.tlsExistPath || 'TLS/CA/certs/certs-per-host' }) }}
          </p>
          <button
            type="button"
            class="exist-switch-btn"
            data-testid="em-tls-switch-existing"
            @click="switchToExisting"
          >
            {{ t('emTls.switchToExisting') }}
          </button>
        </div>
      </div>

      <div class="ssh-field days-row">
        <label class="ssh-label">{{ t('emTls.validity') }}</label>
        <div class="days-input">
          <input
            v-model.number="validityDays"
            type="number"
            min="1"
            max="3650"
            class="ssh-input days-field"
            data-testid="em-tls-days"
          />
          <span class="days-label">{{ t('emTls.days') }}</span>
        </div>
      </div>

      <v-btn
        color="primary"
        class="mt-4"
        :loading="em.tlsGenerating || em.tlsExistChecking"
        :disabled="!canGenerate"
        prepend-icon="mdi-lock-plus"
        data-testid="em-tls-generate-btn"
        @click="handleGenerate"
      >
        {{ em.tlsGenerating ? t('emTls.generating') : t('emTls.generateBtn') }}
        </v-btn>

      <div v-if="em.tlsGenerationResult" class="result-block">
        <div v-if="em.tlsGenerationResult.success" class="result-success">
          <v-icon color="success" class="mr-2">mdi-check-circle</v-icon>
          <span>{{ t('emTls.success') }}</span>
        </div>
        <div v-else class="result-error">
          <v-icon color="error" class="mr-2">mdi-alert-circle</v-icon>
          <span>{{ em.tlsGenerationResult.error || em.tlsError }}</span>
        </div>

        <div v-if="em.tlsGenerationResult.success" class="result-details">
          <p class="result-dir">
            <strong>{{ t('emTls.outputDir') }}:</strong>
            <code class="dir-path">{{ em.tlsGenerationResult.outputDir }}</code>
          </p>
          <code class="structure-code">
            <span class="tree-root">TLS/CA/certs/certs-per-host/</span><br>
            ├── ca.crt<br>
            <template v-for="(nf, idx) in em.tlsGenerationResult.nodeFiles" :key="nf.hostname">
              <span>
                {{ idx === em.tlsGenerationResult.nodeFiles.length - 1 ? '└' : '├' }}── {{ nf.shortName }}/
              </span><br>
              <span class="tree-child">
                {{ idx === em.tlsGenerationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}├── node.crt
              </span><br>
              <span class="tree-child">
                {{ idx === em.tlsGenerationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}├── node.key
              </span><br>
              <span class="tree-child">
                {{ idx === em.tlsGenerationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}└── web.pem
              </span><br>
            </template>
          </code>
        </div>
      </div>
    </v-card>

    <v-card v-else class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('emTls.validationTitle') }}</div>
      <p class="field-description">{{ t('emTls.existingDesc') }}</p>

      <div class="ssh-field">
        <label class="ssh-label">{{ t('emTls.certPath') }}</label>
        <input
          v-model="certPath"
          type="text"
          class="ssh-input"
          :placeholder="t('emTls.certPathPlaceholder')"
          autocomplete="off"
          data-testid="em-tls-cert-path"
        />
      </div>

      <v-btn
        color="primary"
        variant="outlined"
        class="mt-4"
        :loading="em.tlsValidating"
        :disabled="!canValidate"
        prepend-icon="mdi-check-circle"
        data-testid="em-tls-validate-btn"
        @click="handleValidate"
      >
        {{ em.tlsValidating ? t('emTls.validating') : t('emTls.validateBtn') }}
      </v-btn>

      <div v-if="em.tlsValidationResult" class="result-block">
        <div v-if="em.tlsValidationResult.valid" class="result-success">
          <v-icon color="success" class="mr-2">mdi-check-circle</v-icon>
          <span>{{ t('emTls.allValid') }}</span>
        </div>
        <div v-else class="result-error">
          <v-icon color="error" class="mr-2">mdi-alert-circle</v-icon>
          <span>{{ em.tlsError || t('emTls.missing') }}</span>
        </div>

        <div class="ca-status">
          <v-icon
            :color="em.tlsValidationResult.caValid ? 'success' : 'error'"
            size="16"
            class="mr-1"
          >
            {{ em.tlsValidationResult.caValid ? 'mdi-check-circle' : 'mdi-alert-circle' }}
          </v-icon>
          <span class="ca-label">{{ t('emTls.caCert') }}</span>
          <code class="file-path">ca.crt</code>
          <span class="status-text">
            ({{ em.tlsValidationResult.caValid ? t('emTls.valid') : t('emTls.invalid') }})
          </span>
        </div>

        <div v-if="em.tlsValidationResult.nodeFiles.length > 0" class="validation-table">
          <table class="cert-table">
            <thead>
              <tr>
                <th>{{ t('emTls.hostname') }}</th>
                <th>{{ t('emTls.nodeCert') }}</th>
                <th>{{ t('emTls.nodeKey') }}</th>
                <th>{{ t('emTls.keyMatch') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="node in em.tlsValidationResult.nodeFiles" :key="node.index">
                <td class="hostname-cell">
                  <span class="hostname-fqdn">{{ node.hostname }}</span>
                  <span class="hostname-short">({{ node.shortName }})</span>
                </td>
                <td>
                  <v-icon
                    :color="node.certFound && node.certValid ? 'success' : 'error'"
                    size="16"
                    class="mr-1"
                  >
                    {{ node.certFound && node.certValid ? 'mdi-check-circle' : 'mdi-alert-circle' }}
                  </v-icon>
                  <code class="file-path">{{ node.shortName }}/node.crt</code>
                </td>
                <td>
                  <v-icon
                    :color="node.keyFound && node.keyValid ? 'success' : 'error'"
                    size="16"
                    class="mr-1"
                  >
                    {{ node.keyFound && node.keyValid ? 'mdi-check-circle' : 'mdi-alert-circle' }}
                  </v-icon>
                  <code class="file-path">{{ node.shortName }}/node.key</code>
                </td>
                <td>
                  <template v-if="node.certFound && node.keyFound">
                    <v-icon :color="node.pairMatch ? 'success' : 'error'" size="16" class="mr-1">
                      {{ node.pairMatch ? 'mdi-check-circle' : 'mdi-alert-circle' }}
                    </v-icon>
                    <span class="status-text">
                      {{ node.pairMatch ? t('emTls.match') : t('emTls.mismatch') }}
                    </span>
                  </template>
                  <span v-else class="status-text">—</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </v-card>
  </div>
</template>

<style scoped>
.em-tls-head { margin-bottom: 28px; }
.em-tls-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.em-tls-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
}

.source-toggle {
  display: flex;
  gap: 0;
  background: #f1f5f9;
  border-radius: 12px;
  padding: 4px;
  width: fit-content;
  margin-bottom: 16px;
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

.section-card { background: rgb(var(--v-theme-surface)); }
.section-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #94a3b8;
  margin-bottom: 14px;
}

.field-description {
  font-size: 13.5px;
  color: #64748b;
  margin: 0 0 14px;
}

.host-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
}
.host-label {
  font-size: 12.5px;
  font-weight: 500;
  color: #374151;
}
.host-chip {
  font-size: 12px;
  background: #f1f5f9;
  padding: 2px 8px;
  border-radius: 4px;
  color: #1e40af;
}

.will-generate-label {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
  margin: 0 0 8px;
}
.generate-list {
  list-style: none;
  padding: 0;
  margin: 0 0 16px;
}
.generate-list li {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: #64748b;
  padding: 4px 0;
}

.ssh-field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.ssh-label {
  font-size: 12.5px;
  font-weight: 500;
}
.ssh-input {
  padding: 8px 12px;
  border: 1.5px solid #e2e8f0;
  border-radius: 8px;
  font-size: 13.5px;
  color: #0f172a;
  background: #fff;
  outline: none;
  transition: border-color .15s ease;
  font-family: inherit;
  width: 100%;
  box-sizing: border-box;
}
.ssh-input:focus { border-color: #1f6dd9; }

.days-row { margin-top: 4px; }
.days-input {
  display: flex;
  align-items: center;
  gap: 8px;
}
.days-field { max-width: 120px; }
.days-label {
  font-size: 13px;
  color: #64748b;
}

.result-block {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e2e8f0;
}
.result-success,
.result-error {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 13.5px;
  font-weight: 500;
}
.result-success {
  background: #f0fdf4;
  color: #15803d;
}
.result-error {
  background: #fef2f2;
  color: #dc2626;
}
.result-details { margin-top: 12px; }
.result-dir {
  font-size: 13px;
  color: #374151;
  margin: 0 0 8px;
  display: flex;
  align-items: baseline;
  gap: 6px;
  flex-wrap: wrap;
}
.dir-path {
  font-size: 12px;
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  color: #1e40af;
  word-break: break-all;
}
.structure-code {
  display: block;
  font-size: 12px;
  color: #374151;
  background: #f8fafc;
  padding: 10px 14px;
  border-radius: 6px;
  line-height: 1.7;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  white-space: pre;
  overflow-x: auto;
}
.tree-root {
  color: #1e40af;
  font-weight: 600;
}
.tree-child { color: #64748b; }

.ca-status {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  background: #f8fafc;
  font-size: 13px;
}
.ca-label {
  font-weight: 500;
  color: #374151;
}
.file-path {
  font-size: 12px;
  background: #f1f5f9;
  padding: 1px 5px;
  border-radius: 3px;
  color: #1e40af;
  margin-right: 4px;
}
.status-text {
  font-size: 11px;
  color: #94a3b8;
}

.validation-table {
  margin-top: 12px;
  overflow-x: auto;
}
.cert-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.cert-table th,
.cert-table td {
  text-align: left;
  padding: 8px 12px;
  border-bottom: 1px solid #e2e8f0;
  vertical-align: middle;
}
.cert-table th {
  font-weight: 500;
  color: #64748b;
  background: #f8fafc;
  white-space: nowrap;
}
.hostname-fqdn {
  font-size: 13px;
  color: #0f172a;
}
.hostname-short {
  font-size: 11px;
  color: #94a3b8;
  margin-left: 4px;
}

.validation-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 14px;
  margin-bottom: 12px;
}
.validation-message--warning {
  background: #fffbeb;
  border: 1.5px solid #fcd34d;
  color: #92400e;
}
.exist-warning {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  margin-bottom: 16px;
  background: #fffbeb;
  border: 1.5px solid #fcd34d;
  border-radius: 8px;
  color: #92400e;
}
.exist-warning-body { flex: 1; }
.exist-warning-title {
  margin: 0 0 6px;
  font-size: 13.5px;
  font-weight: 600;
}
.exist-warning-hint {
  margin: 0 0 10px;
  font-size: 12.5px;
  line-height: 1.45;
  color: #a16207;
}
.exist-switch-btn {
  border: none;
  background: transparent;
  color: #1f6dd9;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  padding: 0;
  font-family: inherit;
}
.exist-switch-btn:hover { text-decoration: underline; }

.ca-reuse-info {
  background: #eff6ff;
  border-color: #93c5fd;
  color: #1e40af;
}
</style>
