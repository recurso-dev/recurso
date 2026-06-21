import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { endpoints } from '../lib/api'

const CreateCoupon = () => {
    const navigate = useNavigate()
    const [loading, setLoading] = useState(false)
    const [formData, setFormData] = useState({
        code: '',
        discount_type: 'percent',
        discount_value: '',
        duration: 'once',
        duration_months: '',
        max_redemptions: '',
        active: true
    })

    const handleChange = (e) => {
        const { name, value, type, checked } = e.target
        setFormData(prev => ({
            ...prev,
            [name]: type === 'checkbox' ? checked : value
        }))
    }

    const generateCode = () => {
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
        let result = ''
        for (let i = 0; i < 10; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length))
        }
        setFormData(prev => ({ ...prev, code: result }))
    }

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)

        // Map frontend fields to backend expected format
        const payload = {
            code: formData.code,
            discount_type: formData.discount_type.toLowerCase().includes('percent') ? 'percent' : 'amount',
            discount_value: parseInt(formData.discount_value),
            duration: formData.duration.toLowerCase(),
            duration_months: formData.duration === 'repeating' && formData.duration_months ? parseInt(formData.duration_months) : null
        }

        try {
            await endpoints.createCoupon(payload)
            navigate('/coupons')
        } catch (error) {
            console.error("Failed to create coupon:", error)
            alert("Failed to create coupon")
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="relative z-50" aria-labelledby="slide-over-title" role="dialog" aria-modal="true">
            {/* Background backdrop */}
            <div className="fixed inset-0 bg-slate-900/60 dark:bg-slate-900/80 transition-opacity" aria-hidden="true"></div>

            <div className="fixed inset-0 overflow-hidden">
                <div className="absolute inset-0 overflow-hidden">
                    <div className="pointer-events-none fixed inset-y-0 right-0 flex max-w-full pl-10">
                        {/* Slide-over panel */}
                        <div className="pointer-events-auto relative w-screen max-w-2xl">
                            <div className="flex h-full flex-col bg-white dark:bg-slate-900 shadow-xl">
                                <div className="flex flex-col h-full">
                                    <header className="flex h-16 flex-shrink-0 items-center justify-between border-b border-slate-200 dark:border-slate-800 px-6">
                                        <h1 className="text-lg font-medium text-slate-900 dark:text-slate-100">Create New Coupon</h1>
                                        <button
                                            onClick={() => navigate('/coupons')}
                                            className="flex items-center justify-center rounded-lg p-1 text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-700 dark:hover:text-slate-200 focus:outline-none focus:ring-2 focus:ring-primary"
                                        >
                                            <span className="material-symbols-outlined text-2xl">close</span>
                                        </button>
                                    </header>

                                    <main className="flex-1 overflow-y-auto p-8">
                                        <form id="create-coupon-form" onSubmit={handleSubmit} className="space-y-8">
                                            <div className="grid grid-cols-1 gap-6">
                                                <label className="flex flex-col">
                                                    <p className="text-sm font-medium leading-6 text-slate-900 dark:text-slate-200 pb-2">Coupon Code</p>
                                                    <div className="relative flex w-full flex-1 items-stretch rounded-lg">
                                                        <input
                                                            required
                                                            name="code"
                                                            className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-l-lg border-slate-300 bg-white dark:border-slate-700 dark:bg-slate-900 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 dark:placeholder:text-slate-500 focus:border-primary focus:ring-primary h-11 px-3 text-sm transition-all"
                                                            placeholder="e.g. SUMMER25OFF"
                                                            value={formData.code}
                                                            onChange={handleChange}
                                                        />
                                                        <button
                                                            type="button"
                                                            onClick={generateCode}
                                                            className="inline-flex items-center gap-2 whitespace-nowrap border border-l-0 border-slate-300 bg-slate-50 dark:border-slate-700 dark:bg-slate-800 px-4 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-r-lg focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:focus:ring-offset-slate-900 transition-all"
                                                        >
                                                            <span className="material-symbols-outlined text-base">auto_awesome</span>
                                                            Generate
                                                        </button>
                                                    </div>
                                                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-2">Customers will enter this code at checkout.</p>
                                                </label>
                                            </div>

                                            <div className="grid grid-cols-2 gap-6">
                                                <label className="flex flex-col">
                                                    <p className="text-sm font-medium leading-6 text-slate-900 dark:text-slate-200 pb-2">Discount Type</p>
                                                    <select
                                                        name="discount_type"
                                                        className="form-select flex w-full min-w-0 flex-1 overflow-hidden rounded-lg border-slate-300 bg-white dark:border-slate-700 dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:border-primary focus:ring-primary h-11 px-3 text-sm transition-all"
                                                        value={formData.discount_type}
                                                        onChange={handleChange}
                                                    >
                                                        <option value="percent">Percent Off</option>
                                                        <option value="amount">Amount Off</option>
                                                    </select>
                                                </label>
                                                <label className="flex flex-col">
                                                    <p className="text-sm font-medium leading-6 text-slate-900 dark:text-slate-200 pb-2">Discount Value</p>
                                                    <div className="relative rounded-lg">
                                                        <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                                                            <span className="text-slate-500 dark:text-slate-400 sm:text-sm">
                                                                {formData.discount_type === 'percent' ? '%' : '$'}
                                                            </span>
                                                        </div>
                                                        <input
                                                            type="number"
                                                            name="discount_value"
                                                            min="1"
                                                            required
                                                            className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border-slate-300 bg-white dark:border-slate-700 dark:bg-slate-900 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 dark:placeholder:text-slate-500 focus:border-primary focus:ring-primary h-11 pl-7 pr-3 text-sm transition-all"
                                                            placeholder="25"
                                                            value={formData.discount_value}
                                                            onChange={handleChange}
                                                        />
                                                    </div>
                                                </label>
                                            </div>

                                            <div className="grid grid-cols-2 gap-6">
                                                <label className="flex flex-col">
                                                    <p className="text-sm font-medium leading-6 text-slate-900 dark:text-slate-200 pb-2">Duration</p>
                                                    <select
                                                        name="duration"
                                                        className="form-select flex w-full min-w-0 flex-1 overflow-hidden rounded-lg border-slate-300 bg-white dark:border-slate-700 dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus:border-primary focus:ring-primary h-11 px-3 text-sm transition-all"
                                                        value={formData.duration}
                                                        onChange={handleChange}
                                                    >
                                                        <option value="forever">Forever</option>
                                                        <option value="once">Once</option>
                                                        <option value="repeating">Limited Time (Repeating)</option>
                                                    </select>
                                                </label>
                                                {formData.duration === 'repeating' && (
                                                    <label className="flex flex-col">
                                                        <p className="text-sm font-medium leading-6 text-slate-900 dark:text-slate-200 pb-2">Duration in Months</p>
                                                        <input
                                                            type="number"
                                                            name="duration_months"
                                                            min="1"
                                                            required
                                                            className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border-slate-300 bg-white dark:border-slate-700 dark:bg-slate-900 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 dark:placeholder:text-slate-500 focus:border-primary focus:ring-primary h-11 px-3 text-sm transition-all"
                                                            placeholder="e.g. 12"
                                                            value={formData.duration_months}
                                                            onChange={handleChange}
                                                        />
                                                    </label>
                                                )}
                                            </div>

                                            <div className="grid grid-cols-1 gap-6">
                                                <label className="flex flex-col">
                                                    <p className="text-sm font-medium leading-6 text-slate-900 dark:text-slate-200 pb-2">Max Redemptions (optional)</p>
                                                    <input
                                                        type="number"
                                                        name="max_redemptions"
                                                        className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border-slate-300 bg-white dark:border-slate-700 dark:bg-slate-900 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 dark:placeholder:text-slate-500 focus:border-primary focus:ring-primary h-11 px-3 text-sm transition-all"
                                                        placeholder="Enter max redemptions"
                                                        value={formData.max_redemptions}
                                                        onChange={handleChange}
                                                    />
                                                </label>
                                            </div>

                                            <div className="flex items-center justify-between">
                                                <div className="flex flex-col">
                                                    <p className="text-sm font-medium text-slate-900 dark:text-slate-200">Status</p>
                                                    <p className="text-xs text-slate-500 dark:text-slate-400">Set the coupon as active or inactive.</p>
                                                </div>
                                                <button
                                                    type="button"
                                                    role="switch"
                                                    aria-checked={formData.active}
                                                    onClick={() => setFormData(p => ({ ...p, active: !p.active }))}
                                                    className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:focus:ring-offset-slate-900 ${formData.active ? 'bg-primary' : 'bg-slate-200 dark:bg-slate-700'}`}
                                                >
                                                    <span
                                                        aria-hidden="true"
                                                        className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${formData.active ? 'translate-x-5' : 'translate-x-0'}`}
                                                    ></span>
                                                </button>
                                            </div>
                                        </form>
                                    </main>

                                    <footer className="flex flex-shrink-0 items-center justify-end gap-3 border-t border-slate-200 dark:border-slate-800 px-6 py-4">
                                        <button
                                            type="button"
                                            onClick={() => navigate('/coupons')}
                                            className="inline-flex items-center justify-center rounded-lg bg-white dark:bg-slate-800 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:focus:ring-offset-slate-900 transition-all"
                                        >
                                            Cancel
                                        </button>
                                        <button
                                            form="create-coupon-form"
                                            type="submit"
                                            disabled={loading}
                                            className="inline-flex items-center justify-center rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:focus:ring-offset-slate-900 transition-all disabled:opacity-50"
                                        >
                                            {loading ? 'Creating...' : 'Create Coupon'}
                                        </button>
                                    </footer>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default CreateCoupon
