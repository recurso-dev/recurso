import React from 'react'
import { Link } from 'react-router-dom'

// Accepts arrays or numeric counts for each data set so callers can pass
// whatever shape they already have (and tests need no API mocking).
const toCount = (value) => {
    if (Array.isArray(value)) return value.length
    if (typeof value === 'number') return value
    return 0
}

const SetupChecklist = ({ plans, customers, subscriptions, invoices }) => {
    const steps = [
        { key: 'plan', label: 'Create a plan', to: '/plans', done: toCount(plans) > 0 },
        { key: 'customer', label: 'Add a customer', to: '/customers', done: toCount(customers) > 0 },
        { key: 'subscription', label: 'Start a subscription', to: '/subscriptions', done: toCount(subscriptions) > 0 },
        { key: 'invoice', label: 'See your first invoice', to: '/invoices', done: toCount(invoices) > 0 },
    ]

    const completedCount = steps.filter(s => s.done).length

    // Fully set up — nothing to show.
    if (completedCount === steps.length) return null

    return (
        <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900/50 mb-8">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 mb-4">
                <div className="flex items-center gap-2">
                    <span className="material-symbols-outlined text-gray-400 text-[20px]">rocket_launch</span>
                    <h3 className="text-base font-semibold text-gray-900 dark:text-white">Get set up</h3>
                </div>
                <p className="text-sm font-medium text-gray-500 dark:text-zinc-500">
                    {completedCount} of {steps.length} complete
                </p>
            </div>

            {/* Progress bar */}
            <div
                className="h-1.5 w-full rounded-full bg-gray-100 dark:bg-zinc-800 mb-5 overflow-hidden"
                role="progressbar"
                aria-valuenow={completedCount}
                aria-valuemin={0}
                aria-valuemax={steps.length}
                aria-label="Setup progress"
            >
                <div
                    className="h-full rounded-full bg-gray-900 dark:bg-white transition-all duration-300"
                    style={{ width: `${(completedCount / steps.length) * 100}%` }}
                />
            </div>

            {/* Steps */}
            <ul className="divide-y divide-gray-100 dark:divide-zinc-800">
                {steps.map((step) => (
                    <li key={step.key} data-testid={`setup-step-${step.key}`}>
                        {step.done ? (
                            <div className="flex items-center gap-3 py-3">
                                <span
                                    className="material-symbols-outlined text-emerald-500 text-[20px] flex-none"
                                    style={{ fontVariationSettings: "'FILL' 1" }}
                                    aria-hidden="true"
                                >
                                    check_circle
                                </span>
                                <span className="text-sm text-gray-400 dark:text-zinc-500 line-through decoration-gray-300 dark:decoration-zinc-600">
                                    {step.label}
                                </span>
                                <span className="sr-only">(complete)</span>
                            </div>
                        ) : (
                            <Link
                                to={step.to}
                                className="group flex items-center gap-3 py-3 -mx-2 px-2 rounded-md hover:bg-gray-50 dark:hover:bg-zinc-800/50 transition-colors"
                            >
                                <span
                                    className="material-symbols-outlined text-gray-300 dark:text-zinc-600 text-[20px] flex-none"
                                    aria-hidden="true"
                                >
                                    radio_button_unchecked
                                </span>
                                <span className="flex-1 text-sm font-medium text-gray-900 dark:text-white">
                                    {step.label}
                                </span>
                                <span
                                    className="material-symbols-outlined text-gray-400 text-[18px] flex-none transition-transform group-hover:translate-x-0.5"
                                    aria-hidden="true"
                                >
                                    arrow_forward
                                </span>
                            </Link>
                        )}
                    </li>
                ))}
            </ul>
        </div>
    )
}

export default SetupChecklist
