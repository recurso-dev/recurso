import React, { useState, useEffect } from 'react';
import { endpoints } from '../lib/api';
import { Gift, Plus, X, Package, CheckCircle, Clock } from 'lucide-react';

function Gifts() {
    const [gifts, setGifts] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showCreate, setShowCreate] = useState(false);
    const [creating, setCreating] = useState(false);
    const [plans, setPlans] = useState([]);
    const [customers, setCustomers] = useState([]);
    const [form, setForm] = useState({
        buyer_customer_id: '',
        plan_id: '',
        duration_months: 12,
    });

    useEffect(() => {
        fetchGifts();
        fetchPlans();
        fetchCustomers();
    }, []);

    const fetchGifts = async () => {
        try {
            setLoading(true);
            const response = await endpoints.getGifts();
            setGifts(response.data?.data || response.data || []);
        } catch (error) {
            console.error('Error fetching gifts:', error);
        } finally {
            setLoading(false);
        }
    };

    const fetchPlans = async () => {
        try {
            const response = await endpoints.getPlans();
            setPlans(response.data?.data || []);
        } catch (error) {
            console.error('Error fetching plans:', error);
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
            await endpoints.purchaseGift({
                buyer_customer_id: form.buyer_customer_id,
                plan_id: form.plan_id,
                duration_months: parseInt(form.duration_months),
            });
            setShowCreate(false);
            setForm({ buyer_customer_id: '', plan_id: '', duration_months: 12 });
            fetchGifts();
        } catch (error) {
            console.error('Error creating gift:', error);
        } finally {
            setCreating(false);
        }
    };

    const getStatusColor = (status) => {
        switch (status) {
            case 'redeemed': return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400';
            case 'purchased': return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400';
            default: return 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';
        }
    };

    const getStatusIcon = (status) => {
        switch (status) {
            case 'redeemed': return <CheckCircle className="w-3.5 h-3.5" />;
            case 'purchased': return <Clock className="w-3.5 h-3.5" />;
            default: return <Package className="w-3.5 h-3.5" />;
        }
    };

    return (
        <div className="animate-fade-in">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Gift Subscriptions</h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        Manage purchased gift subscriptions and track redemptions.
                    </p>
                </div>
                <button
                    onClick={() => setShowCreate(true)}
                    className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white font-medium rounded-lg hover:bg-primary/90 transition-all hover:scale-105 active:scale-95"
                >
                    <Plus className="w-4 h-4" />
                    Create Gift
                </button>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <div className="flex items-start justify-between">
                        <div>
                            <p className="text-sm font-medium text-slate-500 dark:text-slate-400">Total Gifts Sold</p>
                            <h3 className="text-3xl font-semibold mt-2 text-slate-900 dark:text-white">
                                {gifts.length}
                            </h3>
                        </div>
                        <div className="p-2 rounded-lg bg-slate-100 dark:bg-slate-800">
                            <Gift className="w-5 h-5 text-slate-600 dark:text-slate-300" />
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <div className="flex items-start justify-between">
                        <div>
                            <p className="text-sm font-medium text-slate-500 dark:text-slate-400">Redeemed</p>
                            <h3 className="text-3xl font-semibold mt-2 text-emerald-600 dark:text-emerald-400">
                                {gifts.filter(g => g.status === 'redeemed').length}
                            </h3>
                        </div>
                        <div className="p-2 rounded-lg bg-emerald-100 dark:bg-emerald-900/30">
                            <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                    <div className="flex items-start justify-between">
                        <div>
                            <p className="text-sm font-medium text-slate-500 dark:text-slate-400">Pending</p>
                            <h3 className="text-3xl font-semibold mt-2 text-amber-600 dark:text-amber-400">
                                {gifts.filter(g => g.status === 'purchased').length}
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
                ) : gifts.length === 0 ? (
                    <div className="p-12 text-center">
                        <div className="w-16 h-16 rounded-2xl bg-slate-100 dark:bg-slate-800 flex items-center justify-center mx-auto mb-4">
                            <Gift className="w-8 h-8 text-slate-400" />
                        </div>
                        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
                            No gifts yet
                        </h3>
                        <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
                            Create your first gift subscription for a customer
                        </p>
                        <button
                            onClick={() => setShowCreate(true)}
                            className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white font-medium rounded-lg hover:bg-primary/90"
                        >
                            <Plus className="w-4 h-4" />
                            Create Gift
                        </button>
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Gift Code</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Status</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Duration</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Recipient</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Purchased</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                                {gifts.map((gift, index) => (
                                    <tr
                                        key={gift.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors animate-fade-up"
                                        style={{ animationDelay: `${index * 0.05}s` }}
                                    >
                                        <td className="px-6 py-4">
                                            <span className="font-mono text-sm px-2.5 py-1 rounded-md bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-white">
                                                {gift.code}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${getStatusColor(gift.status)}`}>
                                                {getStatusIcon(gift.status)}
                                                {gift.status}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 text-sm text-slate-600 dark:text-slate-300">
                                            {gift.duration_months} Months
                                        </td>
                                        <td className="px-6 py-4 text-sm text-slate-600 dark:text-slate-300">
                                            {gift.recipient_email || '—'}
                                        </td>
                                        <td className="px-6 py-4 text-sm text-slate-500 dark:text-slate-400">
                                            {new Date(gift.created_at).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Create Gift Modal */}
            {showCreate && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                    <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 w-full max-w-md mx-4 shadow-xl">
                        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700">
                            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Create Gift Subscription</h2>
                            <button onClick={() => setShowCreate(false)} className="p-1 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors">
                                <X className="w-5 h-5 text-slate-500" />
                            </button>
                        </div>
                        <form onSubmit={handleCreate} className="p-6 space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Buyer Customer</label>
                                <select
                                    required
                                    value={form.buyer_customer_id}
                                    onChange={(e) => setForm({ ...form, buyer_customer_id: e.target.value })}
                                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
                                >
                                    <option value="">Select customer...</option>
                                    {customers.map(c => (
                                        <option key={c.id} value={c.id}>{c.name} ({c.email})</option>
                                    ))}
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Plan</label>
                                <select
                                    required
                                    value={form.plan_id}
                                    onChange={(e) => setForm({ ...form, plan_id: e.target.value })}
                                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
                                >
                                    <option value="">Select plan...</option>
                                    {plans.map(p => (
                                        <option key={p.id} value={p.id}>{p.name}</option>
                                    ))}
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Duration (Months)</label>
                                <input
                                    type="number"
                                    min="1"
                                    max="36"
                                    required
                                    value={form.duration_months}
                                    onChange={(e) => setForm({ ...form, duration_months: e.target.value })}
                                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
                                />
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
                                    {creating ? 'Creating...' : 'Create Gift'}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
}

export default Gifts;
