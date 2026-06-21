import React, { useEffect } from 'react'

const SlideOver = ({ isOpen, onClose, title, children }) => {
    useEffect(() => {
        const handleEscape = (e) => {
            if (e.key === 'Escape') onClose()
        }
        if (isOpen) {
            document.addEventListener('keydown', handleEscape)
            document.body.style.overflow = 'hidden'
        }
        return () => {
            document.removeEventListener('keydown', handleEscape)
            document.body.style.overflow = 'unset'
        }
    }, [isOpen, onClose])

    if (!isOpen) return null

    return (
        <div className="relative z-50" aria-labelledby="slide-over-title" role="dialog" aria-modal="true">
            {/* Background backdrop */}
            <div className="fixed inset-0 bg-gray-500/75 transition-opacity" aria-hidden="true" onClick={onClose}></div>

            <div className="fixed inset-0 overflow-hidden">
                <div className="absolute inset-0 overflow-hidden">
                    <div className="pointer-events-none fixed inset-y-0 right-0 flex max-w-full pl-10">
                        {/* Slide-over panel */}
                        <div className="pointer-events-auto w-screen max-w-md transform transition ease-in-out duration-500 sm:duration-700 bg-white dark:bg-slate-900 shadow-xl ring-1 ring-black/5">
                            <div className="flex h-full flex-col overflow-y-scroll bg-white dark:bg-slate-900 py-6">
                                <div className="px-4 sm:px-6">
                                    <div className="flex items-start justify-between">
                                        <h2 className="text-base font-semibold leading-6 text-slate-900 dark:text-white" id="slide-over-title">{title}</h2>
                                        <div className="ml-3 flex h-7 items-center">
                                            <button
                                                type="button"
                                                className="relative rounded-md bg-white dark:bg-slate-900 text-slate-400 hover:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
                                                onClick={onClose}
                                            >
                                                <span className="absolute -inset-2.5"></span>
                                                <span className="sr-only">Close panel</span>
                                                <span className="material-symbols-outlined">close</span>
                                            </button>
                                        </div>
                                    </div>
                                </div>
                                <div className="relative mt-6 flex-1 px-4 sm:px-6">
                                    {children}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default SlideOver
