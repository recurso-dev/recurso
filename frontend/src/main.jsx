import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.jsx'
import { AuthProvider } from './auth/AuthProvider'
import { ToastProvider } from './components/Toast'
import { Toaster } from '@/components/ui/sonner'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')).render(
    <React.StrictMode>
        <BrowserRouter>
            <ToastProvider>
                <AuthProvider>
                    <App />
                    <Toaster />
                </AuthProvider>
            </ToastProvider>
        </BrowserRouter>
    </React.StrictMode>,
)

