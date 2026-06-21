import React, { useState, useEffect } from 'react';
import { endpoints } from '../lib/api';
import { Share2, Plus, X, Users, DollarSign, Clock, CheckCircle, Award } from 'lucide-react';

function Referrals() {
    const [referrals, setReferrals] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showCreate, setShowCreate] = useState(false);
    const [creating, setCreating] = useState(false);
    const [customers, setCustomers] = useState([]);
    const [form, setForm] = useState({
        referrer_id: '',
        referred_id: '',
        reward_amount: 500,
        currency: 'USD',
    });

    useEffect(() => {
        fetchReferrals();
        fetchCustomers();
    }, []);

    const fetchReferrals = async () => {
        try {
            setLoading(true);
            const response = await endpoints.getReferrals();
            setReferrals(response.data?.data || response.data || []);
        } catch (error) {
            console.error('Error fetching referrals:', error);
        } finally {
            setLoading(false);
        }
    };

    const fetchCustomers = async () => {
        try {
            const response = await endpoints.getCustomers();
            setCustomers(response.data?.data || []);
        } catch (error) {
            console.error('Error fetching customers:', error);
        }
    };

    const handleCreate = async (e) => {
        e.preventDefault();
        try {
            setCreating(true);
            await endpoints.createReferral({
                referrer_id: form.referrer_id,
                referred_id: form.referred_id,
                reward_amount: parseInt(form.reward_amount),
                currency: form.currency,
            });
            setShowCreate(false);
            setForm({ referrer_id: '', referred_id: '', reward_amount: 500, currency: 'USD' });
            fetchReferrals();
        } catch (error) {
            console.error('Error creating referral:', error);
        } finally {
            setCreating(false);
        }
    };

    const getStatusColor = (status) => {
        switch (status) {
            case 'rewarded': return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400';
            case 'qualified': return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400';
            default: return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400';
        }
    };

    const getStatusIcon = (status) => {
        switch (status) {
            case 'rewarded': return <Award className="w-3.5 h-3.5" />;
            case 'qualified': return <CheckCircle className="w-3.5 h-3.5" />;
            default: return <Clock className="w-3.5 h-3.5" />;
        }
    };

    const totalRewards = referrals
        .filter(r => r.status === 'rewarded')
        .reduce((acc, curr) => acc + (curr.reward_amount || 0), 0);

    return (
        <div className="animate-fade-in">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Referral Program</h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        Manage your customer referral program and track rewards.
                    </p>
                </div>
                <button
                    onClick={() => setShowCreate(true)}
                    className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white font-medium rounded-lg hover:bg-primary/90 transition-all hover:scale-105 active:scale-95"
                >
                    <Plus className="w-4 h-4" />
                    Create Referral
                </button>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <div className="flex items-start justify-between">
                        <div>
                            <p className="text-sm font-medium text-slate-500 dark:text-slate-400">Total Referrals</p>
                            <h3 className="text-3xl font-semibold mt-2 text-slate-900 dark:text-white">
                                {referrals.length}
                            </h3>
                        </div>
                        <div className="p-2 rounded-lg bg-slate-100 dark:bg-slate-800">
                            <Users className="w-5 h-5 text-slate-600 dark:text-slate-300" />
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <div className="flex items-start justify-between">
                        <div>
                            <p className="text-sm font-medium text-slate-500 dark:text-slate-400">Total Rewards Paid</p>
                            <h3 className="text-3xl font-semibold mt-2 text-emerald-600 dark:text-emerald-400">
                                ${(totalRewards / 100).toFixed(2)}
                            </h3>
                        </div>
                        <div className="p-2 rounded-lg bg-emerald-100 dark:bg-emerald-900/30">
                            <DollarSign className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <div className="flex items-start justify-between">
                        <div>
                            <p className="text-sm font-medium text-slate-500 dark:text-slate-400">Pending</p>
                            <h3 className="text-3xl font-semibold mt-2 text-amber-600 dark:text-amber-400">
                                {referrals.filter(r => r.status === 'pending').length}
                            </h3>
                        </div>
                        <div className="p-2 rounded-lg bg-amber-100 dark:bg-amber-900/30">
                            <Clock className="w-5 h-5 text-amber-600 dark:text-amber-400" />
                        </div>
                    </div>
                </div>
            </div>

            {/* Table */}
            <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden">
                {loading ? (
                    <div className="p-8 text-center">
                        <div className="w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto" />
                    </div>
                ) : referrals.length === 0 ? (
                    <div className="p-12 text-center">
                        <div className="w-16 h-16 rounded-2xl bg-slate-100 dark:bg-slate-800 flex items-center justify-center mx-auto mb-4">
                            <Share2 className="w-8 h-8 text-slate-400" />
                        </div>
                        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
                            No referrals yet
                        </h3>
                        <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
                            Create your first referral to start tracking rewards
                        </p>
                        <button
                            onClick={() => setShowCreate(true)}
                            className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white font-medium rounded-lg hover:bg-primary/90"
                        >
                            <Plus className="w-4 h-4" />
                            Create Referral
                        </button>
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Code</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Status</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Reward</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Created</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                                {referrals.map((referral, index) => (
                                    <tr
                                        key={referral.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors animate-fade-up"
                                        style={{ animationDelay: `${index * 0.05}s` }}
                                    >
                                        <td className="px-6 py-4">
                                            <span className="font-mono text-sm px-2.5 py-1 rounded-md bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-white">
                                                {referral.code}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${getStatusColor(referral.status)}`}>
                                                {getStatusIcon(referral.status)}
                                                {referral.status}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 text-sm font-medium text-slate-900 dark:text-white">
                                            ${(referral.reward_amount / 100).toFixed(2)} {referral.currency || 'USD'}
                                        </td>
                                        <td className="px-6 py-4 text-sm text-slate-500 dark:text-slate-400">
                                            {new Date(referral.created_at).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Create Referral Modal */}
            {showCreate && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                    <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 w-full max-w-md mx-4 shadow-xl">
                        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700">
                            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Create Referral</h2>
                            <button onClick={() => setShowCreate(false)} className="p-1 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors">
                                <X className="w-5 h-5 text-slate-500" />
                            </button>
                        </div>
                        <form onSubmit={handleCreate} className="p-6 space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Referrer (Who referred)</label>
                                <select
                                    required
                                    value={form.referrer_id}
                                    onChange={(e) => setForm({ ...form, referrer_id: e.target.value })}
                                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
                                >
                                    <option value="">Select referrer...</option>
                                    {customers.map(c => (
                                        <option key={c.id} value={c.id}>{c.name} ({c.email})</option>
                                    ))}
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Referred (New customer)</label>
                                <select
                                    required
                                    value={form.referred_id}
                                    onChange={(e) => setForm({ ...form, referred_id: e.target.value })}
                                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
                                >
                                    <option value="">Select referred customer...</option>
                                    {customers.filter(c => c.id !== form.referrer_id).map(c => (
                                        <option key={c.id} value={c.id}>{c.name} ({c.email})</option>
                                    ))}
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Reward Amount (cents)</label>
                                <input
                                    type="number"
                                    min="0"
                                    required
                                    value={form.reward_amount}
                                    onChange={(e) => setForm({ ...form, reward_amount: e.target.value })}
                                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
                                />
                                <p className="text-xs text-slate-500 mt-1">500 = $5.00</p>
                            </div>
                            <div className="flex justify-end gap-3 pt-2">
                                <button
                                    type="button"
                                    onClick={() => setShowCreate(false)}
                                    className="px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    disabled={creating}
                                    className="px-4 py-2 text-sm font-medium bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50 transition-all"
                                >
                                    {creating ? 'Creating...' : 'Create Referral'}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
}

export default Referrals;
