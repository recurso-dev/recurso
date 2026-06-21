import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { endpoints } from '../lib/api'
import SubscriptionDetail from '../components/slide-overs/SubscriptionDetail'

const Subscriptions = () => {
    const [subs, setSubs] = useState([])
    const [loading, setLoading] = useState(true)
    const [customers, setCustomers] = useState({})
    const [plans, setPlans] = useState({})

    // Filter/Pagination State
    const [search, setSearch] = useState('')
    const [debouncedSearch, setDebouncedSearch] = useState('')
    const [page, setPage] = useState(1)
    const [limit, setLimit] = useState(10)
    const [statusFilter, setStatusFilter] = useState('')

    // Detail view state
    const [selectedSub, setSelectedSub] = useState(null)
    const [isDetailOpen, setIsDetailOpen] = useState(false)

    // Debounce search
    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(search)
            setPage(1)
        }, 500)
        return () => clearTimeout(timer)
    }, [search])

    const fetchData = async () => {
        setLoading(true)
        try {
            const params = {
                q: debouncedSearch,
                page: page,
                limit: limit,
                status: statusFilter === 'all' ? '' : statusFilter
            }

            const [subRes, custRes, planRes] = await Promise.all([
                endpoints.getSubscriptions(params),
                // Fetch larger set of customers/plans for mapping. 
                // In a real expanded app, we'd fetch specific IDs or rely on backend expansion.
                endpoints.getCustomers({ limit: 1000 }).catch(() => ({ data: { data: [] } })),
                endpoints.getPlans({ limit: 100 }).catch(() => ({ data: { data: [] } }))
            ])

            // Process Customers
            const customerMap = {}
            const customerList = custRes.data.data || []
            customerList.forEach(c => {
                customerMap[c.id] = c
            })
            setCustomers(customerMap)

            // Process Plans
            const planMap = {}
            const planList = planRes.data.data || []
            planList.forEach(p => {
                planMap[p.id] = p
            })
            setPlans(planMap)

            setSubs(subRes.data.data || [])
        } catch (error) {
            console.error("Failed to fetch data:", error)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchData()
    }, [debouncedSearch, page, limit, statusFilter])

    const handleRowClick = (sub) => {
        setSelectedSub(sub)
        setIsDetailOpen(true)
    }

    const closeDetail = () => {
        setIsDetailOpen(false)
        setTimeout(() => setSelectedSub(null), 300)
    }

    // Additional Filters
    const [planFilter, setPlanFilter] = useState('all')
    const [dateFilter, setDateFilter] = useState('all') // 'all', '30_days', 'this_month'

    const [isPlanOpen, setIsPlanOpen] = useState(false)
    const [isDateOpen, setIsDateOpen] = useState(false)

    // Filter Logic (Client-side filtering of the fetched page)
    const filteredSubs = subs.filter(s => {
        let match = true
        // 1. Plan
        if (planFilter !== 'all') {
            if (s.plan_id !== planFilter) match = false
        }
        // 2. Date (Created At or Start Date?)
        if (dateFilter !== 'all') {
            // Assuming created_at or current_period_start. Let's use current_period_start as it's visible.
            const start = new Date(s.current_period_start)
            const now = new Date()
            if (dateFilter === '30_days') {
                const thirtyDaysAgo = new Date(now.setDate(now.getDate() - 30))
                if (start < thirtyDaysAgo) match = false
            }
            if (dateFilter === 'this_month') {
                const firstOfMonth = new Date(now.getFullYear(), now.getMonth(), 1)
                if (start < firstOfMonth) match = false
            }
        }
        return match
    })

    // Click outside handler
    useEffect(() => {
        const close = () => {
            setIsPlanOpen(false)
            setIsDateOpen(false)
        }
        if (isPlanOpen || isDateOpen) window.addEventListener('click', close)
        return () => window.removeEventListener('click', close)
    }, [isPlanOpen, isDateOpen])

    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            <div className="flex flex-wrap justify-between gap-3 pb-6 border-b border-slate-200 dark:border-slate-800">
                <h1 className="text-slate-900 dark:text-white text-3xl font-bold leading-tight tracking-tight">Subscriptions</h1>
                <Link
                    to="/subscriptions/new"
                    className="flex min-w-[84px] cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold leading-normal tracking-[0.015em] hover:bg-primary/90 transition-all"
                >
                    <span className="material-symbols-outlined text-xl">add</span>
                    <span className="truncate">Add Subscription</span>
                </Link>
            </div>

            {/* Filters */}
            <div className="flex flex-wrap items-center gap-4 py-6">
                <div className="flex-1 min-w-[300px]">
                    <div className="relative flex-grow">
                        <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 dark:text-slate-500 text-xl">search</span>
                        <input
                            className="block w-full rounded-lg border-slate-300 bg-white dark:bg-slate-800 dark:border-slate-700 dark:text-white dark:placeholder-slate-400 pl-10 h-10 text-sm focus:border-primary focus:ring-primary/20"
                            placeholder="Search by customer name, email, or ID..."
                            type="text"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                </div>
                <div className="flex gap-2 overflow-x-auto">
                    {/* Status Select (Param-based) */}
                    <div className="relative">
                        <select
                            className="h-10 appearance-none rounded-lg bg-white dark:bg-slate-800 dark:border-slate-700 border border-slate-300 pl-4 pr-10 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors text-sm font-medium text-slate-600 dark:text-slate-400 focus:outline-none focus:ring-2 focus:ring-primary/20"
                            value={statusFilter}
                            onChange={(e) => {
                                setStatusFilter(e.target.value)
                                setPage(1)
                            }}
                        >
                            <option value="">Status: All</option>
                            <option value="active">Active</option>
                            <option value="paused">Paused</option>
                            <option value="trialing">Trialing</option>
                            <option value="past_due">Past Due</option>
                            <option value="canceled">Canceled</option>
                        </select>
                        <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 material-symbols-outlined text-slate-500 text-xl">expand_more</span>
                    </div>

                    {/* Plan Dropdown */}
                    <div className="relative">
                        <button
                            onClick={(e) => { e.stopPropagation(); setIsPlanOpen(!isPlanOpen); setIsDateOpen(false) }}
                            className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-white dark:bg-slate-800 dark:border-slate-700 border border-slate-300 pl-4 pr-3 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                        >
                            <p className="text-sm font-medium text-slate-600 dark:text-slate-400">
                                Plan: {planFilter === 'all' ? 'All' : (plans[planFilter]?.name || 'Selected')}
                            </p>
                            <span className="material-symbols-outlined text-slate-500 text-xl">expand_more</span>
                        </button>
                        {isPlanOpen && (
                            <div className="absolute right-0 top-12 w-56 z-10 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 py-1 max-h-60 overflow-y-auto">
                                <button onClick={() => setPlanFilter('all')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">All Plans</button>
                                {Object.values(plans).map(plan => (
                                    <button key={plan.id} onClick={() => setPlanFilter(plan.id)} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700 truncate">
                                        {plan.name}
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Date Dropdown */}
                    <div className="relative">
                        <button
                            onClick={(e) => { e.stopPropagation(); setIsDateOpen(!isDateOpen); setIsPlanOpen(false) }}
                            className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-white dark:bg-slate-800 dark:border-slate-700 border border-slate-300 pl-4 pr-3 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                        >
                            <p className="text-sm font-medium text-slate-600 dark:text-slate-400">
                                Date: {dateFilter === 'all' ? 'All Time' : dateFilter === '30_days' ? 'Last 30 Days' : 'This Month'}
                            </p>
                            <span className="material-symbols-outlined text-slate-500 text-xl">expand_more</span>
                        </button>
                        {isDateOpen && (
                            <div className="absolute right-0 top-12 w-48 z-10 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 py-1">
                                <button onClick={() => setDateFilter('all')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">All Time</button>
                                <button onClick={() => setDateFilter('30_days')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">Last 30 Days</button>
                                <button onClick={() => setDateFilter('this_month')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">This Month</button>
                            </div>
                        )}
                    </div>
                </div>
            </div>

            <div className="w-full">
                <div className="flex overflow-hidden rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-sm">
                    <table className="flex-1 w-full">
                        <thead className="bg-slate-50 dark:bg-slate-800/50">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Customer</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Status</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Plan</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Amount</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Start Date</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Next Invoice</th>
                                <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider"></th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                            {loading ? (
                                <tr><td colSpan="7" className="p-8 text-center text-slate-500">Loading subscriptions...</td></tr>
                            ) : filteredSubs.length === 0 ? (
                                <tr><td colSpan="7" className="p-8 text-center text-slate-500">No subscriptions found matching filters.</td></tr>
                            ) : (
                                filteredSubs.map((s) => {
                                    const customer = customers[s.customer_id]
                                    const plan = plans[s.plan_id]
                                    const price = plan?.prices?.[0]
                                    const amount = price ? price.amount : 0
                                    const currency = price ? price.currency : 'USD'
                                    const interval = plan ? plan.interval_unit : 'month'

                                    return (
                                        <tr
                                            key={s.id}
                                            className="hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors"
                                            onClick={() => handleRowClick(s)}
                                        >
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <div className="text-sm font-medium text-slate-900 dark:text-white">
                                                    {customer?.name || 'Unknown'}
                                                </div>
                                                <div className="text-xs text-slate-500 dark:text-slate-400">
                                                    {customer?.email || 'No email'}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                {s.status === 'paused' ? (
                                                    <span className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300">
                                                        <span className="size-1.5 rounded-full bg-amber-500"></span>
                                                        Paused
                                                    </span>
                                                ) : (
                                                    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${s.status === 'active'
                                                        ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                                                        : 'bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-300'
                                                        }`}>
                                                        <span className={`size-1.5 rounded-full ${s.status === 'active' ? 'bg-green-500' : 'bg-slate-500'}`}></span>
                                                        {s.status.charAt(0).toUpperCase() + s.status.slice(1)}
                                                    </span>
                                                )}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-300">
                                                {plan?.name || s.plan_id.slice(0, 8)}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-300">
                                                {new Intl.NumberFormat('en-US', { style: 'currency', currency: currency }).format(amount / 100)} / {interval === 'month' ? 'mo' : 'yr'}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                                {new Date(s.current_period_start).toLocaleDateString()}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                                {new Date(s.current_period_end).toLocaleDateString()}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                                <button className="text-slate-400 hover:text-primary dark:hover:text-primary transition-colors">
                                                    <span className="material-symbols-outlined">more_horiz</span>
                                                </button>
                                            </td>
                                        </tr>
                                    )
                                })
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Pagination Controls */}
            <div className="flex items-center justify-between pt-6 text-sm text-slate-600 dark:text-slate-400">
                <p>Page <span className="font-medium">{page}</span></p>
                <div className="flex gap-2">
                    <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="flex h-9 items-center justify-center rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 px-3 text-sm font-medium hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 transition-colors"
                    >
                        Previous
                    </button>
                    <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={subs.length < limit}
                        className="flex h-9 items-center justify-center rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 px-3 text-sm font-medium hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 transition-colors"
                    >
                        Next
                    </button>
                </div>
            </div>

            <SubscriptionDetail
                subscription={selectedSub}
                customer={selectedSub ? customers[selectedSub.customer_id] : null}
                plan={selectedSub ? plans[selectedSub.plan_id] : null}
                isOpen={isDetailOpen}
                onClose={closeDetail}
            />
        </div>
    )
}

export default Subscriptions
