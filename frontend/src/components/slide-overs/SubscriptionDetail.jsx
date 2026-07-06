import React, { useState } from 'react'
import SlideOver from '../ui/SlideOver'
import { endpoints } from '../../lib/api'

const SubscriptionDetail = ({ subscription, customer, plan, isOpen, onClose, onRefresh }) => {
    const [loading, setLoading] = useState(false)

    if (!subscription) return null

    const price = plan?.prices?.[0]
    const amount = price ? (price.amount / 100).toFixed(2) : '0.00'
    const currency = price ? price.currency.toUpperCase() : 'USD'
    const planName = plan?.name || subscription.plan_id.slice(0, 8)
    const interval = plan?.interval_unit || 'month'

    const handlePause = async () => {
        if (!confirm('Are you sure you want to pause this subscription?')) return;
        setLoading(true);
        try {
            await endpoints.pauseSubscription(subscription.id);
            if (onRefresh) onRefresh();
        } catch (err) {
            alert('Failed to pause subscription');
        } finally {
            setLoading(false);
        }
    }

    const handleResume = async () => {
        if (!confirm('Are you sure you want to resume this subscription?')) return;
        setLoading(true);
        try {
            await endpoints.resumeSubscription(subscription.id);
            if (onRefresh) onRefresh();
        } catch (err) {
            alert('Failed to resume subscription');
        } finally {
            setLoading(false);
        }
    }

    return (
        <SlideOver isOpen={isOpen} onClose={onClose} title="Subscription Details">
            <div className="flex flex-col h-full">
                {/* Toolbar */}
                <div className="flex items-center gap-2 pb-6 border-b border-slate-200 dark:border-slate-800 mb-6">
                    <button className="flex items-center justify-center gap-2 rounded-lg bg-white dark:bg-slate-800 px-3 py-1.5 text-sm font-medium text-slate-700 dark:text-slate-200 shadow-sm ring-1 ring-inset ring-slate-300 dark:ring-slate-600 hover:bg-slate-50 dark:hover:bg-slate-700">Edit</button>
                    {subscription.status === 'active' && (
                        <button
                            onClick={handlePause}
                            disabled={loading}
                            className="flex items-center justify-center gap-2 rounded-lg bg-amber-50 dark:bg-amber-900/30 px-3 py-1.5 text-sm font-medium text-amber-700 dark:text-amber-400 shadow-sm ring-1 ring-inset ring-amber-300 dark:ring-amber-600 hover:bg-amber-100 dark:hover:bg-amber-900/50 disabled:opacity-50"
                        >
                            <span className="material-symbols-outlined !text-sm">pause</span>
                            Pause
                        </button>
                    )}
                    {subscription.status === 'paused' && (
                        <button
                            onClick={handleResume}
                            disabled={loading}
                            className="flex items-center justify-center gap-2 rounded-lg bg-green-50 dark:bg-green-900/30 px-3 py-1.5 text-sm font-medium text-green-700 dark:text-green-400 shadow-sm ring-1 ring-inset ring-green-300 dark:ring-green-600 hover:bg-green-100 dark:hover:bg-green-900/50 disabled:opacity-50"
                        >
                            <span className="material-symbols-outlined !text-sm">play_arrow</span>
                            Resume
                        </button>
                    )}
                    <button className="flex items-center justify-center gap-2 rounded-lg bg-white dark:bg-slate-800 px-3 py-1.5 text-sm font-medium text-red-600 dark:text-red-400 shadow-sm ring-1 ring-inset ring-slate-300 dark:ring-slate-600 hover:bg-red-50 dark:hover:bg-red-900/20">Cancel</button>
                    <button className="flex items-center justify-center gap-2 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-white shadow-sm hover:bg-primary/90">Renew</button>
                </div>

                {/* Header */}
                <div className="flex flex-col gap-2 mb-6">
                    <p className="text-slate-900 dark:text-white tracking-tight text-xl font-bold font-mono">{subscription.id}</p>
                    <div className="flex items-center gap-2">
                        <span className="relative flex h-2 w-2">
                            <span className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${subscription.status === 'active' ? 'bg-green-400' : 'bg-gray-400'}`}></span>
                            <span className={`relative inline-flex rounded-full h-2 w-2 ${subscription.status === 'active' ? 'bg-green-500' : 'bg-gray-500'}`}></span>
                        </span>
                        <p className={`text-sm font-medium leading-normal capitalize ${subscription.status === 'active' ? 'text-green-600 dark:text-green-400' : 'text-slate-600 dark:text-slate-400'}`}>
                            {subscription.status}
                        </p>
                    </div>
                </div>

                {/* Description List */}
                <div className="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2">
                    <div className="flex flex-col gap-1">
                        <p className="text-slate-500 dark:text-slate-400 text-sm font-normal">Customer</p>
                        <p className="text-slate-900 dark:text-white text-sm font-medium">{customer?.name || 'Unknown'}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-slate-500 dark:text-slate-400 text-sm font-normal">Plan</p>
                        <p className="text-slate-900 dark:text-white text-sm font-medium">{planName} - {interval}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-slate-500 dark:text-slate-400 text-sm font-normal">Amount</p>
                        <p className="text-slate-900 dark:text-white text-sm font-medium">
                            {new Intl.NumberFormat('en-US', { style: 'currency', currency: currency }).format(amount)}
                        </p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-slate-500 dark:text-slate-400 text-sm font-normal">Created</p>
                        <p className="text-slate-900 dark:text-white text-sm font-medium">{new Date(subscription.created_at).toLocaleDateString()}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-slate-500 dark:text-slate-400 text-sm font-normal">Current Period</p>
                        <p className="text-slate-900 dark:text-white text-sm font-medium">
                            {new Date(subscription.current_period_start).toLocaleDateString()} - {new Date(subscription.current_period_end).toLocaleDateString()}
                        </p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-slate-500 dark:text-slate-400 text-sm font-normal">Upcoming Invoice</p>
                        <p className="text-slate-900 dark:text-white text-sm font-medium">
                            {new Date(subscription.current_period_end).toLocaleDateString()} for {new Intl.NumberFormat('en-US', { style: 'currency', currency: currency }).format(amount)}
                        </p>
                    </div>
                </div>

                {/* Timeline derived from the subscription's real dates
                    (created_at / current_period_end); there is no per-
                    subscription event history endpoint yet. */}
                <h3 className="text-slate-900 dark:text-white text-base font-semibold leading-tight tracking-tight pt-8 pb-4 border-t border-slate-200 dark:border-slate-800 mt-8">Timeline</h3>
                <div className="flow-root">
                    <ul className="-mb-8">
                        <li>
                            <div className="relative pb-8">
                                <span className="absolute left-3 top-3 -ml-px h-full w-0.5 bg-slate-200 dark:bg-slate-700" aria-hidden="true"></span>
                                <div className="relative flex items-center space-x-3">
                                    <div>
                                        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-green-500 ring-4 ring-white dark:ring-slate-900">
                                            <span className="material-symbols-outlined !text-sm text-white">check</span>
                                        </span>
                                    </div>
                                    <div className="flex min-w-0 flex-1 justify-between space-x-4">
                                        <div>
                                            <p className="text-sm text-slate-700 dark:text-slate-300">Subscription created</p>
                                        </div>
                                        <div className="whitespace-nowrap text-right text-sm text-slate-500 dark:text-slate-400">
                                            <time dateTime={subscription.created_at}>{new Date(subscription.created_at).toLocaleDateString()}</time>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </li>
                        <li>
                            <div className="relative pb-2">
                                <div className="relative flex items-center space-x-3">
                                    <div>
                                        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-slate-400 dark:bg-slate-600 ring-4 ring-white dark:ring-slate-900">
                                            <span className="material-symbols-outlined !text-sm text-white">autorenew</span>
                                        </span>
                                    </div>
                                    <div className="flex min-w-0 flex-1 justify-between space-x-4">
                                        <div>
                                            <p className="text-sm text-slate-700 dark:text-slate-300">Next renewal scheduled</p>
                                        </div>
                                        <div className="whitespace-nowrap text-right text-sm text-slate-500 dark:text-slate-400">
                                            <time dateTime={subscription.current_period_end}>{new Date(subscription.current_period_end).toLocaleDateString()}</time>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </li>
                    </ul>
                </div>
            </div>
        </SlideOver>
    )
}

export default SubscriptionDetail
