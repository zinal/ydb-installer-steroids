import { ref } from 'vue'
import { api } from '@/api/client'
import type { CheckHostResult, TimeSyncSummary } from '@/types/hosts'

export function useHostChecks() {
  const checkResults = ref<CheckHostResult[]>([])
  const isChecking = ref(false)
  const checkError = ref<string | null>(null)
  const timeSyncSummary = ref<TimeSyncSummary | null>(null)

  async function checkHosts(hosts: string[], ssh: any, ports?: number[]) {
    isChecking.value = true
    checkError.value = null
    try {
      const response = await api.checkHosts(hosts, ssh, ports)
      checkResults.value = response.results
      timeSyncSummary.value = response.timeSync ?? null
    } catch (e: any) {
      checkError.value = e.message || 'Failed to check hosts'
    } finally {
      isChecking.value = false
    }
  }

  function clearResults() {
    checkResults.value = []
    checkError.value = null
    timeSyncSummary.value = null
  }

  return {
    checkResults,
    isChecking,
    checkError,
    timeSyncSummary,
    checkHosts,
    clearResults,
  }
}
