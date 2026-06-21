import React, { useEffect, useState } from 'react';
import { endpoints } from '../lib/api';

const StatCard = ({ title, value, subtitle }) => (
    <div className="rounded-xl border border-gray-100 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <p className="text-sm font-medium text-gray-500 dark:text-zinc-400">{title}</p>
        <p className="mt-2 text-3xl font-bold text-gray-900 dark:text-white">{value}</p>
        {subtitle && <p className="mt-1 text-sm text-gray-400 dark:text-zinc-500">{subtitle}</p>}
    </div>
);

const DunningDashboard = () => {
    const [overview, setOverview] = useState(null);
    const [weights, setWeights] = useState([]);
    const [history, setHistory] = useState([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchData = async () => {
            try {
                const [overviewRes, weightsRes, historyRes] = await Promise.all([
                    endpoints.getDunningOverview(),
                    endpoints.getDunningWeights(),
                    endpoints.getDunningHistory({ limit: 50 }),
                ]);
                setOverview(overviewRes.data);
                setWeights(weightsRes.data?.data || []);
                setHistory(historyRes.data?.data || []);
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

    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Smart Dunning</h1>
                <p className="mt-1 text-sm text-gray-500 dark:text-zinc-400">
                    RL-based payment retry optimization — epsilon-greedy multi-armed bandit
                </p>
            </div>

            {/* Overview Cards */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
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
