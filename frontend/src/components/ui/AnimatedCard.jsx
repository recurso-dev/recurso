import React from 'react'

/**
 * AnimatedCard - A card wrapper with entrance animations and hover effects
 * @param {number} index - For staggered animation delay
 * @param {string} variant - Card style variant: default, stat, feature
 * @param {boolean} hoverable - Enable hover lift effect
 * @param {function} onClick - Click handler
 * @param {React.ReactNode} children
 */
const AnimatedCard = ({
    children,
    index = 0,
    variant = "default",
    hoverable = true,
    onClick = null,
    className = ""
}) => {
    const baseStyles = "rounded-xl overflow-hidden animate-fade-up opacity-0"
    const hoverStyles = hoverable ? "hover-lift cursor-pointer" : ""

    const variantStyles = {
        default: "bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800",
        stat: "bg-gradient-to-br from-white to-slate-50 dark:from-slate-900 dark:to-slate-800 border border-slate-200 dark:border-slate-700",
        feature: "bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border border-slate-200/50 dark:border-slate-700/50",
        glass: "glass"
    }

    const delayClass = `animation-delay-${index * 100}`

    return (
        <div
            className={`${baseStyles} ${variantStyles[variant]} ${hoverStyles} ${delayClass} ${className}`}
            onClick={onClick}
            style={{ animationFillMode: 'forwards', animationDelay: `${index * 0.1}s` }}
        >
            {children}
        </div>
    )
}

export default AnimatedCard
