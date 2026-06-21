import React, { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { endpoints } from '../lib/api'
import { useToast } from '../components/Toast'
import ConsentCheckbox from '../components/ui/ConsentCheckbox'

const CreateSubscription = () => {
    const navigate = useNavigate()
    const [customers, setCustomers] = useState([])
    const [plans, setPlans] = useState([])
    const [loadingData, setLoadingData] = useState(true)
    const [submitting, setSubmitting] = useState(false)

    const toast = useToast()
    const [formData, setFormData] = useState({
        customer_id: '',
        plan_id: '',
        start_date: new Date().toISOString().split('T')[0],
        billing_anchor_type: 'acquisition',
        payment_terms: 'due_on_receipt',
        consent_granted: false
    })

    useEffect(() => {
        const fetchData = async () => {
            try {
                const [custRes, plansRes] = await Promise.all([
                    endpoints.getCustomers(),
                    endpoints.getPlans()
                ])
                setCustomers(custRes.data.data || [])
                setPlans(plansRes.data.data || [])
            } catch (error) {
                console.error("Failed to fetch data:", error)
            } finally {
                setLoadingData(false)
            }
        }
        fetchData()
    }, [])

    const selectedCustomer = customers.find(c => c.id === formData.customer_id)
    const selectedPlan = plans.find(p => p.id === formData.plan_id)

    const handleSubmit = async (e) => {
        e.preventDefault()

        if (!formData.consent_granted) {
            toast.warning('Please authorize recurring billing to continue.')
            return
        }

        setSubmitting(true)
        try {
            const payload = {
                customer_id: formData.customer_id,
                plan_id: formData.plan_id,
                start_date: new Date(formData.start_date).toISOString(),
                billing_anchor_type: formData.billing_anchor_type,
                payment_terms: formData.payment_terms,
            }
            const res = await endpoints.createSubscription(payload)
            const sub = res.data

            if (sub && sub.razorpay_subscription_id) {
                // Initialize Razorpay Options
                const options = {
                    key: import.meta.env.VITE_RAZORPAY_KEY_ID,
                    subscription_id: sub.razorpay_subscription_id,
                    name: "Billify Recurso",
                    description: `Subscription for ${selectedPlan?.name || 'Plan'}`,
                    handler: function (response) {
                        // Success Handler
                        console.log("Razorpay Signature:", response.razorpay_signature);
                        // Optionally call backend to verify/record payment immediately
                        navigate('/subscriptions')
                    },
                    prefill: {
                        name: selectedCustomer?.name,
                        email: selectedCustomer?.email,
                        contact: selectedCustomer?.phone
                    },
                    theme: {
                        color: "#3b82f6"
                    },
                    modal: {
                        ondismiss: function () {
                            console.log("Checkout form closed");
                            navigate('/subscriptions');
                        }
                    }
                };

                const rzp = new window.Razorpay(options);
                rzp.open();
            } else {
                navigate('/subscriptions')
            }

        } catch (error) {
            console.error("Failed to create subscription:", error)
            toast.error(error?.response?.data?.error?.message || "Failed to create subscription")
            setSubmitting(false) // Only stop loading if error or no-redirect
        }
        // Note: If Razorpay opens, we keep submitting=true until handler/dismiss to prevent double clicks? 
        // Actually rzp.open() is non-blocking so executing falls through. 
        // But we want to keep 'Creating...' state if we are redirecting?
        // Let's setSubmitting(false) if we open razorpay so UI is responsive, 
        // or keep it true effectively blocking until navigation.
        // Better:
        if (!loadingData) setSubmitting(false)
    }

    // Calculate Summary
    const planPrice = selectedPlan && selectedPlan.prices && selectedPlan.prices.length > 0
        ? selectedPlan.prices[0].amount / 100
        : 0
    const currency = selectedPlan && selectedPlan.prices && selectedPlan.prices.length > 0
        ? selectedPlan.prices[0].currency
        : 'USD'

    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            {/* Page Heading */}
            <div className="flex flex-wrap justify-between items-center gap-4 pb-8 border-b border-slate-200 dark:border-slate-800">
                <div className="flex flex-col gap-2">
                    <h1 className="text-slate-900 dark:text-white text-3xl font-bold tracking-tight">Create New Subscription</h1>
                    <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal">Create a new subscription for an existing customer.</p>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-8 mt-8">
                {/* Main Form Section */}
                <div className="md:col-span-2 flex flex-col gap-8">
                    <form id="create-subscription-form" onSubmit={handleSubmit} className="flex flex-col gap-8">
                        {/* Customer & Plan Section */}
                        <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
                            <div className="p-6">
                                <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-6">Customer & Plan</h2>
                                <div className="flex flex-col gap-6">
                                    <label className="flex flex-col flex-1">
                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Customer</p>
                                        <select
                                            required
                                            className="form-select flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-slate-900 dark:text-white focus:outline-0 focus:ring-2 focus:ring-primary/50 border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950 h-12 px-3 text-base font-normal leading-normal transition-all"
                                            value={formData.customer_id}
                                            onChange={e => setFormData({ ...formData, customer_id: e.target.value })}
                                        >
                                            <option value="">Select a customer</option>
                                            {customers.map(customer => (
                                                <option key={customer.id} value={customer.id}>
                                                    {customer.name} ({customer.email})
                                                </option>
                                            ))}
                                        </select>
                                    </label>
                                    <label className="flex flex-col flex-1">
                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Plan</p>
                                        <select
                                            required
                                            className="form-select flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-slate-900 dark:text-white focus:outline-0 focus:ring-2 focus:ring-primary/50 border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950 h-12 px-3 text-base font-normal leading-normal transition-all"
                                            value={formData.plan_id}
                                            onChange={e => setFormData({ ...formData, plan_id: e.target.value })}
                                        >
                                            <option value="">Select a plan</option>
                                            {plans.map(plan => (
                                                <option key={plan.id} value={plan.id}>
                                                    {plan.name} - ${(plan.prices?.[0]?.amount / 100).toFixed(2)}/{plan.interval_unit}
                                                </option>
                                            ))}
                                        </select>
                                    </label>
                                </div>
                            </div>
                        </div>

                        {/* Scheduling Section */}
                        <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
                            <div className="p-6">
                                <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-6">Scheduling & Billing</h2>
                                <div className="flex flex-col gap-6">
                                    <label className="flex flex-col flex-1">
                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Start Date</p>
                                        <div className="relative">
                                            <input
                                                type="date"
                                                className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-slate-900 dark:text-white focus:outline-0 focus:ring-2 focus:ring-primary/50 border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950 h-12 px-3 text-base font-normal leading-normal transition-all"
                                                value={formData.start_date}
                                                onChange={e => setFormData({ ...formData, start_date: e.target.value })}
                                            />
                                        </div>
                                    </label>
                                    <label className="flex flex-col flex-1">
                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Billing Anchor</p>
                                        <select
                                            className="form-select flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-slate-900 dark:text-white focus:outline-0 focus:ring-2 focus:ring-primary/50 border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950 h-12 px-3 text-base font-normal leading-normal transition-all"
                                            value={formData.billing_anchor_type}
                                            onChange={e => setFormData({ ...formData, billing_anchor_type: e.target.value })}
                                        >
                                            <option value="acquisition">Acquisition Date (default)</option>
                                            <option value="first_of_month">Calendar Billing (1st of Month)</option>
                                        </select>
                                        <p className="text-xs text-slate-500 dark:text-slate-400 mt-1.5">
                                            {formData.billing_anchor_type === 'first_of_month'
                                                ? 'First period will be prorated. All renewals align to the 1st.'
                                                : 'Billing repeats from the subscription start date.'}
                                        </p>
                                    </label>
                                    <label className="flex flex-col flex-1">
                                        <p className="text-slate-900 dark:text-slate-300 text-sm font-medium leading-normal pb-2">Payment Terms</p>
                                        <select
                                            className="form-select flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-slate-900 dark:text-white focus:outline-0 focus:ring-2 focus:ring-primary/50 border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950 h-12 px-3 text-base font-normal leading-normal transition-all"
                                            value={formData.payment_terms}
                                            onChange={e => setFormData({ ...formData, payment_terms: e.target.value })}
                                        >
                                            <option value="due_on_receipt">Due on Receipt (default)</option>
                                            <option value="net15">Net 15</option>
                                            <option value="net30">Net 30</option>
                                            <option value="net45">Net 45</option>
                                            <option value="net60">Net 60</option>
                                        </select>
                                        <p className="text-xs text-slate-500 dark:text-slate-400 mt-1.5">
                                            Number of days after invoice date before payment is due.
                                        </p>
                                    </label>
                                </div>
                            </div>
                        </div>

                        {/* Consent Section */}
                        <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
                            <div className="p-6">
                                <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">Billing Authorization</h2>
                                <ConsentCheckbox
                                    type="recurring"
                                    checked={formData.consent_granted}
                                    onChange={(checked) => setFormData({ ...formData, consent_granted: checked })}
                                    planName={selectedPlan?.name || 'the selected plan'}
                                    amount={selectedPlan?.prices?.[0]?.amount ? `${currency} ${(selectedPlan.prices[0].amount / 100).toFixed(2)}` : ''}
                                    interval={selectedPlan?.interval_unit || 'month'}
                                />
                            </div>
                        </div>
                    </form>
                </div>

                {/* Summary Section */}
                <div className="md:col-span-1">
                    <div className="sticky top-24 bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
                        <div className="p-6">
                            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">Summary</h3>

                            {selectedCustomer && selectedPlan ? (
                                <>
                                    <div className="space-y-3 text-sm">
                                        <div className="flex justify-between">
                                            <span className="text-slate-500 dark:text-slate-400">Customer</span>
                                            <span className="font-medium text-slate-900 dark:text-white text-right max-w-[150px] truncate">{selectedCustomer.name}</span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span className="text-slate-500 dark:text-slate-400">Plan</span>
                                            <span className="font-medium text-slate-900 dark:text-white text-right max-w-[150px] truncate">{selectedPlan.name}</span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span className="text-slate-500 dark:text-slate-400">Starts on</span>
                                            <span className="font-medium text-slate-900 dark:text-white">{new Date(formData.start_date).toLocaleDateString()}</span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span className="text-slate-500 dark:text-slate-400">Billing</span>
                                            <span className="font-medium text-slate-900 dark:text-white">
                                                {formData.billing_anchor_type === 'first_of_month' ? 'Calendar (1st)' : 'Acquisition'}
                                            </span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span className="text-slate-500 dark:text-slate-400">Terms</span>
                                            <span className="font-medium text-slate-900 dark:text-white">
                                                {formData.payment_terms === 'due_on_receipt' ? 'Due on Receipt' : formData.payment_terms.replace('net', 'Net ')}
                                            </span>
                                        </div>
                                    </div>
                                    <div className="my-4 h-px bg-slate-200 dark:bg-slate-800"></div>
                                    <div className="space-y-3 text-sm">
                                        <div className="flex justify-between">
                                            <span className="text-slate-500 dark:text-slate-400">{selectedPlan.name}</span>
                                            <span className="font-medium text-slate-900 dark:text-white">
                                                {new Intl.NumberFormat('en-US', { style: 'currency', currency: currency }).format(planPrice)}
                                            </span>
                                        </div>
                                    </div>
                                    <div className="my-4 h-px bg-slate-200 dark:bg-slate-800"></div>
                                    <div className="flex justify-between mb-6">
                                        <span className="text-base font-semibold text-slate-900 dark:text-white">Total</span>
                                        <span className="text-base font-semibold text-slate-900 dark:text-white">
                                            {new Intl.NumberFormat('en-US', { style: 'currency', currency: currency }).format(planPrice)}
                                        </span>
                                    </div>
                                </>
                            ) : (
                                <p className="text-sm text-slate-500 text-center py-4">Select a customer and plan to see summary.</p>
                            )}

                            <div className="flex flex-col gap-3 mt-4">
                                <button
                                    form="create-subscription-form"
                                    type="submit"
                                    disabled={submitting}
                                    className="flex w-full cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-semibold leading-normal tracking-wide hover:bg-primary/90 disabled:opacity-50 transition-all"
                                >
                                    {submitting ? 'Creating...' : 'Create Subscription'}
                                </button>
                                <button
                                    type="button"
                                    onClick={() => navigate('/subscriptions')}
                                    className="flex w-full cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-transparent text-slate-900 dark:text-white text-sm font-semibold leading-normal tracking-wide border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 transition-all"
                                >
                                    Cancel
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}


export default CreateSubscription
