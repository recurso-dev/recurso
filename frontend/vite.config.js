import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    // The 'charts' chunk (Tremor + recharts + d3) is large but intentionally
    // isolated and lazy — only analytics routes pull it — so don't warn on it.
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        // Split heavy vendor code out of the main bundle so non-chart routes
        // (which are lazy-loaded) don't pay for the charting stack. Charts
        // (Tremor + its transitive recharts/d3) are only pulled by a handful of
        // analytics pages.
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (/@tremor|recharts|d3-|victory|internmap/.test(id)) return 'charts'
          if (id.includes('@radix-ui')) return 'radix'
          if (id.includes('@stripe')) return 'stripe'
          if (/react-router|@tanstack/.test(id)) return 'react-vendor'
        },
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.js',
    css: true,
  },
  server: {
    proxy: {
      '/v1': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
      },
      '/auth': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
      },
      // Backend portal endpoints; SPA pages live at /portal/login etc.,
      // which don't collide with these two prefixes.
      '/portal/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
      },
      '/portal/auth': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
      },
      // Backend metadata (gateway_mode drives the Test-mode chip).
      '/version': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
      },
      // /checkout/:id is both an SPA page (browser navigation) and a JSON
      // API (fetch). Route page loads to the SPA, everything else to the API.
      '/checkout': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
        bypass: (req) => {
          if (req.headers.accept?.includes('text/html')) return '/index.html'
        },
      }
    }
  }
})
