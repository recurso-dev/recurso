import React, { useEffect, useRef } from 'react';

const Modal = ({ isOpen, onClose, title, children }) => {
    const modalRef = useRef(null);
    const previousFocusRef = useRef(null);

    useEffect(() => {
        if (isOpen) {
            previousFocusRef.current = document.activeElement;
            // Focus the modal container
            setTimeout(() => modalRef.current?.focus(), 0);
        } else if (previousFocusRef.current) {
            previousFocusRef.current.focus();
        }
    }, [isOpen]);

    useEffect(() => {
        if (!isOpen) return;

        const handleKeyDown = (e) => {
            if (e.key === 'Escape') {
                onClose();
                return;
            }

            // Trap focus within modal
            if (e.key === 'Tab') {
                const focusable = modalRef.current?.querySelectorAll(
                    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
                );
                if (!focusable || focusable.length === 0) return;

                const first = focusable[0];
                const last = focusable[focusable.length - 1];

                if (e.shiftKey && document.activeElement === first) {
                    e.preventDefault();
                    last.focus();
                } else if (!e.shiftKey && document.activeElement === last) {
                    e.preventDefault();
                    first.focus();
                }
            }
        };

        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [isOpen, onClose]);

    if (!isOpen) return null;

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center overflow-auto bg-black bg-opacity-50"
            role="dialog"
            aria-modal="true"
            aria-labelledby="modal-title"
            onClick={(e) => {
                if (e.target === e.currentTarget) onClose();
            }}
        >
            <div
                ref={modalRef}
                tabIndex={-1}
                className="relative w-full max-w-md rounded-lg bg-white p-6 shadow-xl dark:bg-slate-900 focus:outline-none"
            >
                {/* Header */}
                <div className="mb-4 flex items-center justify-between">
                    <h3 id="modal-title" className="text-lg font-semibold text-slate-900 dark:text-white">{title}</h3>
                    <button
                        onClick={onClose}
                        className="text-slate-400 hover:text-slate-500"
                        aria-label="Close modal"
                        type="button"
                    >
                        <span className="material-symbols-outlined">close</span>
                    </button>
                </div>

                {/* Body */}
                <div>{children}</div>
            </div>
        </div>
    );
};

export default Modal;
