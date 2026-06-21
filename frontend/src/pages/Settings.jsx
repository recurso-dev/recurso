import React, { useState, useEffect } from 'react'
import { endpoints } from '../lib/api'

const Settings = () => {
    const [account, setAccount] = useState({ name: '', email: '' })
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [message, setMessage] = useState(null)

    useEffect(() => {
        const fetchAccount = async () => {
            try {
                const response = await endpoints.getAccount()
                if (response.data.data) {
                    setAccount({
                        name: response.data.data.name,
                        email: response.data.data.email
                    })
                }
            } catch (error) {
                console.error("Failed to fetch account:", error)
            } finally {
                setLoading(false)
            }
        }
        fetchAccount()
    }, [])

    const handleSave = async () => {
        setSaving(true)
        setMessage(null)
        try {
            await endpoints.updateAccount(account)
            setMessage({ type: 'success', text: 'Settings saved successfully.' })
        } catch (error) {
            console.error("Failed to update account:", error)
            setMessage({ type: 'error', text: 'Failed to save settings.' })
        } finally {
            setSaving(false)
        }
    }

    return (
        <div className="flex flex-col max-w-4xl mx-auto px-4 py-8">
            <header className="flex flex-wrap items-center justify-between gap-4 mb-8">
                <div className="flex flex-col gap-1">
                    <h1 className="text-slate-900 dark:text-white text-3xl font-bold tracking-tight">Settings</h1>
                    <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal">Manage your account information.</p>
                </div>
                <button
                    onClick={handleSave}
                    disabled={saving || loading}
                    className="flex items-center justify-center gap-2 overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-semibold leading-normal shadow-sm hover:bg-primary/90 disabled:opacity-50 transition-all"
                >
                    {saving ? <span className="material-symbols-outlined animate-spin text-lg">sync</span> : <span className="material-symbols-outlined text-lg">save</span>}
                    <span className="truncate">{saving ? 'Saving...' : 'Save Changes'}</span>
                </button>
            </header>

            {message && (
                <div className={`mb-6 p-4 rounded-lg ${message.type === 'success' ? 'bg-green-50 text-green-800 dark:bg-green-900/30 dark:text-green-300' : 'bg-red-50 text-red-800 dark:bg-red-900/30 dark:text-red-300'}`}>
                    {message.text}
                </div>
            )}

            {/* Section: General */}
            <section className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6 shadow-sm">
                <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-6">General Information</h2>
                <div className="grid gap-6 max-w-xl">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Company Name</label>
                        <input
                            type="text"
                            className="w-full rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-950 px-3 py-2 text-sm focus:border-primary focus:ring-1 focus:ring-primary text-slate-900 dark:text-white"
                            value={account.name}
                            onChange={(e) => setAccount({ ...account, name: e.target.value })}
                            placeholder="e.g. Acme Corp"
                            disabled={loading}
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Support Email</label>
                        <input
                            type="email"
                            className="w-full rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-950 px-3 py-2 text-sm focus:border-primary focus:ring-1 focus:ring-primary text-slate-900 dark:text-white"
                            value={account.email}
                            onChange={(e) => setAccount({ ...account, email: e.target.value })}
                            placeholder="support@example.com"
                            disabled={loading}
                        />
                    </div>
                </div>
            </section>
        </div>
    )
}

export default Settings
