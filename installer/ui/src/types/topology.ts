export type TopologyType = 'mirror-3-dc' | 'block-4-2' | 'bridge-2-dc' | 'none'

export interface Topology {
  id: TopologyType
  nameKey: string
  descriptionKey: string
  minHosts: number
  minDisks: number
  available: boolean
  diagramType: 'three-dc' | 'block' | 'bridge' | 'single'
}

export interface TopologyState {
  selected: TopologyType | null
  hostsCount: number
}
