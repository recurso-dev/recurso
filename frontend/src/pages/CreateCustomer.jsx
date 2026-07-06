import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { endpoints } from '../lib/api'
import { useToast } from '../components/Toast'

const CreateCustomer = () => {
    const navigate = useNavigate()
    const toast = useToast()
    const [formData, setFormData] = useState({
        name: '',
        email: '',
        phone: '',
        address: '',
        country: 'United States',
        state: 'California',
        tax_id: ''
    })
    const [loading, setLoading] = useState(false)

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)
        try {
            const countryMap = {
                'United States': 'US',
                'India': 'IN',
                'Canada': 'CA',
                'United Kingdom': 'GB'
            }
            const isoCountry = countryMap[formData.country] || 'US'

            const payload = {
                name: formData.name,
                email: formData.email,
                phone: formData.phone,
                tax_id: formData.tax_id,
                gstin: formData.country === 'India' ? formData.tax_id : '', // Send GSTIN separately if India
                place_of_supply: formData.country === 'India' ? formData.state : '', // Send Code if India
                line1: formData.address, // Mapping textarea to line1 for now
                country: isoCountry,
                state: formData.state
                // City/Zip could be parsed from address or added as new fields if needed
            }
            await endpoints.createCustomer(payload)
            navigate('/customers')
        } catch (error) {
            console.error("Failed to create customer:", error)
            toast.error(error?.response?.data?.error?.message || "Failed to create customer")
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="relative z-50" aria-labelledby="slide-over-title" role="dialog" aria-modal="true">
            {/* Background backdrop */}
            <div className="fixed inset-0 bg-slate-900/30 dark:bg-black/50 transition-opacity" aria-hidden="true"></div>

            <div className="fixed inset-0 overflow-hidden">
                <div className="absolute inset-0 overflow-hidden">
                    <div className="pointer-events-none fixed inset-y-0 right-0 flex max-w-full pl-10">
                        {/* Slide-over panel */}
                        <div className="pointer-events-auto relative w-screen max-w-2xl">
                            <div className="flex h-full flex-col bg-white dark:bg-slate-900 shadow-xl">
                                <div className="flex flex-col h-full">
                                    {/* Header */}
                                    <div className="bg-slate-50 dark:bg-slate-800/50 px-6 py-6 border-b border-slate-200 dark:border-slate-800 flex-shrink-0">
                                        <div className="flex items-start justify-between space-x-3">
                                            <div className="space-y-1">
                                                <h1 className="text-slate-900 dark:text-white text-2xl font-bold leading-tight tracking-tight">Add New Customer</h1>
                                                <p className="text-slate-500 dark:text-slate-400 text-sm font-normal leading-normal">Enter the details for your new customer.</p>
                                            </div>
                                            <div className="flex h-7 items-center">
                                                <button
                                                    type="button"
                                                    className="relative text-slate-400 hover:text-slate-500 dark:hover:text-slate-300 focus:outline-none"
                                                    onClick={() => navigate('/customers')}
                                                >
                                                    <span className="absolute -inset-2.5"></span>
                                                    <span className="sr-only">Close panel</span>
                                                    <span className="material-symbols-outlined text-2xl">close</span>
                                                </button>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Scrollable Content */}
                                    <div className="flex-1 overflow-y-auto">
                                        <form id="create-customer-form" onSubmit={handleSubmit} className="space-y-8 px-6 py-8">
                                            {/* Contact Information Section */}
                                            <div className="space-y-6">
                                                <h2 className="text-slate-900 dark:text-white text-lg font-bold leading-tight tracking-[-0.015em]">Contact Information</h2>
                                                <div className="grid grid-cols-1 gap-6">
                                                    <label className="flex flex-col flex-1">
                                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Customer Name <span className="text-red-500">*</span></p>
                                                        <input
                                                            required
                                                            className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                                            placeholder="e.g., Acme Corporation"
                                                            value={formData.name}
                                                            onChange={e => setFormData({ ...formData, name: e.target.value })}
                                                        />
                                                    </label>
                                                    <label className="flex flex-col flex-1">
                                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Email Address <span className="text-red-500">*</span></p>
                                                        <input
                                                            required
                                                            type="email"
                                                            className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                                            placeholder="e.g., billing@acme.com"
                                                            value={formData.email}
                                                            onChange={e => setFormData({ ...formData, email: e.target.value })}
                                                        />
                                                    </label>
                                                    <label className="flex flex-col flex-1">
                                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Phone Number</p>
                                                        <input
                                                            className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                                            placeholder="e.g., +1 (555) 123-4567"
                                                            value={formData.phone}
                                                            onChange={e => setFormData({ ...formData, phone: e.target.value })}
                                                        />
                                                    </label>
                                                </div>
                                            </div>

                                            {/* Divider */}
                                            <div className="border-t border-slate-200 dark:border-slate-800"></div>

                                            {/* Billing Details Section */}
                                            <div className="space-y-6">
                                                <h2 className="text-slate-900 dark:text-white text-lg font-bold leading-tight tracking-[-0.015em]">Billing Details</h2>
                                                <div className="grid grid-cols-1 gap-6">
                                                    <label className="flex flex-col flex-1">
                                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Billing Address</p>
                                                        <textarea
                                                            className="form-textarea flex w-full min-w-0 flex-1 resize-vertical rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all min-h-[88px]"
                                                            placeholder="123 Main Street, Anytown, USA 12345"
                                                            value={formData.address}
                                                            onChange={e => setFormData({ ...formData, address: e.target.value })}
                                                        ></textarea>
                                                    </label>
                                                    <div className="grid grid-cols-2 gap-6">
                                                        <label className="flex flex-col flex-1">
                                                            <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Country</p>
                                                            <select
                                                                className="form-select flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                                                value={formData.country}
                                                                onChange={e => setFormData({ ...formData, country: e.target.value, state: '' })}
                                                            >
                                                                <option>United States</option>
                                                                <option>India</option>
                                                                <option>Canada</option>
                                                                <option>United Kingdom</option>
                                                            </select>
                                                        </label>
                                                        <label className="flex flex-col flex-1">
                                                            <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">
                                                                {formData.country === 'India' ? 'Place of Supply (State)' : 'State / Province'}
                                                            </p>
                                                            {formData.country === 'India' ? (
                                                                <select
                                                                    className="form-select flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                                                    value={formData.state}
                                                                    onChange={e => setFormData({ ...formData, state: e.target.value })}
                                                                >
                                                                    <option value="">Select State</option>
                                                                    <option value="TN">Tamil Nadu</option>
                                                                    <option value="KA">Karnataka</option>
                                                                    <option value="MH">Maharashtra</option>
                                                                    <option value="DL">Delhi</option>
                                                                    <option value="UP">Uttar Pradesh</option>
                                                                    <option value="GJ">Gujarat</option>
                                                                    <option value="KL">Kerala</option>
                                                                    {/* Add more states as needed */}
                                                                </select>
                                                            ) : (
                                                                <input // Use input for other countries for now to allow free text or select for US
                                                                    className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                                                    placeholder="e.g. California"
                                                                    value={formData.state}
                                                                    onChange={e => setFormData({ ...formData, state: e.target.value })}
                                                                />
                                                            )}
                                                        </label>
                                                    </div>
                                                    <label className="flex flex-col flex-1">
                                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">
                                                            {formData.country === 'India' ? 'GSTIN (Goods and Services Tax ID)' : 'Tax ID / VAT Number'}
                                                        </p>
                                                        <input
                                                            className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 dark:border-slate-800 dark:bg-slate-950 dark:text-white dark:placeholder:text-slate-400 transition-all"
                                                            placeholder={formData.country === 'India' ? "e.g., 29ABCDE1234F1Z5" : "e.g., EU123456789"}
                                                            value={formData.tax_id}
                                                            onChange={e => setFormData({ ...formData, tax_id: e.target.value })}
                                                        />
                                                    </label>
                                                </div>
                                            </div>
                                        </form>
                                    </div>

                                    {/* Action buttons */}
                                    <div className="flex-shrink-0 border-t border-slate-200 dark:border-slate-700 px-6 py-5 bg-white dark:bg-slate-900">
                                        <div className="flex justify-end space-x-3">
                                            <button
                                                type="button"
                                                onClick={() => navigate('/customers')}
                                                className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-800 dark:text-white text-sm font-bold leading-normal transition-all"
                                            >
                                                <span className="truncate">Cancel</span>
                                            </button>
                                            <button
                                                form="create-customer-form"
                                                type="submit"
                                                disabled={loading}
                                                className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary hover:bg-primary/90 text-white text-sm font-bold leading-normal shadow-sm transition-all disabled:opacity-50"
                                            >
                                                <span className="truncate">{loading ? "Creating..." : "Create Customer"}</span>
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default CreateCustomer
