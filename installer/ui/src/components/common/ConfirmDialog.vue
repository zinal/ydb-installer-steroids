<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  modelValue: boolean
  title?: string
  message?: string
  confirmLabel?: string
  cancelLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

const { t } = useI18n()

function onConfirm() {
  emit('confirm')
  emit('update:modelValue', false)
}

function onCancel() {
  emit('cancel')
  emit('update:modelValue', false)
}
</script>

<template>
  <v-dialog
    :model-value="modelValue"
    max-width="440"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card class="dialog-card">
      <div class="dialog-icon">
        <v-icon size="28" color="warning">mdi-alert-outline</v-icon>
      </div>

      <v-card-title class="dialog-title">
        {{ title ?? t('install.cancelTitle') }}
      </v-card-title>

      <v-card-text class="dialog-text">
        {{ message ?? t('install.cancelMessage') }}
      </v-card-text>

      <v-card-actions class="dialog-actions">
        <v-btn
          variant="tonal"
          color="primary"
          @click="onCancel"
          class="dialog-btn"
        >
          {{ cancelLabel ?? t('install.continueBtn') }}
        </v-btn>
        <v-btn
          variant="flat"
          color="error"
          @click="onConfirm"
          class="dialog-btn"
        >
          {{ confirmLabel ?? t('install.stopConfirmBtn') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.dialog-card {
  padding: 28px 28px 20px;
  border-radius: 16px !important;
}

.dialog-icon {
  display: flex;
  justify-content: center;
  margin-bottom: 16px;
}

.dialog-title {
  font-size: 17px !important;
  font-weight: 700 !important;
  color: #0f172a;
  text-align: center;
  padding: 0 0 10px !important;
  letter-spacing: -0.02em;
}

.dialog-text {
  font-size: 13.5px !important;
  color: #475569;
  text-align: center;
  line-height: 1.6;
  padding: 0 0 20px !important;
}

.dialog-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  padding: 0 !important;
}

.dialog-btn {
  flex: 1;
  font-weight: 600 !important;
  letter-spacing: -0.005em;
}
</style>
