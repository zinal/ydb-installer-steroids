import { computed, ref } from 'vue'
import { api } from '@/api/client'

export interface DepState {
  name: string
  ok: boolean
  version?: string
  required?: string
  installed: boolean
  source?: string
  skipped: boolean
  instructionOpen: boolean
  instruction: string | null
  instructionLoading: boolean
}

export function useDeps() {
  const deps = ref<DepState[]>([])
  const loading = ref(true)
  const error = ref<string | null>(null)
  const bundled = ref(false)

  const canProceed = computed(() =>
    !loading.value && deps.value.every((d) => d.ok || d.skipped),
  )

  const hasSkipped = computed(() => deps.value.some((d) => d.skipped))

  async function fetchDeps() {
    loading.value = true
    error.value = null
    try {
      const r = await api.getDeps()
      bundled.value = r.bundled === true
      deps.value = r.dependencies.map((d) => ({
        ...d,
        skipped: false,
        instructionOpen: false,
        instruction: null,
        instructionLoading: false,
      }))
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      deps.value = []
    } finally {
      loading.value = false
    }
  }

  async function toggleInstruction(name: string) {
    const dep = deps.value.find((d) => d.name === name)
    if (!dep) return

    dep.instructionOpen = !dep.instructionOpen

    if (dep.instructionOpen && dep.instruction === null) {
      dep.instructionLoading = true
      try {
        const os = detectOS()
        const r = await api.getDepsInstruction(name, os)
        dep.instruction = r.commands
      } catch {
        dep.instruction = ''
      } finally {
        dep.instructionLoading = false
      }
    }
  }

  function skipDep(name: string) {
    const dep = deps.value.find((d) => d.name === name)
    if (dep) {
      dep.skipped = true
      dep.instructionOpen = false
    }
  }

  function unskipDep(name: string) {
    const dep = deps.value.find((d) => d.name === name)
    if (dep) dep.skipped = false
  }

  async function retryDeps() {
    loading.value = true
    error.value = null
    deps.value.forEach((d) => {
      d.skipped = false
      d.instructionOpen = false
      d.instruction = null
    })
    await fetchDeps()
  }

  return {
    deps,
    loading,
    error,
    bundled,
    canProceed,
    hasSkipped,
    fetchDeps,
    retryDeps,
    toggleInstruction,
    skipDep,
    unskipDep,
  }
}

function detectOS(): string {
  const p = (navigator.platform ?? '').toLowerCase()
  const ua = (navigator.userAgent ?? '').toLowerCase()
  if (p.includes('mac') || ua.includes('mac')) return 'macos'
  if (p.includes('linux') || ua.includes('linux')) return 'ubuntu'
  return 'ubuntu'
}
