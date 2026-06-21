import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { endpoints } from '../lib/api'

import CreditNoteDetail from '../components/slide-overs/CreditNoteDetail'

const CreditNotes = () => {
    const [creditNotes, setCreditNotes] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)
    const [search, setSearch] = useState('')
    const [selectedNote, setSelectedNote] = useState(null)
    const [isDetailOpen, setIsDetailOpen] = useState(false)

    useEffect(() => {
        fetchCreditNotes()
    }, [])

    const fetchCreditNotes = async () => {
        setLoading(true)
        try {
            const response = await endpoints.getCreditNotes()
            setCreditNotes(response.data.data || [])
        } catch (err) {
            console.error("Failed to fetch credit notes:", err)
            setError("Failed to load credit notes.")
        } finally {
            setLoading(false)
        }
    }

    const formatCurrency = (amount, currency) => {
        return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency || 'USD' }).format(amount / 100)
    }

    const filteredNotes = creditNotes.filter(cn =>
        cn.id.toLowerCase().includes(search.toLowerCase()) ||
        (cn.customer?.name || '').toLowerCase().includes(search.toLowerCase())
    )

    const handleRowClick = (note) => {
        setSelectedNote(note)
        setIsDetailOpen(true)
    }

    const closeDetail = () => {
        setIsDetailOpen(false)
        setTimeout(() => setSelectedNote(null), 300)
    }

    return (
        <div className="space-y-6">
            <header className="flex flex-wrap justify-between items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">Credit Notes</h1>
                    <p className="text-slate-500 dark:text-slate-400">Manage customer credits and refunds.</p>
                </div>
                <Link
                    to="/credit-notes/new"
                    className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold leading-normal tracking-[0.015em] shadow-sm hover:opacity-90 transition-opacity"
                >
                    <span className="material-symbols-outlined mr-2 text-base">add</span>
                    <span className="truncate">Create Credit Note</span>
                </Link>
            </header>

            {/* Filter & Search */}
            <div className="flex flex-col md:flex-row gap-3">
                <div className="flex-1">
                    <label className="relative block h-10 w-full">
                        <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">search</span>
                        <input
                            className="h-full w-full rounded-lg border border-slate-200 bg-white pl-10 pr-4 text-sm text-slate-900 placeholder-slate-500 focus:border-primary focus:ring-1 focus:ring-primary dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder-slate-400"
                            placeholder="Search by ID or customer..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </label>
                </div>
            </div>

            {/* Table */}
            <div className="rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900 overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
                        <thead className="bg-slate-50 dark:bg-slate-950/50">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">ID</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Customer</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Amount</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Balance</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Status</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Created</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                            {loading ? (
                                <tr><td colSpan="6" className="p-6 text-center text-slate-500">Loading credits...</td></tr>
                            ) : filteredNotes.length === 0 ? (
                                <tr><td colSpan="6" className="p-6 text-center text-slate-500">No credit notes found.</td></tr>
                            ) : (
                                filteredNotes.map((cn) => (
                                    <tr
                                        key={cn.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer"
                                        onClick={() => handleRowClick(cn)}
                                    >
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white font-mono text-xs">
                                            {cn.reference || cn.id.slice(0, 8)}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-300">
                                            {cn.customer ? cn.customer.name : 'Unknown Customer'}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white">
                                            {formatCurrency(cn.amount, cn.currency)}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                            {formatCurrency(cn.balance, cn.currency)}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                                            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium 
                                                ${cn.status === 'issued' ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-300' :
                                                    cn.status === 'used' ? 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300' :
                                                        'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300'}`}>
                                                {cn.status.charAt(0).toUpperCase() + cn.status.slice(1)}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                            {new Date(cn.created_at).toLocaleDateString()}
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            <CreditNoteDetail
                creditNote={selectedNote}
                isOpen={isDetailOpen}
                onClose={closeDetail}
            />
        </div>
    )
}

export default CreditNotes
