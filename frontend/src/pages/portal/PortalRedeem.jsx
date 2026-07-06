import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

import { API_ROOT as API_BASE } from '../../lib/api'

const PortalRedeem = () => {
    const [code, setCode] = useState('')
    const [loading, setLoading] = useState(false)
    const [status, setStatus] = useState({ type: '', message: '' }) // type: 'success' | 'error'
    const navigate = useNavigate()

    const sessionToken = localStorage.getItem('portal_session')

    useEffect(() => {
        if (!sessionToken) {
            navigate('/portal/login')
            return
        }
    }, [sessionToken, navigate])

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)
        setStatus({ type: '', message: '' })

        try {
            const response = await fetch(`${API_BASE}/portal/api/redeem`, {
                method: 'POST',
                headers: {
                    'X-Portal-Session': sessionToken,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ code })
            })

            const data = await response.json()

            if (!response.ok) {
                if (response.status === 401) {
                    localStorage.removeItem('portal_session')
                    navigate('/portal/login')
                    return
                }
                throw new Error(data.error?.message || 'Failed to redeem gift')
            }

            setStatus({ type: 'success', message: 'Gift redeemed successfully! Redirecting...' })
            setTimeout(() => navigate('/portal/dashboard'), 2000)
        } catch (err) {
            setStatus({ type: 'error', message: err.message })
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-950 p-4">
            <div className="w-full max-w-md bg-white dark:bg-slate-900 rounded-2xl shadow-xl border border-slate-200 dark:border-slate-800 p-8">
                <div className="text-center mb-8">
                    <div className="w-12 h-12 bg-primary/10 rounded-xl flex items-center justify-center mx-auto mb-4 text-primary">
                        <span className="material-symbols-outlined text-2xl">redeem</span>
                    </div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Redeem Gift</h1>
                    <p className="text-slate-600 dark:text-slate-400 mt-2">Enter your gift code to claim your subscription.</p>
                </div>

                <form onSubmit={handleSubmit} className="space-y-4">
                    {status.message && (
                        <div className={`p-3 rounded-lg text-sm ${status.type === 'success'
                                ? 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                                : 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                            }`}>
                            {status.message}
                        </div>
                    )}

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Gift Code</label>
                        <input
                            type="text"
                            value={code}
                            onChange={(e) => setCode(e.target.value)}
                            placeholder="GIFT-XXXXXXXX"
                            required
                            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm placeholder-slate-400 shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white dark:placeholder-slate-500 font-mono"
                        />
                    </div>

                    <button
                        type="submit"
                        disabled={loading}
                        className="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-bold text-white shadow-sm hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 disabled:opacity-50 transition-all"
                    >
                        {loading ? 'Redeeming...' : 'Redeem Gift'}
                    </button>
                </form>

                <div className="mt-6 text-center">
                    <button
                        onClick={() => navigate('/portal/dashboard')}
                        className="text-sm text-slate-500 hover:text-slate-900 dark:hover:text-white"
                    >
                        Back to Dashboard
                    </button>
                </div>
            </div>
        </div>
    )
}

export default PortalRedeem
