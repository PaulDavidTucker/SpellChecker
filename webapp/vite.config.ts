import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Base path for GitHub Pages - uncomment and update if deploying to GitHub Pages
  // base: '/SpellChecker/',
  server: {
    port: 3000,
    host: '0.0.0.0',
    proxy: {
      '/check': {
        target: 'http://spellchecker:8080',
        changeOrigin: true,
      },
      '/profiles': {
        target: 'http://spellchecker:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://spellchecker:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})
