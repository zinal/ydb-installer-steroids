import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDistributionStore } from '@/stores/distribution'
import * as client from '@/api/client'

vi.mock('@/api/client', () => ({
  api: {
    getVersions: vi.fn(),
    validateFile: vi.fn(),
  },
}))

describe('useDistributionStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('starts with official source and empty version', () => {
    const store = useDistributionStore()
    expect(store.source).toBe('official')
    expect(store.version).toBe('')
    expect(store.versions).toEqual([])
    expect(store.loadingVersions).toBe(false)
  })

  it('canProceed is false when no version selected', () => {
    const store = useDistributionStore()
    expect(store.canProceed).toBe(false)
  })

  it('canProceed is true when version selected in official mode', () => {
    const store = useDistributionStore()
    store.version = '25.3.1.25'
    expect(store.canProceed).toBe(true)
  })

  it('canProceed is false when archive not valid in local mode', async () => {
    const store = useDistributionStore()
    store.source = 'local'
    store.archivePath = '/some/path'
    expect(store.canProceed).toBe(false)
  })

  it('canProceed is true when archive valid in local mode', async () => {
    const store = useDistributionStore()
    store.source = 'local'
    store.archiveValid = true
    expect(store.canProceed).toBe(true)
  })

  it('fetchVersions loads versions from API', async () => {
    const mockVersions = [
      { version: '25.3.1.25', recommended: true },
      { version: '25.2.3.18', recommended: false },
    ]
    vi.mocked(client.api.getVersions).mockResolvedValue({ versions: mockVersions })

    const store = useDistributionStore()
    await store.fetchVersions()

    expect(client.api.getVersions).toHaveBeenCalled()
    expect(store.versions).toEqual(mockVersions)
    expect(store.loadingVersions).toBe(false)
    expect(store.version).toBe('25.3.1.25')
  })

  it('fetchVersions handles API error gracefully', async () => {
    vi.mocked(client.api.getVersions).mockRejectedValue(new Error('network error'))

    const store = useDistributionStore()
    await store.fetchVersions()

    expect(store.versions).toEqual([])
    expect(store.loadingVersions).toBe(false)
  })

  it('validateArchivePath validates file path', async () => {
    vi.mocked(client.api.validateFile).mockResolvedValue({
      valid: true,
      size: '245MB',
    })

    const store = useDistributionStore()
    await store.validateArchivePath('/opt/ydb.tar.gz')

    expect(client.api.validateFile).toHaveBeenCalledWith('/opt/ydb.tar.gz')
    expect(store.archiveValid).toBe(true)
    expect(store.archiveSize).toBe('245MB')
  })

  it('validateArchivePath handles invalid file', async () => {
    vi.mocked(client.api.validateFile).mockResolvedValue({
      valid: false,
      error: 'File not found',
    })

    const store = useDistributionStore()
    await store.validateArchivePath('/nonexistent/path')

    expect(store.archiveValid).toBe(false)
    expect(store.archiveSize).toBe('')
  })

  it('applyFromInventory loads distribution from inventory data', () => {
    const store = useDistributionStore()
    store.applyFromInventory({
      distribution: {
        source: 'local',
        archive_path: '/opt/ydb.tar.gz',
        binary_path: '/opt/ydbd',
        cli_binary: '/opt/ydb',
      },
    })

    expect(store.source).toBe('local')
    expect(store.archivePath).toBe('/opt/ydb.tar.gz')
    expect(store.binaryPath).toBe('/opt/ydbd')
    expect(store.cliBinary).toBe('/opt/ydb')
  })

  it('toDistributionConfig builds correct config for official source', () => {
    const store = useDistributionStore()
    store.source = 'official'
    store.version = '25.3.1.25'

    const config = store.toDistributionConfig()
    expect(config).toEqual({
      source: 'official',
      version: '25.3.1.25',
    })
  })

  it('toDistributionConfig builds correct config for local source', () => {
    const store = useDistributionStore()
    store.source = 'local'
    store.archivePath = '/opt/ydb.tar.gz'
    store.binaryPath = '/opt/ydbd'
    store.cliBinary = '/opt/ydb'
    store.opsBinary = '/opt/ydbops'
    store.dstoolBinary = '/opt/dstool'

    const config = store.toDistributionConfig()
    expect(config).toEqual({
      source: 'local',
      archive_path: '/opt/ydb.tar.gz',
      binary_path: '/opt/ydbd',
      cli_binary: '/opt/ydb',
      ops_binary: '/opt/ydbops',
      tool_binary: '/opt/dstool',
    })
  })
})
