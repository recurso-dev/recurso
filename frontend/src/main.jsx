import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.jsx'
import { AuthProvider } from './auth/AuthProvider'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from './lib/queryClient'
import { ToastProvider } from './components/Toast'
import { Toaster } from '@/components/ui/sonner'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')).render(
    <React.StrictMode>
        <BrowserRouter>
            <QueryClientProvider client={queryClient}>
            <ToastProvider>
                <AuthProvider>
                    <App />
                    <Toaster />
                </AuthProvider>
            </ToastProvider>
            </QueryClientProvider>
        </BrowserRouter>
    </React.StrictMode>,
)

