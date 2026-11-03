import { describe, it, expect } from 'vitest'
import { computeStatus } from '@/stores/session'

const ydbSteps = [
  { id: '1' }, { id: '2' }, { id: '3' }, { id: '4' }, { id: '5' },
  { id: '6' }, { id: '7' }, { id: '7.1' }, { id: '8' }, { id: '9' }, { id: '10' },
]

describe('computeStatus', () => {
  it('marks the current step as active', () => {
    expect(
      computeStatus('3', ydbSteps, '3', new Set(['1', '2', '3']), new Set()),
    ).toBe('active')
  })

  it('marks visited non-current steps as done', () => {
    expect(
      computeStatus('1', ydbSteps, '3', new Set(['1', '2', '3']), new Set()),
    ).toBe('done')
  })

  it('marks errored steps as error even if visited', () => {
    expect(
      computeStatus('2', ydbSteps, '3', new Set(['1', '2', '3']), new Set(['2'])),
    ).toBe('error')
  })

  it('marks subsequent never-visited steps as clickable when at least one earlier step is visited', () => {
    // user is on 3, 4 has not been visited yet
    expect(
      computeStatus('4', ydbSteps, '3', new Set(['1', '2', '3']), new Set()),
    ).toBe('clickable')
  })

  it('marks the first step clickable when current is elsewhere and nothing else visited', () => {
    // Edge case: index 0 short-circuits the "any earlier visited?" check, so
    // the very first step is always reachable.
    expect(
      computeStatus('1', ydbSteps, '2', new Set(), new Set()),
    ).toBe('clickable')
  })

  it('keeps later steps disabled when no earlier step has been visited', () => {
    // Edge case: empty visited set, user supposedly on step 1 but no visits
    // recorded. Step 5 should still be considered disabled.
    expect(
      computeStatus('5', ydbSteps, '1', new Set(), new Set()),
    ).toBe('disabled')
  })

  it('resolves status for sub-step IDs like 7.1', () => {
    expect(
      computeStatus('7.1', ydbSteps, '7.1', new Set(['1', '2', '3', '4', '5', '6', '7', '7.1']), new Set()),
    ).toBe('active')
    expect(
      computeStatus('7.1', ydbSteps, '8', new Set(['1', '2', '3', '4', '5', '6', '7', '7.1', '8']), new Set()),
    ).toBe('done')
  })

  it('computeStatus still works correctly with filtered steps excluding 7.1', () => {
    // Simulate sidebar items where 7.1 is filtered out
    const filteredSteps = ydbSteps.filter((s) => s.id !== '7.1')
    expect(filteredSteps).toHaveLength(10)

    // Active step check
    expect(
      computeStatus('8', filteredSteps, '8', new Set(['1', '2', '3', '4', '5', '6', '7', '8']), new Set()),
    ).toBe('active')

    // Step 8 follows step 7 in the filtered view
    expect(
      computeStatus('8', filteredSteps, '7', new Set(['1', '2', '3', '4', '5', '6', '7']), new Set()),
    ).toBe('clickable')
  })
})
