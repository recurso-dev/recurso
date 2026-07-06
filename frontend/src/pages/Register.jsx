import React, { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { endpoints } from '../lib/api'
import { useAuth } from '../auth/AuthProvider'

const Register = () => {
    const navigate = useNavigate()
    const { login } = useAuth()
    const [formData, setFormData] = useState({
        orgName: '',
        email: '',
    })
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState(null)

    const handleChange = (e) => {
        setFormData({ ...formData, [e.target.name]: e.target.value })
    }

    const handleSubmit = async (e) => {
        e.preventDefault()
        setLoading(true)
        setError(null)

        try {
            // Call API to register
            const response = await endpoints.register({
                name: formData.orgName,
                email: formData.email
            })

            const { api_key, tenant } = response.data

            // Auto login with the new key
            if (api_key) {
                login(api_key)
                navigate('/')
            }
        } catch (err) {
            console.error("Registration failed:", err)
            setError(err.response?.data?.error?.message || "Registration failed. Please try again.")
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="flex min-h-screen w-full font-sans bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-white transition-colors duration-200">
            {/* Left Column: Branding (Desktop Only) */}
            <div className="relative hidden w-0 flex-1 lg:flex flex-col justify-between bg-slate-900 overflow-hidden">
                <div className="absolute inset-0 bg-cover bg-center opacity-40" style={{ backgroundImage: 'linear-gradient(135deg, #1736cf 0%, #111421 100%)' }}></div>
                <div className="relative flex h-full flex-col justify-between p-12 z-10">
                    <div className="flex items-center gap-2">
                        <div className="flex items-center justify-center w-8 h-8 rounded bg-white/10 text-white">
                            <span className="material-symbols-outlined text-xl">dataset</span>
                        </div>
                        <span className="text-white text-xl font-bold tracking-tight">Recurso</span>
                    </div>

                    <div>
                        <h1 className="text-white text-4xl font-black leading-tight tracking-tight mb-4">
                            The billing engine for high-growth SaaS.
                        </h1>
                        <p className="text-slate-300 text-lg font-normal leading-relaxed max-w-md">
                            Handle subscriptions, recurring invoices, and usage-based metering with a developer-first platform.
                        </p>
                    </div>
                </div>
            </div>

            {/* Right Column: Registration Form */}
            <div className="flex flex-1 flex-col justify-center px-4 py-12 sm:px-6 lg:flex-none lg:px-20 xl:px-24 bg-white dark:bg-slate-950 overflow-y-auto">
                <div className="mx-auto w-full max-w-[420px]">
                    <div className="mb-10">
                        <h2 className="text-3xl font-black leading-tight tracking-tight text-slate-900 dark:text-white">
                            Create your workspace
                        </h2>
                        <p className="mt-2 text-slate-500 dark:text-slate-400 text-base">
                            Get started with your isolated tenant environment.
                        </p>
                    </div>

                    <form onSubmit={handleSubmit} className="space-y-6">
                        {error && (
                            <div className="p-3 rounded-lg bg-red-50 text-red-600 text-sm border border-red-200 dark:bg-red-900/20 dark:border-red-800 dark:text-red-400">
                                {error}
                            </div>
                        )}

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2" htmlFor="orgName">
                                Organization Name
                            </label>
                            <input
                                id="orgName"
                                name="orgName"
                                type="text"
                                required
                                value={formData.orgName}
                                onChange={handleChange}
                                placeholder="Acme Corp"
                                className="w-full rounded-lg border-slate-300 bg-white px-4 py-3 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2" htmlFor="email">
                                Work Email
                            </label>
                            <input
                                id="email"
                                name="email"
                                type="email"
                                required
                                value={formData.email}
                                onChange={handleChange}
                                placeholder="name@company.com"
                                className="w-full rounded-lg border-slate-300 bg-white px-4 py-3 text-slate-900 placeholder:text-slate-400 focus:border-primary focus:ring-primary dark:border-slate-700 dark:bg-slate-900 dark:text-white"
                            />
                        </div>

                        {/* No password field: registration is API-key based —
                            the backend only needs a workspace name and email. */}

                        <div className="flex items-start">
                            <input
                                id="terms"
                                name="terms"
                                type="checkbox"
                                required
                                className="h-4 w-4 rounded border-slate-300 text-primary focus:ring-primary dark:border-slate-700 dark:bg-slate-900"
                            />
                            <label htmlFor="terms" className="ml-3 text-sm text-slate-500 dark:text-slate-400">
                                I agree to the <a href="#" className="font-medium text-primary hover:text-primary/80">Terms</a> and <a href="#" className="font-medium text-primary hover:text-primary/80">Privacy Policy</a>.
                            </label>
                        </div>

                        <button
                            type="submit"
                            disabled={loading}
                            className="flex w-full items-center justify-center rounded-lg bg-primary px-5 py-3 text-base font-bold text-white transition-colors hover:bg-primary/90 disabled:opacity-50"
                        >
                            {loading ? 'Creating...' : 'Create Workspace'}
                        </button>
                    </form>

                    <div className="mt-8 text-center">
                        <p className="text-sm text-slate-500 dark:text-slate-400">
                            Already have an account?{' '}
                            <Link to="/login" className="font-semibold text-primary hover:text-primary/80">
                                Log in
                            </Link>
                        </p>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Register
