import { Star } from 'lucide-react'
import { useStaggerAnimation } from '../hooks/useScrollAnimation'

const testimonials = [
    {
        quote: "Recurso replaced our entire billing stack. Invoice generation, dunning, and churn prediction — all in one place. We shipped it in a weekend.",
        name: "Arjun Mehta",
        role: "CTO",
        company: "NovaPay",
        initials: "AM",
        gradient: "from-emerald-500 to-teal-600",
        metric: { value: "34%", label: "less churn" },
    },
    {
        quote: "The AI smart dunning alone recovered ₹12L in failed payments in our first month. The multi-gateway routing between Stripe and Razorpay is seamless.",
        name: "Priya Sharma",
        role: "Head of Engineering",
        company: "ScaleGrid",
        initials: "PS",
        gradient: "from-blue-500 to-indigo-600",
        metric: { value: "₹12L", label: "recovered" },
    },
    {
        quote: "We needed GST-compliant e-invoicing with IRN generation for our Indian B2B customers. Recurso had it built-in. No other billing platform came close.",
        name: "Vikram Desai",
        role: "Founder",
        company: "InvoiceHQ",
        initials: "VD",
        gradient: "from-violet-500 to-purple-600",
        metric: { value: "100%", label: "GST compliant" },
    },
    {
        quote: "Switching from Chargebee saved us $2,400/month. Self-hosted, zero platform fees, and we own our billing data. The TigerBeetle ledger is incredible.",
        name: "Sarah Chen",
        role: "VP Engineering",
        company: "CloudSync",
        initials: "SC",
        gradient: "from-amber-500 to-orange-600",
        metric: { value: "$2,400", label: "saved/mo" },
    },
]

const Testimonials = () => {
    const containerRef = useStaggerAnimation()

    return (
        <section className="relative py-32 overflow-hidden">
            <div className="absolute inset-0 bg-grid opacity-15" />
            <div className="absolute top-1/2 left-0 w-[400px] h-[400px] bg-emerald-500/[0.02] rounded-full blur-[100px]" />

            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                {/* Header */}
                <div className="text-center mb-16">
                    <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/[0.08] border border-emerald-500/20 mb-6">
                        <span className="text-xs font-semibold text-emerald-400 uppercase tracking-wider">Testimonials</span>
                    </div>
                    <h2 className="text-4xl sm:text-5xl font-bold text-white mb-5 tracking-tight">
                        Loved by{' '}
                        <span className="gradient-text">developers</span>
                    </h2>
                    <p className="text-lg text-gray-400 max-w-2xl mx-auto">
                        See how teams are building with Recurso
                    </p>
                </div>

                {/* Grid */}
                <div ref={containerRef} className="grid md:grid-cols-2 gap-5">
                    {testimonials.map((t) => (
                        <div
                            key={t.name}
                            data-animate
                            className="group glass rounded-2xl p-7 transition-all duration-500 hover:-translate-y-1"
                        >
                            {/* Stars */}
                            <div className="flex gap-0.5 mb-5">
                                {[1, 2, 3, 4, 5].map((i) => (
                                    <Star key={i} className="w-4 h-4 fill-amber-400 text-amber-400" />
                                ))}
                            </div>

                            {/* Quote */}
                            <p className="text-gray-300 leading-relaxed mb-6 text-[15px]">
                                "{t.quote}"
                            </p>

                            {/* Author + Metric */}
                            <div className="flex items-center justify-between pt-5 border-t border-white/[0.06]">
                                <div className="flex items-center gap-3">
                                    <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${t.gradient} flex items-center justify-center flex-shrink-0`}>
                                        <span className="text-xs font-bold text-white">{t.initials}</span>
                                    </div>
                                    <div>
                                        <p className="text-sm font-semibold text-white">{t.name}</p>
                                        <p className="text-xs text-gray-500">{t.role} at {t.company}</p>
                                    </div>
                                </div>

                                {t.metric && (
                                    <div className="text-right">
                                        <div className="text-lg font-bold text-emerald-400">{t.metric.value}</div>
                                        <div className="text-[10px] text-gray-500 uppercase tracking-wider">{t.metric.label}</div>
                                    </div>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </section>
    )
}

export default Testimonials
