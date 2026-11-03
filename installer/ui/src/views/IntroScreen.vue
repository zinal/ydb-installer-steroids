<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useConnectivityStore } from '@/stores/connectivity'
import { useDeps } from '@/composables/useDeps'
import AppVersion from '@/components/common/AppVersion.vue'
import DepsPanel from '@/components/common/DepsPanel.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

type Mode = 'ydb' | 'delete-ydb'

const { t } = useI18n()
const session = useSessionStore()
const connectivity = useConnectivityStore()

const selected = computed<Mode | null>(() => (session.flow as Mode) || null)

const { deps, loading: depsLoading, bundled: depsBundled, canProceed, fetchDeps, retryDeps, toggleInstruction, skipDep, unskipDep } = useDeps()

onMounted(() => {
  fetchDeps()
})

watch([canProceed, depsLoading, selected], ([proceed, loading, mode]) => {
  const isDeleteFlow = mode === 'delete-ydb'
  
  if (isDeleteFlow && !loading) {
    session.setCanGoNext(proceed)
  } else if (!isDeleteFlow) {
    session.setCanGoNext(true)
  }
}, { immediate: true })

interface ModeOption {
  value: Mode
  group: 'install' | 'remove'
  title: string
  description: string
  icon: string
}

const options = computed<ModeOption[]>(() => [
  {
    value: 'ydb',
    group: 'install',
    title: t('intro.ydbWizard'),
    description: t('intro.ydbWizardDesc'),
    icon: 'mdi-database-plus-outline',
  },
  {
    value: 'delete-ydb',
    group: 'remove',
    title: t('intro.deleteYdb'),
    description: t('intro.deleteYdbDesc'),
    icon: 'mdi-database-remove-outline',
  },
])

const installOptions = computed(() => options.value.filter((o) => o.group === 'install'))
const removeOptions = computed(() => options.value.filter((o) => o.group === 'remove'))

/** YDB delete needs 50-inventory + config.yaml. */
const canSelectDeleteYdb = computed(() => session.inventoryFound && session.configFound)

function canSelectDelete(mode: Mode): boolean {
  if (mode === 'delete-ydb') return canSelectDeleteYdb.value
  return true
}

/** Show a hint when YDB delete is locked. */
const showDeleteHint = computed(() => !canSelectDeleteYdb.value)

const deleteHintText = computed(() => t('intro.deleteRequiresConfig'))

async function selectMode(mode: Mode) {
  if (mode === 'delete-ydb' && !canSelectDelete(mode)) {
    return
  }
  await session.setFlow(mode)
}
</script>

<template>
  <div class="intro-shell">
    <header class="intro-head">
      <div class="intro-eyebrow eyebrow">
        <span>{{ t('app.title') }} ·</span>
        <AppVersion />
      </div>
      <h1 class="intro-title">{{ t('intro.heading') }}</h1>
      <p class="intro-sub">{{ t('intro.description') }}</p>
    </header>

    <section class="intro-section">
      <div class="section-head">
        <span class="section-eyebrow eyebrow">{{ t('intro.ydbGroup') }}</span>
      </div>
      <div class="mode-grid" :class="{ 'mode-grid--single': installOptions.length === 1 }">
        <button
          v-for="opt in installOptions"
          :key="opt.value"
          type="button"
          class="mode-card"
          :class="{ 'is-selected': selected === opt.value }"
          @click="selectMode(opt.value)"
        >
          <div class="mode-icon">
            <v-icon :icon="opt.icon" size="22" />
          </div>
          <div class="mode-body">
            <div class="mode-title">{{ opt.title }}</div>
            <div class="mode-desc">{{ opt.description }}</div>
          </div>
          <div class="mode-radio" aria-hidden="true">
            <span class="mode-radio-dot" />
          </div>
        </button>
      </div>
    </section>

    <section class="intro-section">
      <div class="section-head">
        <span class="section-eyebrow eyebrow">{{ t('intro.deleteGroup') }}</span>
      </div>
      <div class="mode-grid" :class="{ 'mode-grid--single': removeOptions.length === 1 }">
        <template v-for="opt in removeOptions" :key="opt.value">
          <div
            v-if="!canSelectDelete(opt.value)"
            class="mode-card mode-card--danger mode-card--disabled"
          >
            <div class="mode-icon mode-icon--danger">
              <v-icon :icon="opt.icon" size="22" />
            </div>
            <div class="mode-body">
              <div class="mode-title">{{ opt.title }}</div>
              <div class="mode-desc">{{ opt.description }}</div>
            </div>
            <div class="mode-lock">
              <v-icon icon="mdi-lock-outline" size="16" color="grey" />
            </div>
          </div>
          <button
            v-else
            type="button"
            class="mode-card mode-card--danger"
            :class="{ 'is-selected': selected === opt.value }"
            @click="selectMode(opt.value)"
          >
            <div class="mode-icon mode-icon--danger">
              <v-icon :icon="opt.icon" size="22" />
            </div>
            <div class="mode-body">
              <div class="mode-title">{{ opt.title }}</div>
              <div class="mode-desc">{{ opt.description }}</div>
            </div>
            <div class="mode-radio mode-radio--danger" aria-hidden="true">
              <span class="mode-radio-dot" />
            </div>
          </button>
        </template>
      </div>
      <div
        v-if="showDeleteHint"
        class="delete-info"
      >
        <v-icon size="16" color="info" class="mr-2">mdi-information-outline</v-icon>
        {{ deleteHintText }}
      </div>
    </section>

    <section class="intro-section">
      <v-card class="pa-5 deps-card">
        <div class="mt-n1 deps-head">
          <span class="section-eyebrow eyebrow">{{ t('connectivity.title') }}</span>
          <v-btn
            variant="outlined"
            size="small"
            prepend-icon="mdi-refresh"
            :disabled="connectivity.status === 'checking'"
            @click="connectivity.check()"
            class="deps-retry-btn"
          >
            {{ t('connectivity.retry') }}
          </v-btn>
        </div>

        <div class="conn-row">
          <div class="conn-left">
            <label class="conn-checkbox">
              <input
                type="checkbox"
                :checked="connectivity.offlineMode === true"
                :disabled="connectivity.status === 'checking'"
                @change="connectivity.setOfflineMode(($event.target as HTMLInputElement).checked)"
              />
              <span>{{ t('connectivity.offlineMode') }}</span>
            </label>
            <HelpTooltip text-key="connectivity.offlineHint" />
          </div>

          <div class="conn-status" :class="'is-' + connectivity.status">
            <v-progress-circular
              v-if="connectivity.status === 'checking'"
              indeterminate
              size="16"
              width="2"
              color="primary"
            />
            <v-icon
              v-else
              size="16"
              :color="connectivity.status === 'online' ? 'success' : 'error'"
            >
              {{ connectivity.status === 'online' ? 'mdi-wifi-check' : 'mdi-wifi-off' }}
            </v-icon>
            <span class="conn-status-text">
              {{
                connectivity.status === 'checking'
                  ? t('connectivity.checking')
                  : connectivity.status === 'online'
                    ? t('connectivity.online')
                    : t('connectivity.offline')
              }}
            </span>
          </div>
        </div>
      </v-card>
    </section>

    <section class="intro-section">
      <v-card class="pa-5 deps-card">
        <div class="mt-n1 deps-head">
          <span class="section-eyebrow eyebrow">{{ t('intro.depsTitle') }}</span>
          <v-btn
            variant="outlined"
            size="small"
            prepend-icon="mdi-refresh"
            @click="retryDeps"
            class="deps-retry-btn"
          >
            {{ t('deps.retry') }}
          </v-btn>
        </div>

        <DepsPanel
          :deps="deps"
          :loading="depsLoading"
          :bundled="depsBundled"
          @toggle-instruction="toggleInstruction"
          @skip="skipDep"
          @unskip="unskipDep"
        />

        <div
          v-if="selected && !canProceed && !depsLoading"
          class="deps-warning"
        >
          <v-icon size="16" color="warning" class="mr-1">mdi-alert-outline</v-icon>
          {{ t('deps.skippedWarning') }}
        </div>
      </v-card>
    </section>
  </div>
</template>

<style scoped>
.intro-shell {
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.intro-head { padding: 4px 2px 0; }
.intro-eyebrow {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.intro-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.intro-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
}

.intro-section { display: flex; flex-direction: column; gap: 12px; }
.section-head { padding-left: 4px; }

.mode-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.mode-grid--single { grid-template-columns: minmax(0, 1fr); }

.mode-card {
  display: grid;
  grid-template-columns: 44px 1fr 22px;
  align-items: flex-start;
  gap: 14px;
  text-align: left;
  padding: 18px;
  background: #fff;
  border: 1.5px solid #e2e8f0;
  border-radius: 14px;
  cursor: pointer;
  transition: all .18s ease;
  font: inherit;
  color: inherit;
}
.mode-card:hover {
  border-color: #cbd5e1;
  box-shadow: var(--shadow-1);
}
.mode-card.is-selected {
  border-color: #1f6dd9;
  background: linear-gradient(180deg, #f5f9ff 0%, #ffffff 100%);
  box-shadow: 0 0 0 4px rgba(31, 109, 217, 0.10), 0 1px 3px rgba(15,23,42,.06);
}
.mode-card--danger.is-selected {
  border-color: #dc2626;
  background: linear-gradient(180deg, #fff5f5 0%, #ffffff 100%);
  box-shadow: 0 0 0 4px rgba(220, 38, 38, 0.10), 0 1px 3px rgba(15,23,42,.06);
}

.mode-icon {
  width: 44px; height: 44px;
  border-radius: 12px;
  background: #eef4ff;
  color: #1f6dd9;
  display: flex; align-items: center; justify-content: center;
  flex: 0 0 auto;
}
.mode-icon--danger {
  background: #fee2e2;
  color: #dc2626;
}

.mode-body { min-width: 0; }
.mode-title {
  font-size: 14.5px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: -0.01em;
  margin-bottom: 4px;
  line-height: 1.3;
}
.mode-desc {
  font-size: 12.5px;
  color: #64748b;
  line-height: 1.5;
}

.mode-radio {
  width: 20px; height: 20px;
  border-radius: 999px;
  border: 1.5px solid #cbd5e1;
  position: relative;
  flex: 0 0 auto;
  margin-top: 2px;
  transition: all .18s ease;
}
.mode-radio-dot {
  position: absolute;
  inset: 4px;
  border-radius: 999px;
  background: transparent;
  transition: background .18s ease;
}
.mode-card.is-selected .mode-radio { border-color: #1f6dd9; }
.mode-card.is-selected .mode-radio-dot { background: #1f6dd9; }
.mode-card--danger.is-selected .mode-radio { border-color: #dc2626; }
.mode-card--danger.is-selected .mode-radio-dot { background: #dc2626; }

.mode-card--disabled {
  opacity: 0.6;
  cursor: not-allowed !important;
}
.mode-card--disabled:hover {
  border-color: #e2e8f0;
  box-shadow: none;
}
.mode-card--disabled .mode-body {
  opacity: 0.7;
}
.mode-lock {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  margin-top: 2px;
}

.conn-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.conn-left {
  display: flex;
  align-items: center;
  gap: 2px;
}
.conn-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13.5px;
  color: #334155;
  cursor: pointer;
  font-family: inherit;
}
.conn-checkbox input { accent-color: #1f6dd9; cursor: pointer; }
.conn-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
  color: #475569;
}
.conn-status.is-online { color: #047857; }
.conn-status.is-offline { color: #b91c1c; }
.conn-status-text { line-height: 1.3; }

.deps-card { background: rgb(var(--v-theme-surface)); }
.deps-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  gap: 12px;
}

.deps-retry-btn {
  flex-shrink: 0;
  height: 28px !important;
  min-height: 28px !important;
}

.deps-warning {
  display: flex;
  align-items: center;
  margin-top: 12px;
  padding: 10px 12px;
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: 8px;
  font-size: 12.5px;
  color: #92400e;
}

.delete-info {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  font-size: 12.5px;
  color: #1e40af;
}
</style>
