import { Head } from 'vite-react-ssg'
import { Check, X, Minus, Github, ArrowRight, LineChart, PieChart, Calculator, Banknote } from 'lucide-react'
import Navbar from '../components/Navbar'
import Footer from '../components/Footer'

const CONTACT = 'mailto:swapnil.go20@gmail.com'

const tiers = [
    {
        name: 'Open Source',
        price: 'Free',
        period: 'forever',
        description: 'Self-host the full engine on your own infrastructure.',
        features: [
            'Every feature — no paywalled add-ons',
            'All payment gateways (Stripe, Razorpay)',
            'Usage metering & entitlements',
            'GST, e-invoicing, EU VAT',
            'Customer portal, API & webhooks',
            'Unlimited everything',
            'Community support',
        ],
        cta: 'Get started on GitHub',
        ctaLink: 'https://github.com/recurso-dev/recurso',
        highlighted: false,
    },
    {
        name: 'Cloud',
        price: '$299',
        period: '/mo + usage',
        description: 'Managed hosting — we run it, you ship.',
        badge: 'Most popular',
        priceNote: '10,000 invoices & 5,000 subscriptions included',
        features: [
            'Everything in Open Source',
            'Managed hosting & auto-scaling',
            'Daily backups & SSL included',
            '99.9% uptime SLA',
            'Email support',
            '$0.02 / extra invoice · $0.05 / extra subscription',
        ],
        cta: 'Join the Cloud waitlist',
        ctaLink: '/#waitlist',
        highlighted: true,
    },
    {
        name: 'Enterprise',
        price: 'Custom',
        period: '',
        description: 'Dedicated infrastructure, compliance & SLAs.',
        features: [
            'Everything in Cloud',
            'Dedicated infrastructure',
            'SOC 2 & on-premise deployment',
            'Custom integrations',
            'Custom rate limits',
            'Priority support with SLA',
        ],
        cta: 'Contact sales',
        ctaLink: `${CONTACT}?subject=Recurso%20Enterprise`,
        highlighted: false,
    },
]

// Condensed to the rows that actually differ across tiers.
const comparison = [
    { name: 'Core billing engine', os: true, cloud: true, ent: true },
    { name: 'Usage metering & entitlements', os: true, cloud: true, ent: true },
    { name: 'GST / e-invoicing / EU VAT', os: true, cloud: true, ent: true },
    { name: 'Hosting', os: 'Self-managed', cloud: 'Managed', ent: 'Managed / on-prem' },
    { name: 'Auto-scaling', os: false, cloud: true, ent: true },
    { name: 'Daily backups', os: false, cloud: true, ent: true },
    { name: 'Uptime SLA', os: '—', cloud: '99.9%', ent: '99.99%' },
    { name: 'Support', os: 'Community', cloud: 'Email', ent: 'Priority + SLA' },
    { name: 'SOC 2', os: false, cloud: false, ent: true },
    { name: 'Invoices included', os: 'Unlimited', cloud: '10,000/mo', ent: 'Custom' },
    { name: 'Active subscriptions', os: 'Unlimited', cloud: '5,000', ent: 'Custom' },
    { name: 'Advanced add-ons', os: 'Included', cloud: 'À la carte', ent: 'Included' },
]

const addOns = [
    { icon: LineChart, name: 'Churn Prediction', price: '+$99/mo', body: 'ML churn scoring per customer with high-risk alerts and retention workflows.' },
    { icon: PieChart, name: 'Advanced Analytics', price: '+$49/mo', body: 'MRR waterfall, cohort retention, LTV & payback, scheduled report exports.' },
    { icon: Calculator, name: 'Accounting Sync', price: '+$49/mo', body: 'Two-way sync with QuickBooks, Xero, Zoho & Tally — invoices, payments, credit notes.' },
    { icon: Banknote, name: 'Multi-Currency FX', price: '+$79/mo', body: '135+ currencies with daily FX rates, conversion at invoice time, gain/loss reporting.' },
]

const faqs = [
    { q: 'Is the open-source version really free?', a: 'Yes — MIT licensed, every feature included, no paywall. You provide the infrastructure; you get everything.' },
    { q: 'How does Cloud usage pricing work?', a: 'The $299/mo base covers 10,000 invoices and 5,000 active subscriptions. Beyond that it’s $0.02 per invoice and $0.05 per subscription, billed monthly. API requests are rate-limited, not charged for overage.' },
    { q: 'Why are add-ons free in open source but paid on Cloud?', a: 'Self-hosting includes everything because you run the compute. On Cloud, add-ons like churn prediction need extra managed resources — the add-on price covers that.' },
    { q: 'Can I move between self-hosted and Cloud?', a: 'Both ways. We provide full data exports and a migration tool — you own your data, and there’s no lock-in.' },
    { q: 'Do you take a percentage of my revenue?', a: 'Never. Pricing is a flat base plus predictable per-unit usage — we don’t tax your revenue like the incumbents.' },
]

const Cell = ({ value }) => {
    if (value === true) {
        return (
            <span className="mx-auto flex h-5 w-5 items-center justify-center rounded-full bg-brand/10">
                <Check className="h-3 w-3 text-brand" />
            </span>
        )
    }
    if (value === false) {
        return <X className="mx-auto h-3.5 w-3.5 text-fg-subtle/60" />
    }
    return <span className="block text-center text-[13px] text-fg-muted">{value}</span>
}

const PricingPage = () => (
    <div className="min-h-screen bg-surface">
        <Head>
            <title>Pricing — Recurso</title>
            <meta name="description" content="Free and open source to self-host, with managed Recurso Cloud from $299/mo. Predictable per-unit usage — we never take a percentage of your revenue." />
            <meta property="og:title" content="Pricing — Recurso" />
            <meta property="og:description" content="Open source and free to self-host. Managed Cloud from $299/mo. No revenue tax." />
            <meta property="og:type" content="website" />
        </Head>

        <Navbar />

        <main>
            {/* Hero */}
            <section className="relative overflow-hidden border-b border-line">
                <div className="absolute inset-0 bg-dots" />
                <div className="hero-glow absolute inset-x-0 top-0 h-full" />
                <div className="relative mx-auto max-w-3xl px-4 pb-14 pt-24 text-center sm:px-6 sm:pt-28 lg:px-8">
                    <p className="section-label">Pricing</p>
                    <h1 className="mx-auto mt-3 max-w-2xl text-4xl font-bold tracking-tight text-fg sm:text-5xl">
                        Predictable pricing, <span className="gradient-brand">no revenue tax</span>
                    </h1>
                    <p className="mx-auto mt-5 max-w-xl text-base leading-relaxed text-fg-muted">
                        Free and open source to self-host. Move to managed Cloud when you want us to run it.
                        Either way, we never take a cut of your revenue.
                    </p>
                </div>
            </section>

            {/* Tier cards */}
            <section className="relative">
                <div className="mx-auto -mt-2 max-w-5xl px-4 pb-8 sm:px-6 lg:px-8">
                    <div className="grid gap-4 md:grid-cols-3">
                        {tiers.map((plan) => (
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
                                    {plan.priceNote && <p className="mt-2 text-xs text-fg-subtle">{plan.priceNote}</p>}
                                </div>
                                <ul className="mt-6 flex-1 space-y-3">
                                    {plan.features.map((f) => (
                                        <li key={f} className="flex items-start gap-2.5">
                                            <Check className="mt-0.5 h-4 w-4 shrink-0 text-brand" />
                                            <span className="text-sm text-fg-muted">{f}</span>
                                        </li>
                                    ))}
                                </ul>
                                <a
                                    href={plan.ctaLink}
                                    {...(plan.ctaLink.startsWith('http') ? { target: '_blank', rel: 'noreferrer' } : {})}
                                    className={`group mt-8 ${plan.highlighted ? 'btn-primary' : 'btn-secondary'} w-full`}
                                >
                                    {plan.name === 'Open Source' && <Github className="h-4 w-4" />}
                                    {plan.cta}
                                    <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
                                </a>
                            </div>
                        ))}
                    </div>
                    <p className="mt-6 text-center text-xs text-fg-subtle">
                        All prices in USD. Cloud is in early access — join the waitlist for onboarding.
                    </p>
                </div>
            </section>

            {/* Comparison table */}
            <section className="border-t border-line py-20 sm:py-24">
                <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
                    <div className="mx-auto max-w-2xl text-center">
                        <p className="section-label">Compare plans</p>
                        <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">What each plan includes</h2>
                    </div>
                    <div className="mt-12 overflow-hidden rounded-xl border border-line bg-surface-100">
                        <div className="grid grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))] gap-3 border-b border-line bg-surface-200 px-4 py-4 sm:px-6">
                            <div className="text-xs font-medium uppercase tracking-wider text-fg-subtle">Capability</div>
                            <div className="text-center text-xs font-medium text-fg-muted sm:text-sm">Open Source</div>
                            <div className="text-center text-sm font-semibold text-brand">Cloud</div>
                            <div className="text-center text-xs font-medium text-fg-muted sm:text-sm">Enterprise</div>
                        </div>
                        {comparison.map((row, idx) => (
                            <div
                                key={row.name}
                                className={`grid grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))] items-center gap-3 px-4 py-3 transition-colors hover:bg-surface-200/50 sm:px-6 ${
                                    idx !== comparison.length - 1 ? 'border-b border-line/60' : ''
                                }`}
                            >
                                <div className="text-[13px] text-fg-muted sm:text-sm">{row.name}</div>
                                <Cell value={row.os} />
                                <Cell value={row.cloud} />
                                <Cell value={row.ent} />
                            </div>
                        ))}
                    </div>
                </div>
            </section>

            {/* Usage-based pricing */}
            <section className="border-t border-line py-20 sm:py-24">
                <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
                    <div className="grid items-center gap-10 lg:grid-cols-2">
                        <div>
                            <p className="section-label">Usage-based, not revenue-based</p>
                            <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                                A flat base plus predictable per-unit usage
                            </h2>
                            <p className="mt-4 text-base leading-relaxed text-fg-muted">
                                Cloud is a $299/mo base fee — managed hosting, auto-scaling, daily backups, SSL,
                                and email support — with generous allowances. You only pay more as you grow, at a
                                fixed rate per invoice and subscription. No percentage of GMV, ever.
                            </p>
                        </div>
                        <div className="rounded-xl border border-line bg-surface-100 p-7">
                            <p className="text-xs font-medium uppercase tracking-wider text-fg-subtle">Worked example</p>
                            <dl className="mt-5 space-y-3 text-sm">
                                <div className="flex items-center justify-between">
                                    <dt className="text-fg-muted">Base (3,000 subs · 8,000 invoices)</dt>
                                    <dd className="font-mono text-fg">$299</dd>
                                </div>
                                <div className="flex items-center justify-between border-t border-line/60 pt-3">
                                    <dt className="text-fg-muted">Scale to 7,000 subs · 15,000 invoices</dt>
                                    <dd className="font-mono text-fg-subtle">+ $200</dd>
                                </div>
                                <div className="flex items-center justify-between border-t border-line pt-3 text-base">
                                    <dt className="font-semibold text-fg">Total at that scale</dt>
                                    <dd className="font-mono font-semibold text-brand">$499/mo</dd>
                                </div>
                            </dl>
                            <p className="mt-4 text-xs text-fg-subtle">
                                5,000 extra invoices × $0.02 + 2,000 extra subs × $0.05.
                            </p>
                        </div>
                    </div>
                </div>
            </section>

            {/* Add-ons */}
            <section className="border-t border-line py-20 sm:py-24">
                <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
                    <div className="mx-auto max-w-2xl text-center">
                        <p className="section-label">Cloud add-ons</p>
                        <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">Pay only for what you need</h2>
                        <p className="mt-4 text-base leading-relaxed text-fg-muted">
                            Every add-on is included free in the open-source edition. On Cloud, turn on just the
                            advanced capabilities you want.
                        </p>
                    </div>
                    <div className="mt-12 grid gap-4 sm:grid-cols-2">
                        {addOns.map((a) => (
                            <div key={a.name} className="flex items-start gap-4 rounded-xl border border-line bg-surface-100 p-6">
                                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-line bg-surface-200">
                                    <a.icon className="h-[18px] w-[18px] text-brand" />
                                </div>
                                <div>
                                    <div className="flex items-baseline gap-2">
                                        <h3 className="text-[15px] font-semibold text-fg">{a.name}</h3>
                                        <span className="text-xs font-medium text-brand">{a.price}</span>
                                    </div>
                                    <p className="mt-1.5 text-[13px] leading-relaxed text-fg-muted">{a.body}</p>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </section>

            {/* FAQ */}
            <section className="border-t border-line py-20 sm:py-24">
                <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
                    <div className="mx-auto max-w-2xl text-center">
                        <p className="section-label">FAQ</p>
                        <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">Common questions</h2>
                    </div>
                    <div className="mt-12 divide-y divide-line rounded-xl border border-line bg-surface-100">
                        {faqs.map((f) => (
                            <div key={f.q} className="p-6">
                                <h3 className="text-[15px] font-semibold text-fg">{f.q}</h3>
                                <p className="mt-2 text-sm leading-relaxed text-fg-muted">{f.a}</p>
                            </div>
                        ))}
                    </div>
                </div>
            </section>

            {/* Closing CTA */}
            <section className="relative overflow-hidden border-t border-line">
                <div className="absolute inset-0 bg-dots" />
                <div className="hero-glow absolute inset-x-0 bottom-0 h-full rotate-180" />
                <div className="relative mx-auto max-w-site px-4 py-24 text-center sm:px-6 sm:py-28 lg:px-8">
                    <h2 className="mx-auto max-w-2xl text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                        Start free tonight, upgrade when you&apos;re ready
                    </h2>
                    <p className="mx-auto mt-4 max-w-xl text-base leading-relaxed text-fg-muted">
                        Clone and self-host the whole engine, or get in line for managed Cloud.
                    </p>
                    <div className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
                        <a href="https://github.com/recurso-dev/recurso" target="_blank" rel="noreferrer" className="btn-primary w-full sm:w-auto">
                            <Github className="h-4 w-4" /> Start self-hosting
                        </a>
                        <a href="/#waitlist" className="btn-secondary w-full sm:w-auto">
                            Join the Cloud waitlist <ArrowRight className="h-4 w-4" />
                        </a>
                    </div>
                </div>
            </section>
        </main>

        <Footer />
    </div>
)

export default PricingPage
