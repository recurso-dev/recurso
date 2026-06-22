import React, { useState, useEffect, useMemo } from 'react'
import { endpoints } from '../lib/api'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'

const Dashboard = () => {
    // UI State
    const [dateRange, setDateRange] = useState('all') // 'all', '30_days', 'this_month', 'last_month'

    // Data State
    const [stats, setStats] = useState({
        netBilling: 0,
        netPayments: 0,
        unpaidInvoices: 0,
        activeSubs: 0,
        mrr: 0
    })
    const [chartData, setChartData] = useState([])
    const [recentActivity, setRecentActivity] = useState([])
    const [loading, setLoading] = useState(true)

    // Raw Data Storage
    const [rawInvoices, setRawInvoices] = useState([])
    const [rawSubscriptions, setRawSubscriptions] = useState([])
    const [customers, setCustomers] = useState({})
    const [plans, setPlans] = useState({})

    useEffect(() => {
        const fetchData = async () => {
            try {
                // Parallel fetch
                const [subsRes, invRes, custRes, planRes, mrrRes] = await Promise.all([
                    endpoints.getSubscriptions({ limit: 1000 }).catch(() => ({ data: { data: [] } })),
                    endpoints.getInvoices({ limit: 1000 }).catch(() => ({ data: { data: [] } })),
                    endpoints.getCustomers({ limit: 1000 }).catch(() => ({ data: { data: [] } })),
                    endpoints.getPlans({ limit: 1000 }).catch(() => ({ data: { data: [] } })),
                    endpoints.getMRR().catch(() => ({ data: { mrr: 0 } }))
                ])

                // Process Metadata Maps
                const customerMap = {}
                const customerList = custRes.data.data || []
                customerList.forEach(c => { customerMap[c.id] = c.name })
                setCustomers(customerMap)

                const planMap = {}
                const planList = planRes.data.data || []
                planList.forEach(p => { planMap[p.id] = p })
                setPlans(planMap)

                // Store Raw Data
                setRawSubscriptions(subsRes.data.data || [])
                setRawInvoices(invRes.data.data || [])

                // Store MRR
                const mrrValue = mrrRes.data?.mrr || mrrRes.data?.data?.mrr || 0
                setStats(prev => ({ ...prev, mrr: mrrValue }))

            } catch (error) {
                console.error("Dashboard fetch error:", error)
            } finally {
                setLoading(false)
            }
        }
        fetchData()
    }, [])

    // Filter Logic
    useEffect(() => {
        if (loading) return

        const now = new Date()
        const getStartDate = () => {
            switch (dateRange) {
                case '30_days':
                    const d30 = new Date()
                    d30.setDate(now.getDate() - 30)
                    return d30
                case 'this_month':
                    return new Date(now.getFullYear(), now.getMonth(), 1)
                case 'last_month':
                    return new Date(now.getFullYear(), now.getMonth() - 1, 1)
                default:
                    return new Date(0) // Beginning of time
            }
        }

        const getEndDate = () => {
            if (dateRange === 'last_month') {
                return new Date(now.getFullYear(), now.getMonth(), 0) // Last day of previous month
            }
            return now
        }

        const startDate = getStartDate()
        const endDate = getEndDate()

        // 1. Filter Invoices
        const filteredInvoices = rawInvoices.filter(inv => {
            const d = new Date(inv.created_at)
            return d >= startDate && d <= endDate
        })

        // 2. Filter Subscriptions (for Activity Feed only usually, but let's filter 'New Subs' count)
        // Note: activeSubs is usually "Currently Active" regardless of when created, 
        // but "New Subscriptions" activity should be time-range bound.
        // For the stats card, let's keep "Active Subscriptions" as a live total snapshot (standard practice),
        // OR we can show "New Subscriptions" in this period. 
        // Let's stick to "Active Subscriptions" = Total Live for now as it's more useful, 
        // but maybe we filter the Activity Feed.

        const filteredNewSubs = rawSubscriptions.filter(s => {
            const d = new Date(s.created_at)
            return d >= startDate && d <= endDate
        })

        // --- Calculate Stats ---
        let netBilling = 0
        let netPayments = 0
        let unpaid = 0

        filteredInvoices.forEach(inv => {
            netBilling += inv.total
            if (inv.status === 'paid') {
                netPayments += inv.total
            } else if (inv.status === 'open' || inv.status === 'past_due') {
                unpaid += inv.total
            }
        })

        // Active Count is global, not time-bound usually, but let's recalculate filtering
        const activeCount = rawSubscriptions.filter(s => s.status === 'active').length

        setStats(prev => ({
            ...prev,
            netBilling,
            netPayments,
            unpaidInvoices: unpaid,
            activeSubs: activeCount
        }))

        // --- Prepare Chart Data ---
        const revenueByDate = {}
        filteredInvoices.forEach(inv => {
            const dateStr = new Date(inv.created_at).toLocaleDateString('en-US') // Standardize key
            if (!revenueByDate[dateStr]) revenueByDate[dateStr] = 0
            revenueByDate[dateStr] += inv.total
        })

        // Zero-fill gaps
        const filledChartData = []
        let currentDate = new Date(startDate)
        // Adjust start date if it's "beginning of time" (Date(0)) to something reasonable for chart?
        // If "All Time", we usually start from first invoice.
        // Let's find min date if range is 'all'

        let loopStartDate = currentDate
        if (dateRange === 'all') {
            if (filteredInvoices.length > 0) {
                // Sort invoices to find first date
                const sortedInvs = [...filteredInvoices].sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
                loopStartDate = new Date(sortedInvs[0].created_at)
            } else {
                loopStartDate = new Date() // No data, showing today
            }
        }

        // Normalize to start of day
        loopStartDate.setHours(0, 0, 0, 0)
        const endLoop = new Date(endDate)
        endLoop.setHours(0, 0, 0, 0)

        // Safety cap: Don't loop more than 365 days for 'all' if old data exists, or just accept it? 
        // For MVP, simple loop is fine. 

        const itr = new Date(loopStartDate)
        while (itr <= endLoop) {
            const dStr = itr.toLocaleDateString('en-US')
            filledChartData.push({
                date: dStr,
                revenue: (revenueByDate[dStr] || 0) / 100
            })
            itr.setDate(itr.getDate() + 1)
        }

        setChartData(filledChartData)

        // --- Prepare Activity Feed ---
        const activityFeed = []

        filteredNewSubs.forEach(s => {
            activityFeed.push({
                id: `sub_${s.id} `,
                type: 'New Subscription',
                customer_id: s.customer_id,
                amount: plans[s.plan_id]?.prices?.[0]?.amount || 0,
                status: s.status === 'active' ? 'Active' : s.status === 'trialing' ? 'Trial' : 'Pending',
                date: new Date(s.created_at),
                currency: 'USD'
            })
        })

        filteredInvoices.forEach(inv => {
            // For Activity Feed, we want to see events that happened IN this range.
            // If range is 'This Month', we want invoices created this month.
            // (We effectively already filtered 'filteredInvoices' by created_at above)

            if (inv.status === 'paid') {
                activityFeed.push({
                    id: `inv_paid_${inv.id} `,
                    type: 'Invoice Paid',
                    customer_id: inv.customer_id,
                    amount: inv.total,
                    status: 'Paid',
                    date: new Date(inv.updated_at || inv.created_at),
                    currency: inv.currency
                })
            } else if (inv.status === 'open') {
                activityFeed.push({
                    id: `inv_sent_${inv.id} `,
                    type: 'Invoice Sent',
                    customer_id: inv.customer_id,
                    amount: inv.total,
                    status: 'Open',
                    date: new Date(inv.created_at),
                    currency: inv.currency
                })
            }
        })

        setRecentActivity(activityFeed.sort((a, b) => b.date - a.date).slice(0, 10))

    }, [dateRange, rawInvoices, rawSubscriptions, loading, customers, plans])


    const formatCurrency = (amount) => {
        return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount / 100)
    }

    return (
        <div className="mx-auto max-w-[1600px] p-6">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 mb-8">
                <div>
                    <h2 className="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">
                        Overview
                    </h2>
                </div>
                <div className="flex items-center gap-3">
                    <select
                        value={dateRange}
                        onChange={(e) => setDateRange(e.target.value)}
                        className="h-9 block w-40 rounded-md border border-gray-200 bg-white dark:bg-zinc-900 dark:border-zinc-800 dark:text-white px-3 text-sm focus:border-black focus:ring-black dark:focus:border-white dark:focus:ring-white transition-all outline-none"
                    >
                        <option value="all">All Time</option>
                        <option value="30_days">Last 30 Days</option>
                        <option value="this_month">This Month</option>
                        <option value="last_month">Last Month</option>
                    </select>
                </div>
            </div>

            {/* Stats Grid - Bento Style */}
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-5 mb-8">
                {/* Net Billing */}
                <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900/50">
                    <div className="flex justify-between items-start">
                        <p className="text-sm font-medium text-gray-500 dark:text-zinc-500">Net Billing</p>
                        <span className="material-symbols-outlined text-gray-400 text-[20px]">trending_up</span>
                    </div>
                    <p className="mt-4 text-3xl font-semibold tracking-tighter text-gray-900 dark:text-white">
                        {loading ? "..." : formatCurrency(stats.netBilling)}
                    </p>
                </div>

                {/* Net Payments */}
                <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900/50">
                    <div className="flex justify-between items-start">
                        <p className="text-sm font-medium text-gray-500 dark:text-zinc-500">Net Payments</p>
                        <span className="material-symbols-outlined text-gray-400 text-[20px]">payments</span>
                    </div>
                    <p className="mt-4 text-3xl font-semibold tracking-tighter text-gray-900 dark:text-white">
                        {loading ? "..." : formatCurrency(stats.netPayments)}
                    </p>
                </div>

                {/* Unpaid Invoices */}
                <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900/50">
                    <div className="flex justify-between items-start">
                        <p className="text-sm font-medium text-gray-500 dark:text-zinc-500">Unpaid Invoices</p>
                        <span className="material-symbols-outlined text-gray-400 text-[20px]">pending</span>
                    </div>
                    <p className="mt-4 text-3xl font-semibold tracking-tighter text-gray-900 dark:text-white">
                        {loading ? "..." : formatCurrency(stats.unpaidInvoices)}
                    </p>
                </div>

                {/* Active Subscriptions */}
                <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900/50">
                    <div className="flex justify-between items-start">
                        <p className="text-sm font-medium text-gray-500 dark:text-zinc-500">Active Subs</p>
                        <span className="material-symbols-outlined text-gray-400 text-[20px]">check_circle</span>
                    </div>
                    <p className="mt-4 text-3xl font-semibold tracking-tighter text-gray-900 dark:text-white">
                        {loading ? "..." : stats.activeSubs}
                    </p>
                </div>

                {/* MRR */}
                <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900/50">
                    <div className="flex justify-between items-start">
                        <p className="text-sm font-medium text-gray-500 dark:text-zinc-500">MRR</p>
                        <span className="material-symbols-outlined text-gray-400 text-[20px]">monitoring</span>
                    </div>
                    <p className="mt-4 text-3xl font-semibold tracking-tighter text-gray-900 dark:text-white">
                        {loading ? "..." : formatCurrency(stats.mrr)}
                    </p>
                </div>
            </div>

            {/* Main Content Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
                {/* Revenue Trend Chart */}
                <div className="lg:col-span-2 rounded-xl border border-gray-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900/50">
                    <div className="flex items-center justify-between mb-6">
                        <h3 className="text-base font-semibold text-gray-900 dark:text-white">Revenue Trend</h3>
                    </div>

                    <div className="h-72 w-full">
                        {chartData.length > 0 ? (
                            <ResponsiveContainer width="100%" height="100%">
                                <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                                    <defs>
                                        <linearGradient id="colorRevenue" x1="0" y1="0" x2="0" y2="1">
                                            <stop offset="5%" stopColor="#000" stopOpacity={0.1} />
                                            <stop offset="95%" stopColor="#000" stopOpacity={0} />
                                        </linearGradient>
                                    </defs>
                                    <XAxis
                                        dataKey="date"
                                        stroke="#a1a1aa"
                                        fontSize={11}
                                        tickLine={false}
                                        axisLine={false}
                                        dy={10}
                                        minTickGap={30}
                                    />
                                    <YAxis
                                        stroke="#a1a1aa"
                                        fontSize={11}
                                        tickLine={false}
                                        axisLine={false}
                                        tickFormatter={(value) => `$${value}`}
                                    />
                                    <Tooltip
                                        contentStyle={{ backgroundColor: '#09090b', border: '1px solid #27272a', borderRadius: '6px', color: '#fff', fontSize: '12px' }}
                                        itemStyle={{ color: '#fff' }}
                                        formatter={(value) => [`$${value}`, 'Revenue']}
                                        cursor={{ stroke: '#e4e4e7' }}
                                    />
                                    <CartesianGrid strokeDasharray="3 3" stroke="#f4f4f5" vertical={false} />
                                    <Area type="monotone" dataKey="revenue" stroke="#000" strokeWidth={2} fillOpacity={1} fill="url(#colorRevenue)" />
                                </AreaChart>
                            </ResponsiveContainer>
                        ) : (
                            <div className="flex h-full flex-col items-center justify-center text-gray-400">
                                <span className="material-symbols-outlined text-3xl mb-2 opacity-20">bar_chart</span>
                                <span className="text-sm">No data available</span>
                            </div>
                        )}
                    </div>
                </div>

                {/* Recent Activity */}
                <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900/50 flex flex-col">
                    <h3 className="text-base font-semibold text-gray-900 dark:text-white mb-6">Activity</h3>
                    <div className="flex-1 overflow-y-auto max-h-[18rem] -mr-2 pr-2 custom-scrollbar">
                        {loading ? (
                            <p className="text-sm text-gray-500">Loading...</p>
                        ) : recentActivity.length === 0 ? (
                            <div className="flex h-full flex-col items-center justify-center text-gray-400">
                                <span className="text-sm">No recent activity</span>
                            </div>
                        ) : (
                            <ul className="space-y-4">
                                {recentActivity.map((item) => (
                                    <li key={item.id} className="flex gap-3 items-start">
                                        <div className="mt-1 h-1.5 w-1.5 rounded-full bg-gray-300 flex-none" />
                                        <div className="flex-1 space-y-0.5">
                                            <div className="flex items-center justify-between">
                                                <p className="text-sm font-medium text-gray-900 dark:text-white">
                                                    {item.type}
                                                </p>
                                                <p className="text-[10px] text-gray-400 uppercase tracking-wide">{item.date.toLocaleDateString()}</p>
                                            </div>
                                            <p className="text-xs text-gray-500 dark:text-zinc-400">
                                                {formatCurrency(item.amount)} · <span className="text-gray-400">{customers[item.customer_id] || 'Customer'}</span>
                                            </p>
                                        </div>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Dashboard
