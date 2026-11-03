import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/client', () => ({
  api: {
    getConnectivity: vi.fn(),
  },
}))

import { api } from '@/api/client'
import { useConnectivityStore } from '@/stores/connectivity'

describe('connectivity store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('starts in checking state with no override', () => {
    const s = useConnectivityStore()
    expect(s.status).toBe('checking')
    expect(s.override).toBeNull()
    expect(s.hasOverride).toBe(false)
    // While the probe is still running the state is indeterminate, not offline.
    expect(s.effectiveOnline).toBeNull()
    expect(s.offlineMode).toBeNull()
  })

  it('checking state yields null for both getters even with no override', () => {
    const s = useConnectivityStore()
    s.status = 'checking'
    s.override = null
    expect(s.effectiveOnline).toBeNull()
    expect(s.offlineMode).toBeNull()
  })

  it('check() sets online status when probe succeeds', async () => {
    vi.mocked(api.getConnectivity).mockResolvedValue({ online: true, checked_at: 'x' })
    const s = useConnectivityStore()
    await s.check()
    expect(s.status).toBe('online')
    expect(s.effectiveOnline).toBe(true)
  })

  it('check() sets offline status when probe reports offline', async () => {
    vi.mocked(api.getConnectivity).mockResolvedValue({ online: false, checked_at: 'x', error: 'no route' })
    const s = useConnectivityStore()
    await s.check()
    expect(s.status).toBe('offline')
    expect(s.effectiveOnline).toBe(false)
    expect(s.error).toBe('no route')
  })

  it('check() handles API error as offline', async () => {
    vi.mocked(api.getConnectivity).mockRejectedValue(new Error('boom'))
    const s = useConnectivityStore()
    await s.check()
    expect(s.status).toBe('offline')
    expect(s.effectiveOnline).toBe(false)
    expect(s.error).toBe('boom')
  })

  it('override wins over detected status', async () => {
    vi.mocked(api.getConnectivity).mockResolvedValue({ online: false, checked_at: 'x' })
    const s = useConnectivityStore()
    await s.check()
    expect(s.effectiveOnline).toBe(false)

    s.setOverride(true)
    expect(s.hasOverride).toBe(true)
    expect(s.effectiveOnline).toBe(true)

    s.setOverride(null)
    expect(s.hasOverride).toBe(false)
    expect(s.effectiveOnline).toBe(false)
  })

  it('setOfflineMode(true) forces offline regardless of detection', async () => {
    vi.mocked(api.getConnectivity).mockResolvedValue({ online: true, checked_at: 'x' })
    const s = useConnectivityStore()
    await s.check()
    expect(s.offlineMode).toBe(false)

    s.setOfflineMode(true)
    expect(s.offlineMode).toBe(true)
    expect(s.effectiveOnline).toBe(false)

    s.setOfflineMode(false)
    expect(s.offlineMode).toBe(false)
    expect(s.effectiveOnline).toBe(true)
  })
})