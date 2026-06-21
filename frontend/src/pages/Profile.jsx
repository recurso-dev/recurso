import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { endpoints } from '../lib/api'

const Profile = () => {
    const [account, setAccount] = useState({ name: '', email: '', id: '' })
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        const fetchAccount = async () => {
            try {
                const response = await endpoints.getAccount()
                if (response.data.data) {
                    setAccount({
                        name: response.data.data.name,
                        email: response.data.data.email,
                        id: response.data.data.id
                    })
                }
            } catch (error) {
                console.error("Failed to fetch account:", error)
            } finally {
                setLoading(false)
            }
        }
        fetchAccount()
    }, [])

    return (
        <div className="flex flex-col max-w-3xl mx-auto">
            <header className="mb-8 flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight text-slate-900 dark:text-white">Account Profile</h1>
                    <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">View your account identity and security information.</p>
                </div>
                <Link to="/settings" className="rounded-lg bg-white border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 shadow-sm hover:bg-slate-50 dark:bg-slate-800 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-700">
                    Edit Profile
                </Link>
            </header>

            <div className="space-y-6">
                {/* Profile Card */}
                <section className="rounded-xl border border-slate-200 bg-white p-6 dark:border-slate-800 dark:bg-slate-900">
                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-6">Account Information</h2>

                    <div className="flex flex-col md:flex-row gap-8 items-start">
                        {/* Avatar */}
                        <div className="flex flex-col items-center gap-3">
                            <div
                                className="size-24 rounded-full bg-cover bg-center border border-slate-200 dark:border-slate-700 bg-slate-100 flex items-center justify-center text-3xl font-bold text-slate-400"
                            >
                                {account.name ? account.name.charAt(0).toUpperCase() : 'A'}
                            </div>
                        </div>

                        {/* Form */}
                        <div className="flex-1 w-full grid grid-cols-1 gap-4">
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Account Name</label>
                                <div className="w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-900 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300">
                                    {loading ? 'Loading...' : account.name}
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Email Address</label>
                                <div className="w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-900 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300 flex items-center justify-between">
                                    <span>{loading ? 'Loading...' : account.email}</span>
                                    <span className="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded-full dark:bg-green-900/30 dark:text-green-400">Verified</span>
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Tenant ID</label>
                                <div className="w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm font-mono text-slate-500 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-500">
                                    {loading ? 'Loading...' : account.id}
                                </div>
                            </div>
                        </div>
                    </div>
                </section>

                {/* Security Card */}
                <section className="rounded-xl border border-slate-200 bg-white p-6 dark:border-slate-800 dark:bg-slate-900">
                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">Security</h2>
                    <p className="text-sm text-slate-500 mb-6">Your account uses API Keys for authentication. You can manage your keys in the Developers section.</p>

                    <div className="flex items-center gap-4 p-4 bg-slate-50 dark:bg-slate-950 rounded-lg border border-slate-200 dark:border-slate-800">
                        <span className="material-symbols-outlined text-slate-400">key</span>
                        <div className="flex-1">
                            <h3 className="text-sm font-medium text-slate-900 dark:text-white">API Key Authentication</h3>
                            <p className="text-xs text-slate-500 mt-1">You are currently authenticated via a Tenant API Key.</p>
                        </div>
                        <a href="/developers" className="text-sm font-medium text-primary hover:underline">Manage Keys</a>
                    </div>
                </section>
            </div>
        </div>
    )
}

export default Profile
