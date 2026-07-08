import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  ssgOptions: {
    // /vs/lago -> dist/vs/lago/index.html (real static HTML per route)
    dirStyle: 'nested',
    // The /vs/* pages set their own <title>/<meta> via <Head>. Strip the
    // default landing-page tags from the template for those routes so the
    // per-page tags are the only ones in <head>. The landing page ("/") is
    // left untouched.
    onBeforePageRender(route, indexHTML) {
      if (!route.startsWith('/vs/')) return indexHTML
      return indexHTML
        .replace(/\s*<title>[^<]*<\/title>/i, '')
        .replace(/\s*<meta\s+name="description"[^>]*>/i, '')
        .replace(/\s*<meta\s+property="og:title"[^>]*>/i, '')
        .replace(/\s*<meta\s+property="og:description"[^>]*>/i, '')
    },
  },
})
