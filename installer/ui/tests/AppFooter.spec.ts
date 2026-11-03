import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { useSessionStore } from '@/stores/session'
import AppFooter from '@/components/layout/AppFooter.vue'

const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

const i18n = createI18n({
  legacy: false, locale: 'en',
  messages: {
    en: {
      common: { back: 'Back', next: 'Next', finish: 'Finish' },
    },
  },
})

function mountFooter() {
  return mount(
    { template: '<v-app><AppFooter /></v-app>', components: { AppFooter } },
    { global: { plugins: [i18n] } },
  )
}

describe('AppFooter', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    pushMock.mockClear()
  })

  async function setupSession(step = '2', canGo = true) {
    const s = useSessionStore()
    await s.setFlow('ydb')
    s.setCurrent(step)
    if (canGo) s.setCanGoNext(true)
    return s
  }

  it('renders the progress pill with the current step position', async () => {
    await setupSession('2', true)
    const wrapper = mountFooter()
    expect(wrapper.find('.progress-pill').text()).toBe('2 / 10')
  })

  it('calls markVisited and router.push when clicking Next with no beforeNext gate', async () => {
    await setupSession('2', true)
    const wrapper = mountFooter()
    await wrapper.find('.primary-btn').trigger('click')
    await nextTick()

    expect(pushMock).toHaveBeenCalledWith({ name: '3' })
    expect(useSessionStore().visited.has('2')).toBe(true)
  })

  it('respects a beforeNext gate that rejects the transition', async () => {
    const s = await setupSession('2', true)
    s.beforeNext = vi.fn().mockResolvedValue(false)
    const wrapper = mountFooter()

    await wrapper.find('.primary-btn').trigger('click')
    await nextTick()

    expect(s.beforeNext).toHaveBeenCalled()
    expect(pushMock).not.toHaveBeenCalled()
  })

  it('shows a finish button on the last visible step', async () => {
    await setupSession('10', true)
    const wrapper = mountFooter()
    expect(wrapper.find('.primary-btn').exists()).toBe(false)
    const finishBtn = wrapper.findAll('.nav-btn').find(b => b.text().includes('Finish'))
    expect(finishBtn).toBeTruthy()
  })
})
