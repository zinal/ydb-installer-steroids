import { describe, it, expect } from 'vitest'
import { FLOWS } from '@/config/flows'
import { OPERATIONS } from '@/config/operations'

describe('flows', () => {
  it('defines the install and delete flows', () => {
    expect(FLOWS).toHaveProperty('ydb')
    expect(FLOWS).toHaveProperty('delete-ydb')
    expect(Object.keys(FLOWS)).toEqual(['ydb', 'delete-ydb'])
  })

  it('every flow starts with step 1', () => {
    for (const steps of Object.values(FLOWS)) {
      expect(steps[0].id).toBe('1')
    }
  })
})

describe('operations', () => {
  it('covers the install and delete flows', () => {
    expect(OPERATIONS).toHaveProperty('ydb')
    expect(OPERATIONS).toHaveProperty('delete-ydb')
    expect(Object.keys(OPERATIONS)).toEqual(['ydb', 'delete-ydb'])
  })

  it('every operation has a startEndpoint', () => {
    for (const [name, op] of Object.entries(OPERATIONS)) {
      expect(op.startEndpoint, `${name}.startEndpoint`).toBeTruthy()
    }
  })
})
