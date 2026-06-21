import { useEffect, useRef, useState } from 'react'

// Hook for scroll-triggered animations
export function useScrollAnimation(options = {}) {
    const ref = useRef(null)
    const [isVisible, setIsVisible] = useState(false)

    useEffect(() => {
        const observer = new IntersectionObserver(
            ([entry]) => {
                if (entry.isIntersecting) {
                    setIsVisible(true)
                    // Once visible, optionally stop observing
                    if (options.once !== false) {
                        observer.unobserve(entry.target)
                    }
                } else if (options.once === false) {
                    setIsVisible(false)
                }
            },
            {
                threshold: options.threshold || 0.1,
                rootMargin: options.rootMargin || '0px',
            }
        )

        if (ref.current) {
            observer.observe(ref.current)
        }

        return () => {
            if (ref.current) {
                observer.unobserve(ref.current)
            }
        }
    }, [options.threshold, options.rootMargin, options.once])

    return { ref, isVisible }
}

// AnimatedSection wrapper component
export function AnimatedSection({ children, className = '', delay = 0, animation = 'fade-up' }) {
    const { ref, isVisible } = useScrollAnimation({ threshold: 0.1 })

    const animations = {
        'fade-up': 'translate-y-8 opacity-0',
        'fade-down': '-translate-y-8 opacity-0',
        'fade-left': 'translate-x-8 opacity-0',
        'fade-right': '-translate-x-8 opacity-0',
        'scale': 'scale-95 opacity-0',
        'none': '',
    }

    const baseStyles = animations[animation] || animations['fade-up']

    return (
        <div
            ref={ref}
            className={`transition-all duration-700 ease-out ${className} ${isVisible ? 'translate-y-0 translate-x-0 scale-100 opacity-100' : baseStyles
                }`}
            style={{ transitionDelay: `${delay}ms` }}
        >
            {children}
        </div>
    )
}

// Staggered children animation
export function StaggeredContainer({ children, className = '', staggerDelay = 100 }) {
    const { ref, isVisible } = useScrollAnimation({ threshold: 0.1 })

    return (
        <div ref={ref} className={className}>
            {Array.isArray(children)
                ? children.map((child, index) => (
                    <div
                        key={index}
                        className={`transition-all duration-500 ease-out ${isVisible ? 'translate-y-0 opacity-100' : 'translate-y-4 opacity-0'
                            }`}
                        style={{ transitionDelay: `${index * staggerDelay}ms` }}
                    >
                        {child}
                    </div>
                ))
                : children}
        </div>
    )
}

// Counter animation for stats
export function AnimatedCounter({ end, duration = 2000, suffix = '' }) {
    const [count, setCount] = useState(0)
    const { ref, isVisible } = useScrollAnimation({ threshold: 0.5 })

    useEffect(() => {
        if (!isVisible) return

        let startTime = null
        const startValue = 0

        const animate = (timestamp) => {
            if (!startTime) startTime = timestamp
            const progress = Math.min((timestamp - startTime) / duration, 1)

            // Easing function (ease-out)
            const easeOut = 1 - Math.pow(1 - progress, 3)
            setCount(Math.floor(startValue + (end - startValue) * easeOut))

            if (progress < 1) {
                requestAnimationFrame(animate)
            }
        }

        requestAnimationFrame(animate)
    }, [isVisible, end, duration])

    return (
        <span ref={ref}>
            {count.toLocaleString()}{suffix}
        </span>
    )
}
