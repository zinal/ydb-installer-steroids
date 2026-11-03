import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ConsoleOutput from '@/components/common/ConsoleOutput.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      install: {
        search: 'Search logs...',
      },
    },
  },
})

describe('ConsoleOutput', () => {
  it('renders log lines', () => {
    const logs = ['[12:00:01] Starting...', '[12:00:02] Done.']
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs },
    })
    expect(wrapper.text()).toContain('[12:00:01] Starting...')
    expect(wrapper.text()).toContain('[12:00:02] Done.')
  })

  it('renders empty state when no logs', () => {
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs: [] },
    })
    expect(wrapper.find('.log-empty').exists()).toBe(true)
  })

  it('renders search input', () => {
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs: [] },
    })
    expect(wrapper.find('.console-search').exists()).toBe(true)
  })

  it('filters logs by search query', async () => {
    const logs = ['[12:00:01] Starting ansible...', '[12:00:02] Copying files...', '[12:00:03] Done.']
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs, searchQuery: '' },
    })

    const search = wrapper.find('.console-search')
    await search.setValue('ansible')

    const lines = wrapper.findAll('.log-line')
    expect(lines).toHaveLength(1)
    expect(lines[0].text()).toContain('ansible')
  })

  it('shows match count when filtering', async () => {
    const logs = ['line with ansible', 'another ansible line', 'unrelated line']
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs, searchQuery: '' },
    })

    const search = wrapper.find('.console-search')
    await search.setValue('ansible')

    expect(wrapper.find('.console-match-count').exists()).toBe(true)
    expect(wrapper.find('.console-match-count').text()).toContain('2')
  })

  it('shows all logs when search is empty', () => {
    const logs = ['line 1', 'line 2', 'line 3']
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs, searchQuery: '' },
    })

    const lines = wrapper.findAll('.log-line')
    expect(lines).toHaveLength(3)
  })

  it('emits update:searchQuery when search changes', async () => {
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs: [], searchQuery: '' },
    })

    const search = wrapper.find('.console-search')
    await search.setValue('test query')

    expect(wrapper.emitted('update:searchQuery')).toBeTruthy()
    expect(wrapper.emitted('update:searchQuery')![0]).toEqual(['test query'])
  })

  it('renders the console toolbar', () => {
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs: [] },
    })
    expect(wrapper.find('.console-toolbar').exists()).toBe(true)
  })

  it('renders the console body', () => {
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs: [] },
    })
    expect(wrapper.find('.console-body').exists()).toBe(true)
  })

  it('case-insensitive search', async () => {
    const logs = ['Running ANSIBLE task', 'other line']
    const wrapper = mount(ConsoleOutput, {
      global: { plugins: [i18n] },
      props: { logs, searchQuery: '' },
    })

    const search = wrapper.find('.console-search')
    await search.setValue('ansible')

    const lines = wrapper.findAll('.log-line')
    expect(lines).toHaveLength(1)
  })
})
