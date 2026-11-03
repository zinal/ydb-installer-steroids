import { computed } from 'vue'
import { useHostsStore, type DCHosts } from '@/stores/hosts'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useDatabasesStore } from '@/stores/databases'

export interface CpuStats {
  available: boolean
  min: number
  max: number
  sum: number
  hostCount: number
}

export type UtilizationStatus = 'low' | 'medium' | 'high' | 'critical'

export interface CoresBudget {
  staticPerHost: number
  dynamicPerHost: number
  totalAllocatedPerHost: number
  remainingPerHost: number
  staticClusterTotal: number
  dynamicClusterTotal: number
  totalAllocatedCluster: number
  remainingCluster: number
  utilizationPercent: number
  utilizationStatus: UtilizationStatus
}

function computeCpuStats(): CpuStats {
  const hosts = useHostsStore()
  const dcs: (keyof DCHosts)[] = ['DC-1', 'DC-2', 'DC-3']

  const cores: number[] = []
  for (const dc of dcs) {
    for (const host of hosts.dcHosts[dc]) {
      const trimmed = host.trim()
      if (!trimmed) continue
      const result = hosts.checkResults[trimmed]
      if (result && result.cpuCores > 0) {
        cores.push(result.cpuCores)
      }
    }
  }

  if (cores.length === 0) {
    return { available: false, min: 0, max: 0, sum: 0, hostCount: 0 }
  }

  return {
    available: true,
    min: Math.min(...cores),
    max: Math.max(...cores),
    sum: cores.reduce((a, b) => a + b, 0),
    hostCount: cores.length,
  }
}

function computeBudget(stats: CpuStats, staticPerHost: number, dynamicPerHost: number): CoresBudget {
  const totalPerHost = staticPerHost + dynamicPerHost
  const effectiveMin = stats.available ? stats.min : 0
  const remainingPerHost = effectiveMin - totalPerHost
  const hostCount = Math.max(stats.hostCount, 1)

  const utilizationPercent = stats.available && effectiveMin > 0
    ? Math.round((totalPerHost / effectiveMin) * 100)
    : 0

  let utilizationStatus: UtilizationStatus = 'low'
  if (utilizationPercent > 100) {
    utilizationStatus = 'critical'
  } else if (utilizationPercent > 85) {
    utilizationStatus = 'high'
  } else if (utilizationPercent > 70) {
    utilizationStatus = 'medium'
  }

  return {
    staticPerHost,
    dynamicPerHost,
    totalAllocatedPerHost: totalPerHost,
    remainingPerHost: Math.max(remainingPerHost, 0),

    staticClusterTotal: staticPerHost * hostCount,
    dynamicClusterTotal: dynamicPerHost * hostCount,
    totalAllocatedCluster: totalPerHost * hostCount,
    remainingCluster: stats.available
      ? Math.max(stats.sum - (totalPerHost * hostCount), 0)
      : 0,

    utilizationPercent,
    utilizationStatus,
  }
}

export function useCoresInfo() {
  const hostsStore = useHostsStore()
  const ydbConfigStore = useYdbConfigStore()
  const databasesStore = useDatabasesStore()

  const effectiveCoresDynamic = computed(() =>
    databasesStore.totalDynamicPerHost > 0
      ? databasesStore.totalDynamicPerHost
      : ydbConfigStore.coresDynamic
  )

  const cpuStats = computed<CpuStats>(() => computeCpuStats())

  const budget = computed<CoresBudget>(() =>
    computeBudget(
      cpuStats.value,
      ydbConfigStore.coresStatic,
      effectiveCoresDynamic.value,
    )
  )

  const suggestedMax = computed(() => {
    if (cpuStats.value.available) {
      return Math.max(cpuStats.value.max, 64)
    }
    return 64
  })

  return {
    cpuStats,
    budget,
    suggestedMax,
  }
}