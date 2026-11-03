import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const validateFile = vi.fn().mockResolvedValue({ valid: true, size: '10MB' })

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn().mockResolvedValue({ inventory_found: false, data: null }),
    updateInventory: vi.fn().mockResolvedValue({ status: 'ok' }),
    getEMInventory: vi.fn().mockResolvedValue({ em_inventory_found: false, data: null }),
    updateEMInventory: vi.fn().mockResolvedValue({ status: 'ok' }),
    validateFile: (...args: unknown[]) => validateFile(...args),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { useEmStore } from '@/stores/em'
import { useSessionStore } from '@/stores/session'
import EmDistributionScreen from '@/views/EmDistributionScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      emDistribution: {
        heading: 'YDB EM Distribution',
        subtitle: 'Provide local archive or binaries.',
        modeArchive: 'Local archive',
        modeBinaries: 'Local binaries',
        archiveLabel: 'ydb_em_archive',
        archivePlaceholder: '/opt/em.tar.gz',
        metaUiArchiveLabel: 'ydb_em_meta_ui_archive',
        metaUiArchivePlaceholder: '/opt/ui.tar.gz',
        metaUiBinaryLabel: 'ydb_em_meta_ui_binary',
        metaUiBinaryPlaceholder: '/opt/gateway',
        cpBinaryLabel: 'ydb_em_cp_binary',
        cpBinaryPlaceholder: '/opt/cp',
        cpAgentBinaryLabel: 'ydb_em_cp_agent_binary',
        cpAgentBinaryPlaceholder: '/opt/agent',
        fileValid: 'File found. Size: {size}.',
        fileInvalid: 'File not found or not accessible',
      },
    },
  },
})

describe('EmDistributionScreen', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    validateFile.mockClear()
    validateFile.mockResolvedValue({ valid: true, size: '10MB' })
  })

  function mountScreen() {
    return mount(EmDistributionScreen, {
      global: {
        plugins: [i18n],
        stubs: {
          'v-card': { template: '<div class="v-card"><slot /></div>' },
          'v-icon': true,
        },
      },
    })
  }

  it('renders heading, outside pill toggle, and archive fields by default', async () => {
    const wrapper = mountScreen()
    await nextTick()
    expect(wrapper.text()).toContain('YDB EM Distribution')
    expect(wrapper.find('[data-testid="em-dist-mode-row"]').exists()).toBe(true)
    expect(wrapper.findAll('.source-option')).toHaveLength(2)
    expect(wrapper.find('[data-testid="em-archive-path"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="em-meta-ui-archive-path"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="em-cp-binary"]').exists()).toBe(false)
    expect(wrapper.find('input.ssh-input').exists()).toBe(true)
  })

  it('switches to binaries panel via outside pill toggle', async () => {
    const wrapper = mountScreen()
    await nextTick()
    const options = wrapper.findAll('.source-option')
    expect(options.length).toBe(2)
    await options[1].trigger('click')
    await options[1].find('input').setValue(true)
    await nextTick()
    expect(useEmStore().distMode).toBe('binaries')
    expect(wrapper.find('[data-testid="em-meta-ui-binary"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="em-cp-binary"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="em-cp-agent-binary"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="em-archive-path"]').exists()).toBe(false)
  })

  it('disables Next until archive paths validate', async () => {
    const session = useSessionStore()
    mountScreen()
    await nextTick()
    expect(session.canGoNext).toBe(false)

    const em = useEmStore()
    em.archiveValid = true
    em.metaUIArchiveValid = true
    await nextTick()
    expect(session.canGoNext).toBe(true)
  })

  it('shows validation message after path validation', async () => {
    vi.useFakeTimers()
    const wrapper = mountScreen()
    await nextTick()

    const em = useEmStore()
    em.archivePath = '/opt/dist/ydb-em.tar.gz'
    await nextTick()
    vi.advanceTimersByTime(350)
    await flushPromises()

    expect(validateFile).toHaveBeenCalled()
    expect(em.archiveValid).toBe(true)
    expect(wrapper.text()).toContain('File found')
    vi.useRealTimers()
  })
})
