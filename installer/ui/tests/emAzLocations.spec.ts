import { describe, it, expect } from 'vitest'
import { deriveEmAzLocations, DEFAULT_EM_AZ } from '@/utils/emAzLocations'
import type { DCHosts } from '@/stores/hosts'

function dc(overrides: Partial<DCHosts> = {}): DCHosts {
  return {
    'DC-1': [],
    'DC-2': [],
    'DC-3': [],
    ...overrides,
  }
}

describe('deriveEmAzLocations', () => {
  it('returns defaults when all DCs are empty', () => {
    expect(deriveEmAzLocations(dc())).toEqual(DEFAULT_EM_AZ)
  })

  it('returns defaults when DCs contain only blank strings', () => {
    const hosts = dc({ 'DC-1': [''], 'DC-2': ['  '], 'DC-3': [] })
    expect(deriveEmAzLocations(hosts)).toEqual(DEFAULT_EM_AZ)
  })

  it('returns only DC-1 when only DC-1 has hosts', () => {
    const hosts = dc({ 'DC-1': ['host1.example.com'] })
    expect(deriveEmAzLocations(hosts)).toEqual(['dc-1'])
  })

  it('returns only DCs that have non-empty hosts', () => {
    const hosts = dc({
      'DC-1': ['h1'],
      'DC-3': ['h3a', 'h3b'],
    })
    expect(deriveEmAzLocations(hosts)).toEqual(['dc-1', 'dc-3'])
  })

  it('maintains stable dc-1→dc-2→dc-3 order', () => {
    const hosts = dc({
      'DC-3': ['h3'],
      'DC-2': ['h2'],
      'DC-1': ['h1'],
    })
    expect(deriveEmAzLocations(hosts)).toEqual(['dc-1', 'dc-2', 'dc-3'])
  })

  it('returns all three when all DCs have hosts', () => {
    const hosts = dc({
      'DC-1': ['h1'],
      'DC-2': ['h2'],
      'DC-3': ['h3'],
    })
    expect(deriveEmAzLocations(hosts)).toEqual(['dc-1', 'dc-2', 'dc-3'])
  })

  it('skips DCs that have only blank entries', () => {
    const hosts = dc({
      'DC-1': ['', '  '],
      'DC-2': ['h2'],
      'DC-3': [''],
    })
    expect(deriveEmAzLocations(hosts)).toEqual(['dc-2'])
  })

  it('returns a new array every call (no shared reference)', () => {
    const a = deriveEmAzLocations(dc())
    const b = deriveEmAzLocations(dc())
    expect(a).toEqual(DEFAULT_EM_AZ)
    expect(b).toEqual(DEFAULT_EM_AZ)
    expect(a).not.toBe(b)
  })
})
