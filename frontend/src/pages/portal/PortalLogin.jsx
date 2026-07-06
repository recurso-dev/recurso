import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { API_ROOT as API_BASE } from '../../lib/api'

const PortalLogin = () => {
    const [email, setEmail] = useState('')
    const [loading, setLoading] = useState(false)
    const [success, setSuccess] = useState(false)
    const [devLink, setDevLink] = useState(null)
    const [error, setError] = useState(null)
    const navigate = useNavigate()

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)
        setError(null)

        try {
            const response = await fetch(`${API_BASE}/portal/auth/request`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email })
            })

            const data = await response.json()

            if (response.ok) {
                setSuccess(true)
                // For development, show the magic link
                if (data._dev_link) {
                    setDevLink(data._dev_link)
                }
            } else {
                setError(data.error?.message || 'Failed to send login link')
            }
        } catch (err) {
            setError('Network error. Please try again.')
        } finally {
            setLoading(false)
        }
    }

    const handleDevLogin = async () => {
        if (!devLink) return

        try {
            const response = await fetch(`${API_BASE}${devLink}`)
            const data = await response.json()

            if (response.ok && data.session_token) {
                localStorage.setItem('portal_session', data.session_token)
                navigate('/portal/dashboard')
            } else {
                setError('Failed to verify link')
            }
        } catch (err) {
            setError('Failed to verify link')
        }
    }

    if (success) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-950 px-4">
                <div className="w-full max-w-md">
                    <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 text-center">
                        <div className="w-16 h-16 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
                            <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                            </svg>
                        </div>
                        <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">Check your email</h2>
                        <p className="text-slate-600 dark:text-slate-400 mb-6">
                            We've sent a login link to <strong>{email}</strong>
                        </p>

                        {/* Development only */}
                        {devLink && (
                            <div className="mt-6 p-4 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800">
                                <p className="text-xs text-amber-700 dark:text-amber-400 mb-2">Development Mode:</p>
                                <button
                                    onClick={handleDevLogin}
                                    className="w-full px-4 py-2 bg-amber-500 text-white rounded-lg hover:bg-amber-600 transition-colors text-sm font-medium"
                                >
                                    Click here to login (dev only)
                                </button>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-950 px-4">
            <div className="w-full max-w-md">
                <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8">
                    {/* Logo */}
                    <div className="flex items-center justify-center gap-2 mb-8">
                        <div className="w-10 h-10 rounded-xl bg-primary flex items-center justify-center">
                            <span className="text-white font-bold text-xl">R</span>
                        </div>
                        <span className="text-2xl font-bold text-slate-900 dark:text-white">Recurso</span>
                    </div>

                    <h1 className="text-2xl font-bold text-center text-slate-900 dark:text-white mb-2">
                        Customer Portal
                    </h1>
                    <p className="text-center text-slate-600 dark:text-slate-400 mb-8">
                        Enter your email to access your billing portal
                    </p>

                    {error && (
                        <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded-lg text-sm">
                            {error}
                        </div>
                    )}

                    <form onSubmit={handleSubmit}>
                        <div className="mb-6">
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Email Address
                            </label>
                            <input
                                type="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="you@company.com"
                                required
                                className="w-full px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                            />
                        </div>

                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full py-3 px-4 bg-primary text-white font-semibold rounded-lg hover:bg-primary/90 disabled:opacity-50 transition-all"
                        >
                            {loading ? 'Sending...' : 'Send Login Link'}
                        </button>
                    </form>

                    <p className="mt-6 text-center text-sm text-slate-500 dark:text-slate-400">
                        We'll email you a magic link for password-free sign in.
                    </p>
                </div>
            </div>
        </div>
    )
}

export default PortalLogin
