import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import * as Sentry from '@sentry/react'
import posthog from 'posthog-js'
import App from './App.jsx'
import { AuthProvider } from './auth/AuthProvider'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from './lib/queryClient'
import { Toaster } from '@/components/ui/sonner'
// Self-hosted fonts (no CDN): Inter for the UI, JetBrains Mono for money/code/IDs
// — so the ledger's numbers render identically on every OS.
import '@fontsource/inter/400.css'
import '@fontsource/inter/500.css'
import '@fontsource/inter/600.css'
import '@fontsource/inter/700.css'
import '@fontsource/inter/800.css'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'
import '@fontsource/jetbrains-mono/600.css'
import './index.css'

// Error tracking (Sentry): inert unless VITE_SENTRY_DSN is set at build time,
// mirroring the API's SENTRY_DSN gating.
if (import.meta.env.VITE_SENTRY_DSN) {
    Sentry.init({
        dsn: import.meta.env.VITE_SENTRY_DSN,
        environment: import.meta.env.MODE,
        // Errors only by default; turn on tracing/replay later if wanted.
        tracesSampleRate: 0,
    })
}

// Product analytics (PostHog): inert unless VITE_POSTHOG_KEY is set at build
// time, mirroring Sentry's gating. Autocaptures pageviews + interactions; the
// signed-in tenant/user is identified in AuthProvider once auth resolves.
if (import.meta.env.VITE_POSTHOG_KEY) {
    posthog.init(import.meta.env.VITE_POSTHOG_KEY, {
        api_host: import.meta.env.VITE_POSTHOG_HOST || 'https://us.i.posthog.com',
        capture_pageview: true,
    })
}

ReactDOM.createRoot(document.getElementById('root')).render(
    <React.StrictMode>
        <BrowserRouter>
            <QueryClientProvider client={queryClient}>
                <AuthProvider>
                    <App />
                    <Toaster />
                </AuthProvider>
            </QueryClientProvider>
        </BrowserRouter>
    </React.StrictMode>,
)

