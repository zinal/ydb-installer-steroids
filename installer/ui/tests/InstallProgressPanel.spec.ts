import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import enMessages from '@/locales/en.json'
import InstallProgressPanel from '@/components/common/InstallProgressPanel.vue'
import { useUiStore } from '@/stores/ui'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: enMessages },
})

const baseProps = {
  status: 'running' as const,
  progress: 50,
  tasks: [
    { id: 1, name: 'first task', status: 'done' as const },
    { id: 2, name: 'second task', status: 'running' as const },
  ],
  completedTasks: 1,
  error: null,
  logs: ['line one', 'line two'],
  searchQuery: '',
  logStatus: 'streaming' as const,
  cancelledTitle: 'Cancelled',
  cancelledMessage: 'It was stopped.',
  cancelTitle: 'Confirm',
  cancelMessage: 'Really stop?',
  cancelContinueLabel: 'Continue',
  cancelConfirmLabel: 'Stop',
}

function mountPanel(overrides: Record<string, unknown> = {}) {
  return mount(InstallProgressPanel, {
    props: { ...baseProps, ...overrides },
    global: { plugins: [i18n] },
  })
}

describe('InstallProgressPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders nothing when idle', () => {
    const wrapper = mountPanel({ status: 'idle' })
    expect(wrapper.find('.progress-shell').exists()).toBe(false)
    expect(wrapper.text()).toBe('')
  })

  it('renders nothing when successful (the screen owns the success card)', () => {
    const wrapper = mountPanel({ status: 'success' })
    expect(wrapper.find('.progress-shell').exists()).toBe(false)
  })

  it('renders progress, tasks and console when running', () => {
    const wrapper = mountPanel()
    expect(wrapper.find('.progress-shell').exists()).toBe(true)
    expect(wrapper.text()).toContain('first task')
    expect(wrapper.text()).toContain('second task')
    expect(wrapper.text()).toContain('Console')
  })

  it('collapses the console by default', () => {
    const wrapper = mountPanel()
    expect(useUiStore().consoleExpanded).toBe(false)
    expect(wrapper.find('.console-header .v-icon').classes()).toContain('mdi-chevron-down')
  })

  it('expands the console when the header is clicked', async () => {
    const wrapper = mountPanel()
    await wrapper.find('.console-header').trigger('click')
    expect(useUiStore().consoleExpanded).toBe(true)
    expect(wrapper.find('.console-header .v-icon').classes()).toContain('mdi-chevron-up')
  })

  it('collapses the console again on a second click', async () => {
    const wrapper = mountPanel()
    await wrapper.find('.console-header').trigger('click')
    await wrapper.find('.console-header').trigger('click')
    expect(useUiStore().consoleExpanded).toBe(false)
    expect(wrapper.find('.console-header .v-icon').classes()).toContain('mdi-chevron-down')
  })

  it('shows the stop button only while running', () => {
    expect(mountPanel().find('.cancel-row').exists()).toBe(true)
    expect(mountPanel({ status: 'error' }).find('.cancel-row').exists()).toBe(false)
    expect(mountPanel({ status: 'cancelled' }).find('.cancel-row').exists()).toBe(false)
  })

  it('opens the confirmation dialog with the supplied wording', async () => {
    const wrapper = mountPanel()
    await wrapper.find('.cancel-btn').trigger('click')
    const dialog = wrapper.findComponent({ name: 'ConfirmDialog' })
    expect(dialog.props('title')).toBe('Confirm')
    expect(dialog.props('message')).toBe('Really stop?')
  })

  it('emits cancel when the dialog is confirmed', async () => {
    const wrapper = mountPanel()
    await wrapper.find('.cancel-btn').trigger('click')
    await wrapper.findComponent({ name: 'ConfirmDialog' }).vm.$emit('confirm')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })

  it('shows the error card with task, host and reason', () => {
    const wrapper = mountPanel({
      status: 'error',
      error: { task: 'initial_setup', host: 'node1', reason: 'exploded' },
    })
    expect(wrapper.find('.error-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('initial_setup')
    expect(wrapper.text()).toContain('node1')
    expect(wrapper.text()).toContain('exploded')
  })

  it('omits the host row when no host is reported', () => {
    const wrapper = mountPanel({
      status: 'error',
      error: { task: 'initial_setup', host: '', reason: 'exploded' },
    })
    expect(wrapper.text()).not.toContain('Host')
  })

  it('emits restart from the error state', async () => {
    const wrapper = mountPanel({
      status: 'error',
      error: { task: 't', host: '', reason: 'r' },
    })
    const btn = wrapper.findAll('button').find(b => b.text().includes('Restart'))
    expect(btn).toBeDefined()
    await btn!.trigger('click')
    expect(wrapper.emitted('restart')).toBeTruthy()
  })

  it('uses a custom restart label when provided', () => {
    const wrapper = mountPanel({
      status: 'cancelled',
      restartLabel: 'Restart Deletion',
    })
    expect(wrapper.text()).toContain('Restart Deletion')
    expect(wrapper.text()).not.toContain('Restart Installation')
  })

  it('shows the cancelled card with the supplied wording', () => {
    const wrapper = mountPanel({ status: 'cancelled' })
    expect(wrapper.find('.cancelled-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('Cancelled')
    expect(wrapper.text()).toContain('It was stopped.')
    expect(wrapper.find('.error-card').exists()).toBe(false)
  })

  it('offers log recovery only when the stream dropped', () => {
    const streaming = mountPanel()
    expect(streaming.text()).not.toContain('Recover')

    const dropped = mountPanel({ logStatus: 'disconnected', logsRecovered: false })
    expect(dropped.text()).toContain('Recover')
  })

  it('hides log recovery once logs were already recovered', () => {
    const wrapper = mountPanel({ logStatus: 'disconnected', logsRecovered: true })
    expect(wrapper.text()).not.toContain('Recover')
  })

  it('emits recover-logs when the recovery button is clicked', async () => {
    const wrapper = mountPanel({ logStatus: 'disconnected', logsRecovered: false })
    const btn = wrapper.findAll('button').find(b => b.text().includes('Recover'))
    expect(btn).toBeDefined()
    await btn!.trigger('click')
    expect(wrapper.emitted('recover-logs')).toBeTruthy()
  })

  it('re-emits console search changes for two-way binding', async () => {
    const wrapper = mountPanel()
    const consoleOutput = wrapper.findComponent({ name: 'ConsoleOutput' })
    await consoleOutput.vm.$emit('update:searchQuery', 'needle')
    expect(wrapper.emitted('update:searchQuery')).toBeTruthy()
    expect(wrapper.emitted('update:searchQuery')![0]).toEqual(['needle'])
  })
})
