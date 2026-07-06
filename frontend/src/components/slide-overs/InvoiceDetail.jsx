import React, { useState } from 'react'
import { API_BASE, endpoints } from '../../lib/api'

const EInvoiceStatusBadge = ({ status }) => {
    if (!status || status === 'PENDING') return <span className="text-sm text-slate-400">Pending</span>

    const styles = {
        GENERATED: 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300',
        FAILED: 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300',
        CANCELLED: 'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300',
        NA: 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-500',
    }

    return (
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${styles[status] || styles.NA}`}>
            {status}
        </span>
    )
}

const InvoiceDetail = ({ invoice, isOpen, onClose }) => {
    const [retrying, setRetrying] = useState(false)
    const [cancelling, setCancelling] = useState(false)
    const [showCancelModal, setShowCancelModal] = useState(false)
    const [cancelReason, setCancelReason] = useState('')
    const [cancelCode, setCancelCode] = useState(1)
    const [actionMessage, setActionMessage] = useState(null)

    if (!invoice) return null

    const handleRetry = async () => {
        setRetrying(true)
        setActionMessage(null)
        try {
            await endpoints.retryEInvoice(invoice.id)
            setActionMessage({ type: 'success', text: 'E-invoice retry initiated successfully.' })
        } catch (err) {
            setActionMessage({ type: 'error', text: err?.response?.data?.error?.message || 'Retry failed' })
        } finally {
            setRetrying(false)
        }
    }

    const handleCancel = async () => {
        setCancelling(true)
        setActionMessage(null)
        try {
            await endpoints.cancelEInvoice(invoice.id, { cancel_code: cancelCode, reason: cancelReason })
            setActionMessage({ type: 'success', text: 'E-invoice cancelled successfully.' })
            setShowCancelModal(false)
        } catch (err) {
            setActionMessage({ type: 'error', text: err?.response?.data?.error?.message || 'Cancellation failed' })
        } finally {
            setCancelling(false)
        }
    }

    const hasEInvoice = invoice.e_invoice_status && invoice.e_invoice_status !== 'NA' && invoice.e_invoice_status !== 'PENDING'

    return (
        <div className={`fixed inset-0 overflow-hidden z-50 ${isOpen ? 'pointer-events-auto' : 'pointer-events-none'}`}>
            <div className={`absolute inset-0 bg-slate-500/75 transition-opacity duration-300 ${isOpen ? 'opacity-100' : 'opacity-0'}`} onClick={onClose}></div>
            <div className="fixed inset-y-0 right-0 flex max-w-full pl-10 pointer-events-none">
                <div className={`w-screen max-w-md transform transition ease-in-out duration-300 sm:duration-500 ${isOpen ? 'translate-x-0' : 'translate-x-full'} pointer-events-auto`}>
                    <div className="flex h-full flex-col overflow-y-scroll bg-white dark:bg-slate-900 shadow-xl border-l border-slate-200 dark:border-slate-800">
                        {/* Header */}
                        <div className="px-4 py-6 sm:px-6 bg-slate-50 dark:bg-slate-800/50">
                            <div className="flex items-start justify-between">
                                <h2 className="text-xl font-semibold leading-6 text-slate-900 dark:text-white" id="slide-over-title">Invoice Details</h2>
                                <div className="ml-3 flex h-7 items-center">
                                    <button type="button" className="rounded-md bg-transparent text-slate-400 hover:text-slate-500" onClick={onClose}>
                                        <span className="material-symbols-outlined text-2xl">close</span>
                                    </button>
                                </div>
                            </div>
                            <div className="mt-1 flex gap-2">
                                <p className="text-sm text-slate-500 dark:text-slate-400 font-mono">{invoice.invoice_number}</p>
                                <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${invoice.status === 'paid' ? 'bg-green-100 text-green-800' :
                                        invoice.status === 'open' ? 'bg-blue-100 text-blue-800' : 'bg-red-100 text-red-800'
                                    }`}>
                                    {invoice.status.toUpperCase()}
                                </span>
                            </div>
                        </div>

                        {/* Content */}
                        <div className="relative flex-1 px-4 py-6 sm:px-6">
                            <dl className="space-y-6">
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Amount</dt>
                                    <dd className="mt-1 text-2xl font-bold text-slate-900 dark:text-white">
                                        {new Intl.NumberFormat('en-US', { style: 'currency', currency: invoice.currency }).format(invoice.total / 100)}
                                    </dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Customer ID</dt>
                                    <dd className="mt-1 text-sm text-slate-900 dark:text-white font-mono">{invoice.customer_id}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Created At</dt>
                                    <dd className="mt-1 text-sm text-slate-900 dark:text-white">{new Date(invoice.created_at).toLocaleString()}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-slate-500 dark:text-slate-400">Due Date</dt>
                                    <dd className="mt-1 text-sm text-slate-900 dark:text-white">{new Date(invoice.due_date).toLocaleDateString()}</dd>
                                </div>
                            </dl>

                            {/* E-Invoice Section */}
                            {hasEInvoice && (
                                <div className="mt-8 border-t border-slate-200 dark:border-slate-700 pt-6">
                                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">E-Invoice</h3>
                                    <dl className="space-y-4">
                                        <div className="flex justify-between">
                                            <dt className="text-sm text-slate-500 dark:text-slate-400">Status</dt>
                                            <dd><EInvoiceStatusBadge status={invoice.e_invoice_status} /></dd>
                                        </div>
                                        {invoice.irn && (
                                            <div>
                                                <dt className="text-sm text-slate-500 dark:text-slate-400">IRN</dt>
                                                <dd className="mt-1 text-xs text-slate-900 dark:text-white font-mono break-all">{invoice.irn}</dd>
                                            </div>
                                        )}
                                        {invoice.ack_no && (
                                            <div className="flex justify-between">
                                                <dt className="text-sm text-slate-500 dark:text-slate-400">Ack No</dt>
                                                <dd className="text-sm text-slate-900 dark:text-white font-mono">{invoice.ack_no}</dd>
                                            </div>
                                        )}
                                        {invoice.ack_date && (
                                            <div className="flex justify-between">
                                                <dt className="text-sm text-slate-500 dark:text-slate-400">Ack Date</dt>
                                                <dd className="text-sm text-slate-900 dark:text-white">{invoice.ack_date}</dd>
                                            </div>
                                        )}
                                        {invoice.e_invoice_error_message && (
                                            <div>
                                                <dt className="text-sm text-slate-500 dark:text-slate-400">Error</dt>
                                                <dd className="mt-1 text-sm text-red-600 dark:text-red-400">{invoice.e_invoice_error_message}</dd>
                                            </div>
                                        )}
                                    </dl>

                                    {/* Action Message */}
                                    {actionMessage && (
                                        <div className={`mt-4 rounded-lg px-3 py-2 text-sm ${actionMessage.type === 'success'
                                                ? 'bg-green-50 text-green-800 dark:bg-green-900/20 dark:text-green-300'
                                                : 'bg-red-50 text-red-800 dark:bg-red-900/20 dark:text-red-300'
                                            }`}>
                                            {actionMessage.text}
                                        </div>
                                    )}

                                    {/* E-Invoice Actions */}
                                    <div className="mt-4 flex gap-3">
                                        {invoice.e_invoice_status === 'FAILED' && (
                                            <button
                                                onClick={handleRetry}
                                                disabled={retrying}
                                                className="flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                                            >
                                                <span className="material-symbols-outlined text-base">refresh</span>
                                                {retrying ? 'Retrying...' : 'Retry'}
                                            </button>
                                        )}
                                        {invoice.e_invoice_status === 'GENERATED' && (
                                            <button
                                                onClick={() => setShowCancelModal(true)}
                                                className="flex items-center gap-1.5 rounded-lg bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700"
                                            >
                                                <span className="material-symbols-outlined text-base">cancel</span>
                                                Cancel IRN
                                            </button>
                                        )}
                                    </div>
                                </div>
                            )}

                            {/* Cancel IRN Modal */}
                            {showCancelModal && (
                                <div className="mt-4 rounded-lg border border-slate-200 dark:border-slate-700 p-4 bg-slate-50 dark:bg-slate-800/50">
                                    <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-3">Cancel IRN</h4>
                                    <div className="space-y-3">
                                        <div>
                                            <label className="block text-xs text-slate-500 dark:text-slate-400 mb-1">Cancel Reason</label>
                                            <select
                                                value={cancelCode}
                                                onChange={(e) => setCancelCode(Number(e.target.value))}
                                                className="w-full rounded-md border-slate-300 dark:border-slate-600 dark:bg-slate-700 dark:text-white text-sm"
                                            >
                                                <option value={1}>Duplicate</option>
                                                <option value={2}>Data Entry Mistake</option>
                                                <option value={3}>Order Cancelled</option>
                                                <option value={4}>Others</option>
                                            </select>
                                        </div>
                                        <div>
                                            <label className="block text-xs text-slate-500 dark:text-slate-400 mb-1">Remarks</label>
                                            <input
                                                type="text"
                                                value={cancelReason}
                                                onChange={(e) => setCancelReason(e.target.value)}
                                                placeholder="Enter reason for cancellation"
                                                className="w-full rounded-md border-slate-300 dark:border-slate-600 dark:bg-slate-700 dark:text-white text-sm"
                                            />
                                        </div>
                                        <div className="flex gap-2">
                                            <button
                                                onClick={handleCancel}
                                                disabled={cancelling || !cancelReason}
                                                className="rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
                                            >
                                                {cancelling ? 'Cancelling...' : 'Confirm Cancel'}
                                            </button>
                                            <button
                                                onClick={() => setShowCancelModal(false)}
                                                className="rounded-md bg-slate-200 dark:bg-slate-700 px-3 py-1.5 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-300 dark:hover:bg-slate-600"
                                            >
                                                Close
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* Actions */}
                            <div className="mt-8 flex gap-3">
                                <a
                                    href={`${API_BASE}/invoices/${invoice.id}/pdf`}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="flex-1 flex justify-center items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-primary/90"
                                >
                                    <span className="material-symbols-outlined text-lg">description</span>
                                    Download PDF
                                </a>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default InvoiceDetail
