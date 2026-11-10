import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The Go server serves the built app from web/dist; in development, Vite proxies the API to the server on :8080.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/healthz': 'http://127.0.0.1:8080',
    },
  },
})
