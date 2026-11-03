import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ScreenEyebrow from '@/components/common/ScreenEyebrow.vue'
import { useSessionStore } from '@/stores/session'

vi.mock('@/api/client', () => ({
  api: {
    getSession: vi.fn().mockResolvedValue({}),
    updateSession: vi.fn().mockResolvedValue({}),
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('ScreenEyebrow', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function mountEyebrow(props: Record<string, unknown> = {}) {
    return mount(ScreenEyebrow, { props })
  }

  it('shows the sequential step number in the ydb flow', () => {
    const session = useSessionStore()
    session.$patch({ flow: 'ydb', currentStep: '8' })

    const wrapper = mountEyebrow()
    expect(wrapper.find('.eyebrow-pill').text()).toBe('common.step 8')
    expect(wrapper.find('.eyebrow-flow').text()).toBe('· YDB')
  })

  it('shows sequential numbers and the i18n flow label in the delete flow', () => {
    const session = useSessionStore()

    session.$patch({ flow: 'delete-ydb', currentStep: '31' })
    const wrapper = mountEyebrow()
    expect(wrapper.find('.eyebrow-pill').text()).toBe('common.step 3')
    expect(wrapper.find('.eyebrow-flow').text()).toBe('· intro.deleteYdb')
  })

  it('hides the flow label before any flow is chosen', () => {
    const wrapper = mountEyebrow()
    expect(wrapper.find('.eyebrow-pill').text()).toBe('common.step 1')
    expect(wrapper.find('.eyebrow-flow').exists()).toBe(false)
  })

  it('applies the danger variant only when requested', () => {
    const session = useSessionStore()
    session.$patch({ flow: 'delete-ydb', currentStep: '30' })

    const danger = mountEyebrow({ variant: 'danger' })
    expect(danger.find('.eyebrow-pill').classes()).toContain('eyebrow-pill--danger')

    const normal = mountEyebrow()
    expect(normal.find('.eyebrow-pill').classes()).not.toContain('eyebrow-pill--danger')
  })
})
