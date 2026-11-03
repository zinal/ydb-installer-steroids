import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from '@/stores/session'
import AppHeader from '@/components/layout/AppHeader.vue'

vi.mock('@/plugins/i18n', () => ({ setLocale: vi.fn() }))
import { setLocale } from '@/plugins/i18n'

const i18n = createI18n({
  legacy: false, locale: 'en',
  messages: {
    en: {
      app: { title: 'YDB Installer', beta: 'BETA' },
      header: { bugTracker: 'Report a bug' },
      common: { language: 'Language' },
    },
  },
})

function mountHeader() {
  return mount(
    { template: '<v-app><AppHeader /></v-app>', components: { AppHeader } },
    { global: { plugins: [i18n] } },
  )
}

describe('AppHeader', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the brand title and beta badge after the session version', () => {
    useSessionStore().version = 'v9.9.9'
    const wrapper = mountHeader()
    const version = wrapper.find('.brand-version .app-version__number')
    const beta = wrapper.find('.brand-version .app-version__badge')

    expect(wrapper.find('.brand-title').text()).toBe('YDB Installer')
    expect(version.text()).toBe('v9.9.9')
    expect(beta.text()).toBe('BETA')
    expect(version.element.nextElementSibling).toBe(beta.element)
    expect(wrapper.text()).not.toContain('BETA VERSION FOR INTERNAL USE ONLY')
  })

  it('calls setLocale when clicking a language button', async () => {
    const wrapper = mountHeader()
    // langs = ['ru', 'en'] → index 0 is RU.
    const buttons = wrapper.findAll('.lang-btn')
    await buttons[0].trigger('click')
    expect(setLocale).toHaveBeenCalledWith('ru')
  })

  it('highlights the active language', () => {
    const wrapper = mountHeader()
    const active = wrapper.find('.lang-btn.is-active')
    expect(active.text()).toBe('EN')
  })

  it('renders the bug tracker link', () => {
    const wrapper = mountHeader()
    expect(wrapper.find('.bug-link').exists()).toBe(true)
  })
})
