import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/dashboard/',
  server: {
    proxy: {
      '/api': 'http://localhost:20181',
      '/v1': 'http://localhost:20181',
    }
  },
  build: {
    outDir: '../../internal/dashboard/dist',
    emptyOutDir: true,
  }
})
