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
        // Vite 8 bundles with rolldown, whose Rollup-compat `manualChunks`
        // shim does NOT reliably place shared React — it co-locates
        // react/react-dom/jsx-runtime into the first vendor chunk that needs
        // them (`charts`), so every page then statically imports the 968 kB
        // charting stack. Use rolldown's native advancedChunks, ordered so
        // React is claimed BEFORE charts, keeping `charts` to just the
        // analytics libraries that only the lazy analytics routes pull.
        advancedChunks: {
          groups: [
            { name: 'react-vendor', test: /node_modules\/(react|react-dom|scheduler|use-sync-external-store|react-router|@tanstack)\// },
            // clsx/tailwind-merge/cva back the `cn()` helper every component
            // uses. They must be claimed BEFORE charts — Tremor/recharts also
            // use clsx, so otherwise the bundler leaves clsx inside `charts`
            // and every cn() caller transitively pulls the charting stack.
            { name: 'ui-utils', test: /node_modules\/(clsx|tailwind-merge|class-variance-authority)\// },
            // Positioning/interaction primitives shared by BOTH Tremor and
            // Radix (@floating-ui, @headlessui, aria-hidden, react-remove-
            // scroll, tslib). Claim them before charts or they co-locate into
            // it and Radix/Stripe re-import the charting stack from there.
            { name: 'ui-vendor', test: /node_modules\/(@floating-ui|@headlessui|aria-hidden|react-remove-scroll|react-remove-scroll-bar|tslib|prop-types|react-is)\// },
            { name: 'charts', test: /node_modules\/(@tremor|recharts|d3-|victory|internmap)/ },
            { name: 'radix', test: /node_modules\/@radix-ui/ },
            { name: 'stripe', test: /node_modules\/@stripe/ },
            // Catch-all for every other dependency. Ordered LAST so the
            // specific groups win, but present so no orphaned shared micro-lib
            // (clsx-style transitive deps of charts) is left ungrouped and
            // greedily absorbed INTO the charts chunk — which would make an
            // eager chunk that needs it re-import the whole charting stack.
            { name: 'vendor', test: /node_modules/ },
          ],
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
