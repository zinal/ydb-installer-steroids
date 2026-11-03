import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from '@/stores/ui'

describe('ui store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('collapses the console by default', () => {
    const store = useUiStore()
    expect(store.consoleExpanded).toBe(false)
  })

  it('toggles the console on toggleConsole', () => {
    const store = useUiStore()
    store.toggleConsole()
    expect(store.consoleExpanded).toBe(true)
    store.toggleConsole()
    expect(store.consoleExpanded).toBe(false)
  })
})
