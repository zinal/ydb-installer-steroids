import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/client', () => ({
  api: {
    getRuntimeInfo: vi.fn(),
  },
}))

import { api } from '@/api/client'
import { useRuntimeStore } from '@/stores/runtime'

describe('runtime store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('starts with empty/not-bundled defaults', () => {
    const s = useRuntimeStore()
    expect(s.embedded).toBe(false)
    expect(s.extracted).toBe(false)
    expect(s.bundleActive).toBe(false)
    expect(s.loaded).toBe(false)
  })

  it('load() populates state from API (bundled runtime)', async () => {
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({
      embedded: true,
      extracted: true,
      python: '3.12.5',
      ansible: '10.3.0',
      ansible_core: '2.17.3',
      collection: '1.2.3',
      path: '/tmp/rt/python',
    })
    const s = useRuntimeStore()
    await s.load()

    expect(s.embedded).toBe(true)
    expect(s.extracted).toBe(true)
    expect(s.python).toBe('3.12.5')
    expect(s.ansible).toBe('10.3.0')
    expect(s.ansibleCore).toBe('2.17.3')
    expect(s.collection).toBe('1.2.3')
    expect(s.path).toBe('/tmp/rt/python')
    expect(s.loaded).toBe(true)
    expect(s.bundleActive).toBe(true)
  })

  it('bundleActive is false when not extracted', async () => {
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({
      embedded: true,
      extracted: false,
    })
    const s = useRuntimeStore()
    await s.load()
    expect(s.bundleActive).toBe(false)
  })

  it('load() handles API error gracefully (falls back to not-bundled)', async () => {
    vi.mocked(api.getRuntimeInfo).mockRejectedValue(new Error('boom'))
    const s = useRuntimeStore()
    await s.load()

    expect(s.embedded).toBe(false)
    expect(s.extracted).toBe(false)
    expect(s.loaded).toBe(true)
    expect(s.error).toContain('boom')
    expect(s.bundleActive).toBe(false)
  })

  it('captures non-fatal backend error message', async () => {
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({
      embedded: true,
      extracted: false,
      error: 'archive sha256 mismatch',
    })
    const s = useRuntimeStore()
    await s.load()
    expect(s.error).toBe('archive sha256 mismatch')
  })
})
