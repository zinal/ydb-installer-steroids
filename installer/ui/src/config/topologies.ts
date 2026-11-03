import type { Topology, TopologyType } from '@/types/topology'

export const TOPOLOGIES: Record<TopologyType, Topology> = {
  'mirror-3-dc': {
    id: 'mirror-3-dc',
    nameKey: 'topology.mirror3dc.name',
    descriptionKey: 'topology.mirror3dc.description',
    minHosts: 3,
    minDisks: 9,
    available: true,
    diagramType: 'three-dc',
  },
  'block-4-2': {
    id: 'block-4-2',
    nameKey: 'topology.block42.name',
    descriptionKey: 'topology.block42.description',
    minHosts: 8,
    minDisks: 8,
    available: true,  // TODO
    diagramType: 'block',
  },
  'bridge-2-dc': {
    id: 'bridge-2-dc',
    nameKey: 'topology.bridge2dc.name',
    descriptionKey: 'topology.bridge2dc.description',
    minHosts: 4,
    minDisks: 0,
    available: false,  // TODO
    diagramType: 'bridge',
  },
  'none': {
    id: 'none',
    nameKey: 'topology.none.name',
    descriptionKey: 'topology.none.description',
    minHosts: 1,
    minDisks: 0,
    available: false,  // TODO
    diagramType: 'single',
  },
}

export const TOPOLOGY_ORDER: TopologyType[] = ['mirror-3-dc', 'block-4-2', 'bridge-2-dc', 'none']

export const MIRROR_3DC_NODE_THRESHOLD = 9

// minDisksToSelect returns the minimum number of disks the user must select for
// a topology. mirror-3-dc below 9 hosts concentrates all 3 disks on the first
// host per DC; at 9+ hosts it spreads one disk per node, so a single selected
// disk is sufficient there. Every other topology accepts a single disk.
export function minDisksToSelect(topology: TopologyType | null, totalHosts: number): number {
  if (topology === 'mirror-3-dc') {
    return totalHosts >= MIRROR_3DC_NODE_THRESHOLD ? 1 : 3
  }
  return 1
}
