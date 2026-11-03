<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  /** i18n key for the tooltip body text */
  textKey: string
  /** URL to YDB documentation page (absolute) */
  docLink?: string
  /** Override tooltip max-width (default 400px) */
  maxWidth?: number
}>(), {
  docLink: undefined,
  maxWidth: 400,
})

const { t } = useI18n()

const tooltipMaxWidth = computed(() => `${props.maxWidth}px`)
</script>

<template>
  <v-menu
    open-on-hover
    :close-on-content-click="false"
    :open-delay="200"
    :close-delay="200"
    location="top"
  >
    <template #activator="{ props: activatorProps }">
      <span
        class="help-tooltip-wrapper"
        v-bind="activatorProps"
      >
        <v-icon
          icon="mdi-help-circle-outline"
          size="16"
          class="help-icon"
        />
      </span>
    </template>
    <v-card
      :width="tooltipMaxWidth"
      class="help-tooltip-card"
      elevation="4"
      rounded="lg"
    >
      <v-card-text class="help-tooltip-body">
        <p class="help-tooltip-text">{{ t(textKey) }}</p>
        <a
          v-if="docLink"
          :href="docLink"
          target="_blank"
          rel="noopener"
          class="help-tooltip-link"
        >
          {{ t('help.docsLink') }}
        </a>
      </v-card-text>
    </v-card>
  </v-menu>
</template>

<style scoped>
.help-tooltip-wrapper {
  display: inline-flex;
  align-items: center;
  vertical-align: middle;
  margin-left: 6px;
  margin-top: -2px;
  cursor: help;
}

.help-icon {
  color: #94a3b8 !important;
  transition: color 0.15s ease;
}

.help-tooltip-wrapper:hover .help-icon {
  color: #1f6dd9 !important;
}

.help-tooltip-card {
  background: #f1f5f9 !important;
  border: 1px solid #e2e8f0;
}

.help-tooltip-body {
  padding: 10px 14px !important;
}

.help-tooltip-text {
  margin: 0 0 8px 0;
  font-size: 13px;
  font-weight: 400;
  line-height: 1.5;
  color: #334155;
}

.help-tooltip-text:last-child {
  margin-bottom: 0;
}

.help-tooltip-link {
  display: inline-block;
  font-size: 12px;
  font-weight: 500;
  color: #1f6dd9;
  text-decoration: none;
  transition: color 0.15s ease;
}

.help-tooltip-link:hover {
  color: #1a5cb3;
  text-decoration: underline;
}
</style>