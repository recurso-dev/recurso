import { useState, useEffect } from 'react'
import { Save, Check, AlertCircle, Building2, FileText } from 'lucide-react'
import api from '../../lib/api'

const GSTSettings = () => {
    const [config, setConfig] = useState({
        gstin: '',
        state_code: '',
        state_name: '',
        sac_code: '998314',
        gst_rate: 18,
        pan: '',
        legal_name: '',
        trade_name: '',
        address: '',
        has_lut: false,
    })
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [validating, setValidating] = useState(false)
    const [validation, setValidation] = useState(null)
    const [message, setMessage] = useState(null)

    useEffect(() => {
        fetchConfig()
    }, [])

    const fetchConfig = async () => {
        try {
            const response = await api.get('/settings/gst')
            if (response.data.data) {
                setConfig(response.data.data)
            }
        } catch (error) {
            console.error('Failed to fetch GST config:', error)
        } finally {
            setLoading(false)
        }
    }

    const validateGSTIN = async () => {
        if (!config.gstin || config.gstin.length !== 15) {
            setValidation({ valid: false, message: 'GSTIN must be 15 characters' })
            return
        }

        setValidating(true)
        try {
            const response = await api.post('/settings/gst/validate', {
                gstin: config.gstin,
            })
            setValidation(response.data)
            if (response.data.valid) {
                setConfig(prev => ({
                    ...prev,
                    state_code: response.data.state_code,
                    state_name: response.data.state_name,
                    pan: response.data.pan,
                }))
            }
        } catch (error) {
            setValidation({ valid: false, message: 'Validation failed' })
        } finally {
            setValidating(false)
        }
    }

    const saveConfig = async () => {
        setSaving(true)
        setMessage(null)
        try {
            await api.put('/settings/gst', config)
            setMessage({ type: 'success', text: 'GST configuration saved successfully' })
        } catch (error) {
            setMessage({ type: 'error', text: 'Failed to save configuration' })
        } finally {
            setSaving(false)
        }
    }

    if (loading) {
        return (
            <div className="settings-page">
                <div className="loading">Loading GST settings...</div>
            </div>
        )
    }

    return (
        <div className="settings-page">
            <div className="settings-header">
                <div className="settings-icon">
                    <Building2 size={24} />
                </div>
                <div>
                    <h1>GST Configuration</h1>
                    <p>Configure your GST details for invoice generation</p>
                </div>
            </div>

            {message && (
                <div className={`message ${message.type}`}>
                    {message.type === 'success' ? <Check size={18} /> : <AlertCircle size={18} />}
                    {message.text}
                </div>
            )}

            <div className="settings-section">
                <h2>GSTIN Details</h2>

                <div className="form-row">
                    <div className="form-group gstin-group">
                        <label>GSTIN</label>
                        <div className="input-with-button">
                            <input
                                type="text"
                                value={config.gstin}
                                onChange={(e) => setConfig({ ...config, gstin: e.target.value.toUpperCase() })}
                                placeholder="22AAAAA0000A1Z5"
                                maxLength={15}
                            />
                            <button
                                className="btn-validate"
                                onClick={validateGSTIN}
                                disabled={validating || !config.gstin}
                            >
                                {validating ? 'Validating...' : 'Validate'}
                            </button>
                        </div>
                        {validation && (
                            <div className={`validation-result ${validation.valid ? 'valid' : 'invalid'}`}>
                                {validation.valid ? <Check size={14} /> : <AlertCircle size={14} />}
                                {validation.message}
                            </div>
                        )}
                    </div>
                </div>

                <div className="form-row">
                    <div className="form-group">
                        <label>State Code</label>
                        <input
                            type="text"
                            value={config.state_code}
                            readOnly
                            placeholder="Auto-filled from GSTIN"
                        />
                    </div>
                    <div className="form-group">
                        <label>State Name</label>
                        <input
                            type="text"
                            value={config.state_name}
                            readOnly
                            placeholder="Auto-filled from GSTIN"
                        />
                    </div>
                </div>

                <div className="form-row">
                    <div className="form-group">
                        <label>PAN</label>
                        <input
                            type="text"
                            value={config.pan}
                            onChange={(e) => setConfig({ ...config, pan: e.target.value.toUpperCase() })}
                            placeholder="AAAAA0000A"
                            maxLength={10}
                        />
                    </div>
                </div>
            </div>

            <div className="settings-section">
                <h2>Business Details</h2>

                <div className="form-row">
                    <div className="form-group">
                        <label>Legal Name</label>
                        <input
                            type="text"
                            value={config.legal_name}
                            onChange={(e) => setConfig({ ...config, legal_name: e.target.value })}
                            placeholder="As per registration"
                        />
                    </div>
                    <div className="form-group">
                        <label>Trade Name</label>
                        <input
                            type="text"
                            value={config.trade_name}
                            onChange={(e) => setConfig({ ...config, trade_name: e.target.value })}
                            placeholder="Brand name"
                        />
                    </div>
                </div>

                <div className="form-group">
                    <label>Registered Address</label>
                    <textarea
                        value={config.address}
                        onChange={(e) => setConfig({ ...config, address: e.target.value })}
                        placeholder="Full registered address"
                        rows={3}
                    />
                </div>
            </div>

            <div className="settings-section">
                <h2>Tax Settings</h2>

                <div className="form-row">
                    <div className="form-group">
                        <label>SAC Code</label>
                        <input
                            type="text"
                            value={config.sac_code}
                            onChange={(e) => setConfig({ ...config, sac_code: e.target.value })}
                            placeholder="998314"
                        />
                        <span className="hint">Default for SaaS: 998314</span>
                    </div>
                    <div className="form-group">
                        <label>GST Rate (%)</label>
                        <input
                            type="number"
                            value={config.gst_rate}
                            onChange={(e) => setConfig({ ...config, gst_rate: parseFloat(e.target.value) })}
                            min={0}
                            max={28}
                        />
                        <span className="hint">Standard rate for software: 18%</span>
                    </div>
                </div>

                <div className="form-group checkbox-group">
                    <label className="checkbox-label">
                        <input
                            type="checkbox"
                            checked={config.has_lut}
                            onChange={(e) => setConfig({ ...config, has_lut: e.target.checked })}
                        />
                        <span>LUT (Letter of Undertaking) for exports</span>
                    </label>
                    <span className="hint">Enable for 0% GST on export of services</span>
                </div>
            </div>

            <div className="settings-actions">
                <button className="btn-save" onClick={saveConfig} disabled={saving}>
                    <Save size={18} />
                    {saving ? 'Saving...' : 'Save Configuration'}
                </button>
            </div>

            <style jsx>{`
                .settings-page {
                    max-width: 800px;
                    margin: 0 auto;
                    padding: 32px;
                }

                .settings-header {
                    display: flex;
                    align-items: center;
                    gap: 16px;
                    margin-bottom: 32px;
                }

                .settings-icon {
                    width: 56px;
                    height: 56px;
                    background: linear-gradient(135deg, #10b981 0%, #059669 100%);
                    border-radius: 14px;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    color: white;
                }

                .settings-header h1 {
                    margin: 0;
                    font-size: 24px;
                    color: #0f172a;
                }

                .settings-header p {
                    margin: 4px 0 0;
                    color: #64748b;
                }

                .message {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    padding: 12px 16px;
                    border-radius: 10px;
                    margin-bottom: 24px;
                    font-size: 14px;
                }

                .message.success {
                    background: #dcfce7;
                    color: #166534;
                }

                .message.error {
                    background: #fee2e2;
                    color: #991b1b;
                }

                .settings-section {
                    background: white;
                    border: 1px solid #e2e8f0;
                    border-radius: 12px;
                    padding: 24px;
                    margin-bottom: 24px;
                }

                .settings-section h2 {
                    margin: 0 0 20px;
                    font-size: 16px;
                    color: #334155;
                    font-weight: 600;
                }

                .form-row {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
                    gap: 20px;
                    margin-bottom: 20px;
                }

                .form-group {
                    display: flex;
                    flex-direction: column;
                }

                .form-group label {
                    font-size: 13px;
                    font-weight: 500;
                    color: #475569;
                    margin-bottom: 6px;
                }

                .form-group input,
                .form-group textarea {
                    padding: 10px 14px;
                    border: 1px solid #e2e8f0;
                    border-radius: 8px;
                    font-size: 14px;
                    transition: all 0.2s;
                }

                .form-group input:focus,
                .form-group textarea:focus {
                    outline: none;
                    border-color: #10b981;
                    box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
                }

                .form-group input[readonly] {
                    background: #f8fafc;
                    cursor: not-allowed;
                }

                .hint {
                    font-size: 12px;
                    color: #94a3b8;
                    margin-top: 4px;
                }

                .input-with-button {
                    display: flex;
                    gap: 8px;
                }

                .input-with-button input {
                    flex: 1;
                }

                .btn-validate {
                    padding: 10px 16px;
                    background: #f1f5f9;
                    border: 1px solid #e2e8f0;
                    border-radius: 8px;
                    font-size: 13px;
                    font-weight: 500;
                    color: #475569;
                    cursor: pointer;
                    transition: all 0.2s;
                }

                .btn-validate:hover:not(:disabled) {
                    background: #e2e8f0;
                }

                .btn-validate:disabled {
                    opacity: 0.5;
                    cursor: not-allowed;
                }

                .validation-result {
                    display: flex;
                    align-items: center;
                    gap: 6px;
                    font-size: 12px;
                    margin-top: 6px;
                }

                .validation-result.valid {
                    color: #16a34a;
                }

                .validation-result.invalid {
                    color: #dc2626;
                }

                .checkbox-group {
                    margin-top: 8px;
                }

                .checkbox-label {
                    display: flex;
                    align-items: center;
                    gap: 10px;
                    cursor: pointer;
                }

                .checkbox-label input {
                    width: 18px;
                    height: 18px;
                    accent-color: #10b981;
                }

                .settings-actions {
                    display: flex;
                    justify-content: flex-end;
                    padding-top: 16px;
                }

                .btn-save {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    padding: 12px 24px;
                    background: #10b981;
                    color: white;
                    border: none;
                    border-radius: 10px;
                    font-size: 14px;
                    font-weight: 600;
                    cursor: pointer;
                    transition: all 0.2s;
                }

                .btn-save:hover:not(:disabled) {
                    background: #059669;
                }

                .btn-save:disabled {
                    opacity: 0.5;
                    cursor: not-allowed;
                }
            `}</style>
        </div>
    )
}

export default GSTSettings
