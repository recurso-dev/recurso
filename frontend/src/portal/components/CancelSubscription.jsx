import { useState } from 'react'
import { AlertTriangle, X, Check } from 'lucide-react'

const CancelSubscription = ({ subscription, onCancel, onClose }) => {
    const [step, setStep] = useState('confirm') // confirm | reason | success
    const [reason, setReason] = useState('')
    const [otherReason, setOtherReason] = useState('')
    const [isLoading, setIsLoading] = useState(false)

    const cancellationReasons = [
        { id: 'too_expensive', label: 'Too expensive' },
        { id: 'not_using', label: 'Not using it enough' },
        { id: 'missing_features', label: 'Missing features I need' },
        { id: 'switching_competitor', label: 'Switching to a competitor' },
        { id: 'temporary', label: 'Just need a break' },
        { id: 'other', label: 'Other reason' },
    ]

    const handleCancel = async () => {
        setIsLoading(true)
        try {
            await onCancel({
                reason: reason === 'other' ? otherReason : reason,
                subscriptionId: subscription.id,
            })
            setStep('success')
        } catch (error) {
            console.error('Cancellation failed:', error)
        } finally {
            setIsLoading(false)
        }
    }

    const formatDate = (dateString) => {
        return new Date(dateString).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
        })
    }

    return (
        <div className="cancel-modal-overlay" onClick={onClose}>
            <div className="cancel-modal" onClick={(e) => e.stopPropagation()}>
                <button className="cancel-modal-close" onClick={onClose}>
                    <X size={20} />
                </button>

                {step === 'confirm' && (
                    <>
                        <div className="cancel-header">
                            <div className="cancel-icon warning">
                                <AlertTriangle size={32} />
                            </div>
                            <h2>Cancel Subscription?</h2>
                            <p>You're about to cancel your {subscription?.plan_name} subscription.</p>
                        </div>

                        <div className="cancel-info">
                            <div className="info-row">
                                <span>Current Plan</span>
                                <strong>{subscription?.plan_name}</strong>
                            </div>
                            <div className="info-row">
                                <span>Access Until</span>
                                <strong>{formatDate(subscription?.current_period_end)}</strong>
                            </div>
                        </div>

                        <div className="cancel-warning">
                            <p>After cancellation:</p>
                            <ul>
                                <li>You'll have access until {formatDate(subscription?.current_period_end)}</li>
                                <li>You won't be charged again</li>
                                <li>You can reactivate anytime before the period ends</li>
                            </ul>
                        </div>

                        <div className="cancel-actions">
                            <button className="btn-secondary" onClick={onClose}>
                                Keep Subscription
                            </button>
                            <button
                                className="btn-danger"
                                onClick={() => setStep('reason')}
                            >
                                Continue to Cancel
                            </button>
                        </div>
                    </>
                )}

                {step === 'reason' && (
                    <>
                        <div className="cancel-header">
                            <h2>Help us improve</h2>
                            <p>Why are you cancelling? (Optional)</p>
                        </div>

                        <div className="cancel-reasons">
                            {cancellationReasons.map((r) => (
                                <label key={r.id} className={`reason-option ${reason === r.id ? 'selected' : ''}`}>
                                    <input
                                        type="radio"
                                        name="reason"
                                        value={r.id}
                                        checked={reason === r.id}
                                        onChange={(e) => setReason(e.target.value)}
                                    />
                                    <span>{r.label}</span>
                                </label>
                            ))}
                        </div>

                        {reason === 'other' && (
                            <textarea
                                className="other-reason-input"
                                placeholder="Please tell us more..."
                                value={otherReason}
                                onChange={(e) => setOtherReason(e.target.value)}
                                rows={3}
                            />
                        )}

                        <div className="cancel-actions">
                            <button className="btn-secondary" onClick={() => setStep('confirm')}>
                                Back
                            </button>
                            <button
                                className="btn-danger"
                                onClick={handleCancel}
                                disabled={isLoading}
                            >
                                {isLoading ? 'Cancelling...' : 'Confirm Cancellation'}
                            </button>
                        </div>
                    </>
                )}

                {step === 'success' && (
                    <>
                        <div className="cancel-header">
                            <div className="cancel-icon success">
                                <Check size={32} />
                            </div>
                            <h2>Subscription Cancelled</h2>
                            <p>We're sorry to see you go!</p>
                        </div>

                        <div className="cancel-info">
                            <div className="info-row">
                                <span>Access Until</span>
                                <strong>{formatDate(subscription?.current_period_end)}</strong>
                            </div>
                        </div>

                        <p className="cancel-note">
                            You'll receive a confirmation email shortly. You can reactivate your
                            subscription anytime from this portal.
                        </p>

                        <div className="cancel-actions single">
                            <button className="btn-primary" onClick={onClose}>
                                Done
                            </button>
                        </div>
                    </>
                )}
            </div>

            <style jsx>{`
                .cancel-modal-overlay {
                    position: fixed;
                    inset: 0;
                    background: rgba(0, 0, 0, 0.5);
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    z-index: 1000;
                    padding: 20px;
                }

                .cancel-modal {
                    background: white;
                    border-radius: 16px;
                    max-width: 440px;
                    width: 100%;
                    padding: 32px;
                    position: relative;
                    animation: slideUp 0.3s ease-out;
                }

                @keyframes slideUp {
                    from {
                        opacity: 0;
                        transform: translateY(20px);
                    }
                    to {
                        opacity: 1;
                        transform: translateY(0);
                    }
                }

                .cancel-modal-close {
                    position: absolute;
                    top: 16px;
                    right: 16px;
                    background: none;
                    border: none;
                    color: #94a3b8;
                    cursor: pointer;
                    padding: 4px;
                    border-radius: 8px;
                    transition: all 0.2s;
                }

                .cancel-modal-close:hover {
                    background: #f1f5f9;
                    color: #475569;
                }

                .cancel-header {
                    text-align: center;
                    margin-bottom: 24px;
                }

                .cancel-header h2 {
                    margin: 0 0 8px 0;
                    color: #0f172a;
                    font-size: 20px;
                }

                .cancel-header p {
                    margin: 0;
                    color: #64748b;
                    font-size: 14px;
                }

                .cancel-icon {
                    width: 64px;
                    height: 64px;
                    border-radius: 50%;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    margin: 0 auto 16px;
                }

                .cancel-icon.warning {
                    background: #fef3c7;
                    color: #d97706;
                }

                .cancel-icon.success {
                    background: #dcfce7;
                    color: #16a34a;
                }

                .cancel-info {
                    background: #f8fafc;
                    border-radius: 12px;
                    padding: 16px;
                    margin-bottom: 20px;
                }

                .info-row {
                    display: flex;
                    justify-content: space-between;
                    padding: 8px 0;
                    font-size: 14px;
                }

                .info-row span {
                    color: #64748b;
                }

                .info-row strong {
                    color: #0f172a;
                }

                .cancel-warning {
                    background: #fffbeb;
                    border: 1px solid #fef3c7;
                    border-radius: 12px;
                    padding: 16px;
                    margin-bottom: 24px;
                }

                .cancel-warning p {
                    margin: 0 0 12px 0;
                    font-weight: 600;
                    color: #92400e;
                    font-size: 14px;
                }

                .cancel-warning ul {
                    margin: 0;
                    padding-left: 20px;
                    color: #a16207;
                    font-size: 13px;
                }

                .cancel-warning li {
                    margin-bottom: 4px;
                }

                .cancel-reasons {
                    display: flex;
                    flex-direction: column;
                    gap: 8px;
                    margin-bottom: 16px;
                }

                .reason-option {
                    display: flex;
                    align-items: center;
                    gap: 12px;
                    padding: 12px 16px;
                    border: 1px solid #e2e8f0;
                    border-radius: 10px;
                    cursor: pointer;
                    transition: all 0.2s;
                }

                .reason-option:hover {
                    border-color: #cbd5e1;
                    background: #f8fafc;
                }

                .reason-option.selected {
                    border-color: #0f172a;
                    background: #f1f5f9;
                }

                .reason-option input {
                    display: none;
                }

                .reason-option span {
                    font-size: 14px;
                    color: #334155;
                }

                .other-reason-input {
                    width: 100%;
                    padding: 12px;
                    border: 1px solid #e2e8f0;
                    border-radius: 10px;
                    font-size: 14px;
                    resize: none;
                    margin-bottom: 16px;
                }

                .other-reason-input:focus {
                    outline: none;
                    border-color: #0f172a;
                }

                .cancel-actions {
                    display: flex;
                    gap: 12px;
                }

                .cancel-actions.single {
                    justify-content: center;
                }

                .cancel-actions button {
                    flex: 1;
                    padding: 12px 20px;
                    border-radius: 10px;
                    font-weight: 600;
                    font-size: 14px;
                    cursor: pointer;
                    transition: all 0.2s;
                }

                .btn-secondary {
                    background: #f1f5f9;
                    border: none;
                    color: #475569;
                }

                .btn-secondary:hover {
                    background: #e2e8f0;
                }

                .btn-danger {
                    background: #ef4444;
                    border: none;
                    color: white;
                }

                .btn-danger:hover {
                    background: #dc2626;
                }

                .btn-danger:disabled {
                    opacity: 0.5;
                    cursor: not-allowed;
                }

                .btn-primary {
                    background: #0f172a;
                    border: none;
                    color: white;
                }

                .btn-primary:hover {
                    background: #1e293b;
                }

                .cancel-note {
                    text-align: center;
                    font-size: 13px;
                    color: #64748b;
                    margin-bottom: 24px;
                    line-height: 1.5;
                }
            `}</style>
        </div>
    )
}

export default CancelSubscription
