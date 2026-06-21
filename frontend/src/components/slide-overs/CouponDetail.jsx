import React from 'react'
import SlideOver from '../ui/SlideOver'

const CouponDetail = ({ coupon, isOpen, onClose }) => {
    if (!coupon) return null

    const progress = coupon.max_redemptions
        ? Math.round((coupon.redemptions / coupon.max_redemptions) * 100)
        : 0

    return (
        <SlideOver isOpen={isOpen} onClose={onClose} title="Coupon Details">
            <div className="flex flex-col h-full">
                {/* Header Info */}
                <div className="flex flex-col gap-4 pb-6">
                    <div className="flex items-center justify-between">
                        <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white font-mono">{coupon.code}</h1>
                        <span className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm font-medium ${coupon.status === 'active'
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                            : 'bg-gray-100 text-gray-800 dark:bg-gray-700/50 dark:text-gray-300'
                            }`}>
                            <span className={`size-2 rounded-full ${coupon.status === 'active' ? 'bg-green-500' : 'bg-gray-500'}`}></span>
                            {coupon.status.charAt(0).toUpperCase() + coupon.status.slice(1)}
                        </span>
                    </div>
                </div>

                {/* Redemptions Progress */}
                {coupon.max_redemptions && (
                    <div className="flex flex-col gap-3 py-6 border-t border-slate-200 dark:border-slate-800">
                        <div className="flex items-center justify-between gap-6">
                            <p className="text-base font-medium leading-normal text-slate-900 dark:text-white">Redemptions</p>
                            <p className="text-sm font-normal leading-normal text-slate-500 dark:text-slate-400">{progress}%</p>
                        </div>
                        <div className="h-2 w-full rounded-full bg-slate-200 dark:bg-slate-700">
                            <div className="h-2 rounded-full bg-primary transition-all duration-500" style={{ width: `${progress}%` }}></div>
                        </div>
                        <p className="text-sm font-normal leading-normal text-slate-500 dark:text-slate-400">
                            {coupon.redemptions} of {coupon.max_redemptions} used
                        </p>
                    </div>
                )}

                {/* Description List */}
                <div className="grid grid-cols-1 gap-x-4 border-t border-slate-200 dark:border-slate-800 sm:grid-cols-2">
                    <div className="flex flex-col gap-1 py-4">
                        <p className="text-sm font-normal text-slate-500 dark:text-slate-400">Discount</p>
                        <p className="text-sm font-medium text-slate-900 dark:text-white">{coupon.discount}</p>
                    </div>
                    <div className="flex flex-col gap-1 border-t border-slate-200 dark:border-slate-800 py-4 sm:border-t-0">
                        <div className="flex items-center gap-1.5">
                            <p className="text-sm font-normal text-slate-500 dark:text-slate-400">Duration</p>
                            <span className="material-symbols-outlined text-sm text-slate-400">info</span>
                        </div>
                        <p className="text-sm font-medium text-slate-900 dark:text-white capitalize">
                            {coupon.duration === 'repeating' ? `For ${coupon.duration_in_months} months` : coupon.duration}
                        </p>
                    </div>
                    <div className="flex flex-col gap-1 border-t border-slate-200 dark:border-slate-800 py-4">
                        <p className="text-sm font-normal text-slate-500 dark:text-slate-400">Created Date</p>
                        <p className="text-sm font-medium text-slate-900 dark:text-white">Jan 22, 2024</p>
                    </div>
                    <div className="flex flex-col gap-1 border-t border-slate-200 dark:border-slate-800 py-4">
                        <p className="text-sm font-normal text-slate-500 dark:text-slate-400">Applies To</p>
                        <p className="text-sm font-medium text-slate-900 dark:text-white">All products</p>
                    </div>
                    <div className="flex flex-col gap-1 border-t border-slate-200 dark:border-slate-800 py-4">
                        <p className="text-sm font-normal text-slate-500 dark:text-slate-400">Customer Limit</p>
                        <p className="text-sm font-medium text-slate-900 dark:text-white">One per customer</p>
                    </div>
                </div>

                {/* Footer Buttons */}
                <div className="mt-auto pt-6 flex shrink-0 items-center justify-end gap-3 border-t border-slate-200 dark:border-slate-800">
                    <button className="flex h-10 items-center justify-center rounded-lg bg-slate-100 dark:bg-slate-800 px-4 text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors">
                        Deactivate
                    </button>
                    <button className="flex h-10 items-center justify-center rounded-lg bg-primary px-4 text-sm font-medium text-white hover:bg-primary/90 transition-colors">
                        Edit Coupon
                    </button>
                </div>
            </div>
        </SlideOver>
    )
}

export default CouponDetail
