import { defineStore } from 'pinia'
import type { TopologyType } from '@/types/topology'
import { TOPOLOGIES } from '@/config/topologies'

export const useTopologyStore = defineStore('topology', {
  state: () => ({
    selected: null as TopologyType | null,
    hostsCount: 0,
  }),

  getters: {
    currentTopology: (state) =>
      state.selected ? TOPOLOGIES[state.selected] : null,
    canSelect: (state) => (id: TopologyType) =>
      TOPOLOGIES[id].available,
  },

  actions: {
    select(id: TopologyType) {
      if (TOPOLOGIES[id].available) {
        this.selected = id
      }
    },
    setHostsCount(count: number) {
      this.hostsCount = count
    },
    applyFromInventory(data: { topology?: { type: string } }) {
      if (data.topology?.type && data.topology.type in TOPOLOGIES) {
        this.selected = data.topology.type as TopologyType
      }
    },
    toInventory() {
      return this.selected ? { topology: this.selected } : {}
    },
  },
})
