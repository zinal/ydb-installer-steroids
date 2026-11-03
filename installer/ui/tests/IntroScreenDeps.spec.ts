import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useDeps } from '@/composables/useDeps'

vi.mock('@/api/client', () => ({
  api: {
    getDeps: vi.fn(),
    getDepsInstruction: vi.fn(),
    getSession: vi.fn(),
    updateSession: vi.fn(),
  },
}))

import { api } from '@/api/client'

const mockDeps = [
  { name: 'python', ok: true, version: '3.10.12', required: '>=3.8', installed: true },
  { name: 'ansible', ok: false, version: '', required: '>=2.12', installed: false },
  { name: 'ydb_platform.ydb', ok: false, version: '', required: 'any', installed: false },
]

const mockAllOkDeps = [
  { name: 'python', ok: true, version: '3.10.12', required: '>=3.8', installed: true },
  { name: 'ansible', ok: true, version: '2.15.0', required: '>=2.12', installed: true },
  { name: 'ydb_platform.ydb', ok: true, version: '1.0.0', required: 'any', installed: true },
]

describe('IntroScreen Dependency Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
    const session = useSessionStore()
    session.$reset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Delete Flow Navigation Control', () => {
    it('disables Next button when delete-ydb flow selected with missing dependencies', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { canProceed, fetchDeps } = useDeps()
      await fetchDeps()

      await session.setFlow('delete-ydb')

      const isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (isDeleteFlow) {
        session.setCanGoNext(canProceed.value)
      }

      expect(canProceed.value).toBe(false)
      expect(session.canGoNext).toBe(false)
    })

    it('disables Next button when delete-em-ydb flow selected with missing dependencies', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { canProceed, fetchDeps } = useDeps()
      await fetchDeps()

      await session.setFlow('delete-em-ydb')

      const isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (isDeleteFlow) {
        session.setCanGoNext(canProceed.value)
      }

      expect(canProceed.value).toBe(false)
      expect(session.canGoNext).toBe(false)
    })

    it('enables Next button when delete-ydb flow selected with all dependencies satisfied', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockAllOkDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { canProceed, fetchDeps } = useDeps()
      await fetchDeps()

      await session.setFlow('delete-ydb')

      const isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (isDeleteFlow) {
        session.setCanGoNext(canProceed.value)
      }

      expect(canProceed.value).toBe(true)
      expect(session.canGoNext).toBe(true)
    })

    it('enables Next button when delete-ydb flow selected after skipping all failed dependencies', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { canProceed, fetchDeps, skipDep } = useDeps()
      await fetchDeps()

      await session.setFlow('delete-ydb')

      expect(canProceed.value).toBe(false)

      skipDep('ansible')
      skipDep('ydb_platform.ydb')

      const isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (isDeleteFlow) {
        session.setCanGoNext(canProceed.value)
      }

      expect(canProceed.value).toBe(true)
      expect(session.canGoNext).toBe(true)
    })
  })

  describe('Install Flow Navigation Control', () => {
    it('always enables Next button for install flows regardless of dependencies', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { canProceed, fetchDeps } = useDeps()
      await fetchDeps()

      await session.setFlow('ydb')

      const isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (!isDeleteFlow) {
        session.setCanGoNext(true)
      }

      expect(canProceed.value).toBe(false)
      expect(session.canGoNext).toBe(true)
    })

    it('always enables Next button for ydb-em install flow regardless of dependencies', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { canProceed, fetchDeps } = useDeps()
      await fetchDeps()

      await session.setFlow('ydb-em')

      const isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (!isDeleteFlow) {
        session.setCanGoNext(true)
      }

      expect(canProceed.value).toBe(false)
      expect(session.canGoNext).toBe(true)
    })
  })

  describe('No Flow Selected', () => {
    it('enables Next button when no flow is selected', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { fetchDeps } = useDeps()
      await fetchDeps()

      expect(session.flow).toBe('')

      const isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (!isDeleteFlow) {
        session.setCanGoNext(true)
      }

      expect(session.canGoNext).toBe(true)
    })
  })

  describe('Flow Switching Behavior', () => {
    it('updates canGoNext when switching from install to delete flow with missing deps', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { canProceed, fetchDeps } = useDeps()
      await fetchDeps()

      await session.setFlow('ydb')
      let isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (!isDeleteFlow) {
        session.setCanGoNext(true)
      }
      expect(session.canGoNext).toBe(true)

      await session.setFlow('delete-ydb')
      isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (isDeleteFlow) {
        session.setCanGoNext(canProceed.value)
      }
      expect(session.canGoNext).toBe(false)
    })

    it('updates canGoNext when switching from delete to install flow', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { canProceed, fetchDeps } = useDeps()
      await fetchDeps()

      await session.setFlow('delete-ydb')
      let isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (isDeleteFlow) {
        session.setCanGoNext(canProceed.value)
      }
      expect(session.canGoNext).toBe(false)

      await session.setFlow('ydb')
      isDeleteFlow = session.flow === 'delete-ydb' || session.flow === 'delete-em-ydb'
      if (!isDeleteFlow) {
        session.setCanGoNext(true)
      }
      expect(session.canGoNext).toBe(true)
    })
  })

  describe('Loading State Handling', () => {
    it('does not disable Next button during dependency loading for delete flows', async () => {
      let resolveDeps: (value: any) => void
      const depsPromise = new Promise((resolve) => {
        resolveDeps = resolve
      })

      vi.mocked(api.getDeps).mockReturnValue(depsPromise as any)
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const { loading, fetchDeps } = useDeps()
      
      fetchDeps()

      await session.setFlow('delete-ydb')

      expect(loading.value).toBe(true)

      resolveDeps!({ dependencies: mockDeps })
    })
  })

  describe('EM flow selection', () => {
    it('allows ydb-em flow selection', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockAllOkDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      await session.setFlow('ydb-em')
      expect(session.flow).toBe('ydb-em')
    })

    it('allows delete-em-ydb flow when EM inventory is present', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockAllOkDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      const canSelectDeleteEm = true
      expect(canSelectDeleteEm).toBe(true)
      await session.setFlow('delete-em-ydb')
      expect(session.flow).toBe('delete-em-ydb')
    })

    it('allows ydb install flow', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockAllOkDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
      })

      const session = useSessionStore()
      await session.load()

      await session.setFlow('ydb')
      expect(session.flow).toBe('ydb')
    })

    it('allows delete-ydb flow when configs are present', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockAllOkDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
        inventory_found: true,
        config_found: true,
      })

      const session = useSessionStore()
      await session.load()

      const canSelectDeleteYdb = session.inventoryFound && session.configFound

      expect(canSelectDeleteYdb).toBe(true)
      await session.setFlow('delete-ydb')
      expect(session.flow).toBe('delete-ydb')
    })

    it('blocks delete-ydb when configs are missing', async () => {
      vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockAllOkDeps })
      vi.mocked(api.getSession).mockResolvedValue({
        flow: '',
        current_step: '1',
        steps: [],
        default_lang: 'en',
        inventory_found: false,
        config_found: false,
      })

      const session = useSessionStore()
      await session.load()

      const canSelectDeleteYdb = session.inventoryFound && session.configFound

      expect(canSelectDeleteYdb).toBe(false)
      expect(session.flow).toBe('')
    })
  })
})
