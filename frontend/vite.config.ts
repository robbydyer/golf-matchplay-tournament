import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    host: '0.0.0.0',
    allowedHosts: true,
    proxy: {
      '/api': {
        target: process.env.API_URL || 'http://backend:8080',
        changeOrigin: true,
      },
    },
    watch: {
      usePolling: true,
    },
  },
})
