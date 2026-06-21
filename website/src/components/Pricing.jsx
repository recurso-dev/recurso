import { Check, X, ArrowRight } from 'lucide-react'
import { useStaggerAnimation } from '../hooks/useScrollAnimation'

const Pricing = () => {
    const containerRef = useStaggerAnimation()

    const plans = [
        {
            name: 'Open Source',
            price: 'Free',
            period: 'forever',
            description: 'Self-host on your own infrastructure',
            features: [
                { text: 'Unlimited subscriptions', included: true },
                { text: 'Multi-gateway payments', included: true },
                { text: 'Usage metering', included: true },
                { text: 'Invoicing & credit notes', included: true },
                { text: 'API & webhooks', included: true },
                { text: 'Community support', included: true },
                { text: 'Priority support', included: false },
                { text: 'Managed hosting', included: false },
            ],
            cta: 'Get Started',
            ctaLink: 'https://github.com/recur-so/recurso',
            highlighted: false,
        },
        {
            name: 'Pro',
            price: '$299',
            period: '/month',
            description: 'For growing SaaS businesses',
            badge: 'Most Popular',
            features: [
                { text: 'Everything in Open Source', included: true },
                { text: 'Managed cloud hosting', included: true },
                { text: 'Auto-scaling infrastructure', included: true },
                { text: '99.9% uptime SLA', included: true },
                { text: 'Priority email support', included: true },
                { text: 'Custom integrations', included: true },
                { text: 'Dedicated account manager', included: false },
                { text: 'Custom SLA', included: false },
            ],
            cta: 'Start Free Trial',
            ctaLink: '#',
            highlighted: true,
        },
        {
            name: 'Enterprise',
            price: 'Custom',
            period: '',
            description: 'For high-volume businesses',
            features: [
                { text: 'Everything in Pro', included: true },
                { text: 'Dedicated infrastructure', included: true },
                { text: 'Custom SLA up to 99.99%', included: true },
                { text: 'SOC 2 Type II compliance', included: true },
                { text: 'Dedicated account manager', included: true },
                { text: 'On-premise deployment', included: true },
                { text: '24/7 phone support', included: true },
                { text: 'Custom development', included: true },
            ],
            cta: 'Contact Sales',
            ctaLink: '#',
            highlighted: false,
        },
    ]

    return (
        <section id="pricing" className="py-28 relative overflow-hidden">
            {/* Background */}
            <div className="absolute inset-0 bg-grid opacity-15" />
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-emerald-500/[0.03] rounded-full blur-[120px]" />

            <div className="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                {/* Header */}
                <div className="text-center mb-16">
                    <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/[0.08] border border-emerald-500/20 mb-6">
                        <span className="text-xs font-semibold text-emerald-400 uppercase tracking-wider">Pricing</span>
                    </div>
                    <h2 className="text-4xl sm:text-5xl font-bold text-white mb-5 tracking-tight">
                        Simple, transparent pricing
                    </h2>
                    <p className="text-lg text-gray-400 max-w-2xl mx-auto">
                        Start free with self-hosted. Scale to managed cloud when you're ready.
                    </p>
                </div>

                {/* Pricing cards */}
                <div ref={containerRef} className="grid md:grid-cols-3 gap-5 max-w-5xl mx-auto">
                    {plans.map((plan) => (
                        <div
                            key={plan.name}
                            data-animate
                            className={`relative rounded-2xl p-7 transition-all duration-500 hover:-translate-y-1 ${plan.highlighted
                                    ? 'bg-gradient-to-b from-emerald-500/[0.12] to-emerald-500/[0.02] border-2 border-emerald-500/30 glow-green'
                                    : 'glass border border-white/[0.06]'
                                }`}
                        >
                            {plan.badge && (
                                <div className="absolute -top-3.5 left-1/2 -translate-x-1/2 px-4 py-1 bg-emerald-500 text-black text-xs font-bold rounded-full shadow-lg shadow-emerald-500/20">
                                    {plan.badge}
                                </div>
                            )}

                            <div className="mb-6">
                                <h3 className="text-xl font-bold text-white mb-1">{plan.name}</h3>
                                <p className="text-sm text-gray-500">{plan.description}</p>
                            </div>

                            <div className="mb-8">
                                <span className="text-5xl font-extrabold text-white tracking-tight">{plan.price}</span>
                                {plan.period && <span className="text-gray-500 ml-1">{plan.period}</span>}
                            </div>

                            <ul className="space-y-3 mb-8">
                                {plan.features.map((feature) => (
                                    <li key={feature.text} className="flex items-center gap-3">
                                        {feature.included ? (
                                            <div className="w-5 h-5 rounded-full bg-emerald-500/10 flex items-center justify-center flex-shrink-0">
                                                <Check className="w-3 h-3 text-emerald-400" />
                                            </div>
                                        ) : (
                                            <X className="w-4 h-4 text-gray-700 flex-shrink-0 ml-0.5" />
                                        )}
                                        <span className={`text-sm ${feature.included ? 'text-gray-300' : 'text-gray-600'}`}>
                                            {feature.text}
                                        </span>
                                    </li>
                                ))}
                            </ul>

                            <a
                                href={plan.ctaLink}
                                className={`group flex items-center justify-center gap-2 w-full py-3 px-6 font-semibold rounded-xl transition-all duration-300 text-sm ${plan.highlighted
                                        ? 'bg-emerald-500 text-black hover:bg-emerald-400 glow-ring'
                                        : 'bg-white/[0.06] text-white hover:bg-white/[0.1]'
                                    }`}
                            >
                                {plan.cta}
                                <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
                            </a>
                        </div>
                    ))}
                </div>

                {/* FAQ teaser */}
                <div className="text-center mt-16">
                    <p className="text-sm text-gray-500">
                        Have questions?{' '}
                        <a href="#" className="text-emerald-400 hover:text-emerald-300 transition-colors font-medium">
                            Check our FAQ
                        </a>
                        {' '}or{' '}
                        <a href="#" className="text-emerald-400 hover:text-emerald-300 transition-colors font-medium">
                            contact us
                        </a>
                    </p>
                </div>
            </div>
        </section>
    )
}

export default Pricing
