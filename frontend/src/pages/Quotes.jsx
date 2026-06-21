import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { endpoints } from '../lib/api'
import { FileText, Plus, Send, Check, X, ArrowRight, MoreHorizontal, Search } from 'lucide-react'
import QuoteDetail from '../components/slide-overs/QuoteDetail'

const statusColors = {
    draft: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300',
    sent: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
    accepted: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
    declined: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
    expired: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
}

const Quotes = () => {
    const [quotes, setQuotes] = useState([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)
    const [searchQuery, setSearchQuery] = useState('')
    const [statusFilter, setStatusFilter] = useState('')
    const [selectedQuote, setSelectedQuote] = useState(null)
    const [isDetailOpen, setIsDetailOpen] = useState(false)
    const navigate = useNavigate()

    useEffect(() => {
        fetchQuotes()
    }, [statusFilter, searchQuery])

    const fetchQuotes = async () => {
        try {
            setLoading(true)
            const params = {}
            if (statusFilter) params.status = statusFilter
            if (searchQuery) params.search = searchQuery

            const response = await endpoints.getQuotes(params)
            setQuotes(response.data.data || [])
        } catch (err) {
            setError('Failed to load quotes')
            console.error(err)
        } finally {
            setLoading(false)
        }
    }

    const handleSend = async (id, e) => {
        e?.stopPropagation()
        try {
            await endpoints.sendQuote(id)
            fetchQuotes()
        } catch (err) {
            console.error('Failed to send quote:', err)
        }
    }

    const handleConvert = async (id, e) => {
        e?.stopPropagation()
        try {
            await endpoints.convertQuoteToInvoice(id)
            fetchQuotes()
        } catch (err) {
            console.error('Failed to convert quote:', err)
        }
    }

    const handleRowClick = (quote) => {
        setSelectedQuote(quote)
        setIsDetailOpen(true)
    }

    const closeDetail = () => {
        setIsDetailOpen(false)
        setTimeout(() => setSelectedQuote(null), 300)
    }

    const formatCurrency = (amount, currency = 'USD') => {
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: currency
        }).format(amount / 100)
    }

    const formatDate = (date) => {
        return new Date(date).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        })
    }

    return (
        <div className="animate-fade-in">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Quotes</h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        Create and manage price quotes for customers
                    </p>
                </div>
                <Link
                    to="/quotes/new"
                    className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white font-medium rounded-lg hover:bg-primary/90 transition-all hover:scale-105 active:scale-95"
                >
                    <Plus className="w-4 h-4" />
                    New Quote
                </Link>
            </div>

            {/* Filters */}
            <div className="flex flex-col sm:flex-row gap-4 mb-6">
                <div className="relative flex-1">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                    <input
                        type="text"
                        placeholder="Search quotes..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="w-full pl-10 pr-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
                    />
                </div>
                <select
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value)}
                    className="px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                >
                    <option value="">All Status</option>
                    <option value="draft">Draft</option>
                    <option value="sent">Sent</option>
                    <option value="accepted">Accepted</option>
                    <option value="declined">Declined</option>
                    <option value="expired">Expired</option>
                </select>
            </div>

            {/* Table */}
            <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden">
                {loading ? (
                    <div className="p-8 text-center">
                        <div className="w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto" />
                    </div>
                ) : error ? (
                    <div className="p-8 text-center text-red-500">{error}</div>
                ) : quotes.length === 0 ? (
                    <div className="p-12 text-center">
                        <div className="w-16 h-16 rounded-2xl bg-slate-100 dark:bg-slate-800 flex items-center justify-center mx-auto mb-4">
                            <FileText className="w-8 h-8 text-slate-400" />
                        </div>
                        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
                            No quotes yet
                        </h3>
                        <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
                            Create your first quote to send to customers
                        </p>
                        <Link
                            to="/quotes/new"
                            className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white font-medium rounded-lg hover:bg-primary/90"
                        >
                            <Plus className="w-4 h-4" />
                            Create Quote
                        </Link>
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Quote</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Customer</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Amount</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Status</th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Created</th>
                                    <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                                {quotes.map((quote, index) => (
                                    <tr
                                        key={quote.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors animate-fade-up cursor-pointer"
                                        style={{ animationDelay: `${index * 0.05}s` }}
                                        onClick={() => handleRowClick(quote)}
                                    >
                                        <td className="px-6 py-4">
                                            <button
                                                className="text-sm font-medium text-primary hover:underline bg-transparent border-0 p-0"
                                                onClick={(e) => { e.stopPropagation(); handleRowClick(quote) }}
                                            >
                                                {quote.quote_number}
                                            </button>
                                        </td>
                                        <td className="px-6 py-4 text-sm text-slate-600 dark:text-slate-300">
                                            {quote.customer_id?.substring(0, 8)}...
                                        </td>
                                        <td className="px-6 py-4 text-sm font-medium text-slate-900 dark:text-white">
                                            {formatCurrency(quote.total, quote.currency)}
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${statusColors[quote.status]}`}>
                                                {quote.status}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 text-sm text-slate-500 dark:text-slate-400">
                                            {formatDate(quote.created_at)}
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center justify-end gap-2" onClick={(e) => e.stopPropagation()}>
                                                {quote.status === 'draft' && (
                                                    <button
                                                        onClick={(e) => handleSend(quote.id, e)}
                                                        className="p-1.5 rounded-lg text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/30 transition-colors"
                                                        title="Send Quote"
                                                    >
                                                        <Send className="w-4 h-4" />
                                                    </button>
                                                )}
                                                {quote.status === 'accepted' && !quote.invoice_id && (
                                                    <button
                                                        onClick={(e) => handleConvert(quote.id, e)}
                                                        className="p-1.5 rounded-lg text-emerald-600 hover:bg-emerald-50 dark:hover:bg-emerald-900/30 transition-colors"
                                                        title="Convert to Invoice"
                                                    >
                                                        <ArrowRight className="w-4 h-4" />
                                                    </button>
                                                )}
                                                <button
                                                    onClick={(e) => { e.stopPropagation(); handleRowClick(quote) }}
                                                    className="p-1.5 rounded-lg text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                                                >
                                                    <MoreHorizontal className="w-4 h-4" />
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            <QuoteDetail
                quote={selectedQuote}
                isOpen={isDetailOpen}
                onClose={closeDetail}
            />
        </div>
    )
}

export default Quotes
