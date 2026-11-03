import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useHostsStore } from '@/stores/hosts'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useDatabasesStore } from '@/stores/databases'
import { useCoresInfo } from '@/composables/useCoresInfo'
import { mount } from '@vue/test-utils'
import CoresInfoCard from '@/components/common/CoresInfoCard.vue'
import type { CpuStats, CoresBudget } from '@/composables/useCoresInfo'
import { createI18n } from 'vue-i18n'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      coresInfo: {
        title: 'CPU Allocation',
        perHostUnit: 'per host',
        of: 'of',
        cores: 'cores',
        static: 'Static',
        dynamic: 'Dynamic',
        free: 'Free',
        staticReserved: 'Reserved for storage nodes',
        remainingForDynamic: 'Remaining for dynamic nodes',
        clusterTotal: 'Cluster total',
        statusLow: 'Resources sufficient',
        statusMedium: 'Moderate utilization',
        statusHigh: 'High utilization',
        statusCritical: 'CPU overcommitted! Reduce allocation.',
        notAvailable: 'Run host checks to see CPU availability',
      },
    },
  },
})

function mountCard(props: {
  cpuStats: CpuStats
  budget: CoresBudget
  compact?: boolean
}) {
  return mount(CoresInfoCard, {
    props,
    global: {
      plugins: [i18n],
    },
  })
}

function setupStores() {
  setActivePinia(createPinia())
  const hostsStore = useHostsStore()
  const ydbConfigStore = useYdbConfigStore()
  const databasesStore = useDatabasesStore()
  return { hostsStore, ydbConfigStore, databasesStore }
}

function addHostWithCores(
  hostsStore: ReturnType<typeof useHostsStore>,
  hostname: string,
  cores: number,
  dc: 'DC-1' | 'DC-2' | 'DC-3' = 'DC-1',
) {
  hostsStore.dcHosts[dc].push(hostname)
  hostsStore.setCheckResult(hostname, {
    ssh: true,
    timeSync: true,
    python: true,
    cpuCores: cores,
  })
}

describe('useCoresInfo', () => {
  beforeEach(() => {
    setupStores()
  })

  it('returns no data when no hosts are checked', () => {
    const coresInfo = useCoresInfo()
    expect(coresInfo.cpuStats.value.available).toBe(false)
    expect(coresInfo.cpuStats.value.min).toBe(0)
    expect(coresInfo.cpuStats.value.max).toBe(0)
    expect(coresInfo.cpuStats.value.sum).toBe(0)
    expect(coresInfo.cpuStats.value.hostCount).toBe(0)
  })

  it('computes correct cpu stats from single host', () => {
    const { hostsStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)

    const coresInfo = useCoresInfo()
    expect(coresInfo.cpuStats.value.available).toBe(true)
    expect(coresInfo.cpuStats.value.min).toBe(16)
    expect(coresInfo.cpuStats.value.max).toBe(16)
    expect(coresInfo.cpuStats.value.sum).toBe(16)
    expect(coresInfo.cpuStats.value.hostCount).toBe(1)
  })

  it('computes correct cpu stats from multiple hosts across DCs', () => {
    const { hostsStore } = setupStores()
    addHostWithCores(hostsStore, 'node1.dc1', 16, 'DC-1')
    addHostWithCores(hostsStore, 'node2.dc1', 16, 'DC-1')
    addHostWithCores(hostsStore, 'node1.dc2', 24, 'DC-2')
    addHostWithCores(hostsStore, 'node1.dc3', 8, 'DC-3')

    const coresInfo = useCoresInfo()
    expect(coresInfo.cpuStats.value.available).toBe(true)
    expect(coresInfo.cpuStats.value.min).toBe(8)
    expect(coresInfo.cpuStats.value.max).toBe(24)
    expect(coresInfo.cpuStats.value.sum).toBe(64)
    expect(coresInfo.cpuStats.value.hostCount).toBe(4)
  })

  it('ignores hosts without check results', () => {
    const { hostsStore } = setupStores()
    hostsStore.dcHosts['DC-1'].push('node1')

    const coresInfo = useCoresInfo()
    expect(coresInfo.cpuStats.value.available).toBe(false)
  })

  it('returns defaults for empty host count', () => {
    const coresInfo = useCoresInfo()
    expect(coresInfo.suggestedMax.value).toBe(64)
  })

  it('suggestedMax uses actual CPU max when available', () => {
    const { hostsStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 8)

    const coresInfo = useCoresInfo()
    // Math.max(8, 64) = 64 since 64 is the generous fallback
    expect(coresInfo.suggestedMax.value).toBeGreaterThanOrEqual(64)
  })

  it('budget is zero when both static and dynamic default to 0', () => {
    const { hostsStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)

    const coresInfo = useCoresInfo()
    expect(coresInfo.budget.value.staticPerHost).toBe(0)
    expect(coresInfo.budget.value.dynamicPerHost).toBe(0)
    expect(coresInfo.budget.value.totalAllocatedPerHost).toBe(0)
    expect(coresInfo.budget.value.remainingPerHost).toBe(16)
    expect(coresInfo.budget.value.utilizationPercent).toBe(0)
    expect(coresInfo.budget.value.utilizationStatus).toBe('low')
  })

  it('budget reflects static allocation only when dynamic is not set', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    ydbConfigStore.coresStatic = 6
    databasesStore.coresDynamic = 0

    const coresInfo = useCoresInfo()
    expect(coresInfo.budget.value.staticPerHost).toBe(6)
    expect(coresInfo.budget.value.dynamicPerHost).toBe(0)
    expect(coresInfo.budget.value.totalAllocatedPerHost).toBe(6)
    expect(coresInfo.budget.value.remainingPerHost).toBe(10)
  })

  it('budget reflects both static and dynamic allocation', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    ydbConfigStore.coresStatic = 6
    databasesStore.coresDynamic = 4

    const coresInfo = useCoresInfo()
    expect(coresInfo.budget.value.staticPerHost).toBe(6)
    expect(coresInfo.budget.value.dynamicPerHost).toBe(4)
    expect(coresInfo.budget.value.totalAllocatedPerHost).toBe(10)
    expect(coresInfo.budget.value.remainingPerHost).toBe(6)
    expect(coresInfo.budget.value.staticClusterTotal).toBe(6)
    expect(coresInfo.budget.value.dynamicClusterTotal).toBe(4)
    expect(coresInfo.budget.value.totalAllocatedCluster).toBe(10)
  })

  it('utilizationStatus is critical when overcommitted', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 8)
    ydbConfigStore.coresStatic = 10
    databasesStore.coresDynamic = 0

    const coresInfo = useCoresInfo()
    // 10 of 8 = 125% → critical
    expect(coresInfo.budget.value.utilizationPercent).toBeGreaterThan(100)
    expect(coresInfo.budget.value.utilizationStatus).toBe('critical')
    // remaining should be 0 (clamped)
    expect(coresInfo.budget.value.remainingPerHost).toBe(0)
  })

  it('utilizationStatus is low when utilization ≤ 70%', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    databasesStore.coresDynamic = 0
    ydbConfigStore.coresStatic = 4

    const coresInfo = useCoresInfo()
    // 4 of 16 = 25% → low
    expect(coresInfo.budget.value.utilizationPercent).toBe(25)
    expect(coresInfo.budget.value.utilizationStatus).toBe('low')
  })

  it('utilizationStatus is medium when 70% < utilization ≤ 85%', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    databasesStore.coresDynamic = 0
    ydbConfigStore.coresStatic = 12

    const coresInfo = useCoresInfo()
    // 12 of 16 = 75% → medium
    expect(coresInfo.budget.value.utilizationPercent).toBe(75)
    expect(coresInfo.budget.value.utilizationStatus).toBe('medium')
  })

  it('utilizationStatus is high when 85% < utilization ≤ 95%', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    databasesStore.coresDynamic = 0
    ydbConfigStore.coresStatic = 14

    const coresInfo = useCoresInfo()
    // 14 of 16 = 88% → high
    expect(coresInfo.budget.value.utilizationPercent).toBe(88)
    expect(coresInfo.budget.value.utilizationStatus).toBe('high')
  })

  it('utilizationStatus is high when utilization = 100% (fully utilized, not overcommitted)', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    databasesStore.coresDynamic = 0
    ydbConfigStore.coresStatic = 16

    const coresInfo = useCoresInfo()
    // 16 of 16 = 100% → high (at limit, but not overcommitted)
    expect(coresInfo.budget.value.utilizationPercent).toBe(100)
    expect(coresInfo.budget.value.utilizationStatus).toBe('high')
  })

  it('utilizationStatus is critical when utilization > 100% (overcommitted)', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    databasesStore.coresDynamic = 0
    ydbConfigStore.coresStatic = 17

    const coresInfo = useCoresInfo()
    // 17 of 16 = 106% → critical
    expect(coresInfo.budget.value.utilizationPercent).toBeGreaterThan(100)
    expect(coresInfo.budget.value.utilizationStatus).toBe('critical')
  })

  it('cluster totals use hostCount from cpuStats', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16, 'DC-1')
    addHostWithCores(hostsStore, 'node2', 16, 'DC-2')
    addHostWithCores(hostsStore, 'node3', 16, 'DC-3')
    ydbConfigStore.coresStatic = 8
    databasesStore.coresDynamic = 2

    const coresInfo = useCoresInfo()
    expect(coresInfo.cpuStats.value.hostCount).toBe(3)
    expect(coresInfo.budget.value.staticClusterTotal).toBe(24) // 8×3
    expect(coresInfo.budget.value.dynamicClusterTotal).toBe(6) // 2×3
    expect(coresInfo.budget.value.totalAllocatedCluster).toBe(30) // 10×3
    expect(coresInfo.budget.value.remainingCluster).toBe(18) // 48 - 30
  })

  it('dynamicPerHost reflects multi-DB when configVariant is multiple', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    ydbConfigStore.coresStatic = 4
    databasesStore.coresDynamic = 4
    databasesStore.setConfigVariant('multiple')
    // 2 entries × 4 cores = 8 dynamic

    const coresInfo = useCoresInfo()
    expect(coresInfo.budget.value.dynamicPerHost).toBe(8)
    expect(coresInfo.budget.value.totalAllocatedPerHost).toBe(12) // 4 static + 8 dynamic
    expect(coresInfo.budget.value.remainingPerHost).toBe(4) // 16 - 12
  })

  it('budget is critical when multi-DB dynamic cores exceed available CPU', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 8)
    ydbConfigStore.coresStatic = 2
    databasesStore.coresDynamic = 4
    databasesStore.setConfigVariant('multiple')
    // 2 entries × 4 cores = 8 dynamic + 2 static = 10 > 8 → critical

    const coresInfo = useCoresInfo()
    expect(coresInfo.budget.value.dynamicPerHost).toBe(8)
    expect(coresInfo.budget.value.utilizationStatus).toBe('critical')
    expect(coresInfo.budget.value.remainingPerHost).toBe(0)
  })

  it('cluster totals reflect multi-DB dynamic cores', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16, 'DC-1')
    addHostWithCores(hostsStore, 'node2', 16, 'DC-2')
    addHostWithCores(hostsStore, 'node3', 16, 'DC-3')
    ydbConfigStore.coresStatic = 4
    databasesStore.coresDynamic = 3
    databasesStore.setConfigVariant('multiple')
    // 2 entries × 3 cores = 6 dynamic per host

    const coresInfo = useCoresInfo()
    expect(coresInfo.cpuStats.value.hostCount).toBe(3)
    expect(coresInfo.budget.value.dynamicPerHost).toBe(6)
    expect(coresInfo.budget.value.dynamicClusterTotal).toBe(18) // 6×3
    expect(coresInfo.budget.value.totalAllocatedPerHost).toBe(10) // 4+6
    expect(coresInfo.budget.value.totalAllocatedCluster).toBe(30) // 10×3
    expect(coresInfo.budget.value.remainingCluster).toBe(18) // 48 - 30
  })

  it('single mode still works correctly (unchanged behavior)', () => {
    const { hostsStore, ydbConfigStore, databasesStore } = setupStores()
    addHostWithCores(hostsStore, 'node1', 16)
    ydbConfigStore.coresStatic = 6
    databasesStore.coresDynamic = 4
    databasesStore.configVariant = 'single'
    databasesStore.entries = [{ id: 1, name: '/Root/db', groups: 8, hosts: [], hostMode: 'all', affinity: '' }]

    const coresInfo = useCoresInfo()
    expect(coresInfo.budget.value.dynamicPerHost).toBe(4)
    expect(coresInfo.budget.value.totalAllocatedPerHost).toBe(10) // 6+4
    expect(coresInfo.budget.value.remainingPerHost).toBe(6) // 16-10
  })
})

describe('CoresInfoCard', () => {
  const baseCpuStats = {
    available: true,
    min: 16,
    max: 16,
    sum: 48,
    hostCount: 3,
  }

  const baseBudget = {
    staticPerHost: 6,
    dynamicPerHost: 0,
    totalAllocatedPerHost: 6,
    remainingPerHost: 10,
    staticClusterTotal: 18,
    dynamicClusterTotal: 0,
    totalAllocatedCluster: 18,
    remainingCluster: 30,
    utilizationPercent: 38,
    utilizationStatus: 'low' as const,
  }

  it('renders unified card with static and dynamic cores', () => {
    const wrapper = mountCard({
      cpuStats: baseCpuStats,
      budget: baseBudget,
      compact: true,
    })

    expect(wrapper.text()).toContain('CPU Allocation')
    expect(wrapper.text()).toContain('Static')
    expect(wrapper.text()).toContain('Dynamic')
    expect(wrapper.text()).toContain('Free')
    expect(wrapper.text()).toContain('Reserved for storage')
    expect(wrapper.text()).toContain('Remaining for dynamic')
    expect(wrapper.text()).toContain('Resources sufficient')
    expect(wrapper.text()).toContain('18') // cluster total
    expect(wrapper.text()).toContain('48')
  })

  it('renders not available when cpuStats.available is false', () => {
    const wrapper = mountCard({
      cpuStats: { available: false, min: 0, max: 0, sum: 0, hostCount: 0 },
      budget: baseBudget,
    })

    expect(wrapper.text()).toContain('Run host checks')
  })

  it('renders both static and dynamic when both are allocated', () => {
    const budget = {
      ...baseBudget,
      dynamicPerHost: 4,
      totalAllocatedPerHost: 10,
      remainingPerHost: 6,
      dynamicClusterTotal: 12,
      totalAllocatedCluster: 30,
      remainingCluster: 18,
      utilizationPercent: 62,
      utilizationStatus: 'low' as const,
    }

    const wrapper = mountCard({
      cpuStats: baseCpuStats,
      budget,
    })

    expect(wrapper.text()).toContain('Static')
    expect(wrapper.text()).toContain('Dynamic')
    expect(wrapper.text()).toContain('Reserved for storage')
    expect(wrapper.text()).toContain('Remaining for dynamic')
  })

  it('shows zero static and dynamic without hiding labels', () => {
    const budget = {
      ...baseBudget,
      staticPerHost: 0,
      dynamicPerHost: 0,
      totalAllocatedPerHost: 0,
      remainingPerHost: 16,
      staticClusterTotal: 0,
      dynamicClusterTotal: 0,
      totalAllocatedCluster: 0,
      remainingCluster: 48,
      utilizationPercent: 0,
      utilizationStatus: 'low' as const,
    }

    const wrapper = mountCard({
      cpuStats: baseCpuStats,
      budget,
    })

    expect(wrapper.text()).toContain('Static')
    expect(wrapper.text()).toContain('Dynamic')
    expect(wrapper.text()).toContain('Free')
    expect(wrapper.text()).toContain('0')
  })

  it('shows critical status in red', () => {
    const budget = {
      ...baseBudget,
      totalAllocatedPerHost: 16,
      remainingPerHost: 0,
      utilizationPercent: 100,
      utilizationStatus: 'critical' as const,
    }

    const wrapper = mountCard({
      cpuStats: { ...baseCpuStats, min: 16 },
      budget,
    })

    expect(wrapper.text()).toContain('CPU overcommitted')
  })
})
