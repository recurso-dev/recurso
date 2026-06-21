import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { endpoints as api } from '../lib/api'
import CouponDetail from '../components/slide-overs/CouponDetail'

const Coupons = () => {
    const navigate = useNavigate()
    const [coupons, setCoupons] = useState([])
    const [loading, setLoading] = useState(true)

    const [selectedCoupon, setSelectedCoupon] = useState(null)
    const [isDetailOpen, setIsDetailOpen] = useState(false)

    useEffect(() => {
        const fetchCoupons = async () => {
            try {
                const response = await api.getCoupons()
                // Map backend fields to frontend expectations
                const mappedCoupons = (response.data.data || []).map(c => ({
                    ...c,
                    status: 'active', // Default to active as backend doesn't return status yet
                    redemptions: 0,
                    max_redemptions: null,
                    discount: c.discount_type === 'percent' ? `${c.discount_value}%` : `$${(c.discount_value / 100).toFixed(2)}`,
                    duration_in_months: c.duration_months
                }))
                setCoupons(mappedCoupons)
            } catch (error) {
                console.error("Failed to fetch coupons:", error)
            } finally {
                setLoading(false)
            }
        }
        fetchCoupons()
    }, [])

    const handleRowClick = (coupon) => {
        setSelectedCoupon(coupon)
        setIsDetailOpen(true)
    }

    const closeDetail = () => {
        setIsDetailOpen(false)
        setTimeout(() => setSelectedCoupon(null), 300)
    }

    const [statusFilter, setStatusFilter] = useState('all') // 'all', 'active', 'expired'
    const [isFilterOpen, setIsFilterOpen] = useState(false)

    // Filter Logic
    const filteredCoupons = coupons.filter(c => {
        if (statusFilter === 'all') return true
        return c.status === statusFilter
    })

    // Click outside handler
    useEffect(() => {
        const close = () => setIsFilterOpen(false)
        if (isFilterOpen) window.addEventListener('click', close)
        return () => window.removeEventListener('click', close)
    }, [isFilterOpen])

    const handleFilterClick = (e, filter) => {
        e.stopPropagation()
        setStatusFilter(filter)
        setIsFilterOpen(false)
    }

    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            <div className="flex flex-wrap items-center justify-between gap-4 pb-8">
                <div className="flex flex-col gap-1">
                    <h1 className="text-3xl font-bold tracking-tight text-slate-900 dark:text-white">Coupons</h1>
                    <p className="text-base font-normal text-slate-500 dark:text-slate-400">Create and manage discount codes for your customers.</p>
                </div>
                <button
                    onClick={() => navigate('/coupons/new')}
                    className="flex h-10 cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg bg-primary px-4 text-sm font-semibold text-white shadow-sm transition-all hover:bg-primary/90"
                >
                    <span className="material-symbols-outlined text-xl">add</span>
                    <span className="truncate">Create Coupon</span>
                </button>
            </div>

            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden">
                {/* ToolBar */}
                <div className="flex justify-between items-center gap-2 px-4 py-3 border-b border-slate-200 dark:border-slate-800">
                    <div className="relative w-full max-w-sm">
                        <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                            <span className="material-symbols-outlined text-slate-400 dark:text-slate-500 text-xl">search</span>
                        </div>
                        <input
                            className="block w-full rounded-lg border-slate-300 bg-white dark:bg-slate-800 dark:border-slate-700 dark:text-white dark:placeholder-slate-400 pl-10 h-10 text-sm focus:border-primary focus:ring-primary/20"
                            placeholder="Search coupons..."
                            type="text"
                        />
                    </div>

                    {/* Filter Dropdown */}
                    <div className="relative">
                        <button
                            onClick={(e) => { e.stopPropagation(); setIsFilterOpen(!isFilterOpen) }}
                            className="flex items-center justify-center gap-2 rounded-lg h-9 px-3 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 text-sm font-medium border border-slate-300 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700"
                        >
                            <span className="material-symbols-outlined text-base">filter_list</span>
                            Filter: {statusFilter === 'all' ? 'All' : statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1)}
                        </button>
                        {isFilterOpen && (
                            <div className="absolute right-0 top-10 w-40 z-10 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 py-1">
                                <button onClick={(e) => handleFilterClick(e, 'all')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">All</button>
                                <button onClick={(e) => handleFilterClick(e, 'active')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">Active</button>
                                <button onClick={(e) => handleFilterClick(e, 'expired')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">Expired</button>
                            </div>
                        )}
                    </div>
                </div>

                {/* Table */}
                <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
                        <thead className="bg-slate-50 dark:bg-slate-800/50">
                            <tr>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Coupon Code</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Discount</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Duration</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Redemptions</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Status</th>
                                <th scope="col" className="relative px-6 py-3"><span className="sr-only">Actions</span></th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                            {loading ? (
                                <tr><td colSpan="6" className="p-8 text-center text-slate-500">Loading coupons...</td></tr>
                            ) : filteredCoupons.length === 0 ? (
                                <tr><td colSpan="6" className="p-8 text-center text-slate-500">No coupons found.</td></tr>
                            ) : (
                                filteredCoupons.map((coupon) => (
                                    <tr
                                        key={coupon.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer"
                                        onClick={() => handleRowClick(coupon)}
                                    >
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-mono font-medium text-slate-900 dark:text-white">{coupon.code}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-400">{coupon.discount}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-400 capitalize">
                                            {coupon.duration === 'repeating' ? `For ${coupon.duration_in_months} months` : coupon.duration}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-400">
                                            {coupon.redemptions} {coupon.max_redemptions ? `/ ${coupon.max_redemptions}` : ''}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                                            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
                                                ${coupon.status === 'active' ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300' :
                                                    coupon.status === 'expired' ? 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300' :
                                                        'bg-gray-100 text-gray-800 dark:bg-gray-700/50 dark:text-gray-300'}`}>
                                                {coupon.status.charAt(0).toUpperCase() + coupon.status.slice(1)}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <button className="text-slate-400 hover:text-primary dark:hover:text-primary transition-colors">
                                                <span className="material-symbols-outlined text-xl">more_horiz</span>
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>

                {/* Pagination */}
                <div className="px-4 py-3 border-t border-slate-200 dark:border-slate-800 flex items-center justify-between">
                    <p className="text-sm text-slate-600 dark:text-slate-400">Showing <span className="font-medium">1</span> to <span className="font-medium">{filteredCoupons.length}</span> of <span className="font-medium">{coupons.length}</span> results</p>
                    <div className="flex gap-2">
                        <button className="flex h-8 w-8 items-center justify-center rounded border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50" disabled>
                            <span className="material-symbols-outlined text-lg">chevron_left</span>
                        </button>
                        <button className="flex h-8 w-8 items-center justify-center rounded border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50" disabled>
                            <span className="material-symbols-outlined text-lg">chevron_right</span>
                        </button>
                    </div>
                </div>
            </div>

            <CouponDetail
                coupon={selectedCoupon}
                isOpen={isDetailOpen}
                onClose={closeDetail}
            />
        </div>
    )
}

export default Coupons
