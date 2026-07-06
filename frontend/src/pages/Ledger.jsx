import React, { useState, useEffect } from 'react'
import { endpoints } from '../lib/api'

const Ledger = () => {
    const [accounts, setAccounts] = useState([])
    const [entries, setEntries] = useState([])
    const [loading, setLoading] = useState(true)
    const [entriesLoading, setEntriesLoading] = useState(false)
    const [error, setError] = useState(null)
    const [selectedAccountId, setSelectedAccountId] = useState('')

    // Fetch Accounts on Mount
    useEffect(() => {
        fetchAccounts()
    }, [])

    // Fetch Entries when Account Changes
    useEffect(() => {
        if (selectedAccountId) {
            fetchEntries(selectedAccountId)
        } else {
            setEntries([])
        }
    }, [selectedAccountId])

    const fetchAccounts = async () => {
        setLoading(true)
        try {
            const response = await endpoints.getLedgerAccounts()
            const accs = response.data.data || []
            setAccounts(accs)

            // Auto-select first account if available
            if (accs.length > 0) {
                setSelectedAccountId(accs[0].id)
            }
        } catch (err) {
            console.error("Failed to fetch ledger accounts:", err)
            setError("Failed to load accounts.")
        } finally {
            setLoading(false)
        }
    }

    const fetchEntries = async (accountId) => {
        setEntriesLoading(true)
        try {
            const response = await endpoints.getLedgerEntries({
                account_id: accountId,
                limit: 50
            })
            setEntries(response.data.data || [])
        } catch (err) {
            console.error("Failed to fetch ledger entries:", err)
            // Don't show critical error for entries, just log
        } finally {
            setEntriesLoading(false)
        }
    }

    const formatCurrency = (amount) => {
        return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount / 100)
    }

    return (
        <div className="space-y-6">
            <div className="flex flex-col gap-2">
                <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">Financial Ledger</h1>
                <p className="text-slate-500 dark:text-slate-400">View double-entry ledger transactions and account balances.</p>
            </div>

            {/* Account Selector */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Select Account</label>
                    <select
                        className="w-full rounded-lg border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                        value={selectedAccountId}
                        onChange={(e) => setSelectedAccountId(e.target.value)}
                        disabled={loading || accounts.length === 0}
                    >
                        {loading ? <option>Loading accounts...</option> : null}
                        {!loading && accounts.length === 0 ? <option>No accounts found</option> : null}
                        {accounts.map(acc => (
                            <option key={acc.id} value={acc.id}>{acc.name} ({acc.code})</option>
                        ))}
                    </select>
                </div>
                {/* Selected account's current balance from the accounts API */}
                <div className="md:col-span-2 flex gap-4">
                    {selectedAccountId && accounts.find(a => a.id === selectedAccountId) && (
                        <div className="p-4 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg shadow-sm">
                            <p className="text-sm text-slate-500">Current Balance</p>
                            <p className="text-2xl font-bold text-slate-900 dark:text-white">
                                {formatCurrency(accounts.find(a => a.id === selectedAccountId).balance || 0)}
                            </p>
                        </div>
                    )}
                </div>
            </div>

            {/* Entries Table */}
            <div className="rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900 overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
                        <thead className="bg-slate-50 dark:bg-slate-950/50">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Transaction ID</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Debit</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Credit</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Amount</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Code</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                            {entriesLoading ? (
                                <tr><td colSpan="5" className="p-6 text-center text-slate-500">Loading entries...</td></tr>
                            ) : entries.length === 0 ? (
                                <tr><td colSpan="5" className="p-6 text-center text-slate-500">No entries found for this account.</td></tr>
                            ) : (
                                entries.map((entry) => (
                                    <tr key={entry.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white font-mono text-xs">
                                            {entry.id}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-300 font-mono text-xs">
                                            {entry.debit_account_id}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-300 font-mono text-xs">
                                            {entry.credit_account_id}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white">
                                            {formatCurrency(entry.amount)}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-300">
                                            <span className="inline-flex items-center rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-800 dark:bg-slate-800 dark:text-slate-300">
                                                Code {entry.code}
                                            </span>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    )
}

export default Ledger
