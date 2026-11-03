<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/session'
import { useEmStore } from '@/stores/em'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'

const session = useSessionStore()
const emStore = useEmStore()
const ydb = useYdbConfigStore()
const { t } = useI18n()

const showPassword = ref(false)

const user = computed({
  get: () => ydb.user,
  set: (v: string) => { ydb.user = v },
})

const password = computed({
  get: () => ydb.password,
  set: (v: string) => { ydb.password = v },
})

const dbRoot = computed({
  get: () => emStore.dbConnectionDBRoot,
  set: (v: string) => emStore.setDbConnectionDBRoot(v),
})

const dbName = computed({
  get: () => emStore.dbConnectionDB,
  set: (v: string) => emStore.setDbConnectionDB(v),
})

const credentialsValid = computed(() => user.value.trim() !== '')
const hasEmHost = computed(() => emStore.firstEmHost !== '')

const fieldsReady = computed(() =>
  credentialsValid.value
  && emStore.paramsFieldsValid
  && hasEmHost.value,
)

const canProceed = computed(() => fieldsReady.value)

watch(canProceed, (v) => session.setCanGoNext(v), { immediate: true })
</script>

<template>
  <div class="em-params-shell">
    <header class="em-params-head">
      <ScreenEyebrow />
      <h1 class="em-params-title">{{ t('emParams.heading') }}</h1>
      <p class="em-params-sub">{{ t('emParams.subtitle') }}</p>
    </header>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('emParams.credentialsLabel') }}</div>
      <div class="ssh-form">
        <div class="ssh-field">
          <label class="ssh-label">{{ t('emParams.user') }}</label>
          <input
            v-model="user"
            type="text"
            class="ssh-input"
            :placeholder="t('emParams.userPlaceholder')"
            autocomplete="username"
          />
        </div>
        <div class="ssh-field">
          <label class="ssh-label">{{ t('emParams.password') }}</label>
          <div class="password-row">
            <span class="password-input-wrap">
              <input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                class="ssh-input"
                :placeholder="t('emParams.passwordPlaceholder')"
                autocomplete="new-password"
              />
              <v-btn
                variant="text"
                size="small"
                density="compact"
                :icon="showPassword ? 'mdi-eye-off' : 'mdi-eye'"
                class="eye-btn"
                @click="showPassword = !showPassword"
              />
            </span>
          </div>
        </div>
      </div>
    </v-card>

    <v-card class="section-card pa-6 mb-4">
      <div class="section-label mb-4">{{ t('emParams.emDbLabel') }}</div>
      <div class="ssh-form">
        <div class="ssh-field">
          <label class="ssh-label">{{ t('emParams.dbRoot') }}</label>
          <input
            v-model="dbRoot"
            type="text"
            class="ssh-input"
            :placeholder="t('emParams.dbRootPlaceholder')"
            autocomplete="off"
          />
        </div>
        <div class="ssh-field">
          <label class="ssh-label">{{ t('emParams.dbName') }}</label>
          <input
            v-model="dbName"
            type="text"
            class="ssh-input"
            :placeholder="t('emParams.dbNamePlaceholder')"
            autocomplete="off"
          />
        </div>

      </div>
    </v-card>

    <div v-if="!hasEmHost" class="validation-message validation-message--warning">
      <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
      <span>{{ t('emParams.emHostRequired') }}</span>
    </div>
    <div v-else-if="!credentialsValid" class="validation-message validation-message--warning">
      <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
      <span>{{ t('emParams.credentialsRequired') }}</span>
    </div>
    <div v-else-if="!emStore.paramsFieldsValid" class="validation-message validation-message--warning">
      <v-icon size="18" color="#d97706">mdi-alert-outline</v-icon>
      <span>{{ t('emParams.fieldsRequired') }}</span>
    </div>
  </div>
</template>

<style scoped>
.em-params-head { margin-bottom: 28px; }
.em-params-title {
  font-size: 32px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  line-height: 1.15;
  margin: 0 0 8px;
}
.em-params-sub {
  font-size: 15px;
  color: #64748b;
  line-height: 1.55;
  margin: 0;
  max-width: 620px;
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

.password-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.password-input-wrap {
  position: relative;
  flex: 1;
}
.password-input-wrap .ssh-input {
  padding-right: 36px;
}
.eye-btn {
  position: absolute;
  right: 2px;
  top: 50%;
  transform: translateY(-50%);
  min-width: 32px;
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
  margin-bottom: 12px;
}
.validation-message--warning {
  background: #fffbeb;
  border-color: #fcd34d;
  color: #92400e;
}
</style>
