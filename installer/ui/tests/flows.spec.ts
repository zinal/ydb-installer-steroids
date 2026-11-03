import { describe, it, expect } from 'vitest'
import { FLOWS } from '@/config/flows'
import { OPERATIONS } from '@/config/operations'

describe('flows', () => {
  it('defines all four flows', () => {
    expect(FLOWS).toHaveProperty('ydb')
    expect(FLOWS).toHaveProperty('ydb-em')
    expect(FLOWS).toHaveProperty('delete-ydb')
    expect(FLOWS).toHaveProperty('delete-em-ydb')
  })

  it('step 7.1 is only present in the ydb flow', () => {
    for (const [name, steps] of Object.entries(FLOWS)) {
      const ids = steps.map(s => s.id)
      if (name === 'ydb') {
        expect(ids).toContain('7.1')
      } else {
        expect(ids).not.toContain('7.1')
      }
    }
  })

  it('every flow starts with step 1', () => {
    for (const steps of Object.values(FLOWS)) {
      expect(steps[0].id).toBe('1')
    }
  })
})

describe('operations', () => {
  it('covers all four flows', () => {
    expect(OPERATIONS).toHaveProperty('ydb')
    expect(OPERATIONS).toHaveProperty('ydb-em')
    expect(OPERATIONS).toHaveProperty('delete-ydb')
    expect(OPERATIONS).toHaveProperty('delete-em-ydb')
  })

  it('every operation has a startEndpoint', () => {
    for (const [name, op] of Object.entries(OPERATIONS)) {
      expect(op.startEndpoint, `${name}.startEndpoint`).toBeTruthy()
    }
  })
})