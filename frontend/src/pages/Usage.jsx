import React, { useState, useEffect } from 'react'
import { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid } from 'recharts'
import { endpoints as api } from '../lib/api'

const Usage = () => {
    const [usageStats, setUsageStats] = useState([])
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        const fetchUsage = async () => {
            try {
                const response = await api.getUsageStats()
                setUsageStats(response.data.data || [])
            } catch (error) {
                console.error("Failed to fetch usage stats:", error)
            } finally {
                setLoading(false)
            }
        }
        fetchUsage()
    }, [])
    // Filter State
    const [customerFilter, setCustomerFilter] = useState('all')
    const [planFilter, setPlanFilter] = useState('all')

    const [isCustomerOpen, setIsCustomerOpen] = useState(false)
    const [isPlanOpen, setIsPlanOpen] = useState(false)

    // Extract unique options from stats
    // Note: Usage stats might not have plan name, only IDs. 
    // For demo, we might just show "Active Plan".
    const uniqueCustomers = [...new Set(usageStats.map(d => d.customer_id))]
    const uniquePlans = [...new Set(usageStats.map(d => d.plan_id))]

    // Filter Logic
    const filteredData = usageStats.filter(item => {
        if (customerFilter !== 'all' && item.customer_id !== customerFilter) return false
        if (planFilter !== 'all' && item.plan_id !== planFilter) return false
        return true
    })

    const totalUnits = filteredData.reduce((acc, curr) => acc + curr.total_quantity, 0)

    // Aggregate per dimension for the chart (stats are lifetime aggregates,
    // so a per-dimension breakdown is the honest visualization).
    const byDimension = Object.values(
        filteredData.reduce((acc, curr) => {
            const key = curr.dimension || 'unknown'
            acc[key] = acc[key] || { dimension: key, units: 0 }
            acc[key].units += curr.total_quantity
            return acc
        }, {})
    )

    // Click outside handler
    useEffect(() => {
        const close = () => {
            setIsCustomerOpen(false)
            setIsPlanOpen(false)
        }
        if (isCustomerOpen || isPlanOpen) window.addEventListener('click', close)
        return () => window.removeEventListener('click', close)
    }, [isCustomerOpen, isPlanOpen])


    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            <div className="flex flex-wrap items-center justify-between gap-4 pb-6">
                <h1 className="text-3xl font-bold tracking-tight text-slate-900 dark:text-white">Usage Metering</h1>
                <button
                    onClick={() => {
                        const rows = [
                            ['customer_id', 'plan_id', 'dimension', 'total_quantity'],
                            ...filteredData.map(d => [d.customer_id, d.plan_id, d.dimension, d.total_quantity]),
                        ]
                        const csv = rows.map(r => r.join(',')).join('\n')
                        const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }))
                        const a = document.createElement('a')
                        a.href = url
                        a.download = 'usage-export.csv'
                        a.click()
                        URL.revokeObjectURL(url)
                    }}
                    className="flex h-10 cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg bg-primary px-4 text-sm font-bold text-white shadow-sm transition-all hover:bg-primary/90"
                >
                    <span className="truncate">Export Data</span>
                </button>
            </div>

            {/* Filters */}
            <div className="flex flex-wrap gap-2 py-6 border-b border-slate-200 dark:border-slate-800">

                {/* Customer Filter */}
                <div className="relative">
                    <button
                        onClick={(e) => { e.stopPropagation(); setIsCustomerOpen(!isCustomerOpen); setIsPlanOpen(false) }}
                        className="flex h-9 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 pl-4 pr-3 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
                    >
                        <p className="text-slate-900 dark:text-white text-sm font-medium leading-normal">
                            Customer: {customerFilter === 'all' ? 'All' : customerFilter}
                        </p>
                        <span className="material-symbols-outlined text-lg text-slate-500">expand_more</span>
                    </button>
                    {isCustomerOpen && (
                        <div className="absolute left-0 top-10 w-56 z-10 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 py-1 max-h-60 overflow-y-auto">
                            <button onClick={() => setCustomerFilter('all')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">All Customers</button>
                            {uniqueCustomers.map(c => (
                                <button key={c} onClick={() => setCustomerFilter(c)} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700 truncate">{c}</button>
                            ))}
                        </div>
                    )}
                </div>

                {/* Plan Filter */}
                <div className="relative">
                    <button
                        onClick={(e) => { e.stopPropagation(); setIsPlanOpen(!isPlanOpen); setIsCustomerOpen(false) }}
                        className="flex h-9 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 pl-4 pr-3 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
                    >
                        <p className="text-slate-900 dark:text-white text-sm font-medium leading-normal">
                            Plan: {planFilter === 'all' ? 'All' : planFilter}
                        </p>
                        <span className="material-symbols-outlined text-lg text-slate-500">expand_more</span>
                    </button>
                    {isPlanOpen && (
                        <div className="absolute left-0 top-10 w-48 z-10 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 py-1">
                            <button onClick={() => setPlanFilter('all')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">All Plans</button>
                            {uniquePlans.map(p => (
                                <button key={p} onClick={() => setPlanFilter(p)} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700 truncate">{p}</button>
                            ))}
                        </div>
                    )}
                </div>

            </div>

            {/* Stats */}
            <div className="grid grid-cols-1 gap-6 py-6 sm:grid-cols-2 lg:grid-cols-3">
                <div className="flex flex-col gap-2 rounded-lg p-6 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm">
                    <p className="text-slate-600 dark:text-slate-400 text-base font-medium leading-normal">Total Units Consumed</p>
                    <p className="text-slate-900 dark:text-white tracking-tight text-3xl font-bold leading-tight">
                        {totalUnits.toLocaleString()}
                    </p>
                    <p className="text-emerald-500 text-base font-medium leading-normal">Lifetime</p>
                </div>
                <div className="flex flex-col gap-2 rounded-lg p-6 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm">
                    <p className="text-slate-600 dark:text-slate-400 text-base font-medium leading-normal">Customers Metered</p>
                    <p className="text-slate-900 dark:text-white tracking-tight text-3xl font-bold leading-tight">
                        {[...new Set(filteredData.map(d => d.customer_id))].length}
                    </p>
                    <p className="text-slate-500 text-base font-medium leading-normal">With recorded usage</p>
                </div>
                <div className="flex flex-col gap-2 rounded-lg p-6 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm">
                    <p className="text-slate-600 dark:text-slate-400 text-base font-medium leading-normal">Active Dimensions</p>
                    <p className="text-slate-900 dark:text-white tracking-tight text-3xl font-bold leading-tight">
                        {[...new Set(filteredData.map(d => d.dimension))].length}
                    </p>
                    <p className="text-emerald-500 text-base font-medium leading-normal">Types</p>
                </div>
            </div>

            {/* Charts */}
            <div className="grid grid-cols-1 lg:grid-cols-5 gap-6 mb-8">
                {/* Main Chart */}
                <div className="flex min-w-72 flex-col gap-2 rounded-lg border border-slate-200 dark:border-slate-800 p-6 bg-white dark:bg-slate-900 shadow-sm lg:col-span-3">
                    <p className="text-slate-900 dark:text-white text-base font-medium leading-normal">Usage by Dimension</p>
                    <p className="text-slate-900 dark:text-white tracking-tight text-3xl font-bold leading-tight truncate">
                        {totalUnits.toLocaleString()} Units
                    </p>
                    <p className="text-slate-600 dark:text-slate-400 text-base font-normal leading-normal">All recorded usage</p>
                    <div className="relative flex-1 w-full min-h-[180px] py-4">
                        {byDimension.length > 0 ? (
                            <ResponsiveContainer width="100%" height={180}>
                                <BarChart data={byDimension}>
                                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#334155" opacity={0.2} />
                                    <XAxis dataKey="dimension" tick={{ fontSize: 12 }} stroke="#94a3b8" />
                                    <YAxis tick={{ fontSize: 12 }} stroke="#94a3b8" width={60} />
                                    <Tooltip formatter={(v) => v.toLocaleString()} />
                                    <Bar dataKey="units" fill="#6366F1" radius={[4, 4, 0, 0]} />
                                </BarChart>
                            </ResponsiveContainer>
                        ) : (
                            <div className="flex h-full min-h-[180px] items-center justify-center text-sm text-slate-500">
                                No usage recorded yet.
                            </div>
                        )}
                    </div>
                </div>

                {/* Side Widget */}
                <div className="flex min-w-72 flex-col gap-2 rounded-lg border border-slate-200 dark:border-slate-800 p-6 bg-white dark:bg-slate-900 shadow-sm lg:col-span-2">
                    <p className="text-slate-900 dark:text-white text-base font-medium leading-normal">Usage by Metric</p>
                    <div className="grid min-h-[180px] grid-flow-col gap-6 grid-rows-[1fr_auto] items-end justify-items-center px-3 pt-8 pb-2">
                        {byDimension.length > 0 ? byDimension.map((stat, idx) => {
                            const max = Math.max(...byDimension.map(d => d.units), 1)
                            return (
                                <div key={idx} className="flex flex-col items-center gap-2 w-full">
                                    <div
                                        className="bg-primary/20 dark:bg-primary/30 w-full rounded-t-sm"
                                        style={{ height: `${Math.max(8, Math.round((stat.units / max) * 120))}px` }}
                                    ></div>
                                    <p className="text-slate-600 dark:text-slate-400 text-xs font-bold uppercase tracking-wider">{stat.dimension}</p>
                                    <p className="text-slate-900 dark:text-white text-xs">{stat.units.toLocaleString()}</p>
                                </div>
                            )
                        }) : (
                            <p className="col-span-full text-slate-500 text-sm">No usage data found.</p>
                        )}
                    </div>
                </div>
            </div>

            {/* Table */}
            <div className="w-full">
                <div className="overflow-hidden rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-sm">
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
                            <thead className="bg-slate-50 dark:bg-slate-900/50">
                                <tr>
                                    <th className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-slate-900 dark:text-white sm:pl-6">Customer</th>
                                    <th className="px-3 py-3.5 text-left text-sm font-semibold text-slate-900 dark:text-white">Plan</th>
                                    <th className="px-3 py-3.5 text-left text-sm font-semibold text-slate-900 dark:text-white">Metric</th>
                                    <th className="px-3 py-3.5 text-left text-sm font-semibold text-slate-900 dark:text-white">Usage</th>
                                    <th className="px-3 py-3.5 text-left text-sm font-semibold text-slate-900 dark:text-white">Status</th>
                                    <th className="px-3 py-3.5 text-left text-sm font-semibold text-slate-900 dark:text-white">Timestamp</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-200 dark:divide-slate-800 bg-white dark:bg-slate-900">
                                {usageStats.length === 0 ? (
                                    <tr><td colSpan="6" className="p-8 text-center text-slate-500">No events found.</td></tr>
                                ) : (
                                    usageStats.map((item, index) => (
                                        <tr key={index} className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
                                            <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-slate-900 dark:text-white sm:pl-6">
                                                {item.customer_id ? item.customer_id.substring(0, 8) + '...' : 'Unknown'}
                                            </td>
                                            <td className="whitespace-nowrap px-3 py-4 text-sm text-slate-500 dark:text-slate-400">
                                                {item.plan_id ? 'Active Plan' : '-'}
                                            </td>
                                            <td className="whitespace-nowrap px-3 py-4 text-sm text-slate-500 dark:text-slate-400">{item.dimension}</td>
                                            <td className="whitespace-nowrap px-3 py-4 text-sm text-slate-500 dark:text-slate-400">{item.total_quantity}</td>
                                            <td className="whitespace-nowrap px-3 py-4 text-sm">
                                                <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300">
                                                    Recorded
                                                </span>
                                            </td>
                                            <td className="whitespace-nowrap px-3 py-4 text-sm text-slate-500 dark:text-slate-400">
                                                {/* Timestamp not in aggregate, skipping or assuming recent */}
                                                Recently
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Usage
