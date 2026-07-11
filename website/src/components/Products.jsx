import {
    ArrowUpRight,
    Repeat,
    CreditCard,
    Landmark,
    Scale,
    HeartHandshake,
    Braces,
    Check,
} from 'lucide-react'

const modules = [
    {
        icon: Repeat,
        name: 'Billing Core',
        blurb: 'The full subscription lifecycle — plans, trials, upgrades, and invoices that generate themselves.',
        bullets: [
            'Trials, upgrades, cancellations, proration',
            'Usage metering & metered billing',
            'Entitlements & feature gating',
            'Coupons, quotes & credit notes',
        ],
    },
    {
        icon: CreditCard,
        name: 'Payments',
        blurb: 'Stripe and Razorpay behind one API, with smart gateway routing per currency.',
        bullets: [
            'Stripe + Razorpay with smart routing',
            'UPI AutoPay mandates',
            'Hosted checkout & multi-currency FX',
        ],
    },
    {
        icon: Landmark,
        name: 'Tax & Compliance',
        blurb: 'India-first tax that is built in, not bolted on — and jurisdiction-aware beyond it.',
        bullets: [
            'GST with Place of Supply & HSN codes',
            'e-Invoicing (IRN/GSP) workflows',
            'EU VAT & zero-rated export invoices',
        ],
    },
    {
        icon: Scale,
        name: 'Revenue & Finance',
        blurb: 'An immutable double-entry ledger your accountant can actually sign off on.',
        bullets: [
            'Double-entry ledger on TigerBeetle',
            'Revenue recognition & deferred revenue',
            'QuickBooks/Xero sync & daily reconciliation',
            'Ask-your-data analytics (natural language → SQL)',
        ],
    },
    {
        icon: HeartHandshake,
        name: 'Retention',
        blurb: 'Recover failed payments and keep customers before they churn.',
        bullets: [
            'Configurable dunning campaigns',
            'Smart retries with exponential backoff',
            'Customer self-service portal & cancel flows',
        ],
    },
    {
        icon: Braces,
        name: 'Platform',
        blurb: 'A developer platform first: everything is an API with events you can trust.',
        bullets: [
            'REST API + OpenAPI 3.1 at /openapi.json',
            'Official SDKs — Go, Node & Python',
            'Signed webhooks & event delivery',
        ],
    },
]

const Products = () => (
    <section id="products" className="border-t border-line bg-surface-75 py-24 sm:py-28">
        <div className="mx-auto max-w-site px-4 sm:px-6 lg:px-8">
            <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
                <div className="max-w-2xl">
                    <p className="section-label">Product</p>
                    <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                        Everything a billing stack needs
                    </h2>
                    <p className="mt-4 text-base leading-relaxed text-fg-muted">
                        Six modules that work together as one engine. Use the whole stack or start
                        with the pieces you need — it all runs on your infrastructure.
                    </p>
                </div>
                <a
                    href="https://docs.recurso.dev/concepts"
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex shrink-0 items-center gap-1.5 text-sm font-medium text-brand hover:text-brand-light"
                >
                    Explore the concepts <ArrowUpRight className="h-3.5 w-3.5" />
                </a>
            </div>

            <div className="mt-14 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {modules.map((m) => (
                    <div key={m.name} className="card group p-6">
                        <div className="flex items-center gap-3">
                            <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-line bg-surface-200 transition-colors group-hover:border-brand/40">
                                <m.icon className="h-[18px] w-[18px] text-brand" />
                            </div>
                            <h3 className="text-[15px] font-semibold text-fg">{m.name}</h3>
                        </div>
                        <p className="mt-4 text-sm leading-relaxed text-fg-muted">{m.blurb}</p>
                        <ul className="mt-5 space-y-2.5 border-t border-line pt-5">
                            {m.bullets.map((b) => (
                                <li key={b} className="flex items-start gap-2.5 text-[13px] text-fg-muted">
                                    <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-brand" />
                                    <span>{b}</span>
                                </li>
                            ))}
                        </ul>
                    </div>
                ))}
            </div>
        </div>
    </section>
)

export default Products
