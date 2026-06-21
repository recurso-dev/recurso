import React, { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { endpoints } from '../lib/api'
import { useToast } from '../components/Toast'
import { TableSkeleton, EmptyState, ErrorState } from '../components/LoadingStates'
import CustomerDetail from '../components/slide-overs/CustomerDetail'
import { useDebounce } from '../hooks/useDebounce'

const Customers = () => {
    const [customers, setCustomers] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)
    const [filter, setFilter] = useState('all')
    const [search, setSearch] = useState('')
    const [page, setPage] = useState(1)
    const limit = 10
    const debouncedSearch = useDebounce(search, 500)

    const [selectedCustomer, setSelectedCustomer] = useState(null)
    const [isDetailOpen, setIsDetailOpen] = useState(false)
    const toast = useToast()

    const fetchCustomers = useCallback(async () => {
        setLoading(true)
        setError(null)
        try {
            const params = { page, limit }
            if (debouncedSearch) params.q = debouncedSearch
            if (filter !== 'all') params.status = filter

            const response = await endpoints.getCustomers(params)
            setCustomers(response.data.data || [])
        } catch (err) {
            const msg = err?.response?.data?.error?.message || err?.message || 'Failed to load customers'
            setError(msg)
            toast.error(msg)
        } finally {
            setLoading(false)
        }
    }, [page, limit, filter, debouncedSearch, toast])

    useEffect(() => {
        fetchCustomers()
    }, [fetchCustomers])

    // Reset to page 1 on search or filter change
    useEffect(() => {
        setPage(1)
    }, [debouncedSearch, filter])

    const handleRowClick = (customer) => {
        setSelectedCustomer(customer)
        setIsDetailOpen(true)
    }

    const closeDetail = () => {
        setIsDetailOpen(false)
        setTimeout(() => setSelectedCustomer(null), 300)
    }

    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            <header className="flex flex-wrap justify-between items-center gap-4 mb-8">
                <div>
                    <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">Customers</h1>
                    <p className="text-slate-500 dark:text-slate-400 mt-1">Manage your customer base and subscriptions.</p>
                </div>
                <div className="flex gap-3">
                    <button className="flex items-center justify-center rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 transition-colors">
                        <span className="material-symbols-outlined mr-2 text-base">file_download</span>
                        Export
                    </button>
                    <Link
                        to="/customers/new"
                        className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold leading-normal tracking-[0.015em] shadow-sm hover:opacity-90 transition-opacity"
                    >
                        <span className="material-symbols-outlined mr-2 text-base">add</span>
                        <span className="truncate">Add Customer</span>
                    </Link>
                </div>
            </header>

            {/* Filter & Search */}
            <div className="flex flex-col md:flex-row gap-4 mb-6">
                <div className="flex-1">
                    <div className="relative">
                        <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">search</span>
                        <input
                            type="text"
                            placeholder="Search by name or email..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="w-full rounded-lg border border-slate-200 bg-white pl-10 pr-4 py-2 text-sm text-slate-900 placeholder-slate-500 focus:border-primary focus:ring-1 focus:ring-primary dark:border-slate-800 dark:bg-slate-900 dark:text-white dark:placeholder-slate-400"
                        />
                    </div>
                </div>
                <div className="flex gap-2">
                    {['all', 'active', 'inactive'].map((f) => (
                        <button
                            key={f}
                            onClick={() => setFilter(f)}
                            className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${filter === f
                                ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900'
                                : 'bg-white text-slate-600 border border-slate-200 hover:bg-slate-50 dark:bg-slate-900 dark:text-slate-300 dark:border-slate-800 dark:hover:bg-slate-800'
                                }`}
                        >
                            {f.charAt(0).toUpperCase() + f.slice(1)}
                        </button>
                    ))}
                </div>
            </div>

            {/* Data Grid */}
            <div className="w-full">
                <div className="flex overflow-hidden rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-sm">
                    <table className="flex-1 w-full">
                        <thead className="bg-slate-50 dark:bg-slate-950/50">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Customer</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Status</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Risk</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Subscriptions</th>
                                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Joined</th>
                                <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Action</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                            {loading ? (
                                <tr><td colSpan="6" className="p-6">
                                    <TableSkeleton rows={5} />
                                </td></tr>
                            ) : error ? (
                                <tr><td colSpan="6" className="p-6">
                                    <ErrorState message={error} onRetry={fetchCustomers} />
                                </td></tr>
                            ) : customers.length === 0 ? (
                                <tr><td colSpan="6" className="p-8 text-center text-slate-500">
                                    {search || filter !== 'all'
                                        ? 'No customers match your filters.'
                                        : <EmptyState title="No customers yet" description="Add your first customer to get started." />
                                    }
                                </td></tr>
                            ) : (
                                customers.map((c) => (
                                    <tr
                                        key={c.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors"
                                        onClick={() => handleRowClick(c)}
                                    >
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm font-medium text-slate-900 dark:text-white">{c.name}</div>
                                            <div className="text-sm text-slate-500 dark:text-slate-400">{c.email}</div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            {c.activeSubs > 0 ? (
                                                <span className="inline-flex items-center gap-x-1.5 rounded-full bg-green-100 dark:bg-green-900/50 px-2.5 py-1 text-xs font-medium text-green-700 dark:text-green-400">
                                                    <span className="size-1.5 rounded-full bg-green-500"></span>
                                                    Active
                                                </span>
                                            ) : (
                                                <span className="inline-flex items-center gap-x-1.5 rounded-full bg-slate-100 dark:bg-slate-800 px-2.5 py-1 text-xs font-medium text-slate-600 dark:text-slate-400">
                                                    <span className="size-1.5 rounded-full bg-slate-500"></span>
                                                    Inactive
                                                </span>
                                            )}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <RiskBadge score={c.risk_score} />
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                            {c.activeSubs}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                            {new Date(c.created_at).toLocaleDateString()}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium flex justify-end gap-2">
                                            <button
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    const url = `${window.location.origin}/portal/${c.tenant_id}/${c.id}`;
                                                    navigator.clipboard.writeText(url);
                                                    toast.success('Portal link copied!');
                                                }}
                                                className="text-slate-400 hover:text-primary dark:hover:text-primary transition-colors tooltip"
                                                title="Copy Portal Link"
                                            >
                                                <span className="material-symbols-outlined text-[20px]">link</span>
                                            </button>
                                            <button className="text-slate-400 hover:text-primary dark:hover:text-primary transition-colors">
                                                <span className="material-symbols-outlined text-[20px]">more_horiz</span>
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between border-t border-slate-200 bg-white px-4 py-3 sm:px-6 dark:border-slate-800 dark:bg-slate-900 mt-4 rounded-xl shadow-sm">
                <div className="flex flex-1 justify-between sm:hidden">
                    <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="relative inline-flex items-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
                    >
                        Previous
                    </button>
                    <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={customers.length < limit}
                        className="relative ml-3 inline-flex items-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
                    >
                        Next
                    </button>
                </div>
                <div className="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
                    <div>
                        <p className="text-sm text-slate-700 dark:text-slate-300">
                            Showing page <span className="font-medium">{page}</span>
                        </p>
                    </div>
                    <div>
                        <nav className="isolate inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
                            <button
                                onClick={() => setPage(p => Math.max(1, p - 1))}
                                disabled={page === 1}
                                className="relative inline-flex items-center rounded-l-md px-2 py-2 text-slate-400 ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus:z-20 focus:outline-offset-0 disabled:opacity-50 dark:ring-slate-700 dark:hover:bg-slate-800 transition-colors"
                            >
                                <span className="sr-only">Previous</span>
                                <span className="material-symbols-outlined text-sm">chevron_left</span>
                            </button>
                            <button
                                onClick={() => setPage(p => p + 1)}
                                disabled={customers.length < limit}
                                className="relative inline-flex items-center rounded-r-md px-2 py-2 text-slate-400 ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus:z-20 focus:outline-offset-0 disabled:opacity-50 dark:ring-slate-700 dark:hover:bg-slate-800 transition-colors"
                            >
                                <span className="sr-only">Next</span>
                                <span className="material-symbols-outlined text-sm">chevron_right</span>
                            </button>
                        </nav>
                    </div>
                </div>
            </div>

            <CustomerDetail
                customer={selectedCustomer}
                isOpen={isDetailOpen}
                onClose={closeDetail}
            />
        </div>
    )
}

const RiskBadge = ({ score }) => {
    if (!score) return <span className="text-slate-400 text-xs">-</span>

    let color = 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    let label = 'Low'

    if (score >= 50) {
        color = 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
        label = 'High'
    } else if (score >= 20) {
        color = 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
        label = 'Medium'
    }

    return (
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${color}`}>
            {score} • {label}
        </span>
    )
}

export default Customers
