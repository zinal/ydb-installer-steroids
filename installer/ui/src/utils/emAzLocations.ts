import type { DCHosts } from '@/stores/hosts'

export const DEFAULT_EM_AZ = ['dc-1', 'dc-2', 'dc-3']

const DC_TO_LOC: { dc: keyof DCHosts; loc: string }[] = [
  { dc: 'DC-1', loc: 'dc-1' },
  { dc: 'DC-2', loc: 'dc-2' },
  { dc: 'DC-3', loc: 'dc-3' },
]

/**
 * Derive EM availability zone locations from YDB DC host membership.
 *
 * Non-empty DCs produce their corresponding zone in stable order (dc-1 → dc-2 → dc-3).
 * If no YDB hosts are present, returns the default triple.
 */
export function deriveEmAzLocations(dcHosts: DCHosts): string[] {
  const out: string[] = []
  for (const { dc, loc } of DC_TO_LOC) {
    const has = (dcHosts[dc] ?? []).some(h => h.trim() !== '')
    if (has) out.push(loc)
  }
  return out.length > 0 ? out : [...DEFAULT_EM_AZ]
}
