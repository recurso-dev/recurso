import React from 'react'

const CustomerRiskCard = ({ customer }) => {
    if (!customer) return null

    const score = customer.risk_score || 0
    const factors = customer.risk_factors || {}
    const riskList = factors.factors || []

    let color = 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    let progressColor = 'text-green-500'
    let label = 'Low Risk'

    if (score >= 50) {
        color = 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
        progressColor = 'text-red-500'
        label = 'High Risk'
    } else if (score >= 20) {
        color = 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
        progressColor = 'text-yellow-500'
        label = 'Medium Risk'
    }

    // Circular Progress Calculation
    const radius = 30
    const circumference = 2 * Math.PI * radius
    const offset = circumference - (score / 100) * circumference

    return (
        <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4 shadow-sm">
            <h3 className="text-sm font-medium text-slate-500 dark:text-slate-400 mb-4">AI Churn Prediction</h3>

            <div className="flex items-center gap-6">
                {/* Circular Progress */}
                <div className="relative size-20 flex items-center justify-center">
                    <svg className="size-full -rotate-90" viewBox="0 0 72 72">
                        {/* Background Circle */}
                        <circle
                            cx="36"
                            cy="36"
                            r={radius}
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="6"
                            className="text-slate-100 dark:text-slate-800"
                        />
                        {/* Progress Circle */}
                        <circle
                            cx="36"
                            cy="36"
                            r={radius}
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="6"
                            strokeDasharray={circumference}
                            strokeDashoffset={offset}
                            strokeLinecap="round"
                            className={`transition-all duration-1000 ease-out ${progressColor}`}
                        />
                    </svg>
                    <div className="absolute flex flex-col items-center">
                        <span className={`text-xl font-bold ${progressColor.replace('text-', 'text-')}`.split(' ')[0]}>
                            {score}
                        </span>
                    </div>
                </div>

                {/* Details */}
                <div className="flex-1">
                    <div className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${color} mb-2`}>
                        {label}
                    </div>

                    {riskList.length > 0 ? (
                        <div className="space-y-1">
                            {riskList.map((factor, idx) => (
                                <div key={idx} className="flex items-center text-xs text-slate-600 dark:text-slate-400">
                                    <span className="material-symbols-outlined text-[14px] mr-1 text-red-500">warning</span>
                                    {formatFactor(factor)}
                                </div>
                            ))}
                            {factors.failed_invoices_count > 0 && (
                                <div className="text-xs text-slate-500 pl-5">
                                    {factors.failed_invoices_count} failed invoices
                                </div>
                            )}
                        </div>
                    ) : (
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                            No significant risk factors detected.
                        </p>
                    )}
                </div>
            </div>
        </div>
    )
}

const formatFactor = (factor) => {
    return factor.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
}

export default CustomerRiskCard
