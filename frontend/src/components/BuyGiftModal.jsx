import React, { useState } from 'react'
import Modal from './Modal'
import { endpoints } from '../lib/api'

const BuyGiftModal = ({ isOpen, onClose, plans, onSuccess }) => {
    const [formData, setFormData] = useState({
        buyer_customer_id: '', // Need a way to select buyer? Or maybe create a new buyer?
        // Ideally, in Admin, we pick an existing customer acting as the "Buyer".
        plan_id: plans[0]?.id || '',
        duration_months: 12
    })
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState(null)
    const [giftCode, setGiftCode] = useState(null)

    // Helper to filter valid plans
    const validPlans = plans

    const handleChange = (e) => {
        const { name, value } = e.target
        setFormData(prev => ({ ...prev, [name]: value }))
    }

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)
        setError(null)
        setGiftCode(null)

        try {
            // NOTE: We need a valid buyer_customer_id. 
            // For this UI, we might need a Customer Search/Select component.
            // For MVP simplicity, we might ask for a UUID string, which is not user-friendly. 
            // Or we assume the Merchant IS the buyer (Internal Gift)?
            // Let's ask for Customer ID string for now.
            const payload = {
                buyer_customer_id: formData.buyer_customer_id,
                plan_id: formData.plan_id,
                duration_months: parseInt(formData.duration_months)
            }

            const response = await endpoints.createGift(payload)
            // We need to add createGift to api.js

            setGiftCode(response.data.code)
            if (onSuccess) onSuccess()
        } catch (err) {
            setError(err.response?.data?.error || err.message)
        } finally {
            setLoading(false)
        }
    }

    const handleCopy = () => {
        if (giftCode) {
            navigator.clipboard.writeText(giftCode)
        }
    }

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Purchase Gift Subscription">
            {giftCode ? (
                <div className="space-y-4 text-center">
                    <div className="mx-auto w-16 h-16 bg-green-100 text-green-600 rounded-full flex items-center justify-center">
                        <span className="material-symbols-outlined text-3xl">check_circle</span>
                    </div>
                    <div>
                        <h3 className="text-lg font-bold text-slate-900 dark:text-white">Gift Created!</h3>
                        <p className="text-sm text-slate-600 dark:text-slate-400">Share this code with the recipient.</p>
                    </div>

                    <div className="flex items-center gap-2 bg-slate-100 dark:bg-slate-800 p-3 rounded-lg border border-slate-200 dark:border-slate-700">
                        <code className="flex-1 font-mono text-lg font-bold text-center text-slate-900 dark:text-white tracking-wider">
                            {giftCode}
                        </code>
                        <button onClick={handleCopy} className="p-2 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-md">
                            <span className="material-symbols-outlined text-slate-500">content_copy</span>
                        </button>
                    </div>

                    <button
                        onClick={onClose}
                        className="w-full bg-slate-900 text-white rounded-lg py-2.5 font-semibold hover:bg-slate-800 dark:bg-white dark:text-slate-900 dark:hover:bg-slate-100"
                    >
                        Done
                    </button>
                </div>
            ) : (
                <form onSubmit={handleSubmit} className="space-y-4">
                    {error && (
                        <div className="p-3 text-sm bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400 rounded-lg">
                            {error}
                        </div>
                    )}

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Buyer Customer ID</label>
                        <input
                            type="text"
                            name="buyer_customer_id"
                            value={formData.buyer_customer_id}
                            onChange={handleChange}
                            placeholder="UUID of the buyer"
                            required
                            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm placeholder-slate-400 shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white dark:placeholder-slate-500"
                        />
                        <p className="text-xs text-slate-500 mt-1">Found in Customers list details.</p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Plan</label>
                        <select
                            name="plan_id"
                            value={formData.plan_id}
                            onChange={handleChange}
                            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white"
                        >
                            {validPlans.map(plan => (
                                <option key={plan.id} value={plan.id}>
                                    {plan.name} - {(plan.amount / 100).toFixed(2)} {plan.currency}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Duration (Months)</label>
                        <input
                            type="number"
                            name="duration_months"
                            value={formData.duration_months}
                            onChange={handleChange}
                            min="1"
                            required
                            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm placeholder-slate-400 shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white dark:placeholder-slate-500"
                        />
                    </div>

                    <div className="flex justify-end gap-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={loading}
                            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary/90 disabled:opacity-50"
                        >
                            {loading ? 'Purchasing...' : 'Purchase Gift'}
                        </button>
                    </div>
                </form>
            )}
        </Modal>
    )
}

export default BuyGiftModal
