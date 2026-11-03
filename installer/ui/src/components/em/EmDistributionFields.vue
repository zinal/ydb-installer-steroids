<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useEmStore, type EmDistMode } from '@/stores/em'

const em = useEmStore()
const { t } = useI18n()

const isArchiveMode = computed(() => em.distMode === 'archive')

function setMode(mode: EmDistMode) {
  em.setDistMode(mode)
}

const archivePath = computed({
  get: () => em.archivePath,
  set: (v: string) => { em.archivePath = v },
})
const metaUIArchivePath = computed({
  get: () => em.metaUIArchivePath,
  set: (v: string) => { em.metaUIArchivePath = v },
})
const metaUIBinary = computed({
  get: () => em.metaUIBinary,
  set: (v: string) => { em.metaUIBinary = v },
})
const cpBinary = computed({
  get: () => em.cpBinary,
  set: (v: string) => { em.cpBinary = v },
})
const cpAgentBinary = computed({
  get: () => em.cpAgentBinary,
  set: (v: string) => { em.cpAgentBinary = v },
})

const timers: Record<string, ReturnType<typeof setTimeout> | null> = {
  archive: null,
  metaUIArchive: null,
  metaUIBinary: null,
  cpBinary: null,
  cpAgentBinary: null,
}

function scheduleValidate(
  field: 'archive' | 'metaUIArchive' | 'metaUIBinary' | 'cpBinary' | 'cpAgentBinary',
  path: string,
) {
  if (timers[field]) clearTimeout(timers[field]!)
  timers[field] = setTimeout(() => {
    void em.validatePath(field, path)
  }, 300)
}

watch(archivePath, (path) => scheduleValidate('archive', path))
watch(metaUIArchivePath, (path) => scheduleValidate('metaUIArchive', path))
watch(metaUIBinary, (path) => scheduleValidate('metaUIBinary', path))
watch(cpBinary, (path) => scheduleValidate('cpBinary', path))
watch(cpAgentBinary, (path) => scheduleValidate('cpAgentBinary', path))

watch(() => em.distMode, (mode) => {
  if (mode === 'archive') {
    if (em.archivePath) void em.validatePath('archive', em.archivePath)
    if (em.metaUIArchivePath) void em.validatePath('metaUIArchive', em.metaUIArchivePath)
  } else {
    if (em.metaUIBinary) void em.validatePath('metaUIBinary', em.metaUIBinary)
    if (em.cpBinary) void em.validatePath('cpBinary', em.cpBinary)
    if (em.cpAgentBinary) void em.validatePath('cpAgentBinary', em.cpAgentBinary)
  }
}, { immediate: true })

function validMsg(valid: boolean | null, size: string): string {
  if (valid === null) return ''
  if (valid) return t('emDistribution.fileValid', { size })
  return t('emDistribution.fileInvalid')
}

const archiveMsg = computed(() => validMsg(em.archiveValid, em.archiveSize))
const metaUIArchiveMsg = computed(() => validMsg(em.metaUIArchiveValid, em.metaUIArchiveSize))
const metaUIBinaryMsg = computed(() => validMsg(em.metaUIBinaryValid, em.metaUIBinarySize))
const cpBinaryMsg = computed(() => validMsg(em.cpBinaryValid, em.cpBinarySize))
const cpAgentBinaryMsg = computed(() => validMsg(em.cpAgentBinaryValid, em.cpAgentBinarySize))

const cardLabel = computed(() =>
  isArchiveMode.value
    ? t('emDistribution.modeArchive')
    : t('emDistribution.modeBinaries'),
)
</script>

<template>
  <div class="em-dist-fields">
    <div class="source-toggle" data-testid="em-dist-mode-row">
      <label
        class="source-option"
        :class="{ 'is-active': isArchiveMode }"
        @click="setMode('archive')"
      >
        <input
          type="radio"
          name="em-dist-mode"
          :checked="isArchiveMode"
          @change="setMode('archive')"
        />
        <span class="source-label">{{ t('emDistribution.modeArchive') }}</span>
      </label>
      <label
        class="source-option"
        :class="{ 'is-active': !isArchiveMode }"
        @click="setMode('binaries')"
      >
        <input
          type="radio"
          name="em-dist-mode"
          :checked="!isArchiveMode"
          @change="setMode('binaries')"
        />
        <span class="source-label">{{ t('emDistribution.modeBinaries') }}</span>
      </label>
    </div>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ cardLabel }}</div>

      <div v-if="isArchiveMode" class="ssh-form">
        <div class="ssh-field">
          <label class="ssh-label">
            {{ t('emDistribution.archiveLabel') }}
            <span class="required-mark">*</span>
          </label>
          <input
            v-model="archivePath"
            type="text"
            class="ssh-input"
            :placeholder="t('emDistribution.archivePlaceholder')"
            autocomplete="off"
            data-testid="em-archive-path"
          />
          <div
            v-if="archiveMsg"
            class="validation-msg"
            :class="{ 'is-valid': em.archiveValid, 'is-error': !em.archiveValid }"
          >
            <v-icon
              :icon="em.archiveValid ? 'mdi-check-circle' : 'mdi-alert-circle'"
              size="14"
              :color="em.archiveValid ? 'success' : 'error'"
              class="mr-1"
            />
            {{ archiveMsg }}
          </div>
        </div>

        <div class="ssh-field">
          <label class="ssh-label">
            {{ t('emDistribution.metaUiArchiveLabel') }}
            <span class="required-mark">*</span>
          </label>
          <input
            v-model="metaUIArchivePath"
            type="text"
            class="ssh-input"
            :placeholder="t('emDistribution.metaUiArchivePlaceholder')"
            autocomplete="off"
            data-testid="em-meta-ui-archive-path"
          />
          <div
            v-if="metaUIArchiveMsg"
            class="validation-msg"
            :class="{ 'is-valid': em.metaUIArchiveValid, 'is-error': !em.metaUIArchiveValid }"
          >
            <v-icon
              :icon="em.metaUIArchiveValid ? 'mdi-check-circle' : 'mdi-alert-circle'"
              size="14"
              :color="em.metaUIArchiveValid ? 'success' : 'error'"
              class="mr-1"
            />
            {{ metaUIArchiveMsg }}
          </div>
        </div>
      </div>

      <div v-else class="ssh-form">
        <div class="ssh-field">
          <label class="ssh-label">
            {{ t('emDistribution.metaUiBinaryLabel') }}
            <span class="required-mark">*</span>
          </label>
          <input
            v-model="metaUIBinary"
            type="text"
            class="ssh-input"
            :placeholder="t('emDistribution.metaUiBinaryPlaceholder')"
            autocomplete="off"
            data-testid="em-meta-ui-binary"
          />
          <div
            v-if="metaUIBinaryMsg"
            class="validation-msg"
            :class="{ 'is-valid': em.metaUIBinaryValid, 'is-error': !em.metaUIBinaryValid }"
          >
            <v-icon
              :icon="em.metaUIBinaryValid ? 'mdi-check-circle' : 'mdi-alert-circle'"
              size="14"
              :color="em.metaUIBinaryValid ? 'success' : 'error'"
              class="mr-1"
            />
            {{ metaUIBinaryMsg }}
          </div>
        </div>

        <div class="ssh-field">
          <label class="ssh-label">
            {{ t('emDistribution.cpBinaryLabel') }}
            <span class="required-mark">*</span>
          </label>
          <input
            v-model="cpBinary"
            type="text"
            class="ssh-input"
            :placeholder="t('emDistribution.cpBinaryPlaceholder')"
            autocomplete="off"
            data-testid="em-cp-binary"
          />
          <div
            v-if="cpBinaryMsg"
            class="validation-msg"
            :class="{ 'is-valid': em.cpBinaryValid, 'is-error': !em.cpBinaryValid }"
          >
            <v-icon
              :icon="em.cpBinaryValid ? 'mdi-check-circle' : 'mdi-alert-circle'"
              size="14"
              :color="em.cpBinaryValid ? 'success' : 'error'"
              class="mr-1"
            />
            {{ cpBinaryMsg }}
          </div>
        </div>

        <div class="ssh-field">
          <label class="ssh-label">
            {{ t('emDistribution.cpAgentBinaryLabel') }}
            <span class="required-mark">*</span>
          </label>
          <input
            v-model="cpAgentBinary"
            type="text"
            class="ssh-input"
            :placeholder="t('emDistribution.cpAgentBinaryPlaceholder')"
            autocomplete="off"
            data-testid="em-cp-agent-binary"
          />
          <div
            v-if="cpAgentBinaryMsg"
            class="validation-msg"
            :class="{ 'is-valid': em.cpAgentBinaryValid, 'is-error': !em.cpAgentBinaryValid }"
          >
            <v-icon
              :icon="em.cpAgentBinaryValid ? 'mdi-check-circle' : 'mdi-alert-circle'"
              size="14"
              :color="em.cpAgentBinaryValid ? 'success' : 'error'"
              class="mr-1"
            />
            {{ cpAgentBinaryMsg }}
          </div>
        </div>
      </div>
    </v-card>
  </div>
</template>

<style scoped>
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

.ssh-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
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
.required-mark {
  color: #dc2626;
  margin-left: 2px;
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

.validation-msg {
  display: flex;
  align-items: center;
  font-size: 12.5px;
  padding: 4px 0 0;
}
.validation-msg.is-valid { color: #15803d; }
.validation-msg.is-error { color: #dc2626; }
</style>
