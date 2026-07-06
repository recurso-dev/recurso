import { Check, X, Minus } from 'lucide-react'

const categories = [
    {
        category: 'Pricing',
        features: [
            { name: 'Subscription billing', recurso: true, stripe: true, chargebee: true },
            { name: 'Usage-based billing', recurso: true, stripe: true, chargebee: true },
            { name: 'Hybrid pricing models', recurso: true, stripe: 'partial', chargebee: true },
            { name: 'Self-hosted option', recurso: true, stripe: false, chargebee: false },
            { name: 'No platform fees', recurso: true, stripe: false, chargebee: false },
        ],
    },
    {
        category: 'Payments',
        features: [
            { name: 'Multi-gateway support', recurso: true, stripe: false, chargebee: true },
            { name: 'Stripe integration', recurso: true, stripe: true, chargebee: true },
            { name: 'Razorpay + UPI AutoPay', recurso: true, stripe: false, chargebee: 'partial' },
            { name: 'Smart payment routing', recurso: true, stripe: false, chargebee: false },
            { name: 'AI Dunning', recurso: true, stripe: 'partial', chargebee: 'partial' },
        ],
    },
    {
        category: 'AI & Analytics',
        features: [
            { name: 'AI Churn Prediction', recurso: true, stripe: false, chargebee: false },
            { name: 'Revenue optimization', recurso: true, stripe: 'partial', chargebee: 'partial' },
            { name: 'Double-entry ledger', recurso: true, stripe: false, chargebee: false },
            { name: 'Webhooks & events', recurso: true, stripe: true, chargebee: true },
            { name: 'RBI/GST/E-Invoicing', recurso: true, stripe: 'partial', chargebee: true },
        ],
    },
    {
        category: 'Developer Experience',
        features: [
            { name: 'REST API', recurso: true, stripe: true, chargebee: true },
            { name: 'Open source (MIT)', recurso: true, stripe: false, chargebee: false },
            { name: 'Self-hostable', recurso: true, stripe: false, chargebee: false },
            { name: 'No vendor lock-in', recurso: true, stripe: false, chargebee: false },
        ],
    },
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
    return (
        <span className="mx-auto flex h-5 w-5 items-center justify-center rounded-full bg-amber-500/10">
            <Minus className="h-3 w-3 text-amber-400" />
        </span>
    )
}

const Comparison = () => (
    <section id="compare" className="border-t border-line py-24 sm:py-28">
        <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
            <div className="mx-auto max-w-2xl text-center">
                <p className="section-label">Compare</p>
                <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                    How Recurso compares
                </h2>
                <p className="mt-4 text-base leading-relaxed text-fg-muted">
                    Recurso vs. Chargebee and Stripe Billing, feature by feature.
                </p>
            </div>

            <div className="mt-14 overflow-hidden rounded-xl border border-line bg-surface-100">
                {/* Header */}
                <div className="grid grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))] gap-3 border-b border-line bg-surface-200 px-4 py-4 sm:px-6">
                    <div className="text-xs font-medium uppercase tracking-wider text-fg-subtle">Feature</div>
                    <div className="text-center text-sm font-semibold text-brand">Recurso</div>
                    <div className="text-center text-xs font-medium text-fg-muted sm:text-sm">Stripe Billing</div>
                    <div className="text-center text-xs font-medium text-fg-muted sm:text-sm">Chargebee</div>
                </div>

                {categories.map((cat) => (
                    <div key={cat.category}>
                        <div className="border-b border-line bg-surface-75 px-4 py-2.5 sm:px-6">
                            <span className="text-xs font-semibold uppercase tracking-wider text-fg-subtle">
                                {cat.category}
                            </span>
                        </div>
                        {cat.features.map((f, idx) => (
                            <div
                                key={f.name}
                                className={`grid grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))] items-center gap-3 px-4 py-3 transition-colors hover:bg-surface-200/50 sm:px-6 ${
                                    idx !== cat.features.length - 1 ? 'border-b border-line/60' : 'border-b border-line'
                                }`}
                            >
                                <div className="text-[13px] text-fg-muted sm:text-sm">{f.name}</div>
                                <Cell value={f.recurso} />
                                <Cell value={f.stripe} />
                                <Cell value={f.chargebee} />
                            </div>
                        ))}
                    </div>
                ))}

                {/* Pricing summary row */}
                <div className="grid grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))] items-center gap-3 bg-brand/[0.04] px-4 py-4 sm:px-6">
                    <div className="text-sm font-semibold text-fg">Cost of the software</div>
                    <div className="text-center text-xs font-semibold text-brand sm:text-sm">Free, self-hosted</div>
                    <div className="text-center text-xs text-fg-muted">0.5–0.8% of revenue</div>
                    <div className="text-center text-xs text-fg-muted">From $599/mo</div>
                </div>
            </div>

            {/* Legend */}
            <div className="mt-6 flex justify-center gap-6 text-xs text-fg-subtle">
                <span className="flex items-center gap-1.5">
                    <Check className="h-3 w-3 text-brand" /> Full support
                </span>
                <span className="flex items-center gap-1.5">
                    <Minus className="h-3 w-3 text-amber-400" /> Partial
                </span>
                <span className="flex items-center gap-1.5">
                    <X className="h-3 w-3 text-fg-subtle" /> Not available
                </span>
            </div>
        </div>
    </section>
)

export default Comparison
