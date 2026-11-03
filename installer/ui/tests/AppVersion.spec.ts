import { beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import AppVersion from '@/components/common/AppVersion.vue'
import { useSessionStore } from '@/stores/session'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: { app: { beta: 'BETA' } },
  },
})

function mountVersion() {
  return mount(AppVersion, {
    global: { plugins: [i18n] },
  })
}

describe('AppVersion', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders beta immediately after the version from the session', () => {
    useSessionStore().version = 'v0.4.3'
    const wrapper = mountVersion()
    const version = wrapper.find('.app-version__number')
    const beta = wrapper.find('.app-version__badge')

    expect(version.text()).toBe('v0.4.3')
    expect(beta.text()).toBe('BETA')
    expect(version.element.nextElementSibling).toBe(beta.element)
  })

  it('keeps the beta badge visible while the version is unavailable', () => {
    const wrapper = mountVersion()

    expect(wrapper.find('.app-version__number').text()).toBe('—')
    expect(wrapper.find('.app-version__badge').text()).toBe('BETA')
  })
})
