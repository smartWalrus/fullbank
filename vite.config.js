import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    hmr: false,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
