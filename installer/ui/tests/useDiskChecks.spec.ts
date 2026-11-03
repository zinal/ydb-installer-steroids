import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDiskChecks } from '@/composables/useDiskChecks'
import { useDisksStore } from '@/stores/disks'

const checkDisksMock = vi.fn()
const validateDisksMock = vi.fn()

vi.mock('@/api/client', () => ({
  api: {
    checkDisks: (...args: unknown[]) => checkDisksMock(...args),
    validateDisks: (...args: unknown[]) => validateDisksMock(...args),
  },
}))

const diskA = { name: '/dev/sda', type: 'ssd', sizeBytes: 500_000_000_000, mountpoint: '' }
const diskB = { name: '/dev/sdb', type: 'hdd', sizeBytes: 1_000_000_000_000, mountpoint: '' }

describe('useDiskChecks', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    checkDisksMock.mockReset()
    validateDisksMock.mockReset()
  })

  it('stores per-host results and computes common disks for consistent hosts', async () => {
    checkDisksMock.mockResolvedValueOnce({
      results: [
        { host: 'node-1', disks: [diskA, diskB] },
        { host: 'node-2', disks: [diskA, diskB] },
      ],
    })

    const dc = useDiskChecks()
    await dc.checkDisks(['node-1', 'node-2'], { login: 'root' })

    const store = useDisksStore()
    expect(store.diskResults.get('node-1')).toHaveLength(2)
    expect(store.diskResults.get('node-2')).toHaveLength(2)
    expect(store.consistencyResult?.isConsistent).toBe(true)
    expect(store.consistencyResult?.commonDisks).toHaveLength(2)
    expect(dc.checkError.value).toBeNull()
  })

  it('flags count mismatch when hosts report different disk counts', async () => {
    checkDisksMock.mockResolvedValueOnce({
      results: [
        { host: 'node-1', disks: [diskA, diskB] },
        { host: 'node-2', disks: [diskA] },
      ],
    })

    const dc = useDiskChecks()
    await dc.checkDisks(['node-1', 'node-2'], {})

    const store = useDisksStore()
    expect(store.consistencyResult?.isConsistent).toBe(false)
    const types = store.consistencyResult?.issues.map(i => i.type)
    expect(types).toContain('count_mismatch')
    // Different counts also mean no common disks.
    expect(store.consistencyResult?.commonDisks).toEqual([])
  })

  it('reports a per-host discovery error via checkError', async () => {
    checkDisksMock.mockResolvedValueOnce({
      results: [
        { host: 'node-1', disks: [diskA] },
        { host: 'node-2', error: 'ssh timeout' },
      ],
    })

    const dc = useDiskChecks()
    await dc.checkDisks(['node-1', 'node-2'], {})

    expect(dc.checkError.value).toContain('node-2')
    expect(dc.checkError.value).toContain('ssh timeout')
    // The healthy host is still recorded.
    expect(useDisksStore().diskResults.get('node-1')).toHaveLength(1)
  })

  it('records the transport error when the API call fails', async () => {
    checkDisksMock.mockRejectedValueOnce(new Error('HTTP 500: no ssh'))

    const dc = useDiskChecks()
    await dc.checkDisks(['node-1'], {})

    expect(dc.checkError.value).toBe('HTTP 500: no ssh')
    expect(dc.isChecking.value).toBe(false)
  })

  it('validateDisks stores manual-path validation results', async () => {
    validateDisksMock.mockResolvedValueOnce({
      results: [
        { path: '/dev/sda', exists: true, hosts: ['node-1'] },
        { path: '/dev/nope', exists: false, hosts: [] },
      ],
    })

    const dc = useDiskChecks()
    await dc.validateDisks(['node-1'], {}, ['/dev/sda', '/dev/nope'])

    expect(validateDisksMock).toHaveBeenCalledWith(['node-1'], {}, ['/dev/sda', '/dev/nope'])
    expect(dc.validationResults.value).toHaveLength(2)
    expect(dc.isValidating.value).toBe(false)

    dc.clearValidation()
    expect(dc.validationResults.value).toEqual([])
    expect(dc.validationError.value).toBeNull()
  })

  it('validateDisks records the error on failure', async () => {
    validateDisksMock.mockRejectedValueOnce(new Error('HTTP 500: bad path'))

    const dc = useDiskChecks()
    await dc.validateDisks(['node-1'], {}, ['/dev/sda'])

    expect(dc.validationError.value).toBe('HTTP 500: bad path')
    expect(dc.validationResults.value).toEqual([])
  })
})
