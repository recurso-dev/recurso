import { MapPin, Globe2, Check } from 'lucide-react'

const columns = [
    {
        icon: MapPin,
        title: 'Built for India',
        subtitle: 'Compliance that ships in the box — not a bolted-on integration.',
        items: [
            { name: 'GST engine', detail: 'CGST/SGST/IGST split with Place of Supply rules and HSN codes' },
            { name: 'e-Invoicing', detail: 'IRN generation workflows via GSP, ready for the e-invoice mandate' },
            { name: 'UPI AutoPay mandates', detail: 'Recurring mandates with real revocation, via Razorpay' },
            { name: 'Razorpay native', detail: 'First-class gateway support, including TDS tracking on the books' },
        ],
    },
    {
        icon: Globe2,
        title: 'Ready for the world',
        subtitle: 'The same engine bills in any currency, through the right gateway.',
        items: [
            { name: 'Multi-currency', detail: 'Price plans in INR, USD, EUR and more, with FX rate handling' },
            { name: 'Stripe routing', detail: 'Smart per-currency gateway routing between Stripe and Razorpay' },
            { name: 'EU VAT', detail: 'Jurisdiction-aware tax resolution by seller and buyer location' },
            { name: 'Export invoices', detail: 'Indian non-INR invoices zero-rated with LUT-aware notes' },
        ],
    },
]

const India = () => (
    <section className="border-t border-line bg-surface-75 py-24 sm:py-28">
        <div className="mx-auto max-w-site px-4 sm:px-6 lg:px-8">
            <div className="mx-auto max-w-2xl text-center">
                <p className="section-label">Compliance</p>
                <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                    Built for India, ready for the world
                </h2>
                <p className="mt-4 text-base leading-relaxed text-fg-muted">
                    Most billing tools treat Indian tax as an afterthought. Recurso starts there —
                    and bills globally with the same ledger underneath.
                </p>
            </div>

            <div className="mt-14 grid gap-4 lg:grid-cols-2">
                {columns.map((col) => (
                    <div key={col.title} className="card p-7 sm:p-8">
                        <div className="flex items-center gap-3">
                            <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-line bg-surface-200">
                                <col.icon className="h-5 w-5 text-brand" />
                            </div>
                            <div>
                                <h3 className="text-lg font-semibold text-fg">{col.title}</h3>
                            </div>
                        </div>
                        <p className="mt-4 text-sm text-fg-muted">{col.subtitle}</p>
                        <ul className="mt-6 space-y-4 border-t border-line pt-6">
                            {col.items.map((item) => (
                                <li key={item.name} className="flex items-start gap-3">
                                    <div className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-brand/10">
                                        <Check className="h-3 w-3 text-brand" />
                                    </div>
                                    <div className="text-sm">
                                        <span className="font-medium text-fg">{item.name}</span>
                                        <span className="text-fg-muted"> — {item.detail}</span>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    </div>
                ))}
            </div>
        </div>
    </section>
)

export default India
