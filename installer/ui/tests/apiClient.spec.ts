import { describe, it, expect, vi, beforeEach } from 'vitest'
import { api } from '@/api/client'

const fetchSpy = vi.fn()
vi.stubGlobal('fetch', fetchSpy)

describe('api client', () => {
  beforeEach(() => {
    fetchSpy.mockReset().mockResolvedValue({
      ok: true,
      json: async () => ({ status: 'ok' }),
      text: async () => '',
    })
  })

  it('getSession sends GET /api/session', async () => {
    await api.getSession()
    const [url, init] = fetchSpy.mock.calls[0]
    expect(url).toBe('/api/session')
    // GET requests have no explicit method in the init object.
    expect(init?.method).toBeUndefined()
  })

  it('updateSession sends PUT with JSON body', async () => {
    await api.updateSession({ flow: 'ydb' } as any)
    const [url, init] = fetchSpy.mock.calls[0]
    expect(url).toBe('/api/session')
    expect(init.method).toBe('PUT')
    expect(init.body).toContain('"flow"')
  })

  it('updateInventory sends PUT /api/inventory', async () => {
    await api.updateInventory({ dc1: ['h1'] })
    const [url, init] = fetchSpy.mock.calls[0]
    expect(url).toBe('/api/inventory')
    expect(init.method).toBe('PUT')
  })

  it('getConfigStatus sends GET /api/install/ydb/config-status', async () => {
    await api.getConfigStatus()
    expect(fetchSpy).toHaveBeenCalledWith('/api/install/ydb/config-status', expect.anything())
  })

  it('throws on non-ok responses', async () => {
    fetchSpy.mockResolvedValueOnce({ ok: false, status: 500, text: async () => 'boom' })
    await expect(api.getSession()).rejects.toThrow('HTTP 500')
  })
})