import { useEffect, useRef } from 'react'

export const useScrollAnimation = (options = {}) => {
    const ref = useRef(null)

    useEffect(() => {
        const el = ref.current
        if (!el) return

        const observer = new IntersectionObserver(
            (entries) => {
                entries.forEach((entry) => {
                    if (entry.isIntersecting) {
                        entry.target.classList.add('is-visible')
                        // Once animated, stop observing
                        observer.unobserve(entry.target)
                    }
                })
            },
            {
                threshold: options.threshold || 0.1,
                rootMargin: options.rootMargin || '0px 0px -50px 0px',
            }
        )

        observer.observe(el)

        return () => observer.disconnect()
    }, [])

    return ref
}

// Hook to observe multiple children within a container
export const useStaggerAnimation = (options = {}) => {
    const ref = useRef(null)

    useEffect(() => {
        const container = ref.current
        if (!container) return

        const children = container.querySelectorAll('[data-animate]')

        const observer = new IntersectionObserver(
            (entries) => {
                entries.forEach((entry) => {
                    if (entry.isIntersecting) {
                        // Add staggered delay
                        const items = entry.target.querySelectorAll('[data-animate]')
                        items.forEach((item, i) => {
                            item.style.transitionDelay = `${i * 0.08}s`
                            item.classList.add('is-visible')
                        })
                        observer.unobserve(entry.target)
                    }
                })
            },
            {
                threshold: options.threshold || 0.05,
                rootMargin: options.rootMargin || '0px 0px -30px 0px',
            }
        )

        observer.observe(container)

        return () => observer.disconnect()
    }, [])

    return ref
}

export default useScrollAnimation
