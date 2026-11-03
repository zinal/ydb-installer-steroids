import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useDeps } from '@/composables/useDeps'

vi.mock('@/api/client', () => ({
  api: {
    getDeps: vi.fn(),
    getDepsInstruction: vi.fn(),
  },
}))

import { api } from '@/api/client'

const mockDeps = [
  { name: 'python', ok: true, version: '3.10.12', required: '>=3.8', installed: true },
  { name: 'ansible', ok: false, version: '', required: '>=2.12', installed: false },
  { name: 'ydb_platform.ydb', ok: false, version: '', required: 'any', installed: false },
]

describe('useDeps', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('starts in loading state', () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: [] })
    const { loading, deps } = useDeps()
    expect(loading.value).toBe(true)
    expect(deps.value).toHaveLength(0)
  })

  it('fetches deps and populates state', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { deps, loading, fetchDeps } = useDeps()

    await fetchDeps()

    expect(loading.value).toBe(false)
    expect(deps.value).toHaveLength(3)
    expect(deps.value[0].name).toBe('python')
    expect(deps.value[0].ok).toBe(true)
    expect(deps.value[0].skipped).toBe(false)
    expect(deps.value[0].instructionOpen).toBe(false)
    expect(deps.value[0].instruction).toBeNull()
  })

  it('canProceed is false when any dep fails and not skipped', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { canProceed, fetchDeps } = useDeps()

    await fetchDeps()

    expect(canProceed.value).toBe(false)
  })

  it('canProceed is true when all deps are ok', async () => {
    const allOk = mockDeps.map((d) => ({ ...d, ok: true, installed: true }))
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: allOk })
    const { canProceed, fetchDeps } = useDeps()

    await fetchDeps()

    expect(canProceed.value).toBe(true)
  })

  it('canProceed becomes true after skipping all failed deps', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { canProceed, fetchDeps, skipDep } = useDeps()

    await fetchDeps()
    expect(canProceed.value).toBe(false)

    skipDep('ansible')
    skipDep('ydb_platform.ydb')

    expect(canProceed.value).toBe(true)
  })

  it('skipDep marks dep as skipped', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { deps, fetchDeps, skipDep } = useDeps()

    await fetchDeps()
    skipDep('ansible')

    const ansible = deps.value.find((d) => d.name === 'ansible')
    expect(ansible?.skipped).toBe(true)
  })

  it('unskipDep removes skipped flag', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { deps, fetchDeps, skipDep, unskipDep } = useDeps()

    await fetchDeps()
    skipDep('ansible')
    unskipDep('ansible')

    const ansible = deps.value.find((d) => d.name === 'ansible')
    expect(ansible?.skipped).toBe(false)
  })

  it('toggleInstruction fetches and sets instruction text', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    vi.mocked(api.getDepsInstruction).mockResolvedValue({
      dep: 'ansible',
      os: 'ubuntu',
      commands: 'sudo apt install ansible',
    })

    const { deps, fetchDeps, toggleInstruction } = useDeps()
    await fetchDeps()

    await toggleInstruction('ansible')

    const ansible = deps.value.find((d) => d.name === 'ansible')
    expect(ansible?.instructionOpen).toBe(true)
    expect(ansible?.instruction).toBe('sudo apt install ansible')
  })

  it('toggleInstruction closes the block on second call', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    vi.mocked(api.getDepsInstruction).mockResolvedValue({
      dep: 'ansible',
      os: 'ubuntu',
      commands: 'sudo apt install ansible',
    })

    const { deps, fetchDeps, toggleInstruction } = useDeps()
    await fetchDeps()

    await toggleInstruction('ansible')
    await toggleInstruction('ansible')

    const ansible = deps.value.find((d) => d.name === 'ansible')
    expect(ansible?.instructionOpen).toBe(false)
  })

  it('handles API error gracefully', async () => {
    vi.mocked(api.getDeps).mockRejectedValue(new Error('Network error'))
    const { deps, loading, error, fetchDeps } = useDeps()

    await fetchDeps()

    expect(loading.value).toBe(false)
    expect(deps.value).toHaveLength(0)
    expect(error.value).toContain('Network error')
  })

  it('hasSkipped is true when at least one dep is skipped', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { hasSkipped, fetchDeps, skipDep } = useDeps()

    await fetchDeps()
    expect(hasSkipped.value).toBe(false)

    skipDep('ansible')
    expect(hasSkipped.value).toBe(true)
  })

  it('retryDeps resets all skip states', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { deps, fetchDeps, skipDep, retryDeps } = useDeps()

    await fetchDeps()
    skipDep('ansible')
    skipDep('ydb_platform.ydb')

    expect(deps.value[1].skipped).toBe(true)
    expect(deps.value[2].skipped).toBe(true)

    await retryDeps()

    expect(deps.value[1].skipped).toBe(false)
    expect(deps.value[2].skipped).toBe(false)
  })

  it('retryDeps resets all instruction states', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    vi.mocked(api.getDepsInstruction).mockResolvedValue({
      dep: 'ansible',
      os: 'ubuntu',
      commands: 'sudo apt install ansible',
    })

    const { deps, fetchDeps, toggleInstruction, retryDeps } = useDeps()

    await fetchDeps()
    await toggleInstruction('ansible')

    expect(deps.value[1].instructionOpen).toBe(true)
    expect(deps.value[1].instruction).toBe('sudo apt install ansible')

    await retryDeps()

    expect(deps.value[1].instructionOpen).toBe(false)
    expect(deps.value[1].instruction).toBeNull()
  })

  it('retryDeps calls fetchDeps', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { fetchDeps, retryDeps } = useDeps()

    await retryDeps()

    expect(api.getDeps).toHaveBeenCalledTimes(1)
  })

  it('exposes bundled=true and bundled sources when runtime is embedded', async () => {
    const bundledDeps = [
      { name: 'python', ok: true, version: '3.12.5', required: '>=3.8', installed: true, source: 'bundled' },
      { name: 'ansible', ok: true, version: '10.3.0', required: '>=2.12', installed: true, source: 'bundled' },
      { name: 'ydb_platform.ydb', ok: true, version: '1.2.3', required: 'any', installed: true, source: 'bundled' },
    ]
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: bundledDeps, bundled: true })
    const { deps, bundled, canProceed, fetchDeps } = useDeps()

    await fetchDeps()

    expect(bundled.value).toBe(true)
    expect(canProceed.value).toBe(true)
    expect(deps.value.every((d) => d.source === 'bundled')).toBe(true)
  })

  it('bundled defaults to false when API omits the flag', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { bundled, fetchDeps } = useDeps()

    await fetchDeps()

    expect(bundled.value).toBe(false)
  })

  it('retryDeps sets loading state', async () => {
    vi.mocked(api.getDeps).mockResolvedValue({ dependencies: mockDeps })
    const { loading, retryDeps } = useDeps()

    const promise = retryDeps()
    expect(loading.value).toBe(true)

    await promise
    expect(loading.value).toBe(false)
  })
})
