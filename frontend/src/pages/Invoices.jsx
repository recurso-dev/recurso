import React, { useState, useCallback, useEffect } from 'react'
import { endpoints } from '../lib/api'
import { useToast } from '../components/Toast'
import { TableSkeleton, EmptyState, ErrorState } from '../components/LoadingStates'
import InvoiceDetail from '../components/slide-overs/InvoiceDetail'

const Invoices = () => {
    const [invoices, setInvoices] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)
    const [search, setSearch] = useState('')
    const [selectedInvoice, setSelectedInvoice] = useState(null)
    const [isDetailOpen, setIsDetailOpen] = useState(false)
    const toast = useToast()

    const fetchInvoices = useCallback(async () => {
        setLoading(true)
        setError(null)
        try {
            const response = await endpoints.getInvoices()
            setInvoices(response.data.data || [])
        } catch (err) {
            const msg = err?.response?.data?.error?.message || err?.message || 'Failed to load invoices'
            setError(msg)
            toast.error(msg)
        } finally {
            setLoading(false)
        }
    }, []) // eslint-disable-line react-hooks/exhaustive-deps

    useEffect(() => {
        fetchInvoices()
    }, [fetchInvoices])

    const filteredInvoices = invoices.filter(inv => {
        if (!search) return true
        const s = search.toLowerCase()
        return inv.invoice_number?.toLowerCase().includes(s) ||
            inv.customer_id?.toLowerCase().includes(s) ||
            inv.status?.toLowerCase().includes(s)
    })

    const handleRowClick = (invoice) => {
        setSelectedInvoice(invoice)
        setIsDetailOpen(true)
    }

    const closeDetail = () => {
        setIsDetailOpen(false)
        setTimeout(() => setSelectedInvoice(null), 300)
    }

    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            <div className="flex flex-wrap justify-between gap-3 pb-6 border-b border-slate-200 dark:border-slate-800">
                <div className="flex flex-col gap-1">
                    <h1 className="text-slate-900 dark:text-white text-3xl font-bold leading-tight tracking-tight">Invoices</h1>
                    <p className="text-base font-normal text-slate-500 dark:text-slate-400">View and manage customer invoices.</p>
                </div>
            </div>

            {/* Search */}
            <div className="flex gap-3 py-6 overflow-x-auto">
                <div className="relative flex-grow max-w-sm">
                    <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                        <span className="material-symbols-outlined text-slate-400 dark:text-slate-500">search</span>
                    </div>
                    <input
                        className="block w-full rounded-lg border-slate-300 bg-white dark:bg-slate-800 dark:border-slate-700 dark:text-white dark:placeholder-slate-400 pl-10 h-10 text-sm focus:border-primary focus:ring-primary/20"
                        placeholder="Search invoices..."
                        type="text"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                    />
                </div>
            </div>

            <div className="w-full">
                <div className="flex overflow-hidden rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-sm">
                    <table className="flex-1 w-full">
                        <thead className="bg-slate-50 dark:bg-slate-800/50">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Number</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Customer</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Amount</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Status</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">E-Invoice</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Date</th>
                                <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider"></th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                            {loading ? (
                                <tr><td colSpan="7" className="p-6">
                                    <TableSkeleton rows={5} />
                                </td></tr>
                            ) : error ? (
                                <tr><td colSpan="7" className="p-6">
                                    <ErrorState message={error} onRetry={fetchInvoices} />
                                </td></tr>
                            ) : filteredInvoices.length === 0 ? (
                                <tr><td colSpan="7" className="p-8 text-center text-slate-500">
                                    <EmptyState title="No invoices yet" description="Invoices will appear here once subscriptions are billed." />
                                </td></tr>
                            ) : (
                                filteredInvoices.map((inv) => (
                                    <tr
                                        key={inv.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors"
                                        onClick={() => handleRowClick(inv)}
                                    >
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white">
                                            {inv.invoice_number}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                            {inv.customer_id?.slice(0, 8)}...
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-300">
                                            {new Intl.NumberFormat('en-US', { style: 'currency', currency: inv.currency || 'USD' }).format((inv.total || 0) / 100)}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <StatusBadge status={inv.status} />
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <EInvoiceBadge status={inv.e_invoice_status} />
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                            {new Date(inv.created_at).toLocaleDateString()}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <button className="text-slate-400 hover:text-primary dark:hover:text-primary transition-colors">
                                                <span className="material-symbols-outlined">more_horiz</span>
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            <InvoiceDetail
                invoice={selectedInvoice}
                isOpen={isDetailOpen}
                onClose={closeDetail}
            />
        </div>
    )
}

const StatusBadge = ({ status }) => {
    const styles = {
        paid: 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300',
        open: 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300',
        overdue: 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300',
        void: 'bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-300',
        draft: 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400',
    }

    return (
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${styles[status] || styles.draft}`}>
            {status}
        </span>
    )
}

const EInvoiceBadge = ({ status }) => {
    if (!status || status === 'PENDING') return <span className="text-xs text-slate-400">-</span>

    const styles = {
        GENERATED: 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300',
        FAILED: 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300',
        CANCELLED: 'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300',
        NA: 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-500',
    }

    return (
        <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${styles[status] || styles.NA}`}>
            {status}
        </span>
    )
}

export default Invoices
