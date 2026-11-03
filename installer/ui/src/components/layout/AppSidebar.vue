<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSessionStore, type SidebarItem } from '@/stores/session'

const session = useSessionStore()
const router = useRouter()
const { t } = useI18n()

const items = computed<SidebarItem[]>(() => session.sidebarItems)

function clickable(it: SidebarItem) {
  return it.status === 'clickable' || it.status === 'done' || it.status === 'error' || it.status === 'active'
}
function go(it: SidebarItem) {
  if (!clickable(it)) return
  router.push({ name: it.id })
}
</script>

<template>
  <v-navigation-drawer permanent width="288" color="surface">
    <div class="sidebar-shell">
      <div class="sidebar-eyebrow eyebrow">{{ t('sidebar.title', 'Steps') }}</div>

      <ol class="steps">
        <li
          v-for="(it, idx) in items"
          :key="it.id"
          class="step"
          :class="['step--' + it.status, { 'step--clickable': clickable(it) }]"
          @click="go(it)"
        >
          <span class="bullet" aria-hidden="true">
            <template v-if="it.status === 'done'">
              <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
                <path d="M3.5 8.5l3 3 6-6.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </template>
            <template v-else-if="it.status === 'error'">
              <svg viewBox="0 0 16 16" fill="none" width="12" height="12">
                <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              </svg>
            </template>
            <template v-else>
              <span class="bullet-num">{{ idx + 1 }}</span>
            </template>
          </span>

          <span class="label">
            {{ t(`sidebar.${it.i18nKey}`) }}
          </span>
        </li>
      </ol>
    </div>
  </v-navigation-drawer>
</template>

<style scoped>
.sidebar-shell {
  padding: 24px 20px 32px;
}
.sidebar-eyebrow {
  margin-bottom: 14px;
  padding-left: 4px;
}

.steps {
  list-style: none;
  margin: 0;
  padding: 0;
}

.step {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px 8px 8px;
  margin: 2px 0;
  border-radius: 10px;
  cursor: default;
  transition: background-color .15s ease, color .15s ease;
  user-select: none;
  min-height: 40px;
}
.step--clickable { cursor: pointer; }
.step--clickable:hover { background: #f1f5f9; }

.bullet {
  flex: 0 0 auto;
  width: 26px;
  height: 26px;
  border-radius: 999px;
  background: #fff;
  border: 1.5px solid #cbd5e1;
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: center;
  font: 600 11px/1 'Inter', sans-serif;
  letter-spacing: -0.02em;
  transition: all .18s ease;
}
.bullet-num { font-variant-numeric: tabular-nums; }

.label {
  font-size: 13.5px;
  font-weight: 500;
  color: #475569;
  line-height: 1.25;
  letter-spacing: -0.01em;
}

.step--active .bullet {
  background: #2399FF;
  border-color: #2399FF;
  color: #fff;
  box-shadow: 0 0 0 4px rgba(35, 153, 255, 0.14);
}
.step--active .label   { color: #0f172a; font-weight: 600; }

.step--done .bullet {
  background: #16a34a;
  border-color: #16a34a;
  color: #fff;
}
.step--done .label    { color: #334155; }

.step--error .bullet {
  background: #fee2e2;
  border-color: #dc2626;
  color: #dc2626;
}
.step--error .label   { color: #b91c1c; font-weight: 600; }

.step--clickable:not(.step--active):not(.step--done):not(.step--error) .bullet {
  border-color: #fbbf24;
  color: #b45309;
}
.step--clickable:not(.step--active):not(.step--done):not(.step--error) .label {
  color: #475569;
}

.step--disabled .bullet { border-color: #e2e8f0; color: #94a3b8; }
.step--disabled .label  { color: #94a3b8; }
</style>
