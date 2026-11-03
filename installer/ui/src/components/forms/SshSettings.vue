<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SshConfig } from '@/types/install'
import { HELP_LINKS } from '@/config/helpLinks'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

const props = defineProps<{
  modelValue: SshConfig
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: SshConfig): void
}>()

const { t } = useI18n()

function update<K extends keyof SshConfig>(key: K, value: SshConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

const showJumpHost = computed(() => props.modelValue.useJumpHost)
const specifyKey = ref(!!props.modelValue.keyPath)
const advancedOpen = ref(false)
</script>

<template>
  <div class="ssh-form">
    <div class="ssh-field">
      <label class="ssh-label">{{ t('ssh.login') }}</label>
      <input
        type="text"
        class="ssh-input"
        :value="modelValue.login"
        @input="update('login', ($event.target as HTMLInputElement).value)"
        autocomplete="username"
      />
    </div>

    <div class="ssh-field">
      <label class="ssh-checkbox">
        <input
          type="checkbox"
          :checked="specifyKey"
          @change="specifyKey = !specifyKey; if (!specifyKey) update('keyPath', undefined)"
        />
        <span>{{ t('ssh.specifyKey') }}</span>
      </label>
    </div>

    <div v-if="specifyKey" class="ssh-field ssh-field--indent">
      <label class="ssh-label">{{ t('ssh.keyPath') }}</label>
      <input
        type="text"
        class="ssh-input"
        :value="modelValue.keyPath ?? ''"
        :placeholder="'~/.ssh/id_rsa'"
        @input="update('keyPath', ($event.target as HTMLInputElement).value)"
        autocomplete="off"
      />
    </div>

    <div class="ssh-field">
      <label class="ssh-checkbox">
        <input
          type="checkbox"
          :checked="modelValue.useJumpHost"
          @change="update('useJumpHost', ($event.target as HTMLInputElement).checked)"
        />
        <span>{{ t('ssh.useJumpHost') }}</span>
      </label>
    </div>

    <div v-if="showJumpHost" class="ssh-field ssh-field--indent">
      <label class="ssh-label">
        {{ t('ssh.jumpHostFqdn') }}
        <HelpTooltip text-key="help.jumpHost" :doc-link="HELP_LINKS.jumpHost" />
      </label>
      <input
        type="text"
        class="ssh-input"
        :value="modelValue.jumpHost ?? ''"
        :placeholder="'bastion.example.com'"
        @input="update('jumpHost', ($event.target as HTMLInputElement).value)"
        autocomplete="off"
      />
    </div>

    <div class="ssh-advanced">
      <button
        type="button"
        class="ssh-advanced-toggle"
        @click="advancedOpen = !advancedOpen"
      >
        <v-icon size="14" class="mr-1">
          {{ advancedOpen ? 'mdi-chevron-down' : 'mdi-chevron-right' }}
        </v-icon>
        {{ t('ssh.advancedSettings') }}
      </button>

      <div v-if="advancedOpen" class="ssh-field ssh-field--indent">
        <label class="ssh-label">{{ t('ssh.customSshArgs') }}</label>
        <textarea
          class="ssh-textarea"
          :value="modelValue.customSshArgs ?? ''"
          :placeholder="'-o StrictHostKeyChecking=no -o ConnectTimeout=10'"
          rows="2"
          @input="update('customSshArgs', ($event.target as HTMLTextAreaElement).value)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
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

.ssh-field--indent {
  padding-left: 20px;
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

.ssh-textarea {
  padding: 8px 12px;
  border: 1.5px solid #e2e8f0;
  border-radius: 8px;
  font-size: 12.5px;
  color: #0f172a;
  background: #fff;
  outline: none;
  transition: border-color .15s ease;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  width: 100%;
  box-sizing: border-box;
  resize: vertical;
}
.ssh-textarea:focus { border-color: #1f6dd9; }

.ssh-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13.5px;
  color: #334155;
  cursor: pointer;
}
.ssh-checkbox input { accent-color: #1f6dd9; cursor: pointer; }

.ssh-advanced-toggle {
  display: inline-flex;
  align-items: center;
  background: none;
  border: none;
  padding: 0;
  font-size: 12.5px;
  color: #64748b;
  cursor: pointer;
  font-family: inherit;
  transition: color .15s ease;
}
.ssh-advanced-toggle:hover { color: #1f6dd9; }
</style>
