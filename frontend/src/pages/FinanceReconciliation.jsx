import React, { useCallback, useEffect, useState } from 'react'
import { endpoints } from '../lib/api'
import { StatsSkeleton } from '../components/ui/Skeleton'
import { ErrorState } from '../components/LoadingStates'
import EmptyState from '../components/ui/EmptyState'

const BalancedIcon = ({ className }) => (
    <span className={`material-symbols-outlined !text-4xl text-green-500 ${className || ''}`}>verified</span>
)

// Human labels for the backend's discrepancy type constants
// (internal/service/reconciliation.go).
const DISCREPANCY_LABELS = {
    missing_invoice_transaction: 'Missing invoice transaction',
    invoice_amount_mismatch: 'Invoice amount mismatch',
    missing_payment_transaction: 'Missing payment transaction',
    payment_amount_mismatch: 'Payment amount mismatch',
    orphaned_transaction: 'Orphaned transaction',
    missing_in_tigerbeetle: 'Missing in TigerBeetle',
    missing_in_postgres: 'Missing in Postgres',
    tb_amount_mismatch: 'TigerBeetle amount mismatch',
}

const shortId = (id) => (id ? `${id.substring(0, 8)}…` : '—')

// Discrepancy amounts are minor units (cents/paise); the report carries no
// currency, so render them as plain integers rather than guessing a symbol.
const formatMinorUnits = (n) => (typeof n === 'number' ? n.toLocaleString() : '—')

const SummaryCard = ({ title, value, subtitle, children }) => (
    <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <p className="text-sm font-medium text-slate-500 dark:text-slate-400">{title}</p>
        {value !== undefined && <p className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">{value}</p>}
        {subtitle && <p className="mt-1 text-sm text-slate-400 dark:text-slate-500">{subtitle}</p>}
        {children}
    </div>
)

const FinanceReconciliation = () => {
    const [report, setReport] = useState(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    const runReconciliation = useCallback(async () => {
        setLoading(true)
        setError(null)
        try {
            const res = await endpoints.runReconciliation()
            setReport(res.data?.data || null)
        } catch (err) {
            setError(err?.response?.data?.error?.message || 'Failed to run reconciliation')
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        runReconciliation()
    }, [runReconciliation])

    const discrepancies = report?.discrepancies || []
    const totalDiscrepancies = report?.total_discrepancies || 0
    const booksBalanced = report && totalDiscrepancies === 0

    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            {/* Header */}
            <div className="flex flex-wrap items-center justify-between gap-4 pb-8">
                <div className="flex flex-col gap-1">
                    <h1 className="text-3xl font-bold tracking-tight text-slate-900 dark:text-white">Reconciliation</h1>
                    <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal">
                        On-demand check that billing records, the Postgres ledger, and TigerBeetle agree.
                    </p>
                </div>
                <button
                    onClick={runReconciliation}
                    disabled={loading}
                    className="flex h-10 items-center justify-center gap-2 overflow-hidden rounded-lg bg-primary px-4 text-sm font-semibold text-white shadow-sm transition-all hover:bg-primary/90 disabled:opacity-50"
                >
                    <span className={`material-symbols-outlined text-lg ${loading ? 'animate-spin' : ''}`}>refresh</span>
                    <span className="truncate">{loading ? 'Running…' : 'Run again'}</span>
                </button>
            </div>

            {loading ? (
                <StatsSkeleton />
            ) : error ? (
                <ErrorState message={error} onRetry={runReconciliation} />
            ) : report && (
                <div className="flex flex-col gap-8">
                    {/* Summary Cards */}
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                        <SummaryCard
                            title="Invoices Checked"
                            value={(report.invoices_checked || 0).toLocaleString()}
                            subtitle={`${(report.paid_invoices_checked || 0).toLocaleString()} paid invoices`}
                        />
                        <SummaryCard
                            title="Discrepancies"
                            value={totalDiscrepancies.toLocaleString()}
                            subtitle={booksBalanced ? 'Nothing out of place' : `${discrepancies.length.toLocaleString()} listed below`}
                        />
                        <SummaryCard title="TigerBeetle">
                            <div className="mt-2">
                                {report.tb_compared ? (
                                    <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-2.5 py-1 text-xs font-medium text-green-700 dark:bg-green-900/40 dark:text-green-400">
                                        <span className="material-symbols-outlined text-sm">check_circle</span>
                                        Compared
                                    </span>
                                ) : (
                                    <span
                                        title={report.tb_skip_reason || 'Comparison skipped'}
                                        data-testid="tb-skipped-badge"
                                        className="inline-flex cursor-help items-center gap-1 rounded-full bg-amber-100 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/40 dark:text-amber-400"
                                    >
                                        <span className="material-symbols-outlined text-sm">info</span>
                                        Skipped
                                    </span>
                                )}
                            </div>
                            <p className="mt-2 text-sm text-slate-400 dark:text-slate-500">
                                {report.tb_compared
                                    ? `${(report.tb_accounts_checked || 0).toLocaleString()} accounts · ${(report.tb_transfers_checked || 0).toLocaleString()} transfers`
                                    : report.tb_skip_reason || 'Comparison skipped'}
                            </p>
                        </SummaryCard>
                        <SummaryCard
                            title="Last Run"
                            value={report.finished_at ? new Date(report.finished_at).toLocaleTimeString() : '—'}
                            subtitle={report.finished_at ? new Date(report.finished_at).toLocaleDateString() : ''}
                        />
                    </div>

                    {/* Truncation notice */}
                    {report.truncated && (
                        <div className="flex items-center gap-3 rounded-lg bg-amber-50 p-4 text-amber-800 ring-1 ring-inset ring-amber-200 dark:bg-amber-900/30 dark:text-amber-300 dark:ring-amber-900/50">
                            <span className="material-symbols-outlined flex-shrink-0">warning</span>
                            <p className="text-sm font-medium">
                                Showing the first {discrepancies.length.toLocaleString()} of {totalDiscrepancies.toLocaleString()} discrepancies. Resolve these and run again to see the rest.
                            </p>
                        </div>
                    )}

                    {/* Discrepancies */}
                    {booksBalanced ? (
                        <div className="rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900">
                            <EmptyState
                                icon={BalancedIcon}
                                variant="success"
                                title="Books balanced"
                                description="Every invoice and payment agrees with the ledger. Nothing to fix here."
                            />
                        </div>
                    ) : (
                        <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900">
                            <div className="border-b border-slate-200 px-6 py-4 dark:border-slate-800">
                                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Discrepancies</h2>
                                <p className="text-sm text-slate-500 dark:text-slate-400">
                                    Disagreements between billing records and the ledger. Amounts are in minor units.
                                </p>
                            </div>
                            <div className="overflow-x-auto">
                                <table className="w-full text-left text-sm">
                                    <thead className="border-b border-slate-200 bg-slate-50 dark:border-slate-800 dark:bg-slate-950/50">
                                        <tr>
                                            <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Type</th>
                                            <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Invoice</th>
                                            <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Transaction</th>
                                            <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Reference</th>
                                            <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Expected</th>
                                            <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400">Found</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                                        {discrepancies.map((d, i) => (
                                            <tr key={`${d.type}-${d.invoice_id || d.transaction_id || i}`} className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
                                                <td className="px-6 py-4">
                                                    <span className="inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/40 dark:text-red-400">
                                                        {DISCREPANCY_LABELS[d.type] || d.type}
                                                    </span>
                                                </td>
                                                <td className="px-6 py-4 font-mono text-xs text-slate-500 dark:text-slate-400" title={d.invoice_id || undefined}>
                                                    {shortId(d.invoice_id)}
                                                </td>
                                                <td className="px-6 py-4 font-mono text-xs text-slate-500 dark:text-slate-400" title={d.transaction_id || undefined}>
                                                    {shortId(d.transaction_id)}
                                                </td>
                                                <td className="px-6 py-4 font-mono text-xs text-slate-500 dark:text-slate-400" title={d.reference_id || undefined}>
                                                    {shortId(d.reference_id)}
                                                </td>
                                                <td className="px-6 py-4 text-right font-mono text-sm text-slate-900 dark:text-white">
                                                    {formatMinorUnits(d.expected_amount)}
                                                </td>
                                                <td className="px-6 py-4 text-right font-mono text-sm text-slate-900 dark:text-white">
                                                    {formatMinorUnits(d.found_amount)}
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    )
}

export default FinanceReconciliation
