import { ref } from 'vue'

export function useHostFileImport() {
  const importing = ref(false)
  const importError = ref<string | null>(null)

  async function readHostFile(file: File): Promise<string[]> {
    importing.value = true
    importError.value = null
    try {
      const text = await file.text()
      const lines = text
        .split('\n')
        .map(l => l.trim())
        .filter(l => l.length > 0 && !l.startsWith('#'))
      if (lines.length === 0) {
        importError.value = 'hosts.fileEmpty'
        return []
      }
      return lines
    } catch {
      importError.value = 'hosts.fileReadError'
      return []
    } finally {
      importing.value = false
    }
  }

  return { importing, importError, readHostFile }
}
