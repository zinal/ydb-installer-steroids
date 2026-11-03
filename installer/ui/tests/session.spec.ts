import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from '@/stores/session'

vi.mock('@/api/client', () => ({
  api: {
    getSession: vi.fn().mockResolvedValue({
      flow: '',
      current_step: '1',
      steps: [],
      default_lang: 'en',
      version: 'v0.0.2',
    }),
    updateSession: vi.fn().mockResolvedValue({}),
    getDeps: vi.fn(),
    getConfigs: vi.fn(),
  },
}))

describe('useSessionStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('starts with the default flow and active intro step', () => {
    const s = useSessionStore()
    expect(s.flow).toBe('')
    expect(s.currentStep).toBe('1')
    expect(s.sidebarItems[0].id).toBe('1')
    expect(s.sidebarItems[0].status).toBe('active')
  })

  it('switching to YDB flow exposes 10 steps in flowStepList', async () => {
    const s = useSessionStore()
    await s.setFlow('ydb')
    expect(s.flow).toBe('ydb')
    expect(s.flowStepList.length).toBe(10)
    expect(s.flowStepList.map((x) => x.id)).toEqual([
      '1', '2', '3', '4', '5', '6', '7', '8', '9', '10',
    ])
    expect(s.currentStep).toBe('1')
  })

  it('sidebar lists every ydb step', async () => {
    const s = useSessionStore()
    await s.setFlow('ydb')
    expect(s.sidebarItems.length).toBe(10)
    expect(s.sidebarItems[6].id).toBe('7')
    expect(s.sidebarItems[7].id).toBe('8')
  })

  it('switching to delete-ydb gives the short 3-step flow', async () => {
    const s = useSessionStore()
    await s.setFlow('delete-ydb')
    expect(s.flowStepList.map((x) => x.id)).toEqual(['1', '30', '31'])
  })

  it('marking visited and advancing currentStep keeps the previous one as done', async () => {
    const s = useSessionStore()
    await s.setFlow('ydb')

    s.markVisited('1')
    s.setCurrent('2')
    const map = Object.fromEntries(s.sidebarItems.map((it) => [it.id, it.status]))
    expect(map['1']).toBe('done')
    expect(map['2']).toBe('active')
    expect(map['3']).toBe('clickable')
  })

  it('errors take precedence over done in sidebar status', async () => {
    const s = useSessionStore()
    await s.setFlow('ydb')

    s.markVisited('1')
    s.setCurrent('2')
    s.setError('1', true)

    const map = Object.fromEntries(s.sidebarItems.map((it) => [it.id, it.status]))
    expect(map['1']).toBe('error')
  })

  it('canGoNext defaults to true and is mutable', () => {
    const s = useSessionStore()
    expect(s.canGoNext).toBe(true)
    s.setCanGoNext(false)
    expect(s.canGoNext).toBe(false)
  })

  it('visibleFlowSteps matches the ydb flow', async () => {
    const s = useSessionStore()
    await s.setFlow('ydb')
    const ids = s.visibleFlowSteps.map((x) => x.id)
    expect(ids).toEqual(['1', '2', '3', '4', '5', '6', '7', '8', '9', '10'])
  })

  it('currentStepNumber is 1 on the intro before any flow is chosen', () => {
    const s = useSessionStore()
    expect(s.currentStepNumber).toBe(1)
  })

  it('currentStepNumber matches the position in the ydb flow', async () => {
    const s = useSessionStore()
    await s.setFlow('ydb')

    const cases: [string, number][] = [
      ['1', 1], ['2', 2], ['7', 7], ['8', 8], ['9', 9], ['10', 10],
    ]
    for (const [id, num] of cases) {
      s.setCurrent(id)
      expect(s.currentStepNumber).toBe(num)
    }
  })

  it('currentStepNumber is sequential in the delete flow', async () => {
    const s = useSessionStore()

    await s.setFlow('delete-ydb')
    s.setCurrent('30')
    expect(s.currentStepNumber).toBe(2)
    s.setCurrent('31')
    expect(s.currentStepNumber).toBe(3)
  })

  it('toUIState serialises visited/errored Sets to arrays', () => {
    const s = useSessionStore()
    s.setCurrent('3')
    s.setError('2', true)
    const ui = s.toUIState()
    expect(ui.current_step).toBe('3')
    expect(Array.isArray(ui.visited)).toBe(true)
    expect(ui.visited).toContain('3')
    expect(ui.errored).toEqual(['2'])
  })

  it('applyServer hydrates visited/errored/configGenerated from backend ui_state', () => {
    const s = useSessionStore()
    s.applyServer({
      flow: 'ydb',
      current_step: '1',
      steps: [],
      default_lang: 'en',
      ui_state: {
        current_step: '6',
        visited: ['1', '2', '3', '4', '5', '6'],
        errored: ['4'],
        config_generated: true,
      },
    })
    expect(s.currentStep).toBe('6')
    expect(s.visited.has('4')).toBe(true)
    expect(s.visited.has('6')).toBe(true)
    expect(s.errored.has('4')).toBe(true)
    expect(s.configGenerated).toBe(true)
    expect(s.hydrated).toBe(true)
  })

  it('applyServer merges instead of clobbering: does not shrink an already-larger visited set', () => {
    const s = useSessionStore()
    s.setCurrent('2')
    s.markVisited('3')
    s.applyServer({
      flow: 'ydb',
      current_step: '1',
      steps: [],
      default_lang: 'en',
      ui_state: {
        current_step: '1',
        visited: ['1'],
        errored: [],
        config_generated: false,
      },
    })
    expect(s.visited.has('2')).toBe(true)
    expect(s.visited.has('3')).toBe(true)
  })

  it('applyServer keeps router-assigned currentStep instead of overwriting with backend default (load-order fix)', () => {
    const s = useSessionStore()
    s.setCurrent('6')
    s.applyServer({
      flow: 'ydb',
      current_step: '1',
      steps: [],
      default_lang: 'en',
      ui_state: {
        current_step: '1',
        visited: ['1'],
        errored: [],
        config_generated: false,
      },
    })
    expect(s.currentStep).toBe('6')
    expect(s.visited.has('6')).toBe(true)
  })

  it('applyServer falls back gracefully when backend has no ui_state yet (fresh server)', () => {
    const s = useSessionStore()
    s.applyServer({
      flow: '',
      current_step: '1',
      steps: [],
      default_lang: 'en',
    })
    expect(s.currentStep).toBe('1')
    expect(s.hydrated).toBe(true)
  })

  it('beforeNext defaults to null', () => {
    const s = useSessionStore()
    expect(s.beforeNext).toBeNull()
  })

  it('setBeforeNext registers and clears the hook', () => {
    const s = useSessionStore()
    const fn = async () => true
    s.setBeforeNext(fn)
    expect(s.beforeNext).toBe(fn)
    s.setBeforeNext(null)
    expect(s.beforeNext).toBeNull()
  })

  it('setFlow clears beforeNext', async () => {
    const s = useSessionStore()
    const fn = async () => true
    s.setBeforeNext(fn)
    await s.setFlow('delete-ydb')
    expect(s.beforeNext).toBeNull()
  })
})
