import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { endpoints } from '../lib/api'

const CreateCreditNote = () => {
    const navigate = useNavigate()
    const [customers, setCustomers] = useState([])
    const [loading, setLoading] = useState(false)
    const [formData, setFormData] = useState({
        customer_id: '',
        amount: '',
        currency: 'USD',
        reason: '',
        invoice_id: '' // Optional
    })

    useEffect(() => {
        const fetchCustomers = async () => {
            try {
                const res = await endpoints.getCustomers()
                setCustomers(res.data.data || [])
            } catch (err) {
                console.error("Failed to load customers", err)
            }
        }
        fetchCustomers()
    }, [])

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)

        try {
            // Convert amount to cents
            const payload = {
                ...formData,
                amount: Math.round(parseFloat(formData.amount) * 100),
                invoice_id: formData.invoice_id ? formData.invoice_id : null
            }
            if (!payload.invoice_id) delete payload.invoice_id

            await endpoints.createCreditNote(payload)
            navigate('/credit-notes')
        } catch (err) {
            console.error(err)
            alert("Failed to create credit note")
        } finally {
            setLoading(false)
        }
    }

    const handleChange = (e) => {
        setFormData({ ...formData, [e.target.name]: e.target.value })
    }

    return (
        <div className="flex flex-col items-center justify-center min-h-[500px]">
            <div className="w-full max-w-2xl bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-8 shadow-sm">
                <div className="mb-8">
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Create New Credit Note</h1>
                    <p className="text-slate-500 dark:text-slate-400 mt-1">Issue a credit to a customer that can be applied to an invoice.</p>
                </div>

                <form onSubmit={handleSubmit} className="space-y-6">
                    {/* Customer Selection */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Customer</label>
                        <select
                            name="customer_id"
                            required
                            value={formData.customer_id}
                            onChange={handleChange}
                            className="w-full rounded-lg border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-950 text-slate-900 dark:text-white h-11 px-3"
                        >
                            <option value="">Select a customer...</option>
                            {customers.map(c => (
                                <option key={c.id} value={c.id}>{c.name} ({c.email})</option>
                            ))}
                        </select>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        {/* Amount */}
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Credit Amount</label>
                            <div className="relative">
                                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500">USD</span>
                                <input
                                    type="number"
                                    step="0.01"
                                    name="amount"
                                    required
                                    value={formData.amount}
                                    onChange={handleChange}
                                    placeholder="0.00"
                                    className="w-full rounded-lg border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-950 text-slate-900 dark:text-white h-11 pl-12 pr-3"
                                />
                            </div>
                        </div>

                        {/* Invoice Link (Optional) */}
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Linked Invoice (Optional)</label>
                            <input
                                type="text"
                                name="invoice_id"
                                value={formData.invoice_id}
                                onChange={handleChange}
                                placeholder="Invoice ID (UUID)..."
                                className="w-full rounded-lg border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-950 text-slate-900 dark:text-white h-11 px-3"
                            />
                        </div>
                    </div>

                    {/* Reason */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Reason for Credit</label>
                        <textarea
                            name="reason"
                            rows="4"
                            value={formData.reason}
                            onChange={handleChange}
                            placeholder="e.g. Service downtime compensation"
                            className="w-full rounded-lg border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-950 text-slate-900 dark:text-white p-3"
                        ></textarea>
                    </div>

                    <div className="pt-4 flex gap-4 justify-end border-t border-slate-200 dark:border-slate-800 mt-8">
                        <button
                            type="button"
                            onClick={() => navigate('/credit-notes')}
                            className="px-6 py-2.5 rounded-lg border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 font-medium transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={loading}
                            className="px-6 py-2.5 rounded-lg bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                        >
                            {loading ? 'Issuing...' : 'Issue Credit Note'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    )
}

export default CreateCreditNote
