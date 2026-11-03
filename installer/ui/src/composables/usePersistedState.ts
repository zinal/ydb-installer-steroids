import { onMounted, onUnmounted } from 'vue'
import type { Store } from 'pinia'

const SCHEMA_VERSION = 1
const KEY_PREFIX = 'ydb-installer.ui'
const DEBOUNCE_MS = 300

interface Envelope {
  version: number
  data: Record<string, unknown>
}

function replacer(_key: string, value: unknown): unknown {
  if (value instanceof Set) {
    return { __type: 'Set', value: Array.from(value) }
  }
  if (value instanceof Map) {
    return { __type: 'Map', value: Array.from(value.entries()) }
  }
  return value
}

function reviver(_key: string, value: unknown): unknown {
  if (value && typeof value === 'object' && '__type' in (value as Record<string, unknown>)) {
    const tagged = value as { __type: string; value: unknown }
    if (tagged.__type === 'Set') return new Set(tagged.value as unknown[])
    if (tagged.__type === 'Map') return new Map(tagged.value as [unknown, unknown][])
  }
  return value
}

function storageKey(name: string): string {
  return `${KEY_PREFIX}.${name}`
}

function getStorage(): Storage | null {
  try {
    return typeof window !== 'undefined' ? window.sessionStorage : null
  } catch {
    return null
  }
}

export interface PersistedStateOptions<S extends Record<string, unknown>> {
  name: string
  paths: (keyof S)[]
  version?: number
}

export function usePersistedState<S extends Record<string, unknown>>(
  // Using a loosely-typed Pinia store here (rather than `Store<string, S>`)
  // sidesteps generic-variance friction with `$patch`'s UnwrapRef<S> overload;
  // callers still get full type safety on `options.paths`.
  store: Store,
  options: PersistedStateOptions<S>,
) {
  const version = options.version ?? SCHEMA_VERSION
  const key = storageKey(options.name)
  let isApplying = false
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let unsubscribe: (() => void) | null = null

  function readEnvelope(): Envelope | null {
    const storage = getStorage()
    if (!storage) return null
    const raw = storage.getItem(key)
    if (!raw) return null
    try {
      const parsed = JSON.parse(raw, reviver) as Envelope
      if (parsed.version !== version) return null
      return parsed
    } catch {
      return null
    }
  }

  function rehydrate() {
    const envelope = readEnvelope()
    if (!envelope) return
    isApplying = true
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      store.$patch((state: any) => {
        for (const p of options.paths) {
          if (Object.prototype.hasOwnProperty.call(envelope.data, p as string)) {
            state[p as string] = envelope.data[p as string]
          }
        }
      })
    } finally {
      isApplying = false
    }
  }

  function persist() {
    const storage = getStorage()
    if (!storage) return
    const data: Record<string, unknown> = {}
    for (const p of options.paths) {
      data[p as string] = (store.$state as Record<string, unknown>)[p as string]
    }
    const envelope: Envelope = { version, data }
    try {
      storage.setItem(key, JSON.stringify(envelope, replacer))
    } catch {
    }
  }

  function scheduleWrite() {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(persist, DEBOUNCE_MS)
  }

  rehydrate()

  onMounted(() => {
    unsubscribe = store.$subscribe(() => {
      if (!isApplying) scheduleWrite()
    })
  })

  onUnmounted(() => {
    if (debounceTimer) clearTimeout(debounceTimer)
    if (unsubscribe) unsubscribe()
  })

  return { rehydrate, persist }
}

export function clearAllPersistedState() {
  const storage = getStorage()
  if (!storage) return
  const keys: string[] = []
  for (let i = 0; i < storage.length; i++) {
    const k = storage.key(i)
    if (k && k.startsWith(KEY_PREFIX)) keys.push(k)
  }
  for (const k of keys) storage.removeItem(k)
}
