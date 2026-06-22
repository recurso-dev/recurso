const codeSnippet = `# Create a subscription
curl -X POST http://localhost:3000/v1/subscriptions \\
  -H "Authorization: Bearer $RECURSO_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "customer_id": "cus_123",
    "plan_id": "plan_pro_monthly"
  }'

# Generate an invoice
curl -X POST http://localhost:3000/v1/subscriptions/sub_123/advance \\
  -H "Authorization: Bearer $RECURSO_API_KEY"

# Record usage
curl -X POST http://localhost:3000/v1/usage/events \\
  -H "Authorization: Bearer $RECURSO_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "subscription_id": "sub_123",
    "metric": "api_calls",
    "quantity": 1000
  }'`;

const CodeExample = () => {
    return (
        <section className="relative py-32 overflow-hidden">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                <div className="grid lg:grid-cols-2 gap-12 items-center">
                    {/* Left: Code */}
                    <div className="order-2 lg:order-1">
                        <div className="code-block rounded-2xl overflow-hidden">
                            {/* Code header */}
                            <div className="flex items-center gap-2 px-4 py-3 border-b border-white/5">
                                <div className="flex gap-1.5">
                                    <div className="w-3 h-3 rounded-full bg-red-500/80" />
                                    <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
                                    <div className="w-3 h-3 rounded-full bg-green-500/80" />
                                </div>
                                <span className="text-xs text-gray-500 ml-2 font-mono">terminal</span>
                            </div>

                            {/* Code content */}
                            <div className="p-6 overflow-x-auto">
                                <pre className="text-sm leading-relaxed">
                                    <code className="text-gray-300 font-mono whitespace-pre">
                                        {codeSnippet.split('\n').map((line, i) => (
                                            <div key={i} className="flex">
                                                <span className="w-8 text-gray-600 select-none">{i + 1}</span>
                                                <span dangerouslySetInnerHTML={{
                                                    __html: highlightSyntax(line)
                                                }} />
                                            </div>
                                        ))}
                                    </code>
                                </pre>
                            </div>
                        </div>
                    </div>

                    {/* Right: Content */}
                    <div className="order-1 lg:order-2">
                        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-primary/10 border border-primary/20 mb-6">
                            <span className="text-xs font-medium text-primary">Developer First</span>
                        </div>

                        <h2 className="text-4xl sm:text-5xl font-bold text-white mb-6">
                            Build with{' '}
                            <span className="gradient-text">clean APIs</span>
                        </h2>

                        <p className="text-lg text-gray-400 mb-8 leading-relaxed">
                            Our REST API makes billing integration straightforward. Create subscriptions,
                            generate invoices, and track usage with simple HTTP requests.
                        </p>

                        <ul className="space-y-4">
                            {[
                                'RESTful JSON API',
                                'API key authentication',
                                'Webhook signature verification',
                                'Self-hosted — full control over your data'
                            ].map((item, i) => (
                                <li key={i} className="flex items-center gap-3 text-gray-300">
                                    <div className="w-5 h-5 rounded-full bg-primary/20 flex items-center justify-center">
                                        <svg className="w-3 h-3 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                                        </svg>
                                    </div>
                                    {item}
                                </li>
                            ))}
                        </ul>
                    </div>
                </div>
            </div>
        </section>
    )
}

// Simple syntax highlighting
function highlightSyntax(line) {
    return line
        .replace(/(curl|\\)/g, '<span class="text-purple-400">$1</span>')
        .replace(/(-X\s+(?:POST|GET|PUT|DELETE))/g, '<span class="text-blue-400">$1</span>')
        .replace(/(-H|-d)/g, '<span class="text-blue-400">$1</span>')
        .replace(/(#.*)/g, '<span class="text-gray-500">$1</span>')
        .replace(/(".*?")/g, '<span class="text-emerald-400">$1</span>')
        .replace(/(\$RECURSO_API_KEY)/g, '<span class="text-amber-400">$1</span>')
}

export default CodeExample
