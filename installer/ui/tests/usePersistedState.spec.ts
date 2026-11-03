import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia, defineStore } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick } from 'vue'
import { usePersistedState, clearAllPersistedState } from '@/composables/usePersistedState'

// A minimal test store exercising primitives, a Set and a Map — the two
// types that need custom (de)serialisation.
const useTestStore = defineStore('persist-test', {
  state: () => ({
    text: '',
    flag: false,
    tags: new Set<string>(),
    lookup: new Map<string, number>(),
    notPersisted: 'untouched',
  }),
})

/** Mount a component that wires usePersistedState during setup, matching how App.vue uses it. */
function mountWithPersistence(paths: string[]) {
  let storeRef: ReturnType<typeof useTestStore> | null = null
  const Comp = defineComponent({
    setup() {
      const store = useTestStore()
      storeRef = store
      usePersistedState(store, { name: 'persist-test', paths: paths as any })
      return () => h('div')
    },
  })
  const wrapper = mount(Comp)
  return { wrapper, get store() { return storeRef! } }
}

describe('usePersistedState', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    sessionStorage.clear()
  })

  afterEach(() => {
    sessionStorage.clear()
    vi.useRealTimers()
  })

  it('does nothing on first mount when sessionStorage is empty', () => {
    const { store } = mountWithPersistence(['text', 'flag'])
    expect(store.text).toBe('')
    expect(store.flag).toBe(false)
  })

  it('persists whitelisted fields (debounced) and rehydrates them on a fresh store', async () => {
    vi.useFakeTimers()
    const { store, wrapper } = mountWithPersistence(['text', 'flag'])
    store.text = 'hello'
    store.flag = true
    await nextTick()

    vi.advanceTimersByTime(400)
    wrapper.unmount()

    setActivePinia(createPinia())
    const { store: store2 } = mountWithPersistence(['text', 'flag'])
    expect(store2.text).toBe('hello')
    expect(store2.flag).toBe(true)
  })

  it('does not persist fields outside the whitelist', async () => {
    vi.useFakeTimers()
    const { store, wrapper } = mountWithPersistence(['text'])
    store.notPersisted = 'changed'
    await nextTick()
    vi.advanceTimersByTime(400)
    wrapper.unmount()

    setActivePinia(createPinia())
    const { store: store2 } = mountWithPersistence(['text'])
    expect(store2.notPersisted).toBe('untouched')
  })

  it('round-trips a Set value', async () => {
    vi.useFakeTimers()
    const { store, wrapper } = mountWithPersistence(['tags'])
    store.tags.add('a')
    store.tags.add('b')
    await nextTick()
    vi.advanceTimersByTime(400)
    wrapper.unmount()

    setActivePinia(createPinia())
    const { store: store2 } = mountWithPersistence(['tags'])
    expect(store2.tags).toBeInstanceOf(Set)
    expect(Array.from(store2.tags).sort()).toEqual(['a', 'b'])
  })

  it('round-trips a Map value', async () => {
    vi.useFakeTimers()
    const { store, wrapper } = mountWithPersistence(['lookup'])
    store.lookup.set('x', 1)
    store.lookup.set('y', 2)
    await nextTick()
    vi.advanceTimersByTime(400)
    wrapper.unmount()

    setActivePinia(createPinia())
    const { store: store2 } = mountWithPersistence(['lookup'])
    expect(store2.lookup).toBeInstanceOf(Map)
    expect(store2.lookup.get('x')).toBe(1)
    expect(store2.lookup.get('y')).toBe(2)
  })

  it('ignores a stored envelope with a mismatched schema version', () => {
    sessionStorage.setItem(
      'ydb-installer.ui.persist-test',
      JSON.stringify({ version: 999, data: { text: 'stale' } }),
    )
    const { store } = mountWithPersistence(['text'])
    expect(store.text).toBe('')
  })

  it('ignores malformed JSON in storage without throwing', () => {
    sessionStorage.setItem('ydb-installer.ui.persist-test', '{not-json')
    expect(() => mountWithPersistence(['text'])).not.toThrow()
  })

  it('clearAllPersistedState removes all ydb-installer.ui.* keys', async () => {
    vi.useFakeTimers()
    const { store, wrapper } = mountWithPersistence(['text'])
    store.text = 'to-be-cleared'
    await nextTick()
    vi.advanceTimersByTime(400)
    wrapper.unmount()

    expect(sessionStorage.getItem('ydb-installer.ui.persist-test')).not.toBeNull()
    clearAllPersistedState()
    expect(sessionStorage.getItem('ydb-installer.ui.persist-test')).toBeNull()
  })
})
