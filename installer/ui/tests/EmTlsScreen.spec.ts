import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const tlsGenerate = vi.fn().mockResolvedValue({
  success: true,
  output_dir: '/tmp/certs',
  ca_file: '/tmp/certs/ca.crt',
  node_files: [{
    hostname: 'em.example.com',
    short_name: 'em',
    cert_file: '/tmp/certs/em/node.crt',
    key_file: '/tmp/certs/em/node.key',
    web_pem: '/tmp/certs/em/web.pem',
  }],
})
const tlsValidate = vi.fn().mockResolvedValue({
  valid: true,
  ca_valid: true,
  node_files: [],
  missing: [],
  errors: [],
  total_nodes: 1,
  found_nodes: 1,
})
const tlsExists = vi.fn().mockResolvedValue({ exists: false, reasons: [] })

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn().mockResolvedValue({ inventory_found: false, data: null }),
    updateInventory: vi.fn().mockResolvedValue({ status: 'ok' }),
    getEMInventory: vi.fn().mockResolvedValue({ em_inventory_found: false, data: null }),
    updateEMInventory: vi.fn().mockResolvedValue({ status: 'ok' }),
    tlsGenerate: (...args: unknown[]) => tlsGenerate(...args),
    tlsValidate: (...args: unknown[]) => tlsValidate(...args),
    tlsExists: (...args: unknown[]) => tlsExists(...args),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { useEmStore } from '@/stores/em'
import { useSessionStore } from '@/stores/session'
import EmTlsScreen from '@/views/EmTlsScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      emTls: {
        heading: 'TLS Certificates for YDB EM',
        subtitle: 'Generate or validate EM host certificates.',
        modeGenerate: 'Generate self-signed',
        modeExisting: 'Use existing',
        generationTitle: 'Certificate generation',
        validationTitle: 'Certificate validation',
        generateDesc: 'Generate for EM host only.',
        existingDesc: 'Validate existing certs.',
        emHost: 'EM host',
        willGenerate: 'Will generate:',
        caCert: 'CA certificate (ca.crt)',
        nodeCerts: 'Node certificate ({count})',
        privateKeys: 'Private keys',
        webPem: 'web.pem',
        validity: 'Validity period',
        days: 'days',
        generateBtn: 'Generate certificates',
        generating: 'Generating...',
        success: 'Certificates successfully generated',
        outputDir: 'Certificates directory',
        certPath: 'Path to certs-per-host',
        certPathPlaceholder: 'TLS/CA/certs/certs-per-host',
        validateBtn: 'Validate',
        validating: 'Validating...',
        hostname: 'Host',
        nodeCert: 'node.crt',
        nodeKey: 'node.key',
        keyMatch: 'Key match',
        match: 'match',
        mismatch: 'mismatch',
        found: 'found',
        notFound: 'not found',
        valid: 'valid',
        invalid: 'invalid',
        missing: 'Missing certificates',
        allValid: 'All certificates are valid',
        noEmHost: 'Set an EM host first',
        certsExistWarning: 'Certificates blocked',
        certsExistHint: 'Found at {path}',
        switchToExisting: 'Use these certificates',
        caReuseInfo: 'Existing CA will be reused',
      },
    },
  },
})

describe('EmTlsScreen', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    tlsGenerate.mockClear()
    tlsValidate.mockClear()
    tlsExists.mockClear()
    tlsExists.mockResolvedValue({ exists: false, reasons: [] })
  })

  function mountScreen() {
    return mount(EmTlsScreen, {
      global: {
        plugins: [i18n],
        stubs: {
          'v-card': { template: '<div class="v-card"><slot /></div>' },
          'v-icon': true,
          'v-btn': {
            template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
            props: ['disabled', 'loading', 'color', 'variant', 'prependIcon'],
          },
        },
      },
    })
  }

  it('renders heading and pill toggle', async () => {
    const wrapper = mountScreen()
    await nextTick()
    await flushPromises()
    expect(wrapper.text()).toContain('TLS Certificates for YDB EM')
    expect(wrapper.find('[data-testid="em-tls-mode-row"]').exists()).toBe(true)
  })

  it('runs existence preflight when hosts present', async () => {
    const em = useEmStore()
    em.hosts = [{ fqdn: 'em.example.com' }]
    mountScreen()
    await flushPromises()
    expect(tlsExists).toHaveBeenCalled()
  })

  it('shows warning and disables generate when host certs exist', async () => {
    tlsExists.mockResolvedValue({
      exists: true,
      block_generate: true,
      path: '/tmp/certs',
      reasons: ['em/node.crt', 'em/node.key'],
      ca_present: true,
      host_conflicts: ['em/node.crt', 'em/node.key'],
    })
    const em = useEmStore()
    em.hosts = [{ fqdn: 'em.example.com' }]
    const wrapper = mountScreen()
    await flushPromises()
    expect(wrapper.find('[data-testid="em-tls-exist-warning"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Certificates blocked')
    expect(wrapper.find('[data-testid="em-tls-generate-btn"]').attributes('disabled')).toBeDefined()
  })

  it('CA-only tree does not block generate (Option E)', async () => {
    tlsExists.mockResolvedValue({
      exists: false,
      block_generate: false,
      path: '/TLS/CA/certs/certs-per-host',
      reasons: [],
      ca_present: true,
    })
    const em = useEmStore()
    em.hosts = [{ fqdn: 'em.example.com' }]
    const wrapper = mountScreen()
    await flushPromises()
    expect(wrapper.find('[data-testid="em-tls-generate-btn"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.find('[data-testid="em-tls-ca-reuse-info"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Existing CA will be reused')
  })

  it('switchToExisting button switches mode and sets path', async () => {
    tlsExists.mockResolvedValue({
      exists: true,
      block_generate: true,
      path: '/tmp/certs',
      reasons: ['em/node.crt'],
      ca_present: true,
      host_conflicts: ['em/node.crt'],
    })
    const em = useEmStore()
    em.hosts = [{ fqdn: 'em.example.com' }]
    const wrapper = mountScreen()
    await flushPromises()
    await wrapper.find('[data-testid="em-tls-switch-existing"]').trigger('click')
    expect(em.tlsMode).toBe('existing')
    expect(em.tlsCertPath).toBe('/tmp/certs')
  })

  it('generate calls API when allowed', async () => {
    tlsExists.mockResolvedValue({
      exists: false,
      block_generate: false,
      reasons: [],
    })
    const em = useEmStore()
    em.hosts = [{ fqdn: 'em.example.com' }]
    const wrapper = mountScreen()
    await flushPromises()
    await wrapper.find('[data-testid="em-tls-generate-btn"]').trigger('click')
    await flushPromises()
    expect(tlsGenerate).toHaveBeenCalled()
  })
})
