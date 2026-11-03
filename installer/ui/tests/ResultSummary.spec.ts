import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import enMessages from '@/locales/en.json'
import ResultSummary from '@/components/common/ResultSummary.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: enMessages },
})

const baseProps = {
  state: 'success' as const,
  successTitle: 'It worked',
  successMessage: 'All good.',
  errorTitle: 'It failed',
  errorMessage: 'Something broke.',
  noResultMessage: 'Nothing to show.',
  logs: ['line one', 'line two', 'line three'],
  searchQuery: '',
  logStatus: 'none' as const,
  logPath: '/tmp/ydb-installer/logs/install-2024-01-15.log',
}

function mountSummary(overrides: Record<string, unknown> = {}, slots: Record<string, string> = {}) {
  return mount(ResultSummary, {
    props: { ...baseProps, ...overrides },
    slots,
    global: { plugins: [i18n] },
  })
}

describe('ResultSummary', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('shows the success headline when successful', () => {
    const wrapper = mountSummary()
    expect(wrapper.find('.success-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('It worked')
    expect(wrapper.text()).toContain('All good.')
    expect(wrapper.find('.error-card').exists()).toBe(false)
  })

  it('shows the error headline when failed', () => {
    const wrapper = mountSummary({ state: 'error' })
    expect(wrapper.find('.error-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('It failed')
    expect(wrapper.find('.success-card').exists()).toBe(false)
  })

  it('shows the neutral placeholder when idle', () => {
    const wrapper = mountSummary({ state: 'idle' })
    expect(wrapper.text()).toContain('Nothing to show.')
    expect(wrapper.find('.success-card').exists()).toBe(false)
    expect(wrapper.find('.error-card').exists()).toBe(false)
  })

  it('renders flow-specific detail passed through the slot', () => {
    const wrapper = mountSummary({}, { detail: '<div class="my-detail">cluster info</div>' })
    expect(wrapper.find('.my-detail').exists()).toBe(true)
    expect(wrapper.text()).toContain('cluster info')
  })

  it('shows the logs card with a line count when there is a result', () => {
    const wrapper = mountSummary()
    expect(wrapper.find('.logs-header').exists()).toBe(true)
    expect(wrapper.text()).toContain('3')
  })

  it('hides the logs card when idle', () => {
    const wrapper = mountSummary({ state: 'idle' })
    expect(wrapper.find('.logs-header').exists()).toBe(false)
  })

  it('starts with the logs collapsed', () => {
    const wrapper = mountSummary()
    expect(wrapper.findComponent({ name: 'ConsoleOutput' }).isVisible()).toBe(false)
    expect(wrapper.text()).toContain('Show logs')
  })

  it('expands and collapses the logs when the header is clicked', async () => {
    const wrapper = mountSummary()
    await wrapper.find('.logs-header').trigger('click')
    expect(wrapper.text()).toContain('Hide logs')
    expect(wrapper.findComponent({ name: 'ConsoleOutput' }).isVisible()).toBe(true)

    await wrapper.find('.logs-header').trigger('click')
    expect(wrapper.text()).toContain('Show logs')
  })

  it('re-emits console search changes for two-way binding', async () => {
    const wrapper = mountSummary()
    await wrapper.findComponent({ name: 'ConsoleOutput' }).vm.$emit('update:searchQuery', 'needle')
    expect(wrapper.emitted('update:searchQuery')![0]).toEqual(['needle'])
  })

  it('shows the log path with a copy button on error', () => {
    const wrapper = mountSummary({ state: 'error' })
    expect(wrapper.text()).toContain('/tmp/ydb-installer/logs/install-2024-01-15.log')
    expect(wrapper.text()).toContain('Copy path')
  })

  it('does not show the log path on success', () => {
    const wrapper = mountSummary()
    expect(wrapper.find('.log-path').exists()).toBe(false)
  })

  it('omits the log path row when no path is supplied', () => {
    const wrapper = mountSummary({ state: 'error', logPath: '' })
    expect(wrapper.find('.log-path').exists()).toBe(false)
  })

  it('copies the log path to the clipboard and confirms', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })

    const wrapper = mountSummary({ state: 'error' })
    const btn = wrapper.findAll('button').find(b => b.text().includes('Copy path'))
    expect(btn).toBeDefined()
    await btn!.trigger('click')

    expect(writeText).toHaveBeenCalledWith('/tmp/ydb-installer/logs/install-2024-01-15.log')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Copied')
  })

  it('survives a clipboard failure without throwing', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('denied'))
    Object.assign(navigator, { clipboard: { writeText } })

    const wrapper = mountSummary({ state: 'error' })
    const btn = wrapper.findAll('button').find(b => b.text().includes('Copy path'))
    await btn!.trigger('click')
    await wrapper.vm.$nextTick()

    // The confirmation must NOT appear when the copy failed.
    expect(wrapper.text()).not.toContain('Copied')
  })
})
