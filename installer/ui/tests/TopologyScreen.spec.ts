import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import TopologyScreen from '@/views/TopologyScreen.vue'
import { useSessionStore } from '@/stores/session'
import { useTopologyStore } from '@/stores/topology'
import { TOPOLOGIES, TOPOLOGY_ORDER } from '@/config/topologies'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('TopologyScreen', () => {
  let router: any

  beforeEach(() => {
    setActivePinia(createPinia())
    
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div>Home</div>' } },
        { path: '/screen/2', name: '2', component: TopologyScreen },
      ],
    })
  })

  it('renders topology cards for all topologies', () => {
    const wrapper = mount(TopologyScreen, {
      global: {
        plugins: [router],
        stubs: {
          TopologyCard: {
            template: '<div class="topology-card">{{ topology.id }}</div>',
            props: ['topology', 'selected', 'selectable'],
          },
        },
      },
    })

    const cards = wrapper.findAll('.topology-card')
    expect(cards).toHaveLength(4)
    
    const ids = cards.map(c => c.text())
    expect(ids).toEqual(TOPOLOGY_ORDER)
  })

  it('disables Next button when no topology selected', () => {
    const session = useSessionStore()
    mount(TopologyScreen, {
      global: {
        plugins: [router],
        stubs: { TopologyCard: true },
      },
    })
    
    expect(session.canGoNext).toBe(false)
  })

  it('enables Next button when topology is selected', async () => {
    const session = useSessionStore()
    const topologyStore = useTopologyStore()
    
    mount(TopologyScreen, {
      global: {
        plugins: [router],
        stubs: { TopologyCard: true },
      },
    })
    
    topologyStore.select('mirror-3-dc')

    await new Promise(resolve => setTimeout(resolve, 0))
    
    expect(session.canGoNext).toBe(true)
  })

  it('shows correct topology information in cards', () => {
    const wrapper = mount(TopologyScreen, {
      global: {
        plugins: [router],
        stubs: {
          TopologyCard: {
            template: '<div class="topology-card">{{ topology.id }} - {{ topology.minHosts }} hosts</div>',
            props: ['topology', 'selected', 'selectable'],
          },
        },
      },
    })

    const cards = wrapper.findAll('.topology-card')
    
    expect(cards[0].text()).toContain('mirror-3-dc')
    expect(cards[0].text()).toContain('3 hosts')

    expect(cards[1].text()).toContain('block-4-2')
    expect(cards[1].text()).toContain('8 hosts')

    expect(cards[2].text()).toContain('bridge-2-dc')
    expect(cards[2].text()).toContain('4 hosts')

    expect(cards[3].text()).toContain('none')
    expect(cards[3].text()).toContain('1 host')
  })

  it('marks available topologies as selectable', () => {
    const topologyStore = useTopologyStore()

    const wrapper = mount(TopologyScreen, {
      global: {
        plugins: [router],
        stubs: {
          TopologyCard: {
            template: '<div class="topology-card" :class="{ selectable: selectable }">{{ topology.id }}</div>',
            props: ['topology', 'selected', 'selectable'],
          },
        },
      },
    })

    const cards = wrapper.findAll('.topology-card')

    expect(cards[0].classes()).toContain('selectable')
    expect(cards[1].classes()).toContain('selectable')
    expect(cards[2].classes()).not.toContain('selectable')
    expect(cards[3].classes()).not.toContain('selectable')
  })

  it('renders heading and description', () => {
    const wrapper = mount(TopologyScreen, {
      global: {
        plugins: [router],
        stubs: { TopologyCard: true },
      },
    })

    expect(wrapper.find('.topology-title').text()).toBe('topology.heading')
    expect(wrapper.find('.topology-sub').text()).toBe('topology.description')
  })

  it('shows the sequential step number and flow label', () => {
    const session = useSessionStore()
    session.$patch({ flow: 'ydb', currentStep: '2' })

    const wrapper = mount(TopologyScreen, {
      global: {
        plugins: [router],
        stubs: { TopologyCard: true },
      },
    })

    expect(wrapper.find('.eyebrow-pill').text()).toBe('common.step 2')
    expect(wrapper.find('.eyebrow-flow').text()).toBe('· YDB')
  })
})
