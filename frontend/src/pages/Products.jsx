import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { endpoints as api } from '../lib/api'

const Products = () => {
    const [products, setProducts] = useState([])
    const [loading, setLoading] = useState(true)
    const [search, setSearch] = useState('')
    const [debouncedSearch, setDebouncedSearch] = useState('')
    const [page, setPage] = useState(1)
    const [limit, setLimit] = useState(10)

    // Debounce search
    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(search)
            setPage(1)
        }, 500)
        return () => clearTimeout(timer)
    }, [search])

    useEffect(() => {
        const fetchProducts = async () => {
            setLoading(true)
            try {
                const params = {
                    q: debouncedSearch,
                    limit: limit,
                    page: page
                }
                const response = await api.getPlans(params)
                // Backend returns plans. We map them to "product" structure for the UI.
                const plans = response.data.data || []

                // MOCK/Transform: Since "Plans" are simpler in backend, we adapt here.
                const transformed = plans.map(p => ({
                    id: p.id,
                    name: p.name,
                    description: p.description || 'No description',
                    status: p.active ? 'active' : 'archived',
                    plans: p.prices ? p.prices.length : 0, // Count prices/variations as "plans" count proxy
                    updated_at: p.created_at // Use created_at if updated_at is missing
                }))
                setProducts(transformed)
            } catch (error) {
                console.error("Failed to fetch products:", error)
            } finally {
                setLoading(false)
            }
        }
        fetchProducts()
    }, [debouncedSearch, page, limit])

    const [statusFilter, setStatusFilter] = useState('all') // 'all', 'active', 'archived'
    const [isStatusOpen, setIsStatusOpen] = useState(false)

    // Filtered list
    const filteredProducts = products.filter(p => {
        if (statusFilter === 'all') return true
        return p.status === statusFilter
    })

    // Click outside handler
    useEffect(() => {
        const close = () => setIsStatusOpen(false)
        if (isStatusOpen) window.addEventListener('click', close)
        return () => window.removeEventListener('click', close)
    }, [isStatusOpen])

    const handleFilterClick = (e, filter) => {
        e.stopPropagation()
        setStatusFilter(filter)
        setIsStatusOpen(false)
    }

    return (
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            <div className="flex flex-wrap items-center justify-between gap-4 pb-6">
                <h1 className="text-3xl font-bold tracking-tight text-slate-900 dark:text-white">Product Catalog</h1>
                <Link
                    to="/plans/new"
                    className="flex h-10 cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg bg-primary px-4 text-sm font-medium text-white shadow-sm transition-all hover:bg-primary/90"
                >
                    <span className="material-symbols-outlined text-lg">add_circle</span>
                    <span className="truncate">Create New Product</span>
                </Link>
            </div>

            {/* Filters */}
            <div className="flex flex-wrap items-center gap-4 mb-4">
                <div className="flex-grow min-w-[250px]">
                    <div className="relative flex w-full flex-1 items-stretch rounded-lg">
                        <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                            <span className="material-symbols-outlined text-slate-400 dark:text-slate-500 text-xl">search</span>
                        </div>
                        <input
                            className="block w-full rounded-lg border-slate-300 bg-white dark:bg-slate-800 dark:border-slate-700 dark:text-white dark:placeholder-slate-400 pl-10 h-10 text-sm focus:border-primary focus:ring-primary/20"
                            placeholder="Search products..."
                            type="text"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                </div>

                {/* Status Dropdown */}
                <div className="relative">
                    <button
                        onClick={(e) => { e.stopPropagation(); setIsStatusOpen(!isStatusOpen) }}
                        className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-white dark:bg-slate-900/50 border border-slate-300 dark:border-slate-700 px-3 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
                    >
                        <p className="text-slate-700 dark:text-slate-300 text-sm font-medium leading-normal capitalize">Status: {statusFilter}</p>
                        <span className="material-symbols-outlined text-lg text-slate-400 dark:text-slate-500">expand_more</span>
                    </button>

                    {isStatusOpen && (
                        <div className="absolute right-0 top-12 w-40 z-10 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 py-1">
                            <button onClick={(e) => handleFilterClick(e, 'all')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">All</button>
                            <button onClick={(e) => handleFilterClick(e, 'active')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">Active</button>
                            <button onClick={(e) => handleFilterClick(e, 'archived')} className="block w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700">Archived</button>
                        </div>
                    )}
                </div>
            </div>

            <div className="overflow-hidden rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 shadow-sm">
                <div className="overflow-x-auto">
                    <table className="w-full">
                        <thead className="bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-800">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Product Name</th>
                                <th className="px-6 py-3 text-left text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Description</th>
                                <th className="px-6 py-3 text-left text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Status</th>
                                <th className="px-6 py-3 text-left text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Plans</th>
                                <th className="px-6 py-3 text-left text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">Last Updated</th>
                                <th className="px-6 py-3 text-right text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider"></th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                            {loading ? (
                                <tr><td colSpan="6" className="p-8 text-center text-slate-500">Loading products...</td></tr>
                            ) : filteredProducts.length === 0 ? (
                                <tr><td colSpan="6" className="p-8 text-center text-slate-500">No products found.</td></tr>
                            ) : (
                                filteredProducts.map((product) => (
                                    <tr key={product.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors">
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900 dark:text-white">{product.name}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400 truncate max-w-xs">{product.description}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                                            <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium 
                                                ${product.status === 'active' ? 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300' : 'bg-slate-200 text-slate-800 dark:bg-slate-700/50 dark:text-slate-300'}`}>
                                                {product.status.charAt(0).toUpperCase() + product.status.slice(1)}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">{product.plans}</td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                            {new Date(product.updated_at).toLocaleDateString()}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <button className="text-slate-400 hover:text-primary dark:hover:text-primary transition-colors">
                                                <span className="material-symbols-outlined">more_horiz</span>
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between mt-6">
                <p className="text-sm text-slate-500 dark:text-slate-400">
                    Page <span className="font-medium">{page}</span>
                </p>
                <div className="flex gap-2">
                    <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="flex items-center justify-center h-9 w-9 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900/50 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-600 dark:text-slate-300 disabled:opacity-50 transition-colors"
                    >
                        <span className="material-symbols-outlined text-lg">chevron_left</span>
                    </button>
                    <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={products.length < limit}
                        className="flex items-center justify-center h-9 w-9 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900/50 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-600 dark:text-slate-300 disabled:opacity-50 transition-colors"
                    >
                        <span className="material-symbols-outlined text-lg">chevron_right</span>
                    </button>
                </div>
            </div>
        </div>
    )
}

export default Products
