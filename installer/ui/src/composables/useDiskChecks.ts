import { ref } from 'vue'
import { api } from '@/api/client'
import { useDisksStore } from '@/stores/disks'
import type { GetDisksResponse, DiskConsistencyResult, DiskInfo, DiskValidationResult } from '@/types/disks'

export function useDiskChecks() {
  const disksStore = useDisksStore()
  const isChecking = ref(false)
  const checkError = ref<string | null>(null)
  const isValidating = ref(false)
  const validationError = ref<string | null>(null)
  const validationResults = ref<DiskValidationResult[]>([])

  async function checkDisks(hosts: string[], ssh: any) {
    isChecking.value = true
    checkError.value = null
    disksStore.clearDiscovery()
    
    try {
      const response = await api.checkDisks(hosts, ssh)
      
      for (const result of response.results) {
        if (result.error) {
          checkError.value = `Failed to discover disks on ${result.host}: ${result.error}`
          continue
        }
        disksStore.setDiskResults(result.host, result.disks)
      }

      const consistency = checkDiskConsistency(response.results)
      disksStore.setConsistencyResult(consistency)

      disksStore.reconcileSelection()

    } catch (e: any) {
      checkError.value = e.message || 'Failed to check disks'
    } finally {
      isChecking.value = false
    }
  }

  function checkDiskConsistency(results: GetDisksResponse['results']): DiskConsistencyResult {
    const consistency: DiskConsistencyResult = {
      isConsistent: true,
      issues: [],
      commonDisks: [],
    }

    if (results.length === 0) {
      return consistency
    }

    const hostDiskCounts = new Map<string, number>()
    const hostDiskTypes = new Map<string, string[]>()
    const hostDiskSizes = new Map<string, number[]>()

    for (const result of results) {
      if (result.error) continue
      
      hostDiskCounts.set(result.host, result.disks.length)
      
      const types = result.disks.map(d => d.type)
      const sizes = result.disks.map(d => d.sizeBytes)
      
      hostDiskTypes.set(result.host, types)
      hostDiskSizes.set(result.host, sizes)
    }

    if (hostDiskCounts.size > 1) {
      const counts = new Map<number, string[]>()
      for (const [host, count] of hostDiskCounts) {
        if (!counts.has(count)) {
          counts.set(count, [])
        }
        counts.get(count)!.push(host)
      }
      if (counts.size > 1) {
        consistency.isConsistent = false
        for (const [count, hosts] of counts) {
          consistency.issues.push({
            type: 'count_mismatch',
            hosts,
            message: `Hosts ${hosts.join(', ')} have ${count} disks (expected same count on all hosts)`,
          })
        }
      }
    }

    if (hostDiskTypes.size > 1) {
      const typeSets = new Map<string, string[]>()
      for (const [host, types] of hostDiskTypes) {
        const typeKey = types.join(',')
        if (!typeSets.has(typeKey)) {
          typeSets.set(typeKey, [])
        }
        typeSets.get(typeKey)!.push(host)
      }
      if (typeSets.size > 1) {
        consistency.isConsistent = false
        for (const [typeKey, hosts] of typeSets) {
          consistency.issues.push({
            type: 'type_mismatch',
            hosts,
            message: `Hosts ${hosts.join(', ')} have disk types: ${typeKey}`,
          })
        }
      }
    }

    if (hostDiskSizes.size > 1) {
      const sizeSets = new Map<string, string[]>()
      for (const [host, sizes] of hostDiskSizes) {
        const sizeKey = sizes.join(',')
        if (!sizeSets.has(sizeKey)) {
          sizeSets.set(sizeKey, [])
        }
        sizeSets.get(sizeKey)!.push(host)
      }
      if (sizeSets.size > 1) {
        consistency.isConsistent = false
        for (const [sizeKey, hosts] of sizeSets) {
          consistency.issues.push({
            type: 'size_mismatch',
            hosts,
            message: `Hosts ${hosts.join(', ')} have disk sizes: ${sizeKey}`,
          })
        }
      }
    }

    if (results.length > 0) {
      const refHost = results[0]
      if (!refHost.error && refHost.disks.length > 0) {
        let allMatch = true
        for (const result of results.slice(1)) {
          if (result.error || result.disks.length !== refHost.disks.length) {
            allMatch = false
            break
          }
          for (let i = 0; i < result.disks.length; i++) {
            if (result.disks[i].name !== refHost.disks[i].name ||
                result.disks[i].type !== refHost.disks[i].type ||
                result.disks[i].sizeBytes !== refHost.disks[i].sizeBytes) {
              allMatch = false
              break
            }
          }
        }
        if (allMatch) {
          consistency.commonDisks = refHost.disks
        }
      }
    }

    return consistency
  }

  function clearResults() {
    disksStore.clearDiscovery()
    checkError.value = null
  }

  async function validateDisks(hosts: string[], ssh: any, paths: string[]) {
    isValidating.value = true
    validationError.value = null
    validationResults.value = []

    try {
      const response = await api.validateDisks(hosts, ssh, paths)
      validationResults.value = response.results
    } catch (e: any) {
      validationError.value = e.message || 'Failed to validate disks'
    } finally {
      isValidating.value = false
    }
  }

  function clearValidation() {
    validationResults.value = []
    validationError.value = null
    isValidating.value = false
  }

  return {
    isChecking,
    checkError,
    checkDisks,
    clearResults,
    isValidating,
    validationError,
    validationResults,
    validateDisks,
    clearValidation,
  }
}
