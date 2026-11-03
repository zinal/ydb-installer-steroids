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

  it('shows sequential numbers in the ydb-em flow, not screen IDs', () => {
    const session = useSessionStore()
    // Screen 20 is the 2nd step of the ydb-em flow.
    session.$patch({ flow: 'ydb-em', currentStep: '20' })

    const wrapper = mountEyebrow()
    expect(wrapper.find('.eyebrow-pill').text()).toBe('common.step 2')
    expect(wrapper.find('.eyebrow-flow').text()).toBe('· YDB EM')
  })

  it('renumbers ydb steps after 7.1 when showEmStep is toggled', async () => {
    const session = useSessionStore()
    session.$patch({ flow: 'ydb', currentStep: '9', showEmStep: true })

    const wrapper = mountEyebrow()
    // With 7.1 inserted, screen 9 is the 10th visible step.
    expect(wrapper.find('.eyebrow-pill').text()).toBe('common.step 10')

    session.setShowEmStep(false)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.eyebrow-pill').text()).toBe('common.step 9')
  })

  it('shows sequential numbers and i18n flow labels in the delete flows', () => {
    const session = useSessionStore()

    session.$patch({ flow: 'delete-ydb', currentStep: '31' })
    expect(mountEyebrow().find('.eyebrow-pill').text()).toBe('common.step 3')

    session.$patch({ flow: 'delete-em-ydb', currentStep: '33' })
    const wrapper = mountEyebrow()
    expect(wrapper.find('.eyebrow-pill').text()).toBe('common.step 3')
    expect(wrapper.find('.eyebrow-flow').text()).toBe('· intro.deleteEmYdb')
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
