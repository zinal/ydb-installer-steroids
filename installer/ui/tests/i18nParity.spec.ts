import { describe, it, expect } from 'vitest'
import en from '@/locales/en.json'
import ru from '@/locales/ru.json'

/** Recursively collect all dot-separated key paths from a nested object. */
function collectKeys(obj: Record<string, unknown>, prefix = ''): string[] {
  const keys: string[] = []
  for (const [k, v] of Object.entries(obj)) {
    const full = prefix ? `${prefix}.${k}` : k
    if (v !== null && typeof v === 'object' && !Array.isArray(v)) {
      keys.push(...collectKeys(v as Record<string, unknown>, full))
    } else {
      keys.push(full)
    }
  }
  return keys
}

describe('i18n key parity', () => {
  const enKeys = new Set(collectKeys(en))
  const ruKeys = new Set(collectKeys(ru))

  it('every en key exists in ru', () => {
    const missing = [...enKeys].filter(k => !ruKeys.has(k))
    expect(missing).toEqual([])
  })

  it('every ru key exists in en', () => {
    const extra = [...ruKeys].filter(k => !enKeys.has(k))
    expect(extra).toEqual([])
  })

  it('values for shared keys have the same type (string vs object)', () => {
    const mismatches: string[] = []
    for (const k of enKeys) {
      if (!ruKeys.has(k)) continue
      const enVal = getByPath(en, k)
      const ruVal = getByPath(ru, k)
      if (typeof enVal !== typeof ruVal) {
        mismatches.push(`${k}: en=${typeof enVal} ru=${typeof ruVal}`)
      }
    }
    expect(mismatches).toEqual([])
  })
})

function getByPath(obj: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce((o: any, k) => o?.[k], obj)
}