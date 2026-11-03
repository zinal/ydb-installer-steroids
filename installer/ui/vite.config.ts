import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'
import { fileURLToPath, URL } from 'node:url'

// Vite configuration for the YDB Installer SPA.
//
// In dev mode the browser hits Vite on :5173 and we proxy /api and /ws to
// the Go backend on :8080. The production build emits files into ../internal/embed/dist
// so the Go binary can pick them up via //go:embed.
export default defineConfig({
  plugins: [
        vue(),
        vuetify({ autoImport: true }),
    ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
        outDir: '../internal/embed/dist',
        emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
        '/api': 'http://127.0.0.1:8080',
        '/ws': { target: 'ws://127.0.0.1:8080', ws: true },
      },
  },
})
