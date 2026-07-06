import React, { useState, useEffect, useCallback } from 'react'
import { endpoints } from '../../lib/api'
import { useToast } from '../../components/Toast'

const IRPSettings = () => {
    const [config, setConfig] = useState({
        environment: 'sandbox',
        client_id: '',
        client_secret: '',
        username: '',
        password: '',
        gstin: '',
        is_enabled: false,
    })
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [testing, setTesting] = useState(false)
    const [testResult, setTestResult] = useState(null)
    const toast = useToast()

    const fetchConfig = useCallback(async () => {
        setLoading(true)
        try {
            const response = await endpoints.getIRPConfig()
            if (response.data?.data) {
                setConfig(prev => ({ ...prev, ...response.data.data }))
            }
        } catch (err) {
            // Config may not exist yet, that's OK
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        fetchConfig()
    }, [fetchConfig])

    const handleSave = async (e) => {
        e.preventDefault()
        setSaving(true)
        try {
            await endpoints.updateIRPConfig(config)
            toast.success('IRP configuration saved successfully')
        } catch (err) {
            toast.error(err?.response?.data?.error?.message || 'Failed to save configuration')
        } finally {
            setSaving(false)
        }
    }

    const handleTest = async () => {
        setTesting(true)
        setTestResult(null)
        try {
            const response = await endpoints.testIRPConfig()
            setTestResult(response.data)
        } catch (err) {
            setTestResult({ success: false, message: err?.response?.data?.error?.message || 'Connection test failed' })
        } finally {
            setTesting(false)
        }
    }

    if (loading) {
        return (
            <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:px-8">
                <div className="animate-pulse space-y-4">
                    <div className="h-8 bg-slate-200 dark:bg-slate-700 rounded w-48"></div>
                    <div className="h-64 bg-slate-200 dark:bg-slate-700 rounded"></div>
                </div>
            </div>
        )
    }

    return (
        <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:px-8">
            <div className="pb-6 border-b border-slate-200 dark:border-slate-800">
                <h1 className="text-slate-900 dark:text-white text-3xl font-bold leading-tight tracking-tight">IRP Settings</h1>
                <p className="mt-1 text-base font-normal text-slate-500 dark:text-slate-400">
                    Configure NIC Invoice Registration Portal credentials for e-invoicing.
                </p>
            </div>

            <form onSubmit={handleSave} className="mt-8 space-y-6">
                {/* Enable Toggle */}
                <div className="flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-700 p-4">
                    <div>
                        <h3 className="text-sm font-medium text-slate-900 dark:text-white">Enable E-Invoicing</h3>
                        <p className="text-sm text-slate-500 dark:text-slate-400">Generate IRN for B2B invoices via NIC IRP</p>
                    </div>
                    <button
                        type="button"
                        onClick={() => setConfig(prev => ({ ...prev, is_enabled: !prev.is_enabled }))}
                        className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${config.is_enabled ? 'bg-primary' : 'bg-slate-200 dark:bg-slate-600'}`}
                    >
                        <span className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${config.is_enabled ? 'translate-x-5' : 'translate-x-0'}`} />
                    </button>
                </div>

                {/* Environment */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Environment</label>
                    <select
                        value={config.environment}
                        onChange={(e) => setConfig(prev => ({ ...prev, environment: e.target.value }))}
                        className="w-full rounded-lg border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-white text-sm focus:border-primary focus:ring-primary/20"
                    >
                        <option value="sandbox">Sandbox (Testing)</option>
                        <option value="production">Production</option>
                    </select>
                </div>

                {/* GSTIN */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">GSTIN</label>
                    <input
                        type="text"
                        value={config.gstin}
                        onChange={(e) => setConfig(prev => ({ ...prev, gstin: e.target.value.toUpperCase() }))}
                        placeholder="e.g., 33ABCDE1234F1Z5"
                        maxLength={15}
                        className="w-full rounded-lg border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-white text-sm font-mono focus:border-primary focus:ring-primary/20"
                    />
                </div>

                {/* Client ID */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Client ID</label>
                    <input
                        type="text"
                        value={config.client_id}
                        onChange={(e) => setConfig(prev => ({ ...prev, client_id: e.target.value }))}
                        placeholder="NIC API Client ID"
                        className="w-full rounded-lg border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-white text-sm focus:border-primary focus:ring-primary/20"
                    />
                </div>

                {/* Client Secret */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Client Secret</label>
                    <input
                        type="password"
                        value={config.client_secret}
                        onChange={(e) => setConfig(prev => ({ ...prev, client_secret: e.target.value }))}
                        placeholder="NIC API Client Secret"
                        className="w-full rounded-lg border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-white text-sm focus:border-primary focus:ring-primary/20"
                    />
                </div>

                {/* Username */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Username</label>
                    <input
                        type="text"
                        value={config.username}
                        onChange={(e) => setConfig(prev => ({ ...prev, username: e.target.value }))}
                        placeholder="NIC API Username"
                        className="w-full rounded-lg border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-white text-sm focus:border-primary focus:ring-primary/20"
                    />
                </div>

                {/* Password */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Password</label>
                    <input
                        type="password"
                        value={config.password}
                        onChange={(e) => setConfig(prev => ({ ...prev, password: e.target.value }))}
                        placeholder="NIC API Password"
                        className="w-full rounded-lg border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-white text-sm focus:border-primary focus:ring-primary/20"
                    />
                </div>

                {/* Test Result */}
                {testResult && (
                    <div className={`rounded-lg px-4 py-3 text-sm ${testResult.success
                            ? 'bg-green-50 text-green-800 border border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
                            : 'bg-red-50 text-red-800 border border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
                        }`}>
                        {testResult.message}
                    </div>
                )}

                {/* Actions */}
                <div className="flex gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
                    <button
                        type="submit"
                        disabled={saving}
                        className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-primary/90 disabled:opacity-50"
                    >
                        {saving ? 'Saving...' : 'Save Configuration'}
                    </button>
                    <button
                        type="button"
                        onClick={handleTest}
                        disabled={testing}
                        className="rounded-lg bg-slate-100 dark:bg-slate-700 px-4 py-2 text-sm font-semibold text-slate-700 dark:text-slate-300 shadow-sm hover:bg-slate-200 dark:hover:bg-slate-600 disabled:opacity-50"
                    >
                        {testing ? 'Testing...' : 'Test Connection'}
                    </button>
                </div>
            </form>
        </div>
    )
}

export default IRPSettings
