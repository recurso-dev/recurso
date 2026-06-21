import { Check, X, Minus } from 'lucide-react'
import { useScrollAnimation } from '../hooks/useScrollAnimation'

const Comparison = () => {
    const sectionRef = useScrollAnimation()

    const features = [
        {
            category: 'Pricing', features: [
                { name: 'Subscription billing', recurso: true, stripe: true, chargebee: true },
                { name: 'Usage-based billing', recurso: true, stripe: true, chargebee: true },
                { name: 'Hybrid pricing models', recurso: true, stripe: 'partial', chargebee: true },
                { name: 'Self-hosted option', recurso: true, stripe: false, chargebee: false },
                { name: 'No platform fees', recurso: true, stripe: false, chargebee: false },
            ]
        },
        {
            category: 'Payments', features: [
                { name: 'Multi-gateway support', recurso: true, stripe: false, chargebee: true },
                { name: 'Stripe integration', recurso: true, stripe: true, chargebee: true },
                { name: 'Razorpay + UPI AutoPay', recurso: true, stripe: false, chargebee: 'partial' },
                { name: 'Smart payment routing', recurso: true, stripe: false, chargebee: false },
                { name: 'AI Dunning', recurso: true, stripe: 'partial', chargebee: 'partial' },
            ]
        },
        {
            category: 'AI & Analytics', features: [
                { name: 'AI Churn Prediction', recurso: true, stripe: false, chargebee: false },
                { name: 'Revenue optimization', recurso: true, stripe: 'partial', chargebee: 'partial' },
                { name: 'Double-entry ledger', recurso: true, stripe: false, chargebee: false },
                { name: 'Webhooks & events', recurso: true, stripe: true, chargebee: true },
                { name: 'RBI/GST/E-Invoicing', recurso: true, stripe: 'partial', chargebee: true },
            ]
        },
        {
            category: 'Developer Experience', features: [
                { name: 'REST API', recurso: true, stripe: true, chargebee: true },
                { name: 'TypeScript SDK', recurso: true, stripe: true, chargebee: true },
                { name: 'Open source (MIT)', recurso: true, stripe: false, chargebee: false },
                { name: 'Self-hostable', recurso: true, stripe: false, chargebee: false },
                { name: 'No vendor lock-in', recurso: true, stripe: false, chargebee: false },
            ]
        },
    ]

    // Count total Recurso wins
    const totalFeatures = features.reduce((acc, cat) => acc + cat.features.length, 0)
    const recursoWins = features.reduce((acc, cat) =>
        acc + cat.features.filter(f => f.recurso === true && (f.stripe === false || f.chargebee === false)).length, 0)

    const renderCell = (value) => {
        if (value === true) {
            return (
                <div className="flex justify-center">
                    <div className="w-6 h-6 rounded-full bg-emerald-500/10 flex items-center justify-center">
                        <Check className="w-3.5 h-3.5 text-emerald-400" />
                    </div>
                </div>
            )
        } else if (value === false) {
            return (
                <div className="flex justify-center">
                    <X className="w-4 h-4 text-gray-700" />
                </div>
            )
        } else {
            return (
                <div className="flex justify-center">
                    <div className="w-6 h-6 rounded-full bg-amber-500/10 flex items-center justify-center">
                        <Minus className="w-3.5 h-3.5 text-amber-400" />
                    </div>
                </div>
            )
        }
    }

    return (
        <section id="comparison" className="py-28 relative overflow-hidden">
            <div className="absolute inset-0 bg-grid opacity-15" />
            <div className="absolute bottom-0 left-1/4 w-[400px] h-[400px] bg-emerald-500/[0.02] rounded-full blur-[100px]" />

            <div ref={sectionRef} className="relative z-10 max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
                {/* Header */}
                <div className="text-center mb-16">
                    <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/[0.08] border border-emerald-500/20 mb-6">
                        <span className="text-xs font-semibold text-emerald-400 uppercase tracking-wider">Comparison</span>
                    </div>
                    <h2 className="text-4xl sm:text-5xl font-bold text-white mb-5 tracking-tight">
                        How we compare
                    </h2>
                    <p className="text-lg text-gray-400 max-w-2xl mx-auto">
                        See why developers choose Recurso over legacy billing platforms
                    </p>
                </div>

                {/* Table */}
                <div className="glass-strong rounded-2xl overflow-hidden border border-white/[0.06]">
                    {/* Header row */}
                    <div className="grid grid-cols-4 gap-4 p-5 border-b border-white/[0.08] bg-white/[0.03]">
                        <div className="text-sm text-gray-500 font-medium">Feature</div>
                        <div className="text-center">
                            <span className="text-emerald-400 font-bold text-base">Recurso</span>
                        </div>
                        <div className="text-center">
                            <span className="text-gray-400 font-medium text-sm">Stripe Billing</span>
                        </div>
                        <div className="text-center">
                            <span className="text-gray-400 font-medium text-sm">Chargebee</span>
                        </div>
                    </div>

                    {/* Feature rows */}
                    {features.map((category) => (
                        <div key={category.category}>
                            <div className="px-5 py-3 bg-white/[0.02] border-b border-white/[0.05]">
                                <span className="text-sm font-semibold text-white">{category.category}</span>
                            </div>

                            {category.features.map((feature, idx) => (
                                <div
                                    key={feature.name}
                                    className={`grid grid-cols-4 gap-4 px-5 py-3.5 ${idx !== category.features.length - 1 ? 'border-b border-white/[0.04]' : ''
                                        } hover:bg-white/[0.02] transition-colors`}
                                >
                                    <div className="text-sm text-gray-300">{feature.name}</div>
                                    <div>{renderCell(feature.recurso)}</div>
                                    <div>{renderCell(feature.stripe)}</div>
                                    <div>{renderCell(feature.chargebee)}</div>
                                </div>
                            ))}
                        </div>
                    ))}

                    {/* Summary row */}
                    <div className="grid grid-cols-4 gap-4 p-5 border-t border-white/[0.08] bg-emerald-500/[0.03]">
                        <div className="text-sm font-semibold text-white">Unique advantages</div>
                        <div className="text-center">
                            <span className="text-lg font-bold text-emerald-400">{recursoWins}</span>
                        </div>
                        <div className="text-center">
                            <span className="text-sm text-gray-600">—</span>
                        </div>
                        <div className="text-center">
                            <span className="text-sm text-gray-600">—</span>
                        </div>
                    </div>
                </div>

                {/* Legend */}
                <div className="flex justify-center gap-8 mt-8 text-xs">
                    <div className="flex items-center gap-2">
                        <div className="w-5 h-5 rounded-full bg-emerald-500/10 flex items-center justify-center">
                            <Check className="w-3 h-3 text-emerald-400" />
                        </div>
                        <span className="text-gray-500">Full support</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <div className="w-5 h-5 rounded-full bg-amber-500/10 flex items-center justify-center">
                            <Minus className="w-3 h-3 text-amber-400" />
                        </div>
                        <span className="text-gray-500">Partial</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <X className="w-3.5 h-3.5 text-gray-700" />
                        <span className="text-gray-500">Not available</span>
                    </div>
                </div>
            </div>
        </section>
    )
}

export default Comparison
