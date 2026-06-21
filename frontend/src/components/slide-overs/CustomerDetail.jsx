import React from 'react'
import CustomerRiskCard from '../CustomerRiskCard'

const CustomerDetail = ({ customer, isOpen, onClose }) => {
    if (!customer) return null

    return (
        <div className={`fixed inset-0 overflow-hidden z-50 ${isOpen ? 'pointer-events-auto' : 'pointer-events-none'}`}>
            <div className={`absolute inset-0 bg-slate-500/75 transition-opacity duration-300 ${isOpen ? 'opacity-100' : 'opacity-0'}`} onClick={onClose}></div>
            <div className="fixed inset-y-0 right-0 flex max-w-full pl-10 pointer-events-none">
                <div className={`w-screen max-w-md transform transition ease-in-out duration-300 sm:duration-500 ${isOpen ? 'translate-x-0' : 'translate-x-full'} pointer-events-auto`}>
                    <div className="flex h-full flex-col overflow-y-scroll bg-white dark:bg-slate-900 shadow-xl border-l border-slate-200 dark:border-slate-800">
                        {/* Header */}
                        <div className="px-4 py-6 sm:px-6 bg-slate-50 dark:bg-slate-800/50">
                            <div className="flex items-start justify-between">
                                <h2 className="text-xl font-semibold leading-6 text-slate-900 dark:text-white" id="slide-over-title">Customer Details</h2>
                                <div className="ml-3 flex h-7 items-center">
                                    <button
                                        type="button"
                                        className="rounded-md bg-transparent text-slate-400 hover:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary"
                                        onClick={onClose}
                                    >
                                        <span className="sr-only">Close panel</span>
                                        <span className="material-symbols-outlined text-2xl">close</span>
                                    </button>
                                </div>
                            </div>
                            <div className="mt-1">
                                <p className="text-sm text-slate-500 dark:text-slate-400">ID: {customer.id}</p>
                            </div>
                        </div>

                        {/* Content */}
                        <div className="relative flex-1 px-4 py-6 sm:px-6">
                            <div className="mb-6">
                                <CustomerRiskCard customer={customer} />
                            </div>

                            <dl className="space-y-6">
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Name</dt>
                                    <dd className="mt-1 text-base font-semibold text-slate-900 dark:text-white">{customer.name}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Email</dt>
                                    <dd className="mt-1 text-sm text-slate-900 dark:text-white font-mono">{customer.email}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Joined</dt>
                                    <dd className="mt-1 text-sm text-slate-900 dark:text-white">{new Date(customer.created_at).toLocaleString()}</dd>
                                </div>
                                <div className="border-t border-slate-200 dark:border-slate-800 pt-6">
                                    <h3 className="text-sm font-medium text-slate-900 dark:text-white">Active Subscriptions</h3>
                                    <div className="mt-2">
                                        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${customer.activeSubs > 0 ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300' : 'bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-300'}`}>
                                            {customer.activeSubs} Active
                                        </span>
                                    </div>
                                </div>
                            </dl>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default CustomerDetail
