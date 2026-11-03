import { describe, it, expect } from 'vitest'
import { HELP_LINKS, systemRequirementsUrl } from '@/config/helpLinks'

describe('HELP_LINKS', () => {
  it('has all expected keys', () => {
    const expectedKeys = [
      'systemRequirements',
      'jumpHost',
      'ydbDomain',
      'coresStatic',
      'enforceUserToken',
      'requestClientCert',
      'affinityStatic',
      'coresDynamic',
      'dbName',
      'dbGroups',
      'affinityDynamic',
    ] as const

    for (const key of expectedKeys) {
      expect(HELP_LINKS).toHaveProperty(key)
    }
  })

  it('has exactly the expected number of keys', () => {
    expect(Object.keys(HELP_LINKS).length).toBe(15)
  })

  it('all full-URL values are non-empty strings starting with https://ydb.tech', () => {
    for (const [key, url] of Object.entries(HELP_LINKS)) {
      if (key === 'systemRequirements') continue
      expect(typeof url).toBe('string')
      expect(url.length).toBeGreaterThan(0)
      expect(url).toMatch(/^https:\/\/ydb\.tech\/docs\//)
    }
  })

  it('systemRequirements is a locale-less path fragment', () => {
    expect(HELP_LINKS.systemRequirements).toMatch(/^\/[a-z0-9/-]+$/)
    expect(HELP_LINKS.systemRequirements).not.toMatch(/^\/(en|ru)\//)
  })

  it('every key matches the pattern help.<key> i18n convention', () => {
    const i18nKeys = Object.keys(HELP_LINKS).map(
      (key) => `help.${key}`,
    )
    expect(i18nKeys).toContain('help.jumpHost')
    expect(i18nKeys).toContain('help.coresDynamic')
    expect(i18nKeys).toContain('help.dbName')
  })

  it('shares keys between affinityStatic and affinityDynamic', () => {
    expect(HELP_LINKS.affinityStatic).toBe(HELP_LINKS.affinityDynamic)
  })

  it('shares keys between coresStatic and coresDynamic', () => {
    expect(HELP_LINKS.coresStatic).toBe(HELP_LINKS.coresDynamic)
  })

  it('shares keys between ydbDomain and dbName', () => {
    expect(HELP_LINKS.ydbDomain).toBe(HELP_LINKS.dbName)
  })
})

describe('systemRequirementsUrl', () => {
  it('builds the Russian docs URL for the ru locale', () => {
    expect(systemRequirementsUrl('ru')).toBe(
      'https://ydb.tech/docs/ru/devops/concepts/system-requirements',
    )
  })

  it('builds the English docs URL for the en locale', () => {
    expect(systemRequirementsUrl('en')).toBe(
      'https://ydb.tech/docs/en/devops/concepts/system-requirements',
    )
  })
})
