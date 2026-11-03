import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

import YdbConfigScreen from '@/views/YdbConfigScreen.vue'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useSessionStore } from '@/stores/session'
import { useTopologyStore } from '@/stores/topology'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      common: { step: 'Step' },
      screens: { '6': 'YDB configuration' },
      ydbConfig: {
        heading: 'YDB Configuration',
        topologyLabel: 'Topology: {topology}',
        clusterParams: 'Cluster parameters',
        domain: 'Domain',
        domainPlaceholder: 'example.ydb.local',
        domainRequired: 'Domain is required',
        password: 'Password',
        passwordPlaceholder: 'Superuser password',
        generatePassword: 'Generate',
        coresStatic: 'Static cores',
        coresRequired: 'Cores must be greater than 0',
        advancedSettings: 'Advanced settings',
        enforceUserToken: 'Enforce user token',
        requestClientCert: 'Request client certificate',
        enabled: 'Enabled',
        disabled: 'Disabled',
        enforceUserTokenFlag: 'ydb_enforce_user_token_requirement',
        requestClientCertFlag: 'ydb_request_client_certificate',
        affinity: 'Affinity',
        affinityPlaceholder: 'Disabled in this release',
        affinityDesc: 'Static node affinity',
        validationOk: 'Configuration is valid',
        validationRequired: 'Fill required fields',
      },
      help: {
        ydbDomain: 'Cluster domain',
        coresStatic: 'Cores for static nodes',
        enforceUserToken: 'Token requirement',
        requestClientCert: 'Client certificate',
        affinityStatic: 'Affinity',
        docsLink: 'Docs',
      },
      coresInfo: {
        title: 'Resources',
        available: 'Available',
        requested: 'Requested',
        perHost: 'per host',
        warning: 'Requested more than available',
        recommendation: 'Recommendation',
      },
    },
  },
})

function mountScreen() {
  return mount(YdbConfigScreen, {
    global: { plugins: [i18n] },
  })
}

describe('YdbConfigScreen', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders heading with the selected topology label', () => {
    useTopologyStore().selected = 'block-4-2'
    const wrapper = mountScreen()
    expect(wrapper.find('.config-title').text()).toBe('YDB Configuration')
    expect(wrapper.find('.config-sub').text()).toContain('block-4-2')
  })

  const errorHints = (wrapper: ReturnType<typeof mountScreen>) =>
    wrapper.findAll('.validation-hint.is-error').map(h => h.text())

  it('shows the domain-required hint once the domain is cleared', () => {
    useYdbConfigStore().domain = ''
    const wrapper = mountScreen()
    expect(errorHints(wrapper)).toContain('Domain is required')
  })

  it('typing into the domain field updates the store and clears the hint', async () => {
    const wrapper = mountScreen()
    const ydb = useYdbConfigStore()

    const domainInput = wrapper.find('.text-input input')
    await domainInput.setValue('test.ydb.local')

    expect(ydb.domain).toBe('test.ydb.local')
    expect(errorHints(wrapper)).not.toContain('Domain is required')
  })

  it('generate password button fills a 16-char alphanumeric password', async () => {
    const wrapper = mountScreen()
    const ydb = useYdbConfigStore()
    expect(ydb.password).toBe('')

    await wrapper.find('.generate-btn').trigger('click')

    expect(ydb.password).toHaveLength(16)
    expect(ydb.password).toMatch(/^[A-Za-z0-9]+$/)
  })

  it('advanced settings toggle reveals flag radio buttons that update the store', async () => {
    const wrapper = mountScreen()
    const ydb = useYdbConfigStore()

    expect(wrapper.findAll('.radio-row')).toHaveLength(0)

    await wrapper.find('.advanced-toggle-btn').trigger('click')
    const radioRows = wrapper.findAll('.radio-row')
    expect(radioRows.length).toBe(2)

    await radioRows[0].findAll('.radio-option')[0].trigger('click')
    expect(ydb.enforceUserTokenRequirement).toBe(true)

    await radioRows[1].findAll('.radio-option')[1].trigger('click')
    expect(ydb.requestClientCertificate).toBe(false)
  })

  it('drives session.canGoNext from the store validation state', async () => {
    const wrapper = mountScreen()
    const ydb = useYdbConfigStore()
    const session = useSessionStore()

    expect(session.canGoNext).toBe(false)
    expect(wrapper.find('.validation-item.is-warn').exists()).toBe(true)

    ydb.domain = 'test.ydb.local'
    ydb.coresStatic = 8
    await nextTick()

    expect(session.canGoNext).toBe(true)
    expect(wrapper.find('.validation-item.is-ok').exists()).toBe(true)
    wrapper.unmount()
  })
})
