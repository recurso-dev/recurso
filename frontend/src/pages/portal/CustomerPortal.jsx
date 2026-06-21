import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import axios from 'axios';
import { DocumentTextIcon, CreditCardIcon, InformationCircleIcon } from '@heroicons/react/24/outline';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/v1';

export default function CustomerPortal() {
    const { tenantId, customerId } = useParams();
    const [data, setData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        const fetchPortalData = async () => {
            try {
                // Unauthenticated call for MVP
                const response = await axios.get(`${API_BASE_URL}/portal/${tenantId}/${customerId}`);
                setData(response.data);
            } catch (err) {
                setError('Failed to load portal data. The link might be invalid or expired.');
            } finally {
                setLoading(false);
            }
        };

        fetchPortalData();
    }, [tenantId, customerId]);

    if (loading) {
        return (
            <div className="min-h-screen bg-neutral-900 flex items-center justify-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
            </div>
        );
    }

    if (error || !data) {
        return (
            <div className="min-h-screen bg-neutral-900 flex items-center justify-center p-4">
                <div className="bg-neutral-800 rounded-xl p-8 max-w-md w-full text-center border border-neutral-700">
                    <InformationCircleIcon className="w-12 h-12 text-red-400 mx-auto mb-4" />
                    <h2 className="text-xl font-medium text-white mb-2">Access Denied</h2>
                    <p className="text-neutral-400">{error}</p>
                </div>
            </div>
        );
    }

    const { customer, subscriptions, invoices } = data;

    return (
        <div className="min-h-screen bg-neutral-900 text-white py-12 px-4 sm:px-6 lg:px-8">
            <div className="max-w-3xl mx-auto space-y-8">

                {/* Header */}
                <div className="bg-neutral-800 rounded-xl p-6 border border-neutral-700 shadow-sm flex items-center justify-between">
                    <div>
                        <h1 className="text-2xl font-bold text-white tracking-tight">Billing Portal</h1>
                        <p className="text-sm text-neutral-400 mt-1">Manage your subscriptions and invoices.</p>
                    </div>
                    <div className="text-right">
                        <p className="text-sm font-medium text-white">{customer?.name}</p>
                        <p className="text-sm text-neutral-400">{customer?.email}</p>
                    </div>
                </div>

                {/* Subscriptions */}
                <div className="bg-neutral-800 rounded-xl border border-neutral-700 overflow-hidden">
                    <div className="px-6 py-5 border-b border-neutral-700 bg-neutral-800/50 flex items-center gap-2">
                        <CreditCardIcon className="w-5 h-5 text-primary-400" />
                        <h2 className="text-lg font-medium text-white">Active Subscriptions</h2>
                    </div>
                    <div className="divide-y divide-neutral-700">
                        {!subscriptions || subscriptions.length === 0 ? (
                            <div className="p-6 text-center text-sm text-neutral-400">
                                No active subscriptions found.
                            </div>
                        ) : (
                            subscriptions.map((sub) => (
                                <div key={sub.id} className="p-6 flex items-center justify-between">
                                    <div>
                                        <h3 className="text-lg font-medium text-white">{sub.plan_name || 'Premium Plan'}</h3>
                                        <p className="text-sm text-neutral-400 mt-1">
                                            Status: <span className="text-green-400 capitalize">{sub.status}</span>
                                        </p>
                                    </div>
                                    <div className="text-right">
                                        <p className="text-xl font-semibold text-white">
                                            {sub.currency === 'INR' ? '₹' : '$'}{(sub.price / 100).toFixed(2)}
                                        </p>
                                        <p className="text-sm text-neutral-400 mt-1 capitalize">
                                            per {sub.billing_interval || 'month'}
                                        </p>
                                    </div>
                                </div>
                            ))
                        )}
                    </div>
                </div>

                {/* Invoices */}
                <div className="bg-neutral-800 rounded-xl border border-neutral-700 overflow-hidden">
                    <div className="px-6 py-5 border-b border-neutral-700 bg-neutral-800/50 flex items-center gap-2">
                        <DocumentTextIcon className="w-5 h-5 text-primary-400" />
                        <h2 className="text-lg font-medium text-white">Invoice History</h2>
                    </div>
                    <div className="overflow-x-auto">
                        <table className="w-full text-left text-sm whitespace-nowrap">
                            <thead className="bg-neutral-800 text-neutral-400">
                                <tr>
                                    <th className="px-6 py-4 font-medium">Invoice Number</th>
                                    <th className="px-6 py-4 font-medium">Issue Date</th>
                                    <th className="px-6 py-4 font-medium text-right">Amount</th>
                                    <th className="px-6 py-4 font-medium text-right">Status</th>
                                    <th className="px-6 py-4 font-medium text-right">Action</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-neutral-700">
                                {!invoices || invoices.length === 0 ? (
                                    <tr>
                                        <td colSpan="5" className="px-6 py-8 text-center text-neutral-400">
                                            No invoices found.
                                        </td>
                                    </tr>
                                ) : (
                                    invoices.map((inv) => (
                                        <tr key={inv.id} className="hover:bg-neutral-800/50 transition-colors">
                                            <td className="px-6 py-4 font-medium text-white">{inv.invoice_number}</td>
                                            <td className="px-6 py-4 text-neutral-300">
                                                {new Date(inv.issue_date).toLocaleDateString()}
                                            </td>
                                            <td className="px-6 py-4 text-right text-white font-medium">
                                                {inv.currency === 'INR' ? '₹' : '$'}{(inv.total / 100).toFixed(2)}
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${inv.status === 'paid' ? 'bg-green-500/10 text-green-400' :
                                                        inv.status === 'open' ? 'bg-primary-500/10 text-primary-400' :
                                                            'bg-neutral-500/10 text-neutral-400'
                                                    }`}>
                                                    {inv.status}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <a
                                                    href={`${API_BASE_URL.replace('/v1', '')}/v1/invoices/${inv.id}/pdf`}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors"
                                                >
                                                    Download PDF
                                                </a>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>

            </div>
        </div>
    );
}
