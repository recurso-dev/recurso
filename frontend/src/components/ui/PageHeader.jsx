import React from 'react'

/**
 * PageHeader - Animated page title and description
 */
const PageHeader = ({
    title,
    description = null,
    action = null,
    badge = null
}) => {
    return (
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8 animate-fade-down">
            <div>
                <div className="flex items-center gap-3">
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                        {title}
                    </h1>
                    {badge && (
                        <span className="px-2 py-0.5 text-xs font-medium rounded-full bg-primary/10 text-primary animate-scale-in">
                            {badge}
                        </span>
                    )}
                </div>
                {description && (
                    <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
                        {description}
                    </p>
                )}
            </div>
            {action && (
                <div className="animate-slide-in-right">
                    {action}
                </div>
            )}
        </div>
    )
}

export default PageHeader
