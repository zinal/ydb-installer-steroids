// Layout contract tests — every screen must share ONE centered content
// column (see plans/layout-column-unification.md). The column is defined
// once in styles/app.css via design tokens + the `.layout-column` utility
// and applied to the content canvas and the footer inner.
// Per-screen max-width wrappers are forbidden.
// NOTE: The header (AppHeader) intentionally stays at full width with
// 24px padding and is NOT constrained to the shared column.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, type Component } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { VApp } from 'vuetify/components'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import AppFooter from '@/components/layout/AppFooter.vue'

const appCss = readFileSync(resolve(__dirname, '../src/styles/app.css'), 'utf-8')
const appVue = readFileSync(resolve(__dirname, '../src/App.vue'), 'utf-8')

vi.mock('@/api/client', () => ({
  api: {
    getSession: vi.fn().mockResolvedValue({}),
    updateSession: vi.fn().mockResolvedValue({}),
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: 'en' },
  }),
}))

vi.mock('@/plugins/i18n', () => ({
  setLocale: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

// v-footer with the `app` prop requires Vuetify's layout injection,
// which a bare mount lacks — wrap the component in a VApp.
function mountInApp(component: Component) {
  const Wrapper = defineComponent({
    render: () => h(VApp, null, { default: () => h(component) }),
  })
  return mount(Wrapper)
}

describe('shared layout column', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('app.css defines the column tokens and the .layout-column utility', () => {
    expect(appCss).toContain('--content-max-width')
    expect(appCss).toContain('--content-gutter')
    expect(appCss).toContain('.layout-column')
    expect(appCss).toMatch(/\.layout-column\s*\{[^}]*max-width:\s*var\(--content-max-width\)/s)
    expect(appCss).toMatch(/\.layout-column\s*\{[^}]*margin-inline:\s*auto/s)
  })

  it('App.vue applies .layout-column to the content canvas', () => {
    expect(appVue).toContain('content-canvas layout-column')
    expect(appVue).toMatch(/padding:\s*40px\s+var\(--content-gutter\)\s+56px/)
  })

  it('AppFooter renders its inner on the shared column', () => {
    const wrapper = mountInApp(AppFooter)
    const inner = wrapper.find('.footer-inner')
    expect(inner.exists()).toBe(true)
    expect(inner.classes()).toContain('layout-column')
  })
})
