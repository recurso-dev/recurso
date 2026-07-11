import { Check, X, ArrowRight } from 'lucide-react'

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
        ctaLink: 'https://github.com/swapnull-in/recur-so',
        highlighted: false,
    },
    {
        name: 'Pro',
        price: '$299',
        period: '/month',
        description: 'For growing SaaS businesses',
        badge: 'Most Popular',
        priceNote: '+ $0.05 per transaction after included volume',
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
        cta: 'Join Cloud waitlist',
        ctaLink: 'mailto:swapnil.go20@gmail.com',
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
        ctaLink: 'mailto:swapnil.go20@gmail.com',
        highlighted: false,
    },
]

const Pricing = () => (
    <section id="pricing" className="border-t border-line py-24 sm:py-28">
        <div className="mx-auto max-w-site px-4 sm:px-6 lg:px-8">
            <div className="mx-auto max-w-2xl text-center">
                <p className="section-label">Pricing</p>
                <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                    Predictable pricing, no revenue tax
                </h2>
                <p className="mt-4 text-base leading-relaxed text-fg-muted">
                    Start free with self-hosted. Move to managed cloud when you want us to run it.
                    Either way, we never take a percentage of your revenue.
                </p>
            </div>

            <div className="mx-auto mt-14 grid max-w-5xl gap-4 md:grid-cols-3">
                {plans.map((plan) => (
                    <div
                        key={plan.name}
                        className={`relative flex flex-col rounded-xl border p-7 ${
                            plan.highlighted
                                ? 'border-brand/50 bg-surface-100 shadow-[0_0_48px_-16px_rgba(16,185,129,0.35)]'
                                : 'border-line bg-surface-100'
                        }`}
                    >
                        {plan.badge && (
                            <div className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-brand px-3 py-1 text-[11px] font-semibold text-[#062e21]">
                                {plan.badge}
                            </div>
                        )}

                        <div>
                            <h3 className="text-lg font-semibold text-fg">{plan.name}</h3>
                            <p className="mt-1 text-sm text-fg-subtle">{plan.description}</p>
                        </div>

                        <div className="mt-6 border-b border-line pb-6">
                            <span className="text-4xl font-bold tracking-tight text-fg">{plan.price}</span>
                            {plan.period && <span className="ml-1 text-sm text-fg-subtle">{plan.period}</span>}
                            {plan.priceNote && (
                                <p className="mt-2 text-xs text-fg-subtle">{plan.priceNote}</p>
                            )}
                        </div>

                        <ul className="mt-6 flex-1 space-y-3">
                            {plan.features.map((f) => (
                                <li key={f.text} className="flex items-center gap-2.5">
                                    {f.included ? (
                                        <Check className="h-4 w-4 shrink-0 text-brand" />
                                    ) : (
                                        <X className="h-4 w-4 shrink-0 text-fg-subtle/50" />
                                    )}
                                    <span className={`text-sm ${f.included ? 'text-fg-muted' : 'text-fg-subtle/70'}`}>
                                        {f.text}
                                    </span>
                                </li>
                            ))}
                        </ul>

                        <a
                            href={plan.ctaLink}
                            {...(plan.ctaLink.startsWith('http') ? { target: '_blank', rel: 'noreferrer' } : {})}
                            className={`group mt-8 ${plan.highlighted ? 'btn-primary' : 'btn-secondary'} w-full`}
                        >
                            {plan.cta}
                            <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
                        </a>
                    </div>
                ))}
            </div>

            <p className="mt-12 text-center text-sm text-fg-subtle">
                Have questions?{' '}
                <a href="mailto:swapnil.go20@gmail.com" className="font-medium text-brand hover:text-brand-light">
                    Contact us
                </a>
            </p>
        </div>
    </section>
)

export default Pricing
