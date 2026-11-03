import type { Flow } from '@/types/session'

export interface FlowStep {
  /** Stable step ID; matches Vue Router route name and screen number. */
  id: string
  /** i18n key under sidebar.<key>. */
  i18nKey: string
}

export const FLOWS: Record<Exclude<Flow, ''>, FlowStep[]> = {
  'ydb': [
    { id: '1', i18nKey: 'intro' },
    { id: '2', i18nKey: 'topology' },
    { id: '3', i18nKey: 'hosts' },
    { id: '4', i18nKey: 'disks' },
    { id: '5', i18nKey: 'distribution' },
    { id: '6', i18nKey: 'config' },
    { id: '7', i18nKey: 'databases' },
    { id: '7.1', i18nKey: 'ydbEm' },
    { id: '8', i18nKey: 'tls' },
    { id: '9', i18nKey: 'install' },
    { id: '10', i18nKey: 'result' },
  ],
  'ydb-em': [
    { id: '1', i18nKey: 'intro' },
    { id: '20', i18nKey: 'emHosts' },
    { id: '21', i18nKey: 'emParams' },
    { id: '22', i18nKey: 'emDistribution' },
    { id: '23', i18nKey: 'emTls' },
    { id: '24', i18nKey: 'emInstall' },
    { id: '25', i18nKey: 'emResult' },
  ],
  'delete-ydb': [
    { id: '1', i18nKey: 'intro' },
    { id: '30', i18nKey: 'deleteConfirm' },
    { id: '31', i18nKey: 'deleteProgress' },
  ],
  'delete-em-ydb': [
    { id: '1', i18nKey: 'intro' },
    { id: '32', i18nKey: 'deleteEmConfirm' },
    { id: '33', i18nKey: 'deleteEmProgress' },
  ],
}

/** Default skeleton when no flow is chosen yet — only intro is shown. */
export const DEFAULT_STEPS: FlowStep[] = [
  { id: '1', i18nKey: 'intro' },
]

export function flowSteps(flow: Flow): FlowStep[] {
  if (flow === '' || !(flow in FLOWS)) return DEFAULT_STEPS
  return FLOWS[flow as Exclude<Flow, ''>]
}

/** All step IDs that have a route — used to register placeholder views. */
export const ALL_STEP_IDS: string[] = [
  '1', '2', '3', '4', '5', '6', '7', '7.1', '8', '9', '10',
  '20', '21', '22', '23', '24', '25',
  '30', '31', '32', '33',
]
