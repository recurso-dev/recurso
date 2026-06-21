import React from 'react'

// Generic empty state with illustration
const EmptyState = ({
    icon: Icon,
    title = "No data yet",
    description = "Get started by adding your first item.",
    action = null,
    actionLabel = "Create",
    variant = "default" // default, success, warning
}) => {
    const variants = {
        default: "from-blue-500/20 to-purple-500/20 border-blue-500/30",
        success: "from-emerald-500/20 to-teal-500/20 border-emerald-500/30",
        warning: "from-amber-500/20 to-orange-500/20 border-amber-500/30"
    }

    return (
        <div className="flex flex-col items-center justify-center py-16 px-4 text-center animate-fade-in">
            {/* Animated icon container */}
            <div className={`relative mb-6`}>
                <div className={`absolute inset-0 bg-gradient-to-br ${variants[variant]} rounded-full blur-xl animate-pulse-slow`} />
                <div className="relative w-20 h-20 rounded-2xl bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 flex items-center justify-center">
                    {Icon && <Icon className="w-10 h-10 text-slate-400 dark:text-slate-500" />}
                </div>
                {/* Floating dots decoration */}
                <div className="absolute -top-2 -right-2 w-3 h-3 bg-blue-500/40 rounded-full animate-bounce" style={{ animationDelay: '0s' }} />
                <div className="absolute -bottom-1 -left-3 w-2 h-2 bg-purple-500/40 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }} />
            </div>

            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
                {title}
            </h3>
            <p className="text-sm text-slate-500 dark:text-slate-400 max-w-sm mb-6">
                {description}
            </p>

            {action && (
                <button
                    onClick={action}
                    className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white font-medium rounded-lg hover:bg-primary/90 transition-all hover:scale-105 active:scale-95 shadow-lg shadow-primary/25"
                >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                    {actionLabel}
                </button>
            )}
        </div>
    )
}

export default EmptyState
