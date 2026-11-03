<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/plugins/i18n'
import AppVersion from '@/components/common/AppVersion.vue'
import logoYdb from '@/assets/logo-ydb.svg'

const { t, locale } = useI18n()
const langs = ['ru', 'en'] as const

const current = computed({
  get: () => locale.value as 'ru' | 'en',
  set: (v) => setLocale(v),
})
</script>

<template>
  <v-app-bar
    app
    :elevation="0"
    color="surface"
    height="64"
    class="app-header"
  >
    <div class="header-inner">
      <div class="brand">
        <img :src="logoYdb" alt="YDB" class="brand-logo" />
        <div class="brand-text">
          <div class="brand-title">{{ t('app.title') }}</div>
          <AppVersion class="brand-version" />
        </div>
      </div>

      <v-spacer />

      <a
        href="https://st.yandex-team.ru/YDBOPS-14824"
        target="_blank"
        rel="noopener noreferrer"
        class="bug-link"
        :title="t('header.bugTracker')"
      >
        <v-icon size="20" color="#f00000">mdi-bug</v-icon>
      </a>

      <div class="lang-switch" role="group" :aria-label="t('common.language')">
        <button
          v-for="l in langs"
          :key="l"
          type="button"
          class="lang-btn"
          :class="{ 'is-active': current === l }"
          @click="current = l"
        >
          {{ l.toUpperCase() }}
        </button>
      </div>
    </div>
  </v-app-bar>
</template>

<style scoped>
.app-header :deep(.v-toolbar__content) {
  padding: 0;
}

.header-inner {
  display: flex;
  align-items: center;
  width: 100%;
  padding: 0 24px;
  gap: 16px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 14px;
}

.brand-logo {
  height: 32px;
  width: auto;
  display: block;
}

.brand-text {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.brand-title {
  font-weight: 600;
  font-size: 15px;
  letter-spacing: -0.01em;
  color: #0f172a;
}

.brand-version {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 1px;
}

.lang-switch {
  display: inline-flex;
  background: #f1f5f9;
  border-radius: 10px;
  padding: 3px;
  gap: 2px;
}
.bug-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  transition: background .15s ease;
}
.bug-link:hover {
  background: #f1f5f9;
}
.bug-link:hover .v-icon {
  color: #0f172a !important;
}
.lang-btn {
  border: 0;
  background: transparent;
  padding: 6px 14px;
  border-radius: 8px;
  font: 600 12px/1 'Inter', sans-serif;
  letter-spacing: 0.04em;
  color: #64748b;
  cursor: pointer;
  transition: all .15s ease;
}
.lang-btn:hover { color: #0f172a; }
.lang-btn.is-active {
  background: #fff;
  color: #2399FF;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}
</style>
