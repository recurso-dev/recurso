import React from 'react'

const QuoteDetail = ({ quote, isOpen, onClose }) => {
    if (!quote) return null

    return (
        <div className={`fixed inset-0 overflow-hidden z-50 ${isOpen ? 'pointer-events-auto' : 'pointer-events-none'}`}>
            <div className={`absolute inset-0 bg-slate-500/75 transition-opacity duration-300 ${isOpen ? 'opacity-100' : 'opacity-0'}`} onClick={onClose}></div>
            <div className="fixed inset-y-0 right-0 flex max-w-full pl-10 pointer-events-none">
                <div className={`w-screen max-w-md transform transition ease-in-out duration-300 sm:duration-500 ${isOpen ? 'translate-x-0' : 'translate-x-full'} pointer-events-auto`}>
                    <div className="flex h-full flex-col overflow-y-scroll bg-white dark:bg-slate-900 shadow-xl border-l border-slate-200 dark:border-slate-800">
                        {/* Header */}
                        <div className="px-4 py-6 sm:px-6 bg-slate-50 dark:bg-slate-800/50">
                            <div className="flex items-start justify-between">
                                <h2 className="text-xl font-semibold leading-6 text-slate-900 dark:text-white" id="slide-over-title">Quote Details</h2>
                                <div className="ml-3 flex h-7 items-center">
                                    <button type="button" className="rounded-md bg-transparent text-slate-400 hover:text-slate-500" onClick={onClose}>
                                        <span className="material-symbols-outlined text-2xl">close</span>
                                    </button>
                                </div>
                            </div>
                            <div className="mt-1">
                                <p className="text-sm text-slate-500 dark:text-slate-400 font-mono">ID: {quote.id}</p>
                            </div>
                        </div>

                        {/* Content */}
                        <div className="relative flex-1 px-4 py-6 sm:px-6">
                            <dl className="space-y-6">
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Total Amount</dt>
                                    <dd className="mt-1 text-2xl font-bold text-slate-900 dark:text-white">
                                        {new Intl.NumberFormat('en-US', { style: 'currency', currency: quote.currency }).format(quote.total_amount / 100)}
                                    </dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Customer ID</dt>
                                    <dd className="mt-1 text-sm text-slate-900 dark:text-white font-mono">{quote.customer_id}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Valid Until</dt>
                                    <dd className="mt-1 text-sm text-slate-900 dark:text-white">{new Date(quote.valid_until).toLocaleDateString()}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Status</dt>
                                    <dd className="mt-1">
                                        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${quote.status === 'accepted' ? 'bg-green-100 text-green-800' :
                                                quote.status === 'draft' ? 'bg-gray-100 text-gray-800' :
                                                    'bg-blue-100 text-blue-800'
                                            }`}>
                                            {quote.status.toUpperCase()}
                                        </span>
                                    </dd>
                                </div>
                            </dl>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default QuoteDetail
