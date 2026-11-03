import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useSessionSync } from '@/composables/useSessionSync'

const updateSessionMock = vi.fn().mockResolvedValue({})

vi.mock('@/api/client', () => ({
  api: {
    getSession: vi.fn().mockResolvedValue({
      flow: '',
      current_step: '1',
      steps: [],
      default_lang: 'en',
      version: 'v0.0.2',
    }),
    updateSession: (...args: unknown[]) => updateSessionMock(...args),
    getDeps: vi.fn(),
    getConfigs: vi.fn(),
  },
}))

function mountWithSync() {
  const Comp = defineComponent({
    setup() {
      useSessionSync()
      return () => h('div')
    },
  })
  return mount(Comp)
}

describe('useSessionSync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    updateSessionMock.mockClear()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('does not PUT ui_state before the store has been hydrated (load() not yet called)', async () => {
    const session = useSessionStore()
    const wrapper = mountWithSync()

    session.setCurrent('3')
    await nextTick()
    vi.advanceTimersByTime(500)
    await nextTick()

    expect(updateSessionMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('PUTs ui_state (debounced) once the store is hydrated and a mutation occurs', async () => {
    const session = useSessionStore()
    await session.load()
    const wrapper = mountWithSync()

    session.setCurrent('4')
    await nextTick()
    vi.advanceTimersByTime(500)
    await nextTick()

    expect(updateSessionMock).toHaveBeenCalledTimes(1)
    const [payload] = updateSessionMock.mock.calls[0]
    expect(payload.ui_state.current_step).toBe('4')
    expect(payload.ui_state.visited).toContain('4')
    wrapper.unmount()
  })

  it('debounces multiple rapid mutations into a single PUT', async () => {
    const session = useSessionStore()
    await session.load()
    const wrapper = mountWithSync()

    session.setCurrent('2')
    session.setCurrent('3')
    session.setCurrent('4')
    await nextTick()
    vi.advanceTimersByTime(500)
    await nextTick()

    expect(updateSessionMock).toHaveBeenCalledTimes(1)
    const [payload] = updateSessionMock.mock.calls[0]
    expect(payload.ui_state.current_step).toBe('4')
    wrapper.unmount()
  })
})
