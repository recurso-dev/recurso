import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { endpoints } from '../lib/api'
import { ArrowLeft, Plus, Trash2 } from 'lucide-react'

const CreateQuote = () => {
    const navigate = useNavigate()
    const [loading, setLoading] = useState(false)
    const [customers, setCustomers] = useState([])
    const [error, setError] = useState(null)

    const [formData, setFormData] = useState({
        customer_id: '',
        currency: 'USD',
        notes: '',
        terms: 'Payment due within 30 days of acceptance.',
        tax_amount: 0,
        discount_amount: 0,
        valid_until: '',
        line_items: [
            { description: '', quantity: 1, unit_price: 0 }
        ]
    })

    useEffect(() => {
        fetchCustomers()
    }, [])

    const fetchCustomers = async () => {
        try {
            const response = await endpoints.getCustomers()
            setCustomers(response.data.data || [])
        } catch (err) {
            console.error('Failed to fetch customers:', err)
        }
    }

    const handleChange = (e) => {
        const { name, value } = e.target
        setFormData(prev => ({ ...prev, [name]: value }))
    }

    const handleLineItemChange = (index, field, value) => {
        const newItems = [...formData.line_items]
        newItems[index] = { ...newItems[index], [field]: value }
        setFormData(prev => ({ ...prev, line_items: newItems }))
    }

    const addLineItem = () => {
        setFormData(prev => ({
            ...prev,
            line_items: [...prev.line_items, { description: '', quantity: 1, unit_price: 0 }]
        }))
    }

    const removeLineItem = (index) => {
        if (formData.line_items.length > 1) {
            setFormData(prev => ({
                ...prev,
                line_items: prev.line_items.filter((_, i) => i !== index)
            }))
        }
    }

    const calculateSubtotal = () => {
        return formData.line_items.reduce((sum, item) =>
            sum + (item.quantity * item.unit_price), 0
        )
    }

    const calculateTotal = () => {
        const subtotal = calculateSubtotal()
        return subtotal + formData.tax_amount - formData.discount_amount
    }

    const formatCurrency = (amount) => {
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: formData.currency
        }).format(amount / 100)
    }

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)
        setError(null)

        try {
            const payload = {
                customer_id: formData.customer_id,
                currency: formData.currency,
                notes: formData.notes,
                terms: formData.terms,
                tax_amount: parseInt(formData.tax_amount) || 0,
                discount_amount: parseInt(formData.discount_amount) || 0,
                valid_until: formData.valid_until ? new Date(formData.valid_until).toISOString() : null,
                line_items: formData.line_items.map(item => ({
                    description: item.description,
                    quantity: parseInt(item.quantity) || 1,
                    unit_price: parseInt(item.unit_price) || 0,
                    amount: (parseInt(item.quantity) || 1) * (parseInt(item.unit_price) || 0)
                }))
            }

            await endpoints.createQuote(payload)
            navigate('/quotes')
        } catch (err) {
            setError(err.response?.data?.error?.message || 'Failed to create quote')
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="max-w-4xl mx-auto animate-fade-in">
            {/* Header */}
            <div className="flex items-center gap-4 mb-8">
                <button
                    onClick={() => navigate('/quotes')}
                    className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                >
                    <ArrowLeft className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                </button>
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Create Quote</h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        Create a new quote for a customer
                    </p>
                </div>
            </div>

            {error && (
                <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded-lg">
                    {error}
                </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-6">
                {/* Customer & Settings */}
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">Quote Details</h2>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Customer *
                            </label>
                            <select
                                name="customer_id"
                                value={formData.customer_id}
                                onChange={handleChange}
                                required
                                className="w-full px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                            >
                                <option value="">Select customer</option>
                                {customers.map(customer => (
                                    <option key={customer.id} value={customer.id}>
                                        {customer.name} ({customer.email})
                                    </option>
                                ))}
                            </select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Currency
                            </label>
                            <select
                                name="currency"
                                value={formData.currency}
                                onChange={handleChange}
                                className="w-full px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                            >
                                <option value="USD">USD</option>
                                <option value="EUR">EUR</option>
                                <option value="GBP">GBP</option>
                                <option value="INR">INR</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Valid Until
                            </label>
                            <input
                                type="date"
                                name="valid_until"
                                value={formData.valid_until}
                                onChange={handleChange}
                                className="w-full px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                            />
                        </div>
                    </div>
                </div>

                {/* Line Items */}
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <div className="flex items-center justify-between mb-4">
                        <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Line Items</h2>
                        <button
                            type="button"
                            onClick={addLineItem}
                            className="inline-flex items-center gap-1 px-3 py-1 text-sm text-primary hover:bg-primary/10 rounded-lg transition-colors"
                        >
                            <Plus className="w-4 h-4" />
                            Add Item
                        </button>
                    </div>

                    <div className="space-y-4">
                        {formData.line_items.map((item, index) => (
                            <div key={index} className="grid grid-cols-12 gap-4 items-end p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                                <div className="col-span-12 md:col-span-5">
                                    <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                                        Description
                                    </label>
                                    <input
                                        type="text"
                                        value={item.description}
                                        onChange={(e) => handleLineItemChange(index, 'description', e.target.value)}
                                        placeholder="Item description"
                                        required
                                        className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-sm"
                                    />
                                </div>
                                <div className="col-span-4 md:col-span-2">
                                    <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                                        Qty
                                    </label>
                                    <input
                                        type="number"
                                        value={item.quantity}
                                        onChange={(e) => handleLineItemChange(index, 'quantity', parseInt(e.target.value) || 1)}
                                        min="1"
                                        required
                                        className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-sm"
                                    />
                                </div>
                                <div className="col-span-4 md:col-span-2">
                                    <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                                        Unit Price (¢)
                                    </label>
                                    <input
                                        type="number"
                                        value={item.unit_price}
                                        onChange={(e) => handleLineItemChange(index, 'unit_price', parseInt(e.target.value) || 0)}
                                        min="0"
                                        required
                                        className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-sm"
                                    />
                                </div>
                                <div className="col-span-3 md:col-span-2 text-right">
                                    <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                                        Amount
                                    </label>
                                    <p className="py-2 text-sm font-medium text-slate-900 dark:text-white">
                                        {formatCurrency(item.quantity * item.unit_price)}
                                    </p>
                                </div>
                                <div className="col-span-1">
                                    <button
                                        type="button"
                                        onClick={() => removeLineItem(index)}
                                        disabled={formData.line_items.length === 1}
                                        className="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30 rounded-lg disabled:opacity-50"
                                    >
                                        <Trash2 className="w-4 h-4" />
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>

                    {/* Totals */}
                    <div className="mt-6 pt-6 border-t border-slate-200 dark:border-slate-700">
                        <div className="space-y-2 text-right">
                            <div className="flex justify-end gap-8">
                                <span className="text-sm text-slate-500">Subtotal:</span>
                                <span className="text-sm font-medium text-slate-900 dark:text-white w-24">
                                    {formatCurrency(calculateSubtotal())}
                                </span>
                            </div>
                            <div className="flex justify-end items-center gap-4">
                                <span className="text-sm text-slate-500">Tax (¢):</span>
                                <input
                                    type="number"
                                    name="tax_amount"
                                    value={formData.tax_amount}
                                    onChange={handleChange}
                                    className="w-24 px-2 py-1 text-right rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-sm"
                                />
                            </div>
                            <div className="flex justify-end items-center gap-4">
                                <span className="text-sm text-slate-500">Discount (¢):</span>
                                <input
                                    type="number"
                                    name="discount_amount"
                                    value={formData.discount_amount}
                                    onChange={handleChange}
                                    className="w-24 px-2 py-1 text-right rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-sm"
                                />
                            </div>
                            <div className="flex justify-end gap-8 pt-2 border-t border-slate-200 dark:border-slate-700">
                                <span className="text-base font-semibold text-slate-900 dark:text-white">Total:</span>
                                <span className="text-base font-bold text-primary w-24">
                                    {formatCurrency(calculateTotal())}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Notes & Terms */}
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">Notes & Terms</h2>

                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Notes (visible to customer)
                            </label>
                            <textarea
                                name="notes"
                                value={formData.notes}
                                onChange={handleChange}
                                rows={3}
                                placeholder="Additional notes for the customer..."
                                className="w-full px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Terms & Conditions
                            </label>
                            <textarea
                                name="terms"
                                value={formData.terms}
                                onChange={handleChange}
                                rows={3}
                                className="w-full px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                            />
                        </div>
                    </div>
                </div>

                {/* Actions */}
                <div className="flex justify-end gap-4">
                    <button
                        type="button"
                        onClick={() => navigate('/quotes')}
                        className="px-6 py-2 border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        disabled={loading}
                        className="px-6 py-2 bg-primary text-white font-medium rounded-lg hover:bg-primary/90 disabled:opacity-50 transition-all"
                    >
                        {loading ? 'Creating...' : 'Create Quote'}
                    </button>
                </div>
            </form>
        </div>
    )
}

export default CreateQuote
