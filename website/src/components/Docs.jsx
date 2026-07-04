import { Book, Zap, Code, Terminal, ArrowRight, Clock, ExternalLink } from 'lucide-react'

const DOCS_URL = 'https://docs.recurso.dev'

const Docs = () => {
    const quickLinks = [
        {
            icon: Zap,
            title: 'Quick Start',
            description: 'Get up and running in under 5 minutes',
            link: `${DOCS_URL}/quickstart`,
            time: '5 min',
        },
        {
            icon: Code,
            title: 'API Reference',
            description: 'Complete API documentation with examples',
            link: `${DOCS_URL}/api-reference`,
            time: '',
        },
        {
            icon: Terminal,
            title: 'Self-Hosting Guide',
            description: 'Deploy Recurso on your own infrastructure',
            link: `${DOCS_URL}/going-to-production`,
            time: '',
        },
        {
            icon: Book,
            title: 'Tutorials',
            description: 'Step-by-step guides for common use cases',
            link: `${DOCS_URL}/end-to-end`,
            time: '',
        },
    ]

    const gettingStarted = [
        {
            step: 1,
            title: 'Clone and run',
            code: `git clone https://github.com/swapnull-in/recur-so.git
cd recurso
docker compose up`,
        },
        {
            step: 2,
            title: 'Create an API key',
            code: `# Start the dashboard: cd frontend && npm run dev (http://localhost:5173)
# Navigate to Settings → API Keys → Create Key`,
        },
        {
            step: 3,
            title: 'Create your first subscription',
            code: `curl -X POST http://localhost:8080/v1/subscriptions \\
  -H "Authorization: Bearer your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{"customer_id": "cust_123", "plan_id": "plan_pro"}'`,
        },
    ]

    return (
        <section id="docs" className="py-24 relative overflow-hidden">
            {/* Background */}
            <div className="absolute inset-0 bg-grid opacity-20" />
            <div className="absolute top-0 left-1/3 w-[400px] h-[400px] bg-primary/5 rounded-full blur-3xl" />

            <div className="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                {/* Header */}
                <div className="text-center mb-16">
                    <h2 className="text-4xl sm:text-5xl font-bold text-white mb-4">
                        Documentation
                    </h2>
                    <p className="text-xl text-gray-400 max-w-2xl mx-auto">
                        Everything you need to integrate Recurso into your application
                    </p>
                </div>

                {/* Quick links grid */}
                <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-16">
                    {quickLinks.map((item, index) => (
                        <a
                            key={item.title}
                            href={item.link}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="group glass border border-white/10 rounded-xl p-6 hover:border-primary/30 transition-all duration-300 hover:-translate-y-1"
                            style={{ animationDelay: `${index * 0.1}s` }}
                        >
                            <div className="flex items-start justify-between mb-4">
                                <div className="p-2 rounded-lg bg-primary/10">
                                    <item.icon className="w-5 h-5 text-primary" />
                                </div>
                                <div className="flex items-center gap-2">
                                    {item.time && (
                                        <div className="flex items-center gap-1 text-xs text-gray-500">
                                            <Clock className="w-3 h-3" />
                                            {item.time}
                                        </div>
                                    )}
                                    <ExternalLink className="w-3 h-3 text-gray-500 opacity-0 group-hover:opacity-100 transition-opacity" />
                                </div>
                            </div>
                            <h3 className="text-lg font-semibold text-white mb-2 group-hover:text-primary transition-colors">
                                {item.title}
                            </h3>
                            <p className="text-gray-400 text-sm">{item.description}</p>
                        </a>
                    ))}
                </div>

                {/* Getting started guide */}
                <div className="glass border border-white/10 rounded-2xl p-8">
                    <div className="flex items-center justify-between mb-8">
                        <div>
                            <h3 className="text-2xl font-bold text-white mb-2">Getting Started</h3>
                            <p className="text-gray-400">Build your first subscription in 3 steps</p>
                        </div>
                        <a
                            href={`${DOCS_URL}/quickstart`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="hidden sm:flex items-center gap-2 text-primary hover:underline"
                        >
                            View full guide <ArrowRight className="w-4 h-4" />
                        </a>
                    </div>

                    <div className="space-y-6">
                        {gettingStarted.map((item) => (
                            <div key={item.step} className="flex gap-4">
                                {/* Step number */}
                                <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
                                    <span className="text-primary font-semibold">{item.step}</span>
                                </div>

                                {/* Content */}
                                <div className="flex-1">
                                    <h4 className="text-white font-medium mb-2">{item.title}</h4>
                                    <div className="bg-dark-900 rounded-lg p-4 overflow-x-auto">
                                        <pre className="text-sm">
                                            <code className="text-gray-300 font-mono whitespace-pre">
                                                {item.code}
                                            </code>
                                        </pre>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>

                    <div className="mt-8 pt-6 border-t border-white/10 flex flex-col sm:flex-row items-center justify-between gap-4">
                        <p className="text-gray-400">
                            Need help? Open an issue on{' '}
                            <a href="https://github.com/swapnull-in/recur-so/discussions" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                                GitHub Discussions
                            </a>
                        </p>
                        <a
                            href={DOCS_URL}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-2 px-6 py-3 bg-primary text-black font-semibold rounded-xl hover:bg-primary/90 transition-colors"
                        >
                            Read the docs
                            <ExternalLink className="w-4 h-4" />
                        </a>
                    </div>
                </div>
            </div>
        </section>
    )
}

export default Docs
