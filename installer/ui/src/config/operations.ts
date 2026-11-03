import type { Flow } from '@/types/session'

export interface OperationDescriptor {
  /** POST endpoint that starts the operation. */
  startEndpoint: string
  /** Route name of the screen that renders progress. */
  progressStep: string
  /** Route name of the screen that renders the final result. */
  resultStep: string
  /**
   * i18n namespace prefix for flow-specific wording (headings, success and
   * cancelled copy). Generic labels such as install.tasks / install.console
   * stay shared across every flow.
   */
  i18nPrefix: string
}

/**
 * Flows that currently run an Ansible operation.
 *
 * Typing these keys as required (not Partial) lets callers index OPERATIONS
 * directly without a null check.
 */
export type OperationFlow = 'ydb' | 'delete-ydb'

export const OPERATIONS: Record<OperationFlow, OperationDescriptor> = {
  'ydb': {
    startEndpoint: '/api/install/ydb/start',
    progressStep: '9',
    resultStep: '10',
    i18nPrefix: 'install',
  },
  'delete-ydb': {
    startEndpoint: '/api/install/delete-ydb/start',
    progressStep: '31',
    resultStep: '31',
    i18nPrefix: 'delete',
  },
}

/** Descriptor for a flow, or undefined when the flow runs no operation. */
export function operationFor(flow: Flow): OperationDescriptor | undefined {
  return (OPERATIONS as Record<string, OperationDescriptor | undefined>)[flow]
}
