import React, { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { endpoints } from '../lib/api'
import { useToast } from '../components/Toast'

const CreatePlan = () => {
    const navigate = useNavigate()
    const toast = useToast()
    const [formData, setFormData] = useState({
        name: '',
        code: '',
        description: '',
        price: 99,
        currency: 'USD',
        interval: 'month'
    })
    const [loading, setLoading] = useState(false)

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)
        try {
            const payload = {
                ...formData,
                price: Math.round(parseFloat(formData.price) * 100) // Convert to cents
            }
            await endpoints.createPlan(payload)
            navigate('/plans')
        } catch (error) {
            console.error("Failed to create plan:", error)
            toast.error(error?.response?.data?.error?.message || "Failed to create plan")
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
            {/* Breadcrumbs */}
            <div className="flex flex-wrap gap-2 py-4">
                <Link to="/plans" className="text-primary text-sm font-medium leading-normal hover:underline">Plans</Link>
                <span className="text-slate-500 dark:text-slate-400 text-sm font-medium leading-normal">/</span>
                <span className="text-slate-900 dark:text-white text-sm font-medium leading-normal">Create New Plan</span>
            </div>

            {/* Page Heading */}
            <div className="flex flex-wrap justify-between gap-3 pb-8">
                <div className="flex min-w-72 flex-col gap-2">
                    <h1 className="text-slate-900 dark:text-white text-3xl font-bold leading-tight tracking-tight">Create a new plan</h1>
                    <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal">Configure the details for your new subscription plan.</p>
                </div>
            </div>

            <form onSubmit={handleSubmit} className="flex flex-col gap-8">
                {/* Plan Details Section */}
                <div className="rounded-xl border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900 shadow-sm">
                    <div className="border-b border-slate-200 p-5 dark:border-slate-800">
                        <h2 className="text-slate-900 dark:text-white text-lg font-semibold leading-tight">Plan Details</h2>
                    </div>
                    <div className="p-5 space-y-6">
                        <div className="flex flex-col gap-4 sm:flex-row">
                            <label className="flex flex-col min-w-40 flex-1">
                                <p className="text-slate-900 dark:text-white text-sm font-medium leading-normal pb-2">Plan Name</p>
                                <input
                                    required
                                    className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                    placeholder="e.g. Pro Tier"
                                    value={formData.name}
                                    onChange={e => setFormData({ ...formData, name: e.target.value })}
                                />
                            </label>
                            <label className="flex flex-col min-w-40 flex-1">
                                <p className="text-slate-900 dark:text-white text-sm font-medium leading-normal pb-2">Plan Code (Slug)</p>
                                <input
                                    required
                                    className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                    placeholder="e.g. pro-monthly"
                                    value={formData.code}
                                    onChange={e => setFormData({ ...formData, code: e.target.value })}
                                />
                            </label>
                        </div>
                        <label className="flex flex-col min-w-40 flex-1">
                            <p className="text-slate-900 dark:text-white text-sm font-medium leading-normal pb-2">Description <span className="text-slate-500 font-normal">(Optional)</span></p>
                            <input
                                className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                placeholder="Briefly describe this plan"
                                value={formData.description}
                                onChange={e => setFormData({ ...formData, description: e.target.value })}
                            />
                        </label>
                    </div>
                </div>

                {/* Pricing Section */}
                <div className="rounded-xl border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900 shadow-sm">
                    <div className="border-b border-slate-200 p-5 dark:border-slate-800">
                        <h2 className="text-slate-900 dark:text-white text-lg font-semibold leading-tight">Pricing</h2>
                    </div>
                    <div className="p-5 space-y-6">
                        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                            <div className="flex flex-col">
                                <p className="text-slate-900 dark:text-white text-sm font-medium leading-normal pb-2">Price</p>
                                <div className="relative flex items-center">
                                    <input
                                        required
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        className="form-input w-full rounded-l-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                        placeholder="0.00"
                                        value={formData.price}
                                        onChange={e => setFormData({ ...formData, price: e.target.value })}
                                    />
                                    <select
                                        className="form-select absolute right-0 rounded-r-lg border-y border-r border-slate-200 bg-slate-50 pr-8 text-sm text-slate-900 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-900 dark:text-white transition-all h-full"
                                        value={formData.currency}
                                        onChange={e => setFormData({ ...formData, currency: e.target.value })}
                                        style={{ height: '100%' }}
                                    >
                                        <option value="USD">USD</option>
                                        <option value="INR">INR</option>
                                        <option value="EUR">EUR</option>
                                        <option value="GBP">GBP</option>
                                    </select>
                                </div>
                            </div>
                            <div>
                                <p className="text-slate-900 dark:text-white text-sm font-medium leading-normal pb-2">Billing Interval</p>
                                <div className="flex w-full rounded-lg border border-slate-200 bg-slate-50 p-1 dark:border-slate-800 dark:bg-slate-950">
                                    <button
                                        type="button"
                                        onClick={() => setFormData({ ...formData, interval: 'month' })}
                                        className={`flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-all ${formData.interval === 'month' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-800 dark:text-white' : 'text-slate-500 dark:text-slate-400'}`}
                                    >
                                        Monthly
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => setFormData({ ...formData, interval: 'year' })}
                                        className={`flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-all ${formData.interval === 'year' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-800 dark:text-white' : 'text-slate-500 dark:text-slate-400'}`}
                                    >
                                        Yearly
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                    {/* Usage-Based Billing Toggle (Visual Only) */}
                    <div className="flex items-center justify-between border-t border-slate-200 pt-6 px-6 pb-6 dark:border-slate-800">
                        <div className="flex flex-col">
                            <p className="text-slate-900 dark:text-white text-sm font-medium">Enable Usage-Based Billing</p>
                            <p className="text-slate-500 dark:text-slate-400 text-sm">Charge customers based on their consumption.</p>
                        </div>
                        <button aria-checked="false" className="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent bg-slate-200 transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:bg-slate-700" role="switch" type="button">
                            <span className="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out dark:bg-slate-400"></span>
                        </button>
                    </div>
                </div>

                {/* Action Buttons */}
                <div className="flex justify-end gap-4 pt-4 pb-12">
                    <button
                        type="button"
                        onClick={() => navigate('/plans')}
                        className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-transparent text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 text-sm font-medium leading-normal transition-all"
                    >
                        <span className="truncate">Cancel</span>
                    </button>
                    <button
                        type="submit"
                        disabled={loading}
                        className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-medium leading-normal shadow-sm hover:bg-primary/90 disabled:opacity-50 transition-all"
                    >
                        <span className="truncate">{loading ? 'Creating...' : 'Create Plan'}</span>
                    </button>
                </div>
            </form>
        </div>
    )
}

export default CreatePlan
