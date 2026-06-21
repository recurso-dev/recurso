import React from 'react'

/**
 * StatCard - Animated metric card with trend indicator
 */
const StatCard = ({
    title,
    value,
    change = null, // e.g. "+12%" 
    changeType = "neutral", // positive, negative, neutral
    icon: Icon = null,
    index = 0
}) => {
    const changeColors = {
        positive: "text-emerald-600 dark:text-emerald-400",
        negative: "text-red-600 dark:text-red-400",
        neutral: "text-slate-500 dark:text-slate-400"
    }

    const bgGradients = [
        "from-blue-500/5 to-indigo-500/5",
        "from-emerald-500/5 to-teal-500/5",
        "from-amber-500/5 to-orange-500/5",
        "from-purple-500/5 to-pink-500/5"
    ]

    return (
        <div
            className={`
        relative overflow-hidden rounded-xl p-6
        bg-gradient-to-br ${bgGradients[index % 4]}
        bg-white dark:bg-slate-900
        border border-slate-200 dark:border-slate-800
        hover-lift animate-fade-up opacity-0
      `}
            style={{ animationFillMode: 'forwards', animationDelay: `${index * 0.1}s` }}
        >
            {/* Background decoration */}
            <div className="absolute -right-4 -top-4 w-24 h-24 rounded-full bg-gradient-to-br from-slate-100 to-slate-50 dark:from-slate-800 dark:to-slate-900 blur-2xl opacity-60" />

            <div className="relative">
                <div className="flex items-center justify-between mb-4">
                    <span className="text-sm font-medium text-slate-500 dark:text-slate-400">
                        {title}
                    </span>
                    {Icon && (
                        <div className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-slate-800 flex items-center justify-center">
                            <Icon className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                        </div>
                    )}
                </div>

                <div className="flex items-end gap-2">
                    <span className="text-3xl font-bold text-slate-900 dark:text-white">
                        {value}
                    </span>
                    {change && (
                        <span className={`text-sm font-medium ${changeColors[changeType]} mb-1`}>
                            {change}
                        </span>
                    )}
                </div>
            </div>
        </div>
    )
}

export default StatCard
