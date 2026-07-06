import React, { useEffect, useState } from 'react'
import SlideOver from '../ui/SlideOver'
import { endpoints } from '../../lib/api'
import { useToast } from '../Toast'

// Mirrors the backend's feature-key rule (internal/service/entitlement.go).
const FEATURE_KEY_RE = /^[A-Za-z0-9][A-Za-z0-9._:-]*$/
const MAX_FEATURE_KEY_LEN = 128

// validateRows returns an error string, or null when the set is valid.
const validateEntitlementRows = (rows) => {
    const seen = new Set()
    for (const row of rows) {
        const key = row.feature_key.trim()
        if (!key) return 'Every entitlement needs a feature key.'
        if (key.length > MAX_FEATURE_KEY_LEN) return `Feature key "${key}" exceeds ${MAX_FEATURE_KEY_LEN} characters.`
        if (!FEATURE_KEY_RE.test(key)) return `Feature key "${key}" may only contain letters, numbers, and . _ : - (must start with a letter or number).`
        if (seen.has(key)) return `Duplicate feature key "${key}".`
        seen.add(key)
        if (row.kind === 'limit') {
            if (row.limit_value === '' || row.limit_value === null) return `"${key}" needs a limit value.`
            const n = Number(row.limit_value)
            if (!Number.isInteger(n) || n < 0) return `"${key}" limit must be a whole number ≥ 0.`
        }
    }
    return null
}

// toApiPayload converts editor rows into the PUT body the backend expects:
// booleans carry bool_value only, limits carry limit_value only.
const entitlementRowsToPayload = (rows) =>
    rows.map((row) => row.kind === 'boolean'
        ? { feature_key: row.feature_key.trim(), kind: 'boolean', bool_value: row.bool_value }
        : { feature_key: row.feature_key.trim(), kind: 'limit', limit_value: Number(row.limit_value) })

const toEditorRows = (ents) => ents.map((ent) => ({
    feature_key: ent.feature_key,
    kind: ent.kind,
    bool_value: ent.kind === 'boolean' ? !!ent.bool_value : true,
    limit_value: ent.kind === 'limit' && ent.limit_value != null ? String(ent.limit_value) : '',
}))

const inputClass = "form-input w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white dark:placeholder:text-slate-500"

const PlanDetail = ({ plan, isOpen, onClose }) => {
    const toast = useToast()
    const [entitlements, setEntitlements] = useState([])
    const [entLoadError, setEntLoadError] = useState(false)
    const [isEditing, setIsEditing] = useState(false)
    const [rows, setRows] = useState([])
    const [validationError, setValidationError] = useState(null)
    const [saving, setSaving] = useState(false)

    useEffect(() => {
        if (!isOpen || !plan?.id) return
        let cancelled = false
        setIsEditing(false)
        setValidationError(null)
        setEntLoadError(false)
        endpoints.getPlanEntitlements(plan.id)
            .then((res) => { if (!cancelled) setEntitlements(res.data?.data || []) })
            .catch(() => {
                if (!cancelled) {
                    setEntitlements([])
                    setEntLoadError(true)
                }
            })
        return () => { cancelled = true }
    }, [isOpen, plan?.id])

    if (!plan) return null

    const price = plan.prices && plan.prices[0]
    const amount = price ? (price.amount / 100).toFixed(2) : '0.00'
    const currency = price ? price.currency.toUpperCase() : 'USD'

    const startEditing = () => {
        setRows(toEditorRows(entitlements))
        setValidationError(null)
        setIsEditing(true)
    }

    const cancelEditing = () => {
        setIsEditing(false)
        setValidationError(null)
    }

    const addRow = () => {
        setRows((prev) => [...prev, { feature_key: '', kind: 'boolean', bool_value: true, limit_value: '' }])
    }

    const removeRow = (index) => {
        setRows((prev) => prev.filter((_, i) => i !== index))
    }

    const updateRow = (index, patch) => {
        setRows((prev) => prev.map((row, i) => (i === index ? { ...row, ...patch } : row)))
    }

    const handleSave = async () => {
        const error = validateEntitlementRows(rows)
        if (error) {
            setValidationError(error)
            return
        }
        setValidationError(null)
        setSaving(true)
        try {
            const res = await endpoints.setPlanEntitlements(plan.id, entitlementRowsToPayload(rows))
            setEntitlements(res.data?.data || [])
            setIsEditing(false)
            toast.success('Entitlements saved')
        } catch (err) {
            toast.error(err?.response?.data?.error?.message || 'Failed to save entitlements')
        } finally {
            setSaving(false)
        }
    }

    return (
        <SlideOver isOpen={isOpen} onClose={onClose} title={plan.name}>
            <div className="flex flex-col gap-6">
                {/* Header Info */}
                <div className="flex flex-col gap-2 pb-6 border-b border-slate-200 dark:border-slate-800">
                    <div className="flex items-center gap-3">
                        <h1 className="text-xl font-bold text-slate-900 dark:text-white">{plan.name}</h1>
                        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${plan.active
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                            : 'bg-gray-100 text-gray-800 dark:bg-gray-700/50 dark:text-gray-300'
                            }`}>
                            {plan.active ? 'Active' : 'Inactive'}
                        </span>
                    </div>
                    <p className="font-mono text-xs text-slate-500 dark:text-slate-400">{plan.id}</p>
                </div>

                {/* Details Section */}
                <div className="grid grid-cols-1 gap-y-4 sm:grid-cols-2">
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Price</p>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {new Intl.NumberFormat('en-US', { style: 'currency', currency: currency }).format(amount)}
                        </p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Billing Interval</p>
                        <p className="text-sm text-slate-900 dark:text-white capitalize">{plan.interval_count > 1 ? `${plan.interval_count} ` : ''}{plan.interval_unit}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Created</p>
                        <p className="text-sm text-slate-900 dark:text-white">{new Date(plan.created_at).toLocaleDateString()}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Currency</p>
                        <p className="text-sm text-slate-900 dark:text-white">{currency}</p>
                    </div>
                    <div className="flex flex-col gap-1">
                        <p className="text-sm text-slate-500 dark:text-slate-400">Code</p>
                        <p className="text-sm font-mono text-slate-900 dark:text-white">{plan.code}</p>
                    </div>
                </div>

                {/* Entitlements */}
                <div>
                    <div className="flex items-center justify-between mb-4">
                        <h3 className="text-base font-semibold leading-tight text-slate-900 dark:text-white">Entitlements</h3>
                        {!isEditing && (
                            <button
                                onClick={startEditing}
                                className="flex items-center gap-1.5 rounded-md bg-slate-100 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700 transition-colors"
                            >
                                <span className="material-symbols-outlined text-sm">edit</span>
                                Edit
                            </button>
                        )}
                    </div>

                    {isEditing ? (
                        <div className="flex flex-col gap-3">
                            {rows.length === 0 && (
                                <p className="text-sm text-slate-500 dark:text-slate-400">
                                    No entitlements. Add one below — saving an empty list removes all entitlements from this plan.
                                </p>
                            )}
                            {rows.map((row, index) => (
                                <div key={index} data-testid={`entitlement-row-${index}`} className="flex items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-slate-800 dark:bg-slate-800/40">
                                    <div className="flex-1 min-w-0">
                                        <input
                                            type="text"
                                            value={row.feature_key}
                                            onChange={(e) => updateRow(index, { feature_key: e.target.value })}
                                            placeholder="feature_key (e.g. api.calls)"
                                            aria-label={`Feature key ${index + 1}`}
                                            className={`${inputClass} font-mono`}
                                        />
                                    </div>
                                    <select
                                        value={row.kind}
                                        onChange={(e) => updateRow(index, { kind: e.target.value })}
                                        aria-label={`Kind ${index + 1}`}
                                        className="form-select rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-sm text-slate-900 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white"
                                    >
                                        <option value="boolean">boolean</option>
                                        <option value="limit">limit</option>
                                    </select>
                                    {row.kind === 'boolean' ? (
                                        <label className="flex items-center gap-1.5 text-sm text-slate-700 dark:text-slate-300 whitespace-nowrap cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={row.bool_value}
                                                onChange={(e) => updateRow(index, { bool_value: e.target.checked })}
                                                aria-label={`Enabled ${index + 1}`}
                                                className="rounded border-slate-300 text-primary focus:ring-primary"
                                            />
                                            Enabled
                                        </label>
                                    ) : (
                                        <input
                                            type="number"
                                            min="0"
                                            step="1"
                                            value={row.limit_value}
                                            onChange={(e) => updateRow(index, { limit_value: e.target.value })}
                                            placeholder="Limit"
                                            aria-label={`Limit value ${index + 1}`}
                                            className={`${inputClass} w-24 flex-none`}
                                        />
                                    )}
                                    <button
                                        onClick={() => removeRow(index)}
                                        aria-label={`Remove entitlement ${index + 1}`}
                                        className="text-slate-400 hover:text-red-500 transition-colors flex-none"
                                    >
                                        <span className="material-symbols-outlined text-lg">delete</span>
                                    </button>
                                </div>
                            ))}

                            {validationError && (
                                <p role="alert" className="text-sm text-red-600 dark:text-red-400">{validationError}</p>
                            )}

                            <div className="flex items-center justify-between">
                                <button
                                    onClick={addRow}
                                    className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm font-medium text-primary hover:bg-primary/10 transition-colors"
                                >
                                    <span className="material-symbols-outlined text-lg">add</span>
                                    Add entitlement
                                </button>
                                <div className="flex gap-2">
                                    <button
                                        onClick={cancelEditing}
                                        disabled={saving}
                                        className="rounded-md bg-slate-100 px-3 py-1.5 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-50 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700 transition-colors"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        onClick={handleSave}
                                        disabled={saving}
                                        className="rounded-md bg-primary px-3 py-1.5 text-sm font-semibold text-white shadow-sm hover:bg-primary/90 disabled:opacity-50 transition-colors"
                                    >
                                        {saving ? 'Saving…' : 'Save entitlements'}
                                    </button>
                                </div>
                            </div>
                        </div>
                    ) : entLoadError ? (
                        <p className="text-sm text-red-600 dark:text-red-400">Failed to load entitlements.</p>
                    ) : entitlements.length === 0 ? (
                        <p className="text-sm text-slate-500 dark:text-slate-400">No entitlements configured for this plan.</p>
                    ) : (
                        <div className="flex flex-col gap-2">
                            {entitlements.map((ent) => (
                                <div key={ent.feature_key} className="flex items-center gap-3">
                                    <span className={`material-symbols-outlined text-sm ${ent.kind === 'boolean' && !ent.bool_value ? 'text-slate-400' : 'text-green-500'}`}>
                                        {ent.kind === 'boolean' && !ent.bool_value ? 'close' : 'check'}
                                    </span>
                                    <p className="text-sm font-mono text-slate-700 dark:text-slate-300">{ent.feature_key}</p>
                                    {ent.kind === 'limit' && (
                                        <span className="text-xs text-slate-500 dark:text-slate-400">limit: {ent.limit_value?.toLocaleString()}</span>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Metadata */}
                <div>
                    <h3 className="text-base font-semibold leading-tight text-slate-900 dark:text-white mb-4">Metadata</h3>
                    <div className="rounded-lg bg-slate-100 dark:bg-slate-800 p-4 overflow-x-auto">
                        <pre className="font-mono text-xs text-slate-800 dark:text-slate-300">
                            {JSON.stringify(plan.metadata || {}, null, 2)}
                        </pre>
                    </div>
                </div>
            </div>
        </SlideOver>
    )
}

export default PlanDetail
