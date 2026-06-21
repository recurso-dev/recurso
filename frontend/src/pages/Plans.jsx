import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { endpoints } from '../lib/api'
import BuyGiftModal from '../components/BuyGiftModal'
import PlanDetail from '../components/slide-overs/PlanDetail'

const Plans = () => {
    const [plans, setPlans] = useState([])
    const [loading, setLoading] = useState(true)
    const [selectedPlan, setSelectedPlan] = useState(null)
    const [isDetailOpen, setIsDetailOpen] = useState(false)
    const [isGiftModalOpen, setIsGiftModalOpen] = useState(false)

    const [currencyFilter, setCurrencyFilter] = useState('all') // 'all', 'USD', 'INR'
    const [intervalFilter, setIntervalFilter] = useState('all') // 'all', 'month', 'year'

    // Dropdown UI states
    const [isCurrencyOpen, setIsCurrencyOpen] = useState(false)
    const [isIntervalOpen, setIsIntervalOpen] = useState(false)

    // Filter Logic
    const filteredPlans = plans.filter(p => {
        let match = true
        // 1. Currency (Check prices)
        if (currencyFilter !== 'all') {
            const hasCurrency = p.prices.some(price => price.currency === currencyFilter)
            if (!hasCurrency) match = false
        }
        // 2. Interval
        if (intervalFilter !== 'all') {
            if (p.interval_unit !== intervalFilter) match = false
        }
        return match
    })

    // Click outside handler
    useEffect(() => {
        const close = () => {
            setIsCurrencyOpen(false)
            setIsIntervalOpen(false)
        }
        if (isCurrencyOpen || isIntervalOpen) window.addEventListener('click', close)
        return () => window.removeEventListener('click', close)
    }, [isCurrencyOpen, isIntervalOpen])

    const fetchPlans = async () => {
        try {
            const response = await endpoints.getPlans()
            setPlans(response.data.data || [])
        } catch (error) {
            console.error(error)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchPlans()
    }, [])

    const handleRowClick = (plan) => {
        setSelectedPlan(plan)
        setIsDetailOpen(true)
    }

    const closeDetail = () => {
        setIsDetailOpen(false)
        setTimeout(() => setSelectedPlan(null), 300) // Wait for transition
    }

    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            <div className="flex flex-wrap items-center justify-between gap-3 pb-6 border-b border-slate-200 dark:border-slate-800">
                <h1 className="text-slate-900 dark:text-white text-3xl font-bold leading-tight tracking-tight min-w-72">Plans</h1>
                <div className="flex items-center gap-3">
                    <button
                        onClick={() => setIsGiftModalOpen(true)}
                        className="flex min-w-[84px] cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg h-10 px-4 bg-white border border-slate-200 text-slate-700 text-sm font-bold hover:bg-slate-50 dark:bg-slate-800 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-700 transition-all"
                    >
                        <span className="material-symbols-outlined text-xl">card_giftcard</span>
                        <span className="truncate">Gift Plan</span>
                    </button>
                    <Link
                        to="/plans/new"
                        className="flex min-w-[84px] cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold leading-normal tracking-[0.015em] hover:bg-primary/90 transition-all"
                    >
                        <span className="material-symbols-outlined text-xl">add</span>
                        <span className="truncate">New Plan</span>
                    </Link>
                </div>
            </div>

            {/* Filters */}
            <div className="flex gap-3 py-6 overflow-x-auto">
                <div className="relative flex-grow">
                    <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 dark:text-slate-500 text-xl">search</span>
                    <input
                        className="block w-full rounded-lg border-slate-300 bg-white dark:bg-slate-800 dark:border-slate-700 dark:text-white dark:placeholder-slate-400 pl-10 h-10 text-sm focus:border-primary focus:ring-primary/20"
                        placeholder="Search by plan name or ID..."
                        type="text"
                    />
                </div>

                {/* Currency Dropdown */}
                <div className="relative">
                    <button
                        onClick={(e) => { e.stopPropagation(); setIsCurrencyOpen(!isCurrencyOpen); setIsIntervalOpen(false) }}
                        className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-white dark:bg-slate-800 dark:border-slate-700 border border-slate-300 pl-4 pr-3 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                    >
                        <p className="text-sm font-medium leading-normal capitalize">Currency: {currencyFilter}</p>
                        <span className="material-symbols-outlined text-xl">expand_more</span>
                    </button>
                    {isCurrencyOpen && (
                        <div className="absolute right-0 top-12 w-32 z-10 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 py-1">
                            <button onClick={() => setCurrencyFilter('all')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">All</button>
                            <button onClick={() => setCurrencyFilter('USD')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">USD</button>
                            <button onClick={() => setCurrencyFilter('INR')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">INR</button>
                        </div>
                    )}
                </div>

                {/* Billing Interval Dropdown */}
                <div className="relative">
                    <button
                        onClick={(e) => { e.stopPropagation(); setIsIntervalOpen(!isIntervalOpen); setIsCurrencyOpen(false) }}
                        className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-white dark:bg-slate-800 dark:border-slate-700 border border-slate-300 pl-4 pr-3 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                    >
                        <p className="text-sm font-medium leading-normal capitalize">Interval: {intervalFilter}</p>
                        <span className="material-symbols-outlined text-xl">expand_more</span>
                    </button>
                    {isIntervalOpen && (
                        <div className="absolute right-0 top-12 w-32 z-10 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 py-1">
                            <button onClick={() => setIntervalFilter('all')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">All</button>
                            <button onClick={() => setIntervalFilter('month')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">Month</button>
                            <button onClick={() => setIntervalFilter('year')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">Year</button>
                        </div>
                    )}
                </div>
            </div>

            {/* Table */}
            <div className="w-full">
                <div className="overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900 shadow-sm">
                    <table className="w-full">
                        <thead className="bg-slate-50 border-b border-slate-200 dark:bg-slate-800/50 dark:border-slate-800">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Plan Name</th>
                                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Plan ID</th>
                                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Price</th>
                                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Billing Interval</th>
                                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Status</th>
                                <th className="px-6 py-3 text-right text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400"></th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                            {loading ? (
                                <tr><td colSpan="6" className="p-8 text-center text-slate-500">Loading plans...</td></tr>
                            ) : filteredPlans.length === 0 ? (
                                <tr><td colSpan="6" className="p-8 text-center text-slate-500">No plans found matching filters.</td></tr>
                            ) : (
                                filteredPlans.map((plan) => (
                                    <tr
                                        key={plan.id}
                                        className="cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
                                        onClick={() => handleRowClick(plan)}
                                    >
                                        <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-slate-900 dark:text-white">
                                            {plan.name}
                                        </td>
                                        <td className="whitespace-nowrap px-6 py-4 text-sm font-mono text-slate-500 dark:text-slate-400">
                                            {plan.code}
                                        </td>
                                        <td className="whitespace-nowrap px-6 py-4 text-sm text-slate-600 dark:text-slate-300">
                                            {plan.prices && plan.prices.length > 0
                                                ? new Intl.NumberFormat('en-US', { style: 'currency', currency: plan.prices[0].currency }).format(plan.prices[0].amount / 100)
                                                : 'Free'}
                                        </td>
                                        <td className="whitespace-nowrap px-6 py-4 text-sm text-slate-600 dark:text-slate-300 capitalize">
                                            {plan.interval_unit}
                                        </td>
                                        <td className="whitespace-nowrap px-6 py-4 text-sm">
                                            <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${plan.active
                                                ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                                                : 'bg-gray-100 text-gray-800 dark:bg-gray-700/50 dark:text-gray-300'
                                                }`}>
                                                <span className={`size-1.5 rounded-full ${plan.active ? 'bg-green-500' : 'bg-gray-500'}`}></span>
                                                {plan.active ? 'Active' : 'Archived'}
                                            </span>
                                        </td>
                                        <td className="whitespace-nowrap px-6 py-4 text-right text-sm font-medium">
                                            <button className="text-slate-400 hover:text-primary dark:hover:text-primary transition-colors">
                                                <span className="material-symbols-outlined">more_horiz</span>
                                            </button>
                                        </td>
                                    </tr >
                                ))
                            )}
                        </tbody >
                    </table >
                </div >
            </div >

            {/* Pagination Mock */}
            <div className="flex items-center justify-between pt-6 text-sm text-slate-600 dark:text-slate-400">
                <p>Showing <span className="font-medium">{plans.length > 0 ? 1 : 0}</span> to <span className="font-medium">{plans.length}</span> of <span className="font-medium">{plans.length}</span> results</p>
                <div className="flex items-center justify-center gap-2">
                    <button disabled className="flex size-9 items-center justify-center rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 disabled:opacity-50">
                        <span className="material-symbols-outlined text-base">chevron_left</span>
                    </button>
                    <button disabled className="flex size-9 items-center justify-center rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 disabled:opacity-50">
                        <span className="material-symbols-outlined text-base">chevron_right</span>
                    </button>
                </div>
            </div>

            <PlanDetail
                plan={selectedPlan}
                isOpen={isDetailOpen}
                onClose={closeDetail}
            />

            <BuyGiftModal
                isOpen={isGiftModalOpen}
                onClose={() => setIsGiftModalOpen(false)}
                plans={plans}
            />
        </div>
    )
}

export default Plans
