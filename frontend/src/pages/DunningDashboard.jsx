import React, { useEffect, useState } from 'react';
import { endpoints } from '../lib/api';

const StatCard = ({ title, value, subtitle }) => (
    <div className="rounded-xl border border-gray-100 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <p className="text-sm font-medium text-gray-500 dark:text-zinc-400">{title}</p>
        <p className="mt-2 text-3xl font-bold text-gray-900 dark:text-white">{value}</p>
        {subtitle && <p className="mt-1 text-sm text-gray-400 dark:text-zinc-500">{subtitle}</p>}
    </div>
);

const formatMoney = (amount, currency) => {
    try {
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency,
            maximumFractionDigits: 0,
        }).format(amount / 100);
    } catch {
        return `${currency} ${(amount / 100).toFixed(0)}`;
    }
};

// Last 12 calendar months as "YYYY-MM", oldest first (matches the API window).
const lastTwelveMonths = () => {
    const months = [];
    const d = new Date();
    d.setDate(1);
    d.setMonth(d.getMonth() - 11);
    for (let i = 0; i < 12; i++) {
        months.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`);
        d.setMonth(d.getMonth() + 1);
    }
    return months;
};

const DunningDashboard = () => {
    const [overview, setOverview] = useState(null);
    const [weights, setWeights] = useState([]);
    const [history, setHistory] = useState([]);
    const [recovered, setRecovered] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchData = async () => {
            try {
                const [overviewRes, weightsRes, historyRes, recoveredRes] = await Promise.all([
                    endpoints.getDunningOverview(),
                    endpoints.getDunningWeights(),
                    endpoints.getDunningHistory({ limit: 50 }),
                    endpoints.getDunningRecovered(),
                ]);
                setOverview(overviewRes.data);
                setWeights(weightsRes.data?.data || []);
                setHistory(historyRes.data?.data || []);
                setRecovered(recoveredRes.data);
            } catch (err) {
                console.error('Failed to fetch dunning data:', err);
            } finally {
                setLoading(false);
            }
        };
        fetchData();
    }, []);

    if (loading) {
        return (
            <div className="flex h-64 items-center justify-center">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-900 border-t-transparent dark:border-white"></div>
            </div>
        );
    }

    // Group weights by context key to find winning arm per context
    const contextGroups = {};
    weights.forEach(w => {
        if (!contextGroups[w.context_key]) {
            contextGroups[w.context_key] = [];
        }
        contextGroups[w.context_key].push(w);
    });

    // Recovered revenue: pick the currency with the largest recovered total as
    // the headline; any other currencies are listed in the subtitle.
    const recoveredTotals = recovered?.recovered_amount_total || {};
    const currencies = Object.keys(recoveredTotals).sort((a, b) => recoveredTotals[b] - recoveredTotals[a]);
    const primaryCurrency = currencies[0] || 'USD';
    const recoveredValue = currencies.length > 0
        ? formatMoney(recoveredTotals[primaryCurrency], primaryCurrency)
        : formatMoney(0, 'USD');
    const recoveredSubtitleParts = [`${recovered?.recovered_count || 0} invoices`];
    if (recovered?.recovered_count > 0) {
        recoveredSubtitleParts.push(`avg ${(recovered?.avg_attempts || 0).toFixed(1)} attempts`);
    }
    if (currencies.length > 1) {
        recoveredSubtitleParts.push(
            `+ ${currencies.slice(1).map((c) => formatMoney(recoveredTotals[c], c)).join(', ')}`
        );
    }

    // Monthly recovered-revenue series (headline currency drives bar heights).
    const months = lastTwelveMonths();
    const monthlyByMonth = {};
    (recovered?.monthly || []).forEach((b) => {
        if (!monthlyByMonth[b.month]) {
            monthlyByMonth[b.month] = { amount: 0, count: 0 };
        }
        if (b.currency === primaryCurrency) {
            monthlyByMonth[b.month].amount += b.amount;
        }
        monthlyByMonth[b.month].count += b.count;
    });
    const maxMonthlyAmount = Math.max(1, ...months.map((m) => monthlyByMonth[m]?.amount || 0));

    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Smart Dunning</h1>
                <p className="mt-1 text-sm text-gray-500 dark:text-zinc-400">
                    RL-based payment retry optimization — epsilon-greedy multi-armed bandit
                </p>
            </div>

            {/* Overview Cards */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <StatCard
                    title="Recovered Revenue"
                    value={recoveredValue}
                    subtitle={recoveredSubtitleParts.join(' · ')}
                />
                <StatCard
                    title="Total Retries"
                    value={overview?.total_retries || 0}
                />
                <StatCard
                    title="Successful Recoveries"
                    value={overview?.total_successes || 0}
                />
                <StatCard
                    title="Success Rate"
                    value={overview?.success_rate ? `${(overview.success_rate * 100).toFixed(1)}%` : '0%'}
                />
            </div>

            {/* Recovered Revenue by Month */}
            <div className="rounded-xl border border-gray-100 bg-white dark:border-zinc-800 dark:bg-zinc-900">
                <div className="border-b border-gray-100 px-6 py-4 dark:border-zinc-800">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Recovered Revenue by Month</h2>
                    <p className="text-sm text-gray-500 dark:text-zinc-400">
                        Revenue attributed to the retry/dunning engine over the last 12 months
                        {currencies.length > 1 ? ` (${primaryCurrency} only)` : ''}
                    </p>
                </div>
                {(recovered?.recovered_count || 0) === 0 ? (
                    <div className="px-6 py-12 text-center text-gray-400 dark:text-zinc-500">
                        No recovered payments yet. Recoveries appear when a failed invoice is paid after retries.
                    </div>
                ) : (
                    <div className="flex h-48 items-end gap-2 px-6 pb-3 pt-6">
                        {months.map((month) => {
                            const bucket = monthlyByMonth[month] || { amount: 0, count: 0 };
                            const heightPct = Math.round((bucket.amount / maxMonthlyAmount) * 100);
                            return (
                                <div
                                    key={month}
                                    className="flex h-full flex-1 flex-col items-center justify-end gap-1"
                                    title={`${month}: ${formatMoney(bucket.amount, primaryCurrency)} (${bucket.count} invoices)`}
                                >
                                    <div
                                        data-testid={`recovered-bar-${month}`}
                                        className={bucket.amount > 0
                                            ? 'w-full rounded-t bg-green-500 dark:bg-green-400'
                                            : 'w-full rounded-t bg-gray-100 dark:bg-zinc-800'}
                                        style={{ height: bucket.amount > 0 ? `${Math.max(heightPct, 3)}%` : '2px' }}
                                    />
                                    <span className="text-[10px] text-gray-400 dark:text-zinc-500">{month.slice(5)}</span>
                                </div>
                            );
                        })}
                    </div>
                )}
            </div>

            {/* Arm Performance Table */}
            <div className="rounded-xl border border-gray-100 bg-white dark:border-zinc-800 dark:bg-zinc-900">
                <div className="border-b border-gray-100 px-6 py-4 dark:border-zinc-800">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Arm Performance by Context</h2>
                    <p className="text-sm text-gray-500 dark:text-zinc-400">
                        Each context (currency:error_code) learns independently which retry interval works best
                    </p>
                </div>
                {Object.keys(contextGroups).length === 0 ? (
                    <div className="px-6 py-12 text-center text-gray-400 dark:text-zinc-500">
                        No data yet. Weights will appear after the first retry outcomes are recorded.
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead>
                                <tr className="border-b border-gray-100 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:border-zinc-800 dark:text-zinc-400">
                                    <th className="px-6 py-3">Context</th>
                                    <th className="px-6 py-3">Arm</th>
                                    <th className="px-6 py-3">Avg Reward</th>
                                    <th className="px-6 py-3">Samples</th>
                                    <th className="px-6 py-3">Status</th>
                                </tr>
                            </thead>
                            <tbody>
                                {Object.entries(contextGroups).map(([contextKey, arms]) => {
                                    const bestArm = arms.reduce((best, arm) =>
                                        arm.average_reward > best.average_reward ? arm : best, arms[0]);
                                    return arms.map((arm, idx) => (
                                        <tr key={`${contextKey}-${arm.action_id}`}
                                            className="border-b border-gray-50 dark:border-zinc-800/50">
                                            {idx === 0 && (
                                                <td className="px-6 py-3 font-mono text-sm text-gray-700 dark:text-zinc-300"
                                                    rowSpan={arms.length}>
                                                    {contextKey}
                                                </td>
                                            )}
                                            <td className="px-6 py-3 font-mono text-sm text-gray-900 dark:text-white">
                                                {arm.action_id}
                                            </td>
                                            <td className="px-6 py-3 text-sm">
                                                <span className={arm.average_reward > 0.5
                                                    ? 'text-green-600 dark:text-green-400'
                                                    : arm.average_reward > 0.2
                                                        ? 'text-yellow-600 dark:text-yellow-400'
                                                        : 'text-gray-500 dark:text-zinc-400'}>
                                                    {(arm.average_reward * 100).toFixed(1)}%
                                                </span>
                                            </td>
                                            <td className="px-6 py-3 text-sm text-gray-600 dark:text-zinc-400">
                                                {arm.sample_count}
                                            </td>
                                            <td className="px-6 py-3 text-sm">
                                                {arm.action_id === bestArm.action_id && arm.sample_count > 0 ? (
                                                    <span className="inline-flex items-center rounded-full bg-green-50 px-2 py-1 text-xs font-medium text-green-700 dark:bg-green-900/20 dark:text-green-400">
                                                        Best
                                                    </span>
                                                ) : null}
                                            </td>
                                        </tr>
                                    ));
                                })}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Recent History */}
            <div className="rounded-xl border border-gray-100 bg-white dark:border-zinc-800 dark:bg-zinc-900">
                <div className="border-b border-gray-100 px-6 py-4 dark:border-zinc-800">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Recent Retry History</h2>
                </div>
                {history.length === 0 ? (
                    <div className="px-6 py-12 text-center text-gray-400 dark:text-zinc-500">
                        No retry history yet.
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead>
                                <tr className="border-b border-gray-100 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:border-zinc-800 dark:text-zinc-400">
                                    <th className="px-6 py-3">Time</th>
                                    <th className="px-6 py-3">Invoice</th>
                                    <th className="px-6 py-3">Context</th>
                                    <th className="px-6 py-3">Action</th>
                                    <th className="px-6 py-3">Outcome</th>
                                </tr>
                            </thead>
                            <tbody>
                                {history.map((h) => (
                                    <tr key={h.id} className="border-b border-gray-50 dark:border-zinc-800/50">
                                        <td className="px-6 py-3 text-sm text-gray-500 dark:text-zinc-400">
                                            {new Date(h.created_at).toLocaleString()}
                                        </td>
                                        <td className="px-6 py-3 font-mono text-sm text-gray-700 dark:text-zinc-300">
                                            {h.invoice_id?.substring(0, 8)}...
                                        </td>
                                        <td className="px-6 py-3 font-mono text-sm text-gray-600 dark:text-zinc-400">
                                            {h.context_key}
                                        </td>
                                        <td className="px-6 py-3 font-mono text-sm text-gray-900 dark:text-white">
                                            {h.action_id}
                                        </td>
                                        <td className="px-6 py-3">
                                            {h.outcome === 'success' ? (
                                                <span className="inline-flex items-center rounded-full bg-green-50 px-2 py-1 text-xs font-medium text-green-700 dark:bg-green-900/20 dark:text-green-400">
                                                    Success
                                                </span>
                                            ) : (
                                                <span className="inline-flex items-center rounded-full bg-red-50 px-2 py-1 text-xs font-medium text-red-700 dark:bg-red-900/20 dark:text-red-400">
                                                    Failed
                                                </span>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </div>
    );
};

export default DunningDashboard;
