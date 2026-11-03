<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  /** Resolved button text. Defaults to install restart wording. */
  label?: string
}>(), {
  label: undefined,
})

const { t } = useI18n()
const buttonLabel = computed(() => props.label ?? t('install.restartInstall'))

defineEmits<{
  restart: []
}>()
</script>

<template>
  <div class="restart-row">
    <v-btn
      color="primary"
      variant="elevated"
      size="default"
      @click="$emit('restart')"
    >
      <v-icon start size="18">mdi-restart</v-icon>
      {{ buttonLabel }}
    </v-btn>
  </div>
</template>

<style scoped>
.restart-row {
  display: flex;
  justify-content: center;
  padding: 12px 0 0;
}
</style>