export type Flow = '' | 'ydb' | 'ydb-em' | 'delete-ydb' | 'delete-em-ydb'

export type StepStatus =
  | 'active'
  | 'done'
  | 'error'
  | 'clickable'
  | 'disabled'

export interface Step {
  id: string
  status: StepStatus
}

/**
 * UIState carries frontend navigation history so it can survive a browser
 * refresh. Mirrors internal/models/session.go UIState.
 *
 * Visited/Errored are transported as arrays (JSON has no Set type); the
 * session store converts to/from `Set` internally.
 */
export interface UIState {
  current_step: string
  visited: string[]
  errored: string[]
  show_em_step: boolean
  config_generated: boolean
}

export interface Session {
  flow: Flow
  current_step: string
  steps: Step[]
  default_lang: string
  version?: string
  inventory_found?: boolean
  config_found?: boolean
  ui_state?: UIState
}

export interface SessionUpdate {
  flow?: Flow
  current_step?: string
  steps?: Step[]
  ui_state?: UIState
}
