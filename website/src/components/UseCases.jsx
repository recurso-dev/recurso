import { Building2, Gauge, ShoppingCart, FileSpreadsheet } from 'lucide-react'

const useCases = [
    {
        icon: Building2,
        title: 'B2B SaaS Subscriptions',
        description: 'Monthly, yearly, or custom billing cycles with proration and upgrades.',
        tags: ['Subscriptions', 'Proration', 'Trials'],
        gradient: 'from-emerald-500/20 via-transparent to-transparent',
    },
    {
        icon: Gauge,
        title: 'AI Models & APIs',
        description: 'Meter API calls, input/output tokens. Perfect for GenAI startups.',
        tags: ['Token Metering', 'Pay-as-you-go', 'Overage'],
        gradient: 'from-blue-500/20 via-transparent to-transparent',
    },
    {
        icon: ShoppingCart,
        title: 'Global E-Commerce',
        description: 'Accept USD via Stripe and INR via Razorpay from the same checkout.',
        tags: ['Smart Routing', 'Multi-currency', 'Global'],
        gradient: 'from-purple-500/20 via-transparent to-transparent',
    },
    {
        icon: FileSpreadsheet,
        title: 'Enterprise Invoicing',
        description: 'Net-30 terms, PO numbers, custom fields. Everything enterprises need.',
        tags: ['Net Terms', 'PDF Export', 'Tax Compliance'],
        gradient: 'from-orange-500/20 via-transparent to-transparent',
    },
]

const UseCases = () => {
    return (
        <section id="use-cases" className="relative py-32 overflow-hidden">
            {/* Background decoration */}
            <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[600px] h-[600px] bg-primary/5 rounded-full blur-3xl" />

            <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                {/* Section header */}
                <div className="text-center mb-16">
                    <h2 className="text-4xl sm:text-5xl font-bold text-white mb-4">
                        Built for every{' '}
                        <span className="gradient-text">business model</span>
                    </h2>
                    <p className="text-lg text-gray-400 max-w-2xl mx-auto">
                        From simple subscriptions to complex usage-based pricing, Recurso handles it all.
                    </p>
                </div>

                {/* Use case cards */}
                <div className="grid md:grid-cols-2 gap-6">
                    {useCases.map((useCase, index) => {
                        const Icon = useCase.icon

                        return (
                            <div
                                key={useCase.title}
                                className="group relative glass rounded-2xl p-8 transition-all duration-300 hover:-translate-y-1 overflow-hidden"
                            >
                                {/* Gradient background */}
                                <div className={`absolute inset-0 bg-gradient-to-br ${useCase.gradient} opacity-0 group-hover:opacity-100 transition-opacity duration-500`} />

                                <div className="relative z-10">
                                    <div className="flex items-start gap-6">
                                        <div className="w-14 h-14 rounded-2xl bg-white/5 flex items-center justify-center flex-shrink-0 group-hover:bg-primary/20 transition-colors">
                                            <Icon className="w-7 h-7 text-primary" />
                                        </div>

                                        <div className="flex-1">
                                            <h3 className="text-xl font-semibold text-white mb-2">
                                                {useCase.title}
                                            </h3>
                                            <p className="text-gray-400 mb-4">
                                                {useCase.description}
                                            </p>

                                            <div className="flex flex-wrap gap-2">
                                                {useCase.tags.map((tag) => (
                                                    <span
                                                        key={tag}
                                                        className="px-3 py-1 text-xs font-medium text-gray-400 bg-white/5 rounded-full"
                                                    >
                                                        {tag}
                                                    </span>
                                                ))}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )
                    })}
                </div>
            </div>
        </section>
    )
}

export default UseCases
