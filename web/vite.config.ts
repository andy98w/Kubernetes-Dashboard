import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const apiTarget = loadEnv(mode, '.', '').VITE_API_TARGET || 'http://localhost:8080'
  return {
    plugins: [react()],
    server: { proxy: { '/api': apiTarget, '/healthz': apiTarget } },
  }
})
