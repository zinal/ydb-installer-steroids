import { defineStore } from 'pinia'
import type { Flow, Session, Step, StepStatus, UIState } from '@/types/session'
import { flowSteps, FlowStep } from '@/config/flows'
import { api } from '@/api/client'
import { resetInstallState } from '@/composables/useInstall'

interface State {
    flow: Flow
    currentStep: string
    defaultLang: string
    version: string
    visited: Set<string>
    errored: Set<string>
    canGoNext: boolean
    loaded: boolean
    inventoryFound: boolean
    configFound: boolean
    configGenerated: boolean
    hydrated: boolean
    beforeNext: (() => Promise<boolean>) | null
}

export interface SidebarItem {
    id: string
    i18nKey: string
    status: StepStatus
}

export const useSessionStore = defineStore('session', {
    state: (): State => ({
        flow: '',
        currentStep: '1',
        defaultLang: 'en',
        version: '',
        visited: new Set(['1']),
        errored: new Set(),
        canGoNext: true,
        loaded: false,
        inventoryFound: false,
        configFound: false,
        configGenerated: false,
        hydrated: false,
        beforeNext: null,
    }),

    getters: {
        flowStepList(state): FlowStep[] {
            return flowSteps(state.flow)
        },

        /** Step list for sidebar / footer navigation. */
        visibleFlowSteps(): FlowStep[] {
            return this.flowStepList
        },

        sidebarItems(): SidebarItem[] {
            const steps = this.visibleFlowSteps
            return steps.map((s) =>
            ({
                id: s.id,
                i18nKey: s.i18nKey,
                status: computeStatus(s.id, steps, this.currentStep, this.visited as Set<string>, this.errored as Set<string>),
            }),
            )
        },

        /**
         * 1-based position of the current step within the visible flow —
         * the sequential number shown to the user (ScreenEyebrow pill).
         * Derived from `visibleFlowSteps`, the same source the sidebar and
         * footer use, so all indicators always agree.
         */
        currentStepNumber(state): number {
            const idx = this.visibleFlowSteps.findIndex((s) => s.id === state.currentStep)
            if (idx >= 0) return idx + 1
            const fallback = this.flowStepList.findIndex((s) => s.id === state.currentStep)
            return fallback >= 0 ? fallback + 1 : 1
        },
    },

    actions: {
        async load() {
            try {
                const s = await api.getSession()
                this.applyServer(s)
            } catch (e) {
                // Backend not reachable — keep our optimistic defaults.
                // eslint-disable-next-line no-console
                console.warn('session load failed', e)
            } finally {
                this.loaded = true
            }
      },

      /**
       * Apply the backend session to this store.
       *
       * Load-order note: the Vue Router `beforeEach` guard (see
       * ui/src/router/index.ts) runs *before* this is called on initial page
       * load and may have already called `setCurrent()` for the step encoded
       * in the URL hash (e.g. `#/screen/6`). This method therefore **merges**
       * with, rather than overwrites, any navigation state already present —
       * it never shrinks `visited`/`errored`, and only adopts the backend's
       * `current_step` when the router hasn't already picked a more specific
       * one (i.e. we're still on the default `'1'`).
       */
      applyServer(s: Session) {
          this.flow = s.flow
          this.defaultLang = s.default_lang || 'en'
          this.version = s.version || ''
          this.inventoryFound = s.inventory_found ?? false
          this.configFound = s.config_found ?? false

          if (s.ui_state) {
              const backendVisited = new Set(s.ui_state.visited)
              for (const v of this.visited) backendVisited.add(v)
              this.visited = backendVisited

              const backendErrored = new Set(s.ui_state.errored)
              for (const e of this.errored) backendErrored.add(e)
              this.errored = backendErrored

              // If the router already positioned us on a specific step (URL
              // hash), keep it; otherwise adopt the backend's remembered step.
              if (this.currentStep === '1' && s.ui_state.current_step) {
                  this.currentStep = s.ui_state.current_step
              }
              this.visited.add(this.currentStep)

              this.configGenerated = s.ui_state.config_generated
          } else {
              // Backend has no ui_state yet (fresh server, e.g. first ever run) —
              // fall back to the previous single-element behaviour.
              this.currentStep = this.currentStep !== '1' ? this.currentStep : (s.current_step || '1')
              this.visited.add(this.currentStep)
          }

          this.hydrated = true
      },

      toUIState(): UIState {
          return {
              current_step: this.currentStep,
              visited: Array.from(this.visited),
              errored: Array.from(this.errored),
              config_generated: this.configGenerated,
          }
      },

      async setFlow(flow: Flow) {
          this.flow = flow
          const steps = flowSteps(flow)
          this.currentStep = steps[0]?.id ?? '1'
          this.visited = new Set([this.currentStep])
          this.errored = new Set()
          this.configGenerated = false
          this.beforeNext = null
          resetInstallState()
          try {
          await api.updateSession({ flow, ui_state: this.toUIState() })
      } catch {
          /* offline-mode is fine */
      }
      },

      markVisited(stepId: string) {
          this.visited.add(stepId)
      },

      setCurrent(stepId: string) {
          this.currentStep = stepId
          this.visited.add(stepId)
          this.errored.delete(stepId)
      },

      setError(stepId: string, hasError: boolean) {
          if (hasError) this.errored.add(stepId)
          else this.errored.delete(stepId)
      },

        setCanGoNext(v: boolean) {
            this.canGoNext = v
        },

      setBeforeNext(fn: (() => Promise<boolean>) | null) {
            this.beforeNext = fn
        },

      setConfigGenerated(v: boolean) {
            this.configGenerated = v
        },
    },
})

/**
 * Pure sidebar status calculator — extracted for unit testing.
 *
 * Rules:
 *  - The current step is `active`.
 *  - A step that is in `errored` is `error`.
 *  - A visited step that is not current is `done`.
 *  - A non-visited step is `clickable` if any earlier step is visited
 *    (so the user can jump back from later in the wizard); otherwise
 *    `disabled`.
 */
export function computeStatus(
    stepId: string,
    steps: { id: string }[],
    currentStep: string,
    visited: Set<string>,
    errored: Set<string>,
): StepStatus {
    if (errored.has(stepId)) return 'error'
    if (stepId === currentStep) return 'active'
    if (visited.has(stepId)) return 'done'

    const idx = steps.findIndex((s) => s.id === stepId)
    if (idx <= 0) return 'clickable'
    for (let i = 0; i < idx; i++) {
        if (visited.has(steps[i].id)) return 'clickable'
    }
    return 'disabled'
}

export type { Step }
