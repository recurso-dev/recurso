import { ArrowRight, Terminal, TrendingUp, Users, Receipt, CreditCard } from 'lucide-react'
import { useEffect, useState } from 'react'

const AnimatedCounter = ({ end, suffix = '', prefix = '' }) => {
    const [count, setCount] = useState(0)

    useEffect(() => {
        let start = 0
        const duration = 2000
        const increment = end / (duration / 16)
        const timer = setInterval(() => {
            start += increment
            if (start >= end) {
                setCount(end)
                clearInterval(timer)
            } else {
                setCount(Math.floor(start))
            }
        }, 16)
        return () => clearInterval(timer)
    }, [end])

    return <span>{prefix}{count.toLocaleString()}{suffix}</span>
}

const MiniChart = () => (
    <svg viewBox="0 0 200 60" className="w-full h-full" fill="none">
        <defs>
            <linearGradient id="chartGrad" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" stopColor="rgba(16, 185, 129, 0.3)" />
                <stop offset="100%" stopColor="rgba(16, 185, 129, 0)" />
            </linearGradient>
        </defs>
        <path
            d="M 0 50 Q 20 45 40 42 Q 60 39 80 35 Q 100 28 120 22 Q 140 18 160 14 Q 180 10 200 6"
            stroke="#10B981"
            strokeWidth="2"
            strokeLinecap="round"
            fill="none"
            className="animate-dash"
            style={{ strokeDasharray: 300, strokeDashoffset: 300, animation: 'dash 2s ease-out forwards 0.5s' }}
        />
        <path
            d="M 0 50 Q 20 45 40 42 Q 60 39 80 35 Q 100 28 120 22 Q 140 18 160 14 Q 180 10 200 6 L 200 60 L 0 60 Z"
            fill="url(#chartGrad)"
            opacity="0.5"
        />
    </svg>
)

const Hero = () => {
    return (
        <section className="relative min-h-screen flex items-center justify-center pt-16 overflow-hidden">
            {/* Layered background orbs */}
            <div className="absolute inset-0 bg-grid opacity-30" />
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[900px] h-[900px] bg-emerald-500/[0.03] rounded-full blur-[120px]" />
            <div className="absolute top-1/3 right-1/4 w-[500px] h-[500px] bg-emerald-400/[0.05] rounded-full blur-[100px] animate-pulse-slow" />
            <div className="absolute bottom-1/3 left-1/3 w-[300px] h-[300px] bg-cyan-500/[0.03] rounded-full blur-[80px] animate-pulse-slow delay-200" />

            {/* Floating particles */}
            <div className="absolute top-32 left-[15%] w-1.5 h-1.5 bg-emerald-400 rounded-full animate-float opacity-40" />
            <div className="absolute top-52 right-[20%] w-2 h-2 bg-emerald-500/40 rounded-full animate-float delay-300 opacity-30" />
            <div className="absolute bottom-40 left-[25%] w-1 h-1 bg-emerald-300/60 rounded-full animate-float delay-500 opacity-50" />
            <div className="absolute top-40 left-[60%] w-1.5 h-1.5 bg-cyan-400/40 rounded-full animate-float-slow opacity-30" />

            <div className="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16 sm:py-20">
                <div className="text-center max-w-4xl mx-auto mb-16">
                    {/* Badge */}
                    <div className="inline-flex items-center gap-2.5 px-4 py-2 rounded-full glass-strong border border-emerald-500/20 mb-8 animate-fade-up">
                        <span className="relative flex h-2 w-2">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
                        </span>
                        <span className="text-sm font-medium text-gray-300">Now Open Source — Star us on GitHub</span>
                    </div>

                    {/* Headline */}
                    <h1 className="text-5xl sm:text-6xl lg:text-8xl font-extrabold tracking-tight mb-6 animate-fade-up delay-100 leading-[0.95]">
                        <span className="text-white">Smart billing</span>
                        <br />
                        <span className="gradient-text">for global SaaS</span>
                    </h1>

                    {/* Subheadline */}
                    <p className="text-lg sm:text-xl text-gray-400 max-w-2xl mx-auto mb-10 leading-relaxed animate-fade-up delay-200">
                        Subscriptions, invoicing, and RBI-compliant payments powered by AI.
                        Predict churn, optimize retries, and scale your revenue — globally.
                    </p>

                    {/* CTAs */}
                    <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-16 animate-fade-up delay-300">
                        <a
                            href="https://github.com/recur-so/recurso"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="group flex items-center gap-2 px-8 py-4 bg-emerald-500 text-black font-bold rounded-xl hover:bg-emerald-400 transition-all duration-300 glow-green glow-ring text-base"
                        >
                            Start Building
                            <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
                        </a>

                        <div className="flex items-center gap-3 px-6 py-4 glass-strong rounded-xl cursor-pointer hover:border-emerald-500/30 transition-all group">
                            <Terminal className="w-5 h-5 text-emerald-400" />
                            <code className="text-sm text-gray-300 font-mono group-hover:text-white transition-colors">
                                docker compose up
                            </code>
                        </div>
                    </div>
                </div>

                {/* ═══ Dashboard Mockup ═══ */}
                <div className="dashboard-preview max-w-4xl mx-auto animate-fade-up delay-500">
                    <div className="dashboard-card glass-strong rounded-2xl border border-white/[0.08] overflow-hidden shadow-2xl shadow-black/50">
                        {/* Title bar */}
                        <div className="flex items-center justify-between px-5 py-3 border-b border-white/[0.06] bg-white/[0.02]">
                            <div className="flex items-center gap-2">
                                <div className="flex gap-1.5">
                                    <div className="w-3 h-3 rounded-full bg-red-500/60" />
                                    <div className="w-3 h-3 rounded-full bg-yellow-500/60" />
                                    <div className="w-3 h-3 rounded-full bg-green-500/60" />
                                </div>
                                <span className="text-xs text-gray-500 font-mono ml-3">dashboard.recurso.dev</span>
                            </div>
                            <div className="text-xs text-gray-600">Live Preview</div>
                        </div>

                        {/* Dashboard content */}
                        <div className="p-6">
                            {/* Stats row */}
                            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                                <div className="bg-white/[0.03] rounded-xl p-4 border border-white/[0.05]">
                                    <div className="flex items-center gap-2 mb-2">
                                        <TrendingUp className="w-4 h-4 text-emerald-400" />
                                        <span className="text-[11px] text-gray-500 uppercase tracking-wider font-medium">MRR</span>
                                    </div>
                                    <div className="text-2xl font-bold text-white">
                                        <AnimatedCounter end={153} prefix="$" suffix="K" />
                                    </div>
                                    <span className="text-xs text-emerald-400 font-medium">+12.3%</span>
                                </div>
                                <div className="bg-white/[0.03] rounded-xl p-4 border border-white/[0.05]">
                                    <div className="flex items-center gap-2 mb-2">
                                        <Users className="w-4 h-4 text-blue-400" />
                                        <span className="text-[11px] text-gray-500 uppercase tracking-wider font-medium">Subscribers</span>
                                    </div>
                                    <div className="text-2xl font-bold text-white">
                                        <AnimatedCounter end={3255} />
                                    </div>
                                    <span className="text-xs text-emerald-400 font-medium">+8.1%</span>
                                </div>
                                <div className="bg-white/[0.03] rounded-xl p-4 border border-white/[0.05]">
                                    <div className="flex items-center gap-2 mb-2">
                                        <Receipt className="w-4 h-4 text-amber-400" />
                                        <span className="text-[11px] text-gray-500 uppercase tracking-wider font-medium">Invoices</span>
                                    </div>
                                    <div className="text-2xl font-bold text-white">
                                        <AnimatedCounter end={847} />
                                    </div>
                                    <span className="text-xs text-gray-500">This month</span>
                                </div>
                                <div className="bg-white/[0.03] rounded-xl p-4 border border-white/[0.05]">
                                    <div className="flex items-center gap-2 mb-2">
                                        <CreditCard className="w-4 h-4 text-purple-400" />
                                        <span className="text-[11px] text-gray-500 uppercase tracking-wider font-medium">Recovery</span>
                                    </div>
                                    <div className="text-2xl font-bold text-white">
                                        <AnimatedCounter end={94} suffix="%" />
                                    </div>
                                    <span className="text-xs text-emerald-400 font-medium">AI Dunning</span>
                                </div>
                            </div>

                            {/* Chart area */}
                            <div className="bg-white/[0.02] rounded-xl p-4 border border-white/[0.05]">
                                <div className="flex items-center justify-between mb-3">
                                    <span className="text-sm font-medium text-gray-400">Revenue Growth</span>
                                    <div className="flex gap-2">
                                        <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 font-medium">Monthly</span>
                                        <span className="text-[10px] px-2 py-0.5 rounded-full bg-white/5 text-gray-500 font-medium">Quarterly</span>
                                    </div>
                                </div>
                                <div className="h-16">
                                    <MiniChart />
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Logo cloud */}
                <div className="mt-20 animate-fade-up delay-700">
                    <p className="text-[11px] text-gray-600 uppercase tracking-[0.2em] text-center mb-8 font-medium">
                        Integrates with your stack
                    </p>
                    <div className="flex items-center justify-center gap-10 sm:gap-16 flex-wrap">
                        {['Stripe', 'Razorpay', 'PostgreSQL', 'TigerBeetle', 'Redis'].map((name) => (
                            <div
                                key={name}
                                className="text-gray-600 hover:text-gray-300 transition-all duration-300 cursor-default"
                            >
                                <span className="text-sm font-semibold tracking-wide">{name}</span>
                            </div>
                        ))}
                    </div>
                </div>
            </div>

            {/* Bottom gradient fade */}
            <div className="absolute bottom-0 left-0 right-0 h-40 bg-gradient-to-t from-[#050505] to-transparent" />
        </section>
    )
}

export default Hero
