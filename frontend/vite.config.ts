import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'

// Dev workflow: run `go run ./cmd/server` in backend/ for the API, and
// `npm run dev` here. Vite serves the SPA on :5173 and forwards /api/*
// to the Go server, so the browser hits one origin and CORS doesn't get
// in the way. Override VITE_API_TARGET if your backend isn't on :8080.
const API_TARGET = process.env.VITE_API_TARGET || 'http://localhost:8080'

export default defineConfig({
  plugins: [
    vue(),
    vuetify({ autoImport: true }),
  ],
  server: {
    host: true,
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
      },
    },
  },
})
