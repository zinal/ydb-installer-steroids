import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'

vi.mock('@/api/client', () => ({
  api: {
    getEMInventory: vi.fn(),
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
import DeleteEmConfirmScreen from '@/views/DeleteEmConfirmScreen.vue'
import { useSessionStore } from '@/stores/session'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      delete: {
        titleEmYdb: 'Delete YDB EM',
        componentsInfo: 'Component information',
        emHosts: 'YDB EM hosts',
        connectedEndpoint: 'YDB endpoint',
        connectedDatabase: 'YDB database',
        warningTitle: 'WARNING: Complete deletion is irreversible!',
        willDelete: 'The following will be deleted:',
        emServices: 'YDB EM services',
        emAgents: 'YDB EM agent registrations',
        emConfigFiles: 'YDB EM configuration files',
        emBinaries: 'YDB EM binary files',
        emYdbUntouched: 'The YDB cluster and its data will not be modified.',
        confirmPrompt: 'Type DELETE to confirm:',
        confirmPlaceholder: 'DELETE',
        startDeletionBtn: 'Start Deletion',
      },
    },
  },
})

function createMockEMInventory(overrides: {
  hosts?: { fqdn: string }[]
  endpoint?: string
  database?: string
} = {}) {
  return {
    em_inventory_found: true,
    data: {
      hosts: overrides.hosts ?? [{ fqdn: 'em1.example.com' }],
      db_connection_endpoint: overrides.endpoint ?? 'grpcs://ydb.example.com:2135',
      db_connection_db: overrides.database ?? '/Root/db1',
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

function mountScreen(installStatus = 'idle') {
  const state = makeInstallState(installStatus)
  vi.mocked(useInstall).mockReturnValue(state as unknown as ReturnType<typeof useInstall>)
  const wrapper = mount(DeleteEmConfirmScreen, {
    global: { plugins: [i18n] },
  })
  return { wrapper, state }
}

describe('DeleteEmConfirmScreen (Screen 32)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(api.getEMInventory).mockResolvedValue(createMockEMInventory() as never)
  })

  it('renders the screen title', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Delete YDB EM')
  })

  it('displays EM hosts from inventory', async () => {
    vi.mocked(api.getEMInventory).mockResolvedValue(createMockEMInventory({
      hosts: [{ fqdn: 'em1.example.com' }, { fqdn: 'em2.example.com' }],
    }) as never)
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Component information')
    expect(wrapper.text()).toContain('em1.example.com')
    expect(wrapper.text()).toContain('em2.example.com')
  })

  it('displays connected endpoint and database', async () => {
    vi.mocked(api.getEMInventory).mockResolvedValue(createMockEMInventory({
      endpoint: 'grpcs://cluster:2135',
      database: '/Root/app',
    }) as never)
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('grpcs://cluster:2135')
    expect(wrapper.text()).toContain('/Root/app')
  })

  it('shows EM-only warning (YDB untouched note)', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.warning-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('WARNING')
    expect(wrapper.text()).toContain('will not be modified')
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
    await wrapper.find('.confirm-input').setValue('DELETE')
    const btn = wrapper.find('.start-btn')
    expect((btn.element as HTMLButtonElement).disabled).toBe(false)
  })

  it('canGoNext is false until DELETE is typed', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    const session = useSessionStore()
    expect(session.canGoNext).toBe(false)
    await wrapper.find('.confirm-input').setValue('DELETE')
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

  it('calls install.start with delete-em-ydb endpoint when Start Deletion is clicked', async () => {
    const { wrapper, state } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    await wrapper.find('.confirm-input').setValue('DELETE')
    await wrapper.find('.start-btn').trigger('click')
    expect(state.start).toHaveBeenCalledWith('/api/install/delete-em-ydb/start')
  })

  it('does NOT show SSH settings section', async () => {
    const { wrapper } = mountScreen()
    await new Promise((r) => setTimeout(r, 10))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).not.toContain('SSH settings')
    expect(wrapper.text()).not.toContain('Login')
  })
})
