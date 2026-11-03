import { defineStore } from 'pinia'

interface UiState {
  /** Console log panel on install screens; collapsed by default, persisted across refresh. */
  consoleExpanded: boolean
}

export const useUiStore = defineStore('ui', {
  state: (): UiState => ({
    consoleExpanded: false,
  }),

  actions: {
    toggleConsole() {
      this.consoleExpanded = !this.consoleExpanded
    },
  },
})
