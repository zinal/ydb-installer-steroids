export type TaskStatus = 'pending' | 'running' | 'done' | 'error' | 'cancelled'

export interface Task {
  id: number
  name: string
  status: TaskStatus
}

export type InstallStatus = 'idle' | 'running' | 'success' | 'cancelled' | 'error'

export interface InstallError {
  task: string
  host: string
  reason: string
}

export interface InstallState {
  status: InstallStatus
  progress: number
  tasks: Task[]
  error?: InstallError
}

export interface InstructionResponse {
  dep: string
  os: string
  commands: string
}

export interface SshConfig {
  login: string
  keyPath?: string
  useJumpHost: boolean
  jumpHost?: string
  customSshArgs?: string
}

export function defaultSshConfig(): SshConfig {
  return {
    login: 'root',
    useJumpHost: false,
  }
}
