import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import RestartButton from '@/components/common/RestartButton.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      install: {
        restartInstall: 'Restart Installation',
      },
    },
  },
})

describe('RestartButton', () => {
  it('renders default install restart label when no label prop is passed', () => {
    const wrapper = mount(RestartButton, { global: { plugins: [i18n] } })
    expect(wrapper.text()).toContain('Restart Installation')
  })

  it('renders custom label when label prop is provided', () => {
    const wrapper = mount(RestartButton, {
      props: { label: 'Restart Deletion' },
      global: { plugins: [i18n] },
    })
    expect(wrapper.text()).toContain('Restart Deletion')
    expect(wrapper.text()).not.toContain('Restart Installation')
  })

  it('emits restart on click', async () => {
    const wrapper = mount(RestartButton, { global: { plugins: [i18n] } })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('restart')).toHaveLength(1)
  })
})
