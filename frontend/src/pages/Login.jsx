import React, { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import axios from 'axios'
import { API_BASE } from '../lib/api'
import { useAuth } from '../auth/AuthProvider'

const Login = () => {
    const [key, setKey] = useState('')
    const [error, setError] = useState(null)
    const [checking, setChecking] = useState(false)
    const { login } = useAuth()
    const navigate = useNavigate()

    const handleLogin = async (e) => {
        e.preventDefault()
        if (!key || checking) return
        setChecking(true)
        setError(null)
        try {
            // Validate the key before storing it so a typo fails here,
            // not on the first dashboard request.
            await axios.get(`${API_BASE}/account`, {
                headers: { Authorization: `Bearer ${key}` },
            })
            login(key)
            navigate('/')
        } catch (err) {
            if (err?.response?.status === 401 || err?.response?.status === 403) {
                setError('That API key was rejected. Check it and try again.')
            } else {
                setError('Could not reach the API to verify the key. Is the server running?')
            }
        } finally {
            setChecking(false)
        }
    }

    return (
        <div className="flex min-h-screen w-full font-sans bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-white transition-colors duration-200">
            {/* Left Column: Branding (Desktop Only) - Same as Register for consistency */}
            <div className="relative hidden w-0 flex-1 lg:flex flex-col justify-between bg-slate-900 overflow-hidden">
                <div className="absolute inset-0 bg-cover bg-center opacity-40" style={{ backgroundImage: 'linear-gradient(135deg, #1736cf 0%, #111421 100%)' }}></div>
                <div className="relative flex h-full flex-col justify-between p-12 z-10">
                    <div className="flex items-center gap-2">
                        <div className="flex items-center justify-center w-8 h-8 rounded bg-white/10 text-white">
                            <span className="material-symbols-outlined text-xl">dataset</span>
                        </div>
                        <span className="text-white text-xl font-bold tracking-tight">Recurso</span>
                    </div>

                    <div>
                        <h1 className="text-white text-4xl font-black leading-tight tracking-tight mb-4">
                            Welcome back
                        </h1>
                        <p className="text-slate-300 text-lg font-normal leading-relaxed max-w-md">
                            Log in to access your billing dashboard, manage subscriptions, and view analytics.
                        </p>
                    </div>
                </div>
            </div>

            {/* Right Column: Login Form */}
            <div className="flex flex-1 flex-col justify-center px-4 py-12 sm:px-6 lg:flex-none lg:px-20 xl:px-24 bg-white dark:bg-slate-950 overflow-y-auto">
                <div className="mx-auto w-full max-w-[420px]">
                    <div className="mb-10">
                        <h2 className="text-3xl font-black leading-tight tracking-tight text-slate-900 dark:text-white">
                            Log in
                        </h2>
                        <p className="mt-2 text-slate-500 dark:text-slate-400 text-base">
                            Enter your credentials to continue.
                        </p>
                    </div>

                    <form onSubmit={handleLogin} className="space-y-6">
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2" htmlFor="apiKey">
                                API Secret Key
                            </label>
                            <input
                                id="apiKey"
                                name="apiKey"
                                type="password"
                                required
                                value={key}
                                onChange={(e) => setKey(e.target.value)}
                                placeholder="recurso_sk_..."
                                className="w-full rounded-lg border-slate-300 bg-white px-4 py-3 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white font-mono text-sm"
                            />
                        </div>

                        {error && (
                            <p className="text-sm text-red-500" role="alert">{error}</p>
                        )}

                        <button
                            type="submit"
                            disabled={checking}
                            className="flex w-full items-center justify-center rounded-lg bg-primary px-5 py-3 text-base font-bold text-white transition-colors hover:bg-primary/90 disabled:opacity-60"
                        >
                            {checking ? 'Verifying…' : 'Log in'}
                        </button>
                    </form>

                    <div className="mt-8 text-center">
                        <p className="text-sm text-slate-500 dark:text-slate-400">
                            Don't have a workspace?{' '}
                            <Link to="/register" className="font-semibold text-primary hover:text-primary/80">
                                Create new tenant
                            </Link>
                        </p>
                        <p className="text-xs text-slate-400 mt-4">
                            Use the API key from your tenant registration.
                        </p>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Login
