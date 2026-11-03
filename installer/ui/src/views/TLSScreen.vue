<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useTlsStore } from '@/stores/tls'
import { useHostsStore } from '@/stores/hosts'
import { useTopologyStore } from '@/stores/topology'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const session = useSessionStore()
const tlsStore = useTlsStore()
const hostsStore = useHostsStore()
const topologyStore = useTopologyStore()
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

const isGenerateMode = computed({
  get: () => tlsStore.mode === 'generate',
  set: (v) => tlsStore.setMode(v ? 'generate' : 'existing'),
})

const certPath = computed({
  get: () => tlsStore.certPath,
  set: (v) => tlsStore.setCertPath(v),
})

const validityDays = computed({
  get: () => tlsStore.days,
  set: (v) => tlsStore.setDays(v),
})

/** All non-empty YDB hosts from all DCs, in order. */
const ydbHosts = computed(() => {
  const dcHosts = hostsStore.dcHosts
  const hosts: string[] = []
  const dcs = ['DC-1', 'DC-2', 'DC-3'] as const
  for (const dc of dcs) {
    if (dcHosts[dc]) {
      hosts.push(...dcHosts[dc])
    }
  }
  return hosts.filter(h => h.trim() !== '')
})

const allHosts = computed(() => ydbHosts.value)

const hostCount = computed(() => allHosts.value.length)

const canGenerate = computed(() => hostCount.value > 0 && !tlsStore.isGenerating)

const canValidate = computed(() => certPath.value.trim() !== '' && !tlsStore.isValidating)

async function handleGenerate() {
  if (!canGenerate.value) return
  await tlsStore.generateCertificates(allHosts.value, true)
}

async function handleValidate() {
  if (!canValidate.value) return
  await tlsStore.validateCertificates(allHosts.value)
}

watch(() => tlsStore.canProceed, (can) => {
  session.setCanGoNext(can)
}, { immediate: true })

watch(() => tlsStore.error, (error) => {
  if (error) {
    session.setCanGoNext(false)
  }
})
</script>

<template>
  <div class="tls-shell">
    <header class="tls-head">
      <ScreenEyebrow />
      <h1 class="tls-title">{{ t('tls.heading') }}</h1>
      <p class="tls-sub">{{ t('tls.topologyLabel', { topology: topologyLabel }) }}</p>
    </header>

    <section class="tls-section">
      <div class="mode-toggle">
        <label class="mode-option" :class="{ 'is-active': isGenerateMode }">
          <input type="radio" v-model="isGenerateMode" :value="true" />
          <span class="mode-label">{{ t('tls.generate') }}</span>
        </label>
        <label class="mode-option" :class="{ 'is-active': !isGenerateMode }">
          <input type="radio" v-model="isGenerateMode" :value="false" />
          <span class="mode-label">{{ t('tls.existing') }}</span>
        </label>
      </div>
    </section>

    <section v-if="isGenerateMode" class="tls-section">
      <v-card class="tls-card">
        <div class="card-header">
          <span class="section-eyebrow eyebrow">{{ t('tls.generation.title') }}</span>
        </div>

        <div class="field-group">
          <p class="field-description">{{ t('tls.generateDesc') }}</p>
        </div>

        <div class="field-group">
          <p class="will-generate-label">{{ t('tls.generation.willGenerate') }}</p>
          <ul class="generate-list">
            <li>
              <v-icon size="16" color="primary" class="mr-1">mdi-certificate</v-icon>
              {{ t('tls.generation.caCert') }}
            </li>
            <li>
              <v-icon size="16" color="primary" class="mr-1">mdi-file-document-outline</v-icon>
              {{ t('tls.generation.nodeCerts', { count: hostCount }) }}
            </li>
            <li>
              <v-icon size="16" color="primary" class="mr-1">mdi-key-variant</v-icon>
              {{ t('tls.generation.privateKeys') }}
            </li>
            <li>
              <v-icon size="16" color="primary" class="mr-1">mdi-cog-outline</v-icon>
              {{ t('tls.generation.optionsConf') }}
            </li>
            <li>
              <v-icon size="16" color="primary" class="mr-1">mdi-file-sign</v-icon>
              {{ t('tls.generation.nodeCsr') }}
            </li>
            <li>
              <v-icon size="16" color="primary" class="mr-1">mdi-file-certificate-outline</v-icon>
              {{ t('tls.generation.nodePem') }}
            </li>
            <li>
              <v-icon size="16" color="primary" class="mr-1">mdi-file-lock-outline</v-icon>
              {{ t('tls.generation.webPem') }}
            </li>
          </ul>
        </div>

        <div class="field-group">
          <label class="field-label">{{ t('tls.generation.validity') }}</label>
          <div class="days-input">
            <v-text-field
              v-model.number="validityDays"
              type="number"
              min="1"
              max="3650"
              density="compact"
              variant="outlined"
              hide-details="auto"
              class="days-field"
            />
            <span class="days-label">{{ t('tls.generation.days') }}</span>
          </div>
        </div>

        <div class="field-group field-group--last">
          <v-btn
            color="primary"
            :loading="tlsStore.isGenerating"
            :disabled="!canGenerate"
            @click="handleGenerate"
            prepend-icon="mdi-lock-plus"
            class="generate-btn"
          >
            {{ tlsStore.isGenerating ? t('tls.generation.generating') : t('tls.generation.generateBtn') }}
          </v-btn>
        </div>

        <v-expand-transition>
          <div v-if="tlsStore.generationResult" class="generation-result">
            <div v-if="tlsStore.generationResult.success" class="result-success">
              <v-icon color="success" class="mr-2">mdi-check-circle</v-icon>
              <span>{{ t('tls.generation.success') }}</span>
            </div>
            <div v-else class="result-error">
              <v-icon color="error" class="mr-2">mdi-alert-circle</v-icon>
              <span>{{ tlsStore.generationResult.error }}</span>
            </div>

            <div v-if="tlsStore.generationResult.success" class="result-details">
              <p class="result-dir">
                <strong>{{ t('tls.generation.outputDir') }}:</strong>
                <code class="dir-path">{{ tlsStore.generationResult.outputDir }}</code>
              </p>
              <div class="file-structure">
                <p class="structure-label">{{ t('tls.generation.structure') }}:</p>
                <code class="structure-code">
                  <span class="tree-root">{{ t('tls.generation.fixedPath') }}</span><br>
                  ├── ca.crt<br>
                  <template v-for="(nf, idx) in tlsStore.generationResult.nodeFiles" :key="nf.hostname">
                  <span :class="idx === tlsStore.generationResult.nodeFiles.length - 1 ? 'tree-last' : 'tree-mid'">
                    {{ idx === tlsStore.generationResult.nodeFiles.length - 1 ? '└' : '├' }}── {{ nf.shortName }}/
                  </span><br>
                  <span class="tree-child">
                    {{ idx === tlsStore.generationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}├── options.cnf
                  </span><br>
                  <span class="tree-child">
                    {{ idx === tlsStore.generationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}├── node.key
                  </span><br>
                  <span class="tree-child">
                    {{ idx === tlsStore.generationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}├── node.csr
                  </span><br>
                  <span class="tree-child">
                    {{ idx === tlsStore.generationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}├── node.crt
                  </span><br>
                  <span class="tree-child">
                    {{ idx === tlsStore.generationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}├── node.pem
                  </span><br>
                  <span class="tree-child">
                    {{ idx === tlsStore.generationResult.nodeFiles.length - 1 ? '    ' : '│   ' }}└── web.pem
                  </span><br>
                  </template>
                </code>
              </div>
            </div>
          </div>
        </v-expand-transition>
      </v-card>
    </section>

    <section v-else class="tls-section">
      <v-card class="tls-card">
        <div class="card-header">
          <span class="section-eyebrow eyebrow">{{ t('tls.validation.title') }}</span>
        </div>

        <div class="field-group">
          <p class="field-description">{{ t('tls.existingDesc') }}</p>
        </div>

        <div class="field-group">
          <label class="field-label">{{ t('tls.validation.certPath') }}</label>
          <v-text-field
            v-model="certPath"
            :placeholder="t('tls.validation.certPathPlaceholder')"
            density="compact"
            variant="outlined"
            hide-details="auto"
            class="path-input"
          />
        </div>

        <div class="field-group field-group--last">
          <v-btn
            color="primary"
            variant="outlined"
            :loading="tlsStore.isValidating"
            :disabled="!canValidate"
            @click="handleValidate"
            prepend-icon="mdi-check-circle"
            class="validate-btn"
          >
            {{ tlsStore.isValidating ? t('tls.validation.validating') : t('tls.validation.validateBtn') }}
          </v-btn>
        </div>

        <v-expand-transition>
          <div v-if="tlsStore.validationResult" class="validation-result">
            <div v-if="tlsStore.validationResult.valid" class="result-success">
              <v-icon color="success" class="mr-2">mdi-check-circle</v-icon>
              <span>{{ t('tls.validation.allValid') }}</span>
            </div>
            <div v-else class="result-error">
              <v-icon color="error" class="mr-2">mdi-alert-circle</v-icon>
              <span>{{ tlsStore.error || t('tls.validation.missing') }}</span>
            </div>

            <div v-if="tlsStore.validationResult.nodeFiles.length > 0" class="validation-table">
              <table class="cert-table">
                <thead>
                  <tr>
                    <th>{{ t('tls.validation.hostname') }}</th>
                    <th>{{ t('tls.validation.nodeCert') }}</th>
                    <th>{{ t('tls.validation.nodeKey') }}</th>
                    <th>{{ t('tls.validation.keyMatch') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="node in tlsStore.validationResult.nodeFiles" :key="node.index">
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
                      <span class="status-text">
                        ({{ node.certFound ? t('tls.validation.found') : t('tls.validation.notFound') }}{{ node.certFound ? (node.certValid ? ', ' + t('tls.validation.valid') : ', ' + t('tls.validation.invalid')) : '' }})
                      </span>
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
                      <span class="status-text">
                        ({{ node.keyFound ? t('tls.validation.found') : t('tls.validation.notFound') }}{{ node.keyFound ? (node.keyValid ? ', ' + t('tls.validation.valid') : ', ' + t('tls.validation.invalid')) : '' }})
                      </span>
                    </td>
                    <td>
                      <v-icon
                        v-if="node.certFound && node.keyFound"
                        :color="node.pairMatch ? 'success' : 'error'"
                        size="16"
                        class="mr-1"
                      >
                        {{ node.pairMatch ? 'mdi-check-circle' : 'mdi-alert-circle' }}
                      </v-icon>
                      <v-icon
                        v-else
                        color="grey"
                        size="16"
                        class="mr-1"
                      >
                        mdi-minus-circle-outline
                      </v-icon>
                      <span class="status-text">
                        {{ node.certFound && node.keyFound ? (node.pairMatch ? t('tls.validation.match') : t('tls.validation.mismatch')) : '—' }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div class="ca-status">
              <v-icon
                :color="tlsStore.validationResult.caValid ? 'success' : 'error'"
                size="16"
                class="mr-1"
              >
                {{ tlsStore.validationResult.caValid ? 'mdi-check-circle' : 'mdi-alert-circle' }}
              </v-icon>
              <span class="ca-label">{{ t('tls.validation.caCert') }}</span>
              <code class="file-path">ca.crt</code>
              <span class="status-text">
                ({{ tlsStore.validationResult.caValid ? t('tls.validation.valid') : t('tls.validation.invalid') }})
              </span>
            </div>

            <div v-if="tlsStore.validationResult.missing.length > 0" class="missing-list">
              <p class="missing-label">{{ t('tls.validation.missing') }}:</p>
              <ul>
                <li v-for="(file, idx) in tlsStore.validationResult.missing" :key="idx">
                  <code>{{ file }}</code>
                </li>
              </ul>
            </div>
          </div>
        </v-expand-transition>
      </v-card>
    </section>

    <v-expand-transition>
      <div v-if="tlsStore.error && !tlsStore.generationResult && !tlsStore.validationResult" class="error-banner">
        <v-icon color="error" class="mr-2">mdi-alert-circle</v-icon>
        <span>{{ tlsStore.error }}</span>
      </div>
    </v-expand-transition>
  </div>
</template>

<style scoped>
.tls-shell {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.tls-head { padding: 4px 2px 0; margin-bottom: 16px; }
.tls-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.tls-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
}

.tls-section { display: flex; flex-direction: column; gap: 12px; }

.mode-toggle {
  display: flex;
  gap: 0;
  background: #f1f5f9;
  border-radius: 12px;
  padding: 4px;
  width: fit-content;
}

.mode-option {
  display: flex;
  align-items: center;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
  transition: all .18s ease;
  user-select: none;
}

.mode-option input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.mode-label {
  font-size: 13.5px;
  font-weight: 500;
  color: #64748b;
  transition: color .18s ease;
}

.mode-option.is-active {
  background: #fff;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}

.mode-option.is-active .mode-label {
  color: #0f172a;
  font-weight: 600;
}

/* ── Card ── */
.tls-card {
  padding: 20px;
  background: rgb(var(--v-theme-surface));
}

/* ── Field Group ── */
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

.field-description {
  font-size: 13.5px;
  color: #64748b;
  margin: 0;
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
  margin: 0;
}

.generate-list li {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: #64748b;
  padding: 4px 0;
}

.days-input {
  display: flex;
  align-items: center;
  gap: 8px;
}

.days-field {
  max-width: 120px;
}

.days-label {
  font-size: 13px;
  color: #64748b;
}

.generate-btn,
.validate-btn {
  align-self: flex-start;
}

.generation-result,
.validation-result {
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

.result-details {
  margin-top: 12px;
}

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

.file-structure {
  margin-top: 8px;
}

.structure-label {
  font-size: 12px;
  font-weight: 500;
  color: #64748b;
  margin: 0 0 4px;
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

.tree-mid,
.tree-last {
  color: #374151;
}

.tree-child {
  color: #64748b;
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

.cert-table td {
  color: #374151;
}

.hostname-cell {
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

.missing-list {
  margin-top: 12px;
}

.missing-label {
  font-size: 12px;
  font-weight: 500;
  color: #dc2626;
  margin: 0 0 4px;
}

.missing-list ul {
  margin: 0;
  padding-left: 20px;
}

.missing-list li {
  font-size: 12px;
  color: #dc2626;
  padding: 2px 0;
}

.error-banner {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  color: #dc2626;
  font-size: 13.5px;
}

.path-input {
  max-width: 600px;
}

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
</style>
