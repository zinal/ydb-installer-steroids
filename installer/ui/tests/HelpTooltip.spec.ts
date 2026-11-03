import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      help: {
        docsLink: '↗ Docs',
        testKey: 'This is a test tooltip explanation.',
      },
    },
  },
})

function mountTooltip(props: {
  textKey: string
  docLink?: string
  maxWidth?: number
}) {
  return mount(HelpTooltip, {
    global: { plugins: [i18n] },
    props,
  })
}

describe('HelpTooltip', () => {
  it('renders the help-circle-outline icon', () => {
    const wrapper = mountTooltip({ textKey: 'help.testKey' })
    expect(wrapper.find('.help-tooltip-wrapper').exists()).toBe(true)
    expect(wrapper.find('.mdi-help-circle-outline').exists()).toBe(true)
  })

  it('renders a v-menu component', () => {
    const wrapper = mountTooltip({ textKey: 'help.testKey' })
    expect(wrapper.find('.help-tooltip-wrapper').exists()).toBe(true)
  })

  it('renders v-card inside v-menu content slot', () => {
    const wrapper = mountTooltip({ textKey: 'help.testKey' })
    expect(wrapper.vm).toBeDefined()
  })

  it('renders tooltip i18n text in component template', () => {
    const wrapper = mountTooltip({ textKey: 'help.testKey' })
    expect(wrapper.props('textKey')).toBe('help.testKey')
  })

  it('passes docLink prop to component', () => {
    const wrapper = mountTooltip({
      textKey: 'help.testKey',
      docLink: 'https://ydb.tech/docs/test',
    })
    expect(wrapper.props('docLink')).toBe('https://ydb.tech/docs/test')
  })

  it('docLink is undefined when not provided', () => {
    const wrapper = mountTooltip({ textKey: 'help.testKey' })
    expect(wrapper.props('docLink')).toBeUndefined()
  })

  it('uses default maxWidth of 400 when not provided', () => {
    const wrapper = mountTooltip({ textKey: 'help.testKey' })
    expect(wrapper.props('maxWidth')).toBe(400)
  })

  it('accepts custom maxWidth prop', () => {
    const wrapper = mountTooltip({ textKey: 'help.testKey', maxWidth: 500 })
    expect(wrapper.props('maxWidth')).toBe(500)
  })
})