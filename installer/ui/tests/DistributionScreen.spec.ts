import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  api: {
    getVersions: vi.fn().mockResolvedValue({ versions: [] }),
    validateFile: vi.fn().mockResolvedValue({ valid: false }),
    getConnectivity: vi.fn().mockResolvedValue({ online: true, checked_at: 'x' }),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

import { useDistributionStore } from '@/stores/distribution'
import { useSessionStore } from '@/stores/session'
import { useTopologyStore } from '@/stores/topology'
import { useConnectivityStore } from '@/stores/connectivity'
import DistributionScreen from '@/views/DistributionScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      distribution: {
        heading: 'YDB Distribution',
        topologyLabel: 'Topology: {topology}',
        source: 'Source',
        official: 'Official Distribution',
        local: 'Local Files',
        optionalBinaries: 'Optional Binaries',
        version: 'YDB Version',
        recommended: 'Recommended version',
        loadingVersions: 'Loading versions...',
        noVersions: 'Failed to load versions',
        retryFetch: 'Retry',
        selectVersion: 'Select YDB version',
        package: 'YDB package (.tar.gz)',
        packagePlaceholder: '/opt/dist/ydbd-25.3.1.tar.gz',
        binaryPlaceholder: '/opt/dist/ydb',
        cliBinaryPlaceholder: '.files/ydb',
        serverBinaryPlaceholder: '.files/ydbd',
        opsBinaryPlaceholder: '.files/ydbops',
        toolBinaryPlaceholder: '.files/dstool',
        optional: 'Optional',
        cliBinary: 'ydb CLI',
        serverBinary: 'ydbd',
        opsBinary: 'ydbops',
        toolBinary: 'dstool',
        fileValid: 'File found. Size: {size}',
        fileInvalid: 'File not found or inaccessible',
        fileRequired: 'Please specify archive path',
        offlineWarning: 'Offline mode is active. This installer runs in a closed/isolated environment — use local files for the distribution.',
        checkingConnectivity: 'Checking internet access…',
      },
      help: {
        docsLink: '↗ Docs',
        ydbCli: 'The YDB command-line interface (CLI).',
        ydbd: 'The YDB server daemon (ydbd).',
        ydbops: 'A low-level administrative tool (ydbops).',
        dstool: 'A distributed storage management tool (dstool).',
      },
      screens: {
        '5': 'YDB distribution',
      },
    },
  },
})

function mountScreen() {
  return mount(DistributionScreen, {
    global: { plugins: [i18n] },
  })
}

describe('DistributionScreen — binary tooltips and placeholders', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders the Optional Binaries toggle button', () => {
    const wrapper = mountScreen()
    const toggleBtn = wrapper.find('.binaries-toggle-btn')
    expect(toggleBtn.exists()).toBe(true)
    expect(toggleBtn.text()).toContain('Optional Binaries')
  })

  it('shows binary fields when Optional Binaries is expanded', async () => {
    const wrapper = mountScreen()
    const toggleBtn = wrapper.find('.binaries-toggle-btn')
    await toggleBtn.trigger('click')
    await wrapper.vm.$nextTick()

    const labels = wrapper.findAll('.field-label')
    const labelTexts = labels.map(l => l.text())
    expect(labelTexts.some(t => t.includes('ydb CLI'))).toBe(true)
    expect(labelTexts.some(t => t.includes('ydbd'))).toBe(true)
    expect(labelTexts.some(t => t.includes('ydbops'))).toBe(true)
    expect(labelTexts.some(t => t.includes('dstool'))).toBe(true)
  })

  it('renders HelpTooltip components for each binary field', async () => {
    const wrapper = mountScreen()
    const toggleBtn = wrapper.find('.binaries-toggle-btn')
    await toggleBtn.trigger('click')
    await wrapper.vm.$nextTick()

    const helpIcons = wrapper.findAll('.mdi-help-circle-outline')
    expect(helpIcons.length).toBe(4)
  })

  it('uses .files/ placeholders for binary fields', async () => {
    const wrapper = mountScreen()
    const toggleBtn = wrapper.find('.binaries-toggle-btn')
    await toggleBtn.trigger('click')
    await wrapper.vm.$nextTick()

    const inputs = wrapper.findAll('.path-input input')
    const store = useDistributionStore()
    store.source = 'local'

    await wrapper.vm.$nextTick()

    const allInputs = wrapper.findAll('.path-input input')
    expect(allInputs.length).toBeGreaterThanOrEqual(4)

    const binaryInputs = allInputs.slice(1, 5)
    const placeholders = binaryInputs.map(i => (i.element as HTMLInputElement).placeholder)
    expect(placeholders).toContain('.files/ydb')
    expect(placeholders).toContain('.files/ydbd')
    expect(placeholders).toContain('.files/ydbops')
    expect(placeholders).toContain('.files/dstool')
  })

  it('archive placeholder is unchanged', async () => {
    const wrapper = mountScreen()
    const store = useDistributionStore()
    store.source = 'local'

    await wrapper.vm.$nextTick()

    const allInputs = wrapper.findAll('.path-input input')
    const archiveInput = allInputs[0]
    expect((archiveInput.element as HTMLInputElement).placeholder).toBe('/opt/dist/ydbd-25.3.1.tar.gz')
  })

  it('HelpTooltip has correct textKey props for each binary', async () => {
    const wrapper = mountScreen()
    const toggleBtn = wrapper.find('.binaries-toggle-btn')
    await toggleBtn.trigger('click')
    await wrapper.vm.$nextTick()

    const tooltips = wrapper.findAllComponents({ name: 'HelpTooltip' })
    expect(tooltips.length).toBe(4)

    const textKeys = tooltips.map(t => t.props('textKey'))
    expect(textKeys).toContain('help.ydbCli')
    expect(textKeys).toContain('help.ydbd')
    expect(textKeys).toContain('help.ydbops')
    expect(textKeys).toContain('help.dstool')
  })
})

describe('DistributionScreen — offline connectivity handling', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('shows the offline warning when connectivity is offline', async () => {
    const conn = useConnectivityStore()
    conn.status = 'offline'
    conn.override = null

    const wrapper = mountScreen()
    await wrapper.vm.$nextTick()

    const warning = wrapper.find('.offline-warning')
    expect(warning.exists()).toBe(true)
    expect(warning.text()).toContain('Offline mode is active')
  })

  it('auto-switches source to local when offline and official was selected', async () => {
    const conn = useConnectivityStore()
    conn.status = 'offline'
    conn.override = null
    const dist = useDistributionStore()
    dist.source = 'official'

    const wrapper = mountScreen()
    await wrapper.vm.$nextTick()

    expect(dist.source).toBe('local')
    expect(wrapper.find('.offline-warning').exists()).toBe(true)
  })

  it('keeps official source when override says internet is available', async () => {
    const conn = useConnectivityStore()
    conn.status = 'offline'
    conn.override = true
    const dist = useDistributionStore()
    dist.source = 'official'

    const wrapper = mountScreen()
    await wrapper.vm.$nextTick()

    expect(dist.source).toBe('official')
    expect(wrapper.find('.offline-warning').exists()).toBe(false)
  })

  it('disables the official option when offline', async () => {
    const conn = useConnectivityStore()
    conn.status = 'offline'
    conn.override = null

    const wrapper = mountScreen()
    await wrapper.vm.$nextTick()

    const official = wrapper.find('.source-option.is-disabled input')
    expect(official.exists()).toBe(true)
    expect((official.element as HTMLInputElement).disabled).toBe(true)
  })

  it('enables the official option when online', async () => {
    const conn = useConnectivityStore()
    conn.status = 'online'
    conn.override = null

    const wrapper = mountScreen()
    await wrapper.vm.$nextTick()

    const official = wrapper.find('.source-option input')
    expect((official.element as HTMLInputElement).disabled).toBe(false)
  })

  it('does not auto-switch or disable the official tab while checking', async () => {
    const conn = useConnectivityStore()
    conn.status = 'checking'
    conn.override = null
    const dist = useDistributionStore()
    dist.source = 'official'

    const wrapper = mountScreen()
    await wrapper.vm.$nextTick()

    // The probe is still running — the source must stay untouched.
    expect(dist.source).toBe('official')
    // No offline warning, no disabled official option.
    expect(wrapper.find('.offline-warning').exists()).toBe(false)
    const official = wrapper.find('.source-option input')
    expect((official.element as HTMLInputElement).disabled).toBe(false)
    // The indeterminate state is surfaced as a checking hint instead.
    expect(wrapper.find('.checking-hint').exists()).toBe(true)
  })
})