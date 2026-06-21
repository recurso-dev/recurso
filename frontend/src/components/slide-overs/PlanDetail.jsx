import React from 'react'
import SlideOver from '../ui/SlideOver'

const PlanDetail = ({ plan, isOpen, onClose }) => {
    if (!plan) return null

    const price = plan.prices && plan.prices[0]
    const amount = price ? (price.amount / 100).toFixed(2) : '0.00'
    const currency = price ? price.currency.toUpperCase() : 'USD'

    return (
        <SlideOver isOpen={isOpen} onClose={onClose} title={plan.name}>
            <div className="flex flex-col gap-6">
                {/* Header Info */}
                <div className="flex flex-col gap-2 pb-6 border-b border-slate-200 dark:border-slate-800">
                    <div className="flex items-center gap-3">
                        <h1 className="text-xl font-bold text-slate-900 dark:text-white">{plan.name}</h1>
                        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${plan.active
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                            : 'bg-gray-100 text-gray-800 dark:bg-gray-700/50 dark:text-gray-300'
                            }`}>
                            {plan.active ? 'Active' : 'Inactive'}
                        </span>
                    </div>
                    <p className="font-mono text-xs text-slate-500 dark:text-slate-400">{plan.id}</p>

                    <div className="flex justify-start gap-3 mt-2">
                        <button className="flex h-9 min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-md bg-primary px-4 text-sm font-medium text-white shadow-sm hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:ring-offset-2">
                            <span className="truncate">Edit Plan</span>
                        </button>
                        <button className="flex h-9 min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-md bg-slate-100 px-4 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-300 focus:ring-offset-2 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700">
                            <span className="truncate">More</span>
                            <span className="material-symbols-outlined text-base ml-1.5 -mr-1">expand_more</span>
                        </button>
                    </div>
                </div>

                {/* Details Section */}
                <div className="grid grid-cols-1 gap-y-4 sm:grid-cols-2">
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Price</p>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {new Intl.NumberFormat('en-US', { style: 'currency', currency: currency }).format(amount)}
                        </p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Billing Interval</p>
                        <p className="text-sm text-slate-900 dark:text-white capitalize">{plan.interval_count > 1 ? `${plan.interval_count} ` : ''}{plan.interval_unit}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Created</p>
                        <p className="text-sm text-slate-900 dark:text-white">{new Date(plan.created_at).toLocaleDateString()}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Currency</p>
                        <p className="text-sm text-slate-900 dark:text-white">{currency}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Code</p>
                        <p className="text-sm font-mono text-slate-900 dark:text-white">{plan.code}</p>
                    </div>
                </div>

                {/* Features (Mock for now as backend model might not have them) */}
                <div>
                    <h3 className="text-base font-semibold leading-tight text-slate-900 dark:text-white mb-4">Features</h3>
                    <div className="flex flex-col gap-2">
                        <div className="flex items-center gap-3">
                            <span className="material-symbols-outlined text-green-500 text-sm">check</span>
                            <p className="text-sm text-slate-700 dark:text-slate-300">Standard Support</p>
                        </div>
                        <div className="flex items-center gap-3">
                            <span className="material-symbols-outlined text-green-500 text-sm">check</span>
                            <p className="text-sm text-slate-700 dark:text-slate-300">Basic Analytics</p>
                        </div>
                    </div>
                </div>

                {/* Usage-Based Tiers (Mock) */}
                {/* Only show if usage based */}
                <div>
                    <h3 className="text-base font-semibold leading-tight text-slate-900 dark:text-white mb-4">Usage Tiers</h3>
                    <div className="rounded-lg border border-slate-200 dark:border-slate-800 overflow-hidden">
                        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
                            <thead className="bg-slate-50 dark:bg-slate-800/50">
                                <tr>
                                    <th className="px-3 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Tier</th>
                                    <th className="px-3 py-2 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Price</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-200 dark:divide-slate-800 bg-white dark:bg-slate-900">
                                <tr>
                                    <td className="px-3 py-2 text-sm text-slate-900 dark:text-white">Base</td>
                                    <td className="px-3 py-2 text-sm text-right text-slate-900 dark:text-white">$0.00</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                {/* Metadata */}
                <div>
                    <h3 className="text-base font-semibold leading-tight text-slate-900 dark:text-white mb-4">Metadata</h3>
                    <div className="rounded-lg bg-slate-100 dark:bg-slate-800 p-4 overflow-x-auto">
                        <pre className="font-mono text-xs text-slate-800 dark:text-slate-300">
                            {JSON.stringify(plan.metadata || {}, null, 2)}
                        </pre>
                    </div>
                </div>
            </div>
        </SlideOver>
    )
}

export default PlanDetail
