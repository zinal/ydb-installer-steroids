import { describe, it, expect } from 'vitest'
import { isEMStepBlocked } from '@/router/index'

describe('isEMStepBlocked', () => {
  it('blocks EM steps when the runtime flag has loaded and EM is disabled', () => {
    expect(isEMStepBlocked('7.1', true, false)).toBe(true)
    expect(isEMStepBlocked('20', true, false)).toBe(true)
    expect(isEMStepBlocked('25', true, false)).toBe(true)
    expect(isEMStepBlocked('32', true, false)).toBe(true)
    expect(isEMStepBlocked('33', true, false)).toBe(true)
  })

  it('allows EM steps when the flag is enabled', () => {
    expect(isEMStepBlocked('7.1', true, true)).toBe(false)
    expect(isEMStepBlocked('20', true, true)).toBe(false)
    expect(isEMStepBlocked('33', true, true)).toBe(false)
  })

  it('allows EM steps before the runtime flag has loaded (direct refresh)', () => {
    expect(isEMStepBlocked('7.1', false, false)).toBe(false)
    expect(isEMStepBlocked('20', false, false)).toBe(false)
    expect(isEMStepBlocked('24', false, false)).toBe(false)
  })

  it('never blocks non-EM steps', () => {
    expect(isEMStepBlocked('1', true, false)).toBe(false)
    expect(isEMStepBlocked('8', true, false)).toBe(false)
    expect(isEMStepBlocked('9', true, false)).toBe(false)
  })
})