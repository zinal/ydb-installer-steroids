import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useHostChecks } from '@/composables/useHostChecks'

const checkHostsMock = vi.fn()

vi.mock('@/api/client', () => ({
  api: {
    checkHosts: (...args: unknown[]) => checkHostsMock(...args),
  },
}))

describe('useHostChecks', () => {
  beforeEach(() => {
    checkHostsMock.mockReset()
  })

  it('stores results and time sync summary on success', async () => {
    const summary = { maxSkewMs: 12, thresholdMs: 100, inSync: true }
    checkHostsMock.mockResolvedValueOnce({
      results: [
        { host: 'node-1', sshOk: true, pythonOk: true, portsOk: true },
        { host: 'node-2', sshOk: true, pythonOk: true, portsOk: true },
      ],
      timeSync: summary,
    })

    const hc = useHostChecks()
    const promise = hc.checkHosts(['node-1', 'node-2'], { login: 'root' }, [22])
    expect(hc.isChecking.value).toBe(true)
    await promise

    expect(hc.isChecking.value).toBe(false)
    expect(checkHostsMock).toHaveBeenCalledWith(['node-1', 'node-2'], { login: 'root' }, [22])
    expect(hc.checkResults.value).toHaveLength(2)
    expect(hc.timeSyncSummary.value).toEqual(summary)
    expect(hc.checkError.value).toBeNull()
  })

  it('records the error message on failure', async () => {
    checkHostsMock.mockRejectedValueOnce(new Error('HTTP 500: ssh failed'))

    const hc = useHostChecks()
    await hc.checkHosts(['node-1'], { login: 'root' })

    expect(hc.isChecking.value).toBe(false)
    expect(hc.checkError.value).toBe('HTTP 500: ssh failed')
    expect(hc.checkResults.value).toEqual([])
  })

  it('clearResults resets results, error and time sync summary', async () => {
    checkHostsMock.mockResolvedValueOnce({
      results: [{ host: 'node-1', sshOk: true }],
      timeSync: { maxSkewMs: 1, thresholdMs: 100, inSync: true },
    })

    const hc = useHostChecks()
    await hc.checkHosts(['node-1'], {})
    expect(hc.checkResults.value).toHaveLength(1)

    hc.clearResults()
    expect(hc.checkResults.value).toEqual([])
    expect(hc.timeSyncSummary.value).toBeNull()
    expect(hc.checkError.value).toBeNull()
  })
})
