import {
    CreditCard,
    BarChart3,
    Webhook,
    FileText,
    Shield,
    Globe,
    Zap,
    Database
} from 'lucide-react'
import { useStaggerAnimation } from '../hooks/useScrollAnimation'

const features = [
    {
        icon: Zap,
        title: 'AI Smart Dunning',
        description: 'Multi-Armed Bandit algorithm optimizes retry timing and channels to maximize revenue recovery.',
        stat: null,
        size: 'large',
        color: 'from-amber-500/10 to-transparent',
        iconBg: 'bg-amber-500/10',
        iconColor: 'text-amber-400',
    },
    {
        icon: Globe,
        title: 'India Stack Ready',
        description: 'RBI e-mandates, UPI AutoPay, and GST/e-invoicing compliance built-in from day one.',
        stat: null,
        size: 'medium',
        color: 'from-orange-500/10 to-transparent',
        iconBg: 'bg-orange-500/10',
        iconColor: 'text-orange-400',
    },
    {
        icon: Shield,
        title: 'AI Churn Prediction',
        description: 'Score every customer\'s risk and trigger retention workflows before they cancel.',
        stat: null,
        size: 'medium',
        color: 'from-rose-500/10 to-transparent',
        iconBg: 'bg-rose-500/10',
        iconColor: 'text-rose-400',
    },
    {
        icon: CreditCard,
        title: 'Multi-Gateway Routing',
        description: 'Route payments between Stripe and Razorpay automatically based on currency, region, and success rates.',
        stat: null,
        size: 'large',
        color: 'from-emerald-500/10 to-transparent',
        iconBg: 'bg-emerald-500/10',
        iconColor: 'text-emerald-400',
    },
    {
        icon: Webhook,
        title: 'Webhooks & Events',
        description: 'Real-time notifications with HMAC signatures and automatic retries.',
        stat: null,
        size: 'medium',
        color: 'from-pink-500/10 to-transparent',
        iconBg: 'bg-pink-500/10',
        iconColor: 'text-pink-400',
    },
    {
        icon: BarChart3,
        title: 'Usage Metering',
        description: 'Track API calls, tokens, seats. Bill based on actual consumption.',
        stat: null,
        size: 'small',
        color: 'from-blue-500/10 to-transparent',
        iconBg: 'bg-blue-500/10',
        iconColor: 'text-blue-400',
    },
    {
        icon: Database,
        title: 'Immutable Ledger',
        description: 'TigerBeetle-powered double-entry accounting. Always audit-ready.',
        stat: null,
        size: 'small',
        color: 'from-violet-500/10 to-transparent',
        iconBg: 'bg-violet-500/10',
        iconColor: 'text-violet-400',
    },
    {
        icon: FileText,
        title: 'Credit Notes',
        description: 'Issue refunds, adjustments, and credits with full audit trail.',
        stat: null,
        size: 'small',
        color: 'from-cyan-500/10 to-transparent',
        iconBg: 'bg-cyan-500/10',
        iconColor: 'text-cyan-400',
    },
]

const Features = () => {
    const containerRef = useStaggerAnimation()

    return (
        <section id="features" className="relative py-32 overflow-hidden">
            {/* Background */}
            <div className="absolute inset-0 bg-grid opacity-20" />
            <div className="absolute top-1/2 right-0 w-[500px] h-[500px] bg-emerald-500/[0.02] rounded-full blur-[120px]" />

            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                {/* Section header */}
                <div className="text-center mb-20">
                    <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/[0.08] border border-emerald-500/20 mb-6">
                        <span className="text-xs font-semibold text-emerald-400 uppercase tracking-wider">Platform</span>
                    </div>
                    <h2 className="text-4xl sm:text-5xl lg:text-6xl font-bold text-white mb-5 tracking-tight">
                        Everything you need to{' '}
                        <span className="gradient-text">monetize</span>
                    </h2>
                    <p className="text-lg text-gray-400 max-w-2xl mx-auto leading-relaxed">
                        A complete billing engine with subscriptions, invoicing, payments, and compliance — all in one platform.
                    </p>
                </div>

                {/* Bento grid */}
                <div ref={containerRef} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                    {features.map((feature) => {
                        const Icon = feature.icon
                        const isLarge = feature.size === 'large'
                        const isMedium = feature.size === 'medium'

                        return (
                            <div
                                key={feature.title}
                                data-animate
                                className={`
                                    group relative spotlight-card glass rounded-2xl p-6 transition-all duration-500 hover:-translate-y-1 cursor-default
                                    ${isLarge ? 'lg:col-span-2 lg:row-span-2' : ''}
                                    ${isMedium ? 'lg:col-span-2' : ''}
                                `}
                                onMouseMove={(e) => {
                                    const rect = e.currentTarget.getBoundingClientRect()
                                    e.currentTarget.style.setProperty('--mouse-x', `${e.clientX - rect.left}px`)
                                    e.currentTarget.style.setProperty('--mouse-y', `${e.clientY - rect.top}px`)
                                }}
                            >
                                {/* Gradient overlay on hover */}
                                <div className={`absolute inset-0 bg-gradient-to-br ${feature.color} rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-500`} />

                                <div className="relative z-10">
                                    <div className={`w-11 h-11 rounded-xl ${feature.iconBg} flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300`}>
                                        <Icon className={`w-5 h-5 ${feature.iconColor}`} />
                                    </div>

                                    <h3 className={`font-bold text-white mb-2 ${isLarge ? 'text-2xl' : 'text-lg'}`}>
                                        {feature.title}
                                    </h3>

                                    <p className={`text-gray-400 leading-relaxed ${isLarge ? 'text-base' : 'text-sm'}`}>
                                        {feature.description}
                                    </p>


                                </div>
                            </div>
                        )
                    })}
                </div>
            </div>
        </section>
    )
}

export default Features
