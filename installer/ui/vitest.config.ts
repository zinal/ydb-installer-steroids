import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// Vitest config for unit tests under ui/tests/.
export default defineConfig({
    plugins: [vue()],
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url)),
        },
    },
    test: {
        environment: 'jsdom',
        globals: true,
        include: ['tests/**/*.spec.ts'],
      // Global setup: registers Vuetify for all component tests.
      setupFiles: ['tests/setup.ts'],
      server: {
          deps: {
              // Inline Vuetify so Vitest can process its ESM exports.
              inline: ['vuetify'],
          },
      },
        // Coverage via V8 provider: `npm run test:coverage` (make coverage-ui).
        coverage: {
            provider: 'v8',
            reporter: ['text', 'html', 'lcov'],
            include: ['src/**/*.{ts,vue}'],
            exclude: [
                'src/main.ts',
                'src/shims.d.ts',
                'src/plugins/**',
                'src/types/**',
            ],
            thresholds: {
                lines: 88,
                branches: 83,
                functions: 66,
                statements: 88,
            },
        },
    },
})
