import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ProgressBar from '@/components/common/ProgressBar.vue'
import type { InstallStatus } from '@/types/install'

const i18n = createI18n({
  legacy: false, locale: 'en',
  messages: {
    en: {
      install: {
        progress: '{done} / {total} tasks',
        errorTitle: 'Installation error',
      },
    },
  },
})

function mountBar(progress: number, completed: number, total: number, status: InstallStatus) {
  return mount(ProgressBar, {
    props: { progress, completedTasks: completed, totalTasks: total, status },
    global: { plugins: [i18n] },
  })
}

describe('ProgressBar', () => {
  it('shows the percentage and task count label', () => {
    const wrapper = mountBar(45, 3, 7, 'running')
    expect(wrapper.find('.progress-pct').text()).toBe('45%')
    expect(wrapper.find('.progress-label').text()).toBe('3 / 7 tasks')
  })

  it('shows success label for the success status', () => {
    const wrapper = mountBar(100, 5, 5, 'success')
    expect(wrapper.find('.progress-label').text()).toContain('✅')
  })

  it('shows error label for the error status', () => {
    const wrapper = mountBar(33, 1, 3, 'error')
    expect(wrapper.find('.progress-label').text()).toContain('❌')
  })
})
