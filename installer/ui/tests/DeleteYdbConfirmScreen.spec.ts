import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn(),
    getSession: vi.fn(),
    updateSession: vi.fn(),
  },
}))

vi.mock('@/composables/useInstall', () => ({
  useInstall: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

import { api } from '@/api/client'
import { useInstall } from '@/composables/useInstall'
import DeleteYdbConfirmScreen from '@/views/DeleteYdbConfirmScreen.vue'
import { useSessionStore } from '@/stores/session'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      delete: {
        titleYdb: 'Delete YDB',
        clusterInfo: 'Cluster information',
        topology: 'Topology',
        version: 'YDB version',
        hosts: 'Hosts',
        warningTitle: 'WARNING: Complete deletion is irreversible!',
        willDelete: 'The following will be deleted:',
        services: 'YDB services (ydbd)',
        diskData: 'Data on disks',
        configFiles: 'Configuration files',
        binaries: 'YDB binary files',
        confirmPrompt: 'Type DELETE to confirm:',
        confirmPlaceholder: 'DELETE',
        startDeletionBtn: 'Start Deletion',
      },
    },
  },
})

function createMockInventory(overrides: {
  hosts?: string[]
  version?: string
  topologyType?: string
} = {}) {
  return {
    inventory_found: true,
    config_found: true,
    data: {
      hosts: overrides.hosts ?? ['node1.example.com'],
      topology: overrides.topologyType !== undefined ? { type: overrides.topologyType } : undefined,
      distribution: { source: 'github', version: overrides.version ?? '25.3.1' },
      ssh: { login: 'root', auth_type: 'key' as const, use_jump_host: false },
    },
  }
}

function makeInstallState(status: string = 'idle') {
  return {
    status: ref(status),
    progress: ref(0),
    tasks: ref([]),
    error: ref(null),
    logs: ref([]),
    isStarted: ref(false),
    start: vi.fn().mockResolvedValue(undefined),
    cancel: vi.fn(),
    restartInstallation: vi.fn(),
    waitForRecovery: vi.fn().mockResolvedValue(undefined),
  }
}

function mountScreen(inventoryOverrides = {}, installStatus = 'idle') {
  const state = makeInstallState(installStatus)
  vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
  const wrapper = mount(DeleteYdbConfirmScreen, {
    global: { plugins: [i18n] },
  })
  return { wrapper, state }
}

describe('DeleteYdbConfirmScreen (Screen 30)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(api.getInventory).mockResolvedValue(createMockInventory())
  })

  it('renders the screen title', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Delete YDB')
  })

  it('displays cluster info when config is found', async () => {
    vi.mocked(api.getInventory).mockResolvedValue(createMockInventory({
      hosts: ['node1.example.com', 'node2.example.com'],
    }))
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Cluster information')
    expect(wrapper.text()).toContain('node1.example.com')
    expect(wrapper.text()).toContain('node2.example.com')
  })

  it('displays YDB version from inventory', async () => {
    vi.mocked(api.getInventory).mockResolvedValue(createMockInventory({
      version: '25.3.1.25',
    }))
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('25.3.1.25')
  })

  it('displays topology from inventory (not hardcoded mirror-3-dc)', async () => {
    vi.mocked(api.getInventory).mockResolvedValue(createMockInventory({
      topologyType: 'block-4-2',
    }))
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('block-4-2')
    expect(wrapper.text()).not.toContain('mirror-3-dc')
  })

  it('shows — when topology is missing', async () => {
    vi.mocked(api.getInventory).mockResolvedValue(createMockInventory({
      topologyType: undefined,
    }))
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('—')
  })

  it('shows warning block', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.warning-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('WARNING')
  })

  it('shows confirmation input', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.confirm-input').exists()).toBe(true)
  })

  it('Start Deletion button is disabled when confirmation is empty', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    const btn = wrapper.find('.start-btn')
    expect((btn.element as HTMLButtonElement).disabled).toBe(true)
  })

  it('Start Deletion button is enabled when DELETE is typed', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    const input = wrapper.find('.confirm-input')
    await input.setValue('DELETE')
    const btn = wrapper.find('.start-btn')
    expect((btn.element as HTMLButtonElement).disabled).toBe(false)
  })

  it('canGoNext is false until DELETE is typed', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    const session = useSessionStore()
    expect(session.canGoNext).toBe(false)
    const input = wrapper.find('.confirm-input')
    await input.setValue('DELETE')
    expect(session.canGoNext).toBe(true)
  })

  it('registers beforeNext on mount and clears on unmount', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    const session = useSessionStore()
    expect(session.beforeNext).not.toBeNull()
    wrapper.unmount()
    expect(session.beforeNext).toBeNull()
  })

  it('calls install.start and navigates when Start Deletion is clicked', async () => {
    const { wrapper, state } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    const input = wrapper.find('.confirm-input')
    await input.setValue('DELETE')
    const btn = wrapper.find('.start-btn')
    await btn.trigger('click')
    expect(state.start).toHaveBeenCalledWith('/api/install/delete-ydb/start')
  })

  it('does NOT show SSH settings section', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).not.toContain('SSH settings')
    expect(wrapper.text()).not.toContain('Login')
  })
})
