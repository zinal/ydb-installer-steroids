import { config } from '@vue/test-utils'
import { beforeAll, afterAll, vi } from 'vitest'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'

// Vuetify uses ResizeObserver internally (e.g. VProgressCircular, VCard).
// jsdom does not implement it, so we provide a no-op stub.
if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

if (typeof globalThis.IntersectionObserver === 'undefined') {
  globalThis.IntersectionObserver = class IntersectionObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof IntersectionObserver
}

// Vuetify's VOverlay/locationStrategies uses visualViewport.
if (typeof window !== 'undefined' && !window.visualViewport) {
  Object.defineProperty(window, 'visualViewport', {
    writable: true,
    value: {
      width: 1024,
      height: 768,
      offsetLeft: 0,
      offsetTop: 0,
      pageLeft: 0,
      pageTop: 0,
      scale: 1,
      addEventListener: () => {},
      removeEventListener: () => {},
    },
  })
}

if (typeof window !== 'undefined' && !window.matchMedia) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  })
}

// jsdom's File implementation doesn't have the text() method.
if (typeof globalThis.File !== 'undefined') {
  const OriginalFile = globalThis.File as unknown as {
    new(parts: BlobPart[], name: string, options?: FilePropertyBag): File
    prototype: File
  }
  if (!OriginalFile.prototype.text) {
    OriginalFile.prototype.text = async function (this: File): Promise<string> {
      return new Promise((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = () => resolve(reader.result as string)
        reader.onerror = () => reject(reader.error)
        reader.readAsText(this)
      })
    }
  }
}

const vuetify = createVuetify({ components, directives })
config.global.plugins = [vuetify]

// Some tests intentionally trigger error paths that log to console.error.
const originalError = console.error
beforeAll(() => {
  console.error = (...args: unknown[]) => {
    // Suppress "Failed to fetch YDB versions" error from distribution store tests
    if (typeof args[0] === 'string' && args[0].includes('Failed to fetch YDB versions')) {
      return
    }
    originalError.apply(console, args)
  }
})
afterAll(() => {
  console.error = originalError
})
