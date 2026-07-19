import { useState } from 'react'
import { Check } from 'lucide-react'

const ConsentCheckbox = ({
    type = 'recurring_billing',
    planName = 'subscription',
    amount = '',
    billingInterval = 'month',
    onConsentChange,
    required = true
}) => {
    const [consented, setConsented] = useState(false)

    const consentTexts = {
        recurring_billing: `I authorize recurring charges of ${amount} per ${billingInterval} to my payment method for the ${planName} plan. I understand that:
• I will be charged automatically on each billing cycle
• I will receive a reminder email 24 hours before each charge
• I can cancel my subscription at any time from my account dashboard
• Refunds are processed according to the refund policy`,

        terms_of_service: `I have read and agree to the Terms of Service and Privacy Policy.`,

        email_marketing: `I agree to receive product updates and promotional emails. I can unsubscribe at any time.`,
    }

    const handleChange = (e) => {
        const checked = e.target.checked
        setConsented(checked)
        if (onConsentChange) {
            onConsentChange({
                type,
                granted: checked,
                consentText: consentTexts[type],
                version: '2024.01.1',
            })
        }
    }

    return (
        <div className="consent-checkbox">
            <label className={`consent-label ${consented ? 'consented' : ''}`}>
                <div className="checkbox-wrapper">
                    <input
                        type="checkbox"
                        checked={consented}
                        onChange={handleChange}
                        required={required}
                    />
                    <div className={`custom-checkbox ${consented ? 'checked' : ''}`}>
                        {consented && <Check size={14} />}
                    </div>
                </div>
                <div className="consent-text">
                    {type === 'recurring_billing' && (
                        <>
                            <strong>Authorize Recurring Payments</strong>
                            <p>
                                I authorize recurring charges of <strong>{amount}</strong> per {billingInterval} to my payment method.
                                I will receive a reminder 24 hours before each charge and can cancel anytime.
                            </p>
                            <a href="/terms" target="_blank" className="consent-link">
                                View full authorization terms
                            </a>
                        </>
                    )}
                    {type === 'terms_of_service' && (
                        <>
                            I agree to the{' '}
                            <a href="/terms" target="_blank">Terms of Service</a>
                            {' '}and{' '}
                            <a href="/privacy" target="_blank">Privacy Policy</a>
                        </>
                    )}
                    {type === 'email_marketing' && (
                        <>
                            I agree to receive product updates and promotional emails.
                            <span className="optional-badge">Optional</span>
                        </>
                    )}
                </div>
            </label>

            <style>{`
                .consent-checkbox {
                    margin: 16px 0;
                }

                .consent-label {
                    display: flex;
                    gap: 12px;
                    padding: 16px;
                    border: 1px solid #e2e8f0;
                    border-radius: 12px;
                    cursor: pointer;
                    transition: all 0.2s;
                    background: #fafafa;
                }

                .consent-label:hover {
                    border-color: #cbd5e1;
                    background: #f8fafc;
                }

                .consent-label.consented {
                    border-color: #10b981;
                    background: #f0fdf4;
                }

                .checkbox-wrapper {
                    flex-shrink: 0;
                    padding-top: 2px;
                }

                .checkbox-wrapper input {
                    display: none;
                }

                .custom-checkbox {
                    width: 20px;
                    height: 20px;
                    border: 2px solid #cbd5e1;
                    border-radius: 4px;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    transition: all 0.2s;
                }

                .custom-checkbox.checked {
                    background: #10b981;
                    border-color: #10b981;
                    color: white;
                }

                .consent-text {
                    flex: 1;
                    font-size: 13px;
                    color: #334155;
                    line-height: 1.5;
                }

                .consent-text strong {
                    display: block;
                    font-size: 14px;
                    color: #0f172a;
                    margin-bottom: 4px;
                }

                .consent-text p {
                    margin: 0 0 8px 0;
                    color: #64748b;
                }

                .consent-text a {
                    color: #0f172a;
                }

                .consent-link {
                    font-size: 12px;
                    color: #64748b !important;
                    text-decoration: underline;
                }

                .optional-badge {
                    display: inline-block;
                    margin-left: 8px;
                    padding: 2px 8px;
                    background: #f1f5f9;
                    border-radius: 4px;
                    font-size: 11px;
                    color: #64748b;
                    font-weight: 500;
                }
            `}</style>
        </div>
    )
}

export default ConsentCheckbox
