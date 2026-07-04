import React, { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'

import { API_ROOT as API_BASE } from '../../lib/api'

const PortalDashboard = () => {
    const [profile, setProfile] = useState(null)
    const [invoices, setInvoices] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)
    const navigate = useNavigate()

    const sessionToken = localStorage.getItem('portal_session')

    useEffect(() => {
        if (!sessionToken) {
            navigate('/portal/login')
            return
        }
        fetchData()
    }, [sessionToken, navigate])

    const fetchData = async () => {
        try {
            const headers = {
                'X-Portal-Session': sessionToken,
                'Content-Type': 'application/json'
            }

            // Fetch profile
            const profileRes = await fetch(`${API_BASE}/portal/api/profile`, { headers })
            if (!profileRes.ok) {
                if (profileRes.status === 401) {
                    localStorage.removeItem('portal_session')
                    navigate('/portal/login')
                    return
                }
                throw new Error('Failed to fetch profile')
            }
            const profileData = await profileRes.json()
            setProfile(profileData)

            // Fetch invoices
            const invoicesRes = await fetch(`${API_BASE}/portal/api/invoices`, { headers })
            if (invoicesRes.ok) {
                const invoicesData = await invoicesRes.json()
                setInvoices(invoicesData.data || [])
            }
        } catch (err) {
            setError(err.message)
        } finally {
            setLoading(false)
        }
    }

    const handleLogout = async () => {
        try {
            await fetch(`${API_BASE}/portal/api/logout`, {
                method: 'POST',
                headers: { 'X-Portal-Session': sessionToken }
            })
        } catch (err) {
            // Ignore errors
        }
        localStorage.removeItem('portal_session')
        navigate('/portal/login')
    }

    const formatCurrency = (amount) => {
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: 'USD'
        }).format(amount / 100)
    }

    const formatDate = (date) => {
        return new Date(date).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        })
    }

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-950">
                <div className="text-center">
                    <div className="w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4" />
                    <p className="text-slate-600 dark:text-slate-400">Loading...</p>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
            {/* Header */}
            <header className="bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800">
                <div className="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
                            <span className="text-white font-bold">R</span>
                        </div>
                        <span className="text-lg font-bold text-slate-900 dark:text-white">Recurso</span>
                    </div>
                    <button
                        onClick={handleLogout}
                        className="text-sm text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white transition-colors"
                    >
                        Sign out
                    </button>
                </div>
            </header>

            {/* Main Content */}
            <main className="max-w-4xl mx-auto px-4 py-8">
                <h1 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">
                    Billing Portal
                </h1>
                <p className="text-slate-600 dark:text-slate-400 mb-8">
                    View your invoices and manage your subscription.
                </p>

                {error && (
                    <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded-lg">
                        {error}
                    </div>
                )}

                {/* Quick Stats */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
                    <div className="bg-white dark:bg-slate-900 rounded-xl p-6 border border-slate-200 dark:border-slate-800">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Total Invoices</p>
                        <p className="text-2xl font-bold text-slate-900 dark:text-white">{invoices.length}</p>
                    </div>
                    <div className="bg-white dark:bg-slate-900 rounded-xl p-6 border border-slate-200 dark:border-slate-800">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Total Paid</p>
                        <p className="text-2xl font-bold text-green-600 dark:text-green-400">
                            {formatCurrency(invoices.filter(inv => inv.status === 'paid').reduce((acc, inv) => acc + inv.amount_due, 0))}
                        </p>
                    </div>
                    <div className="bg-white dark:bg-slate-900 rounded-xl p-6 border border-slate-200 dark:border-slate-800">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Outstanding</p>
                        <p className="text-2xl font-bold text-amber-600 dark:text-amber-400">
                            {formatCurrency(invoices.filter(inv => inv.status !== 'paid').reduce((acc, inv) => acc + inv.amount_due, 0))}
                        </p>
                    </div>
                    {/* Referral Card */}
                    <div className="bg-gradient-to-br from-indigo-500 to-purple-600 rounded-xl p-6 text-white">
                        <p className="text-sm text-indigo-100 mb-1">Refer & Earn</p>
                        {profile?.referral_code ? (
                            <div>
                                <div className="flex items-center gap-2 bg-white/20 rounded-lg p-2 mb-2 backdrop-blur-sm">
                                    <code className="flex-1 font-mono text-sm font-bold">{profile.referral_code}</code>
                                    <button 
                                        onClick={() => navigator.clipboard.writeText(profile.referral_code)}
                                        className="p-1 hover:bg-white/20 rounded-md transition-colors"
                                        title="Copy Code"
                                    >
                                        <span className="material-symbols-outlined text-[18px]">content_copy</span>
                                    </button>
                                </div>
                                <p className="text-xs text-indigo-100">Share this code to earn credits!</p>
                            </div>
                        ) : (
                            <p className="text-sm font-medium">Generating code...</p>
                        )}
                    </div>
                </div>

                {/* Invoices Table */}
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden">
                    <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800">
                        <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Invoices</h2>
                    </div>

                    {invoices.length === 0 ? (
                        <div className="p-8 text-center text-slate-500 dark:text-slate-400">
                            No invoices found.
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="w-full">
                                <thead className="bg-slate-50 dark:bg-slate-800/50">
                                    <tr>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Invoice</th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Date</th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Amount</th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Status</th>
                                        <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">Actions</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                                    {invoices.map((invoice) => (
                                        <tr key={invoice.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/30">
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <span className="text-sm font-medium text-slate-900 dark:text-white">
                                                    {invoice.id?.substring(0, 8)}...
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                                {formatDate(invoice.created_at)}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white">
                                                {formatCurrency(invoice.amount_due)}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${invoice.status === 'paid'
                                                        ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                                                        : invoice.status === 'open'
                                                            ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                                                            : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                                                    }`}>
                                                    {invoice.status}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-right">
                                                <button className="text-sm text-primary hover:underline">
                                                    Download PDF
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>
            </main>
        </div>
    )
}

export default PortalDashboard
