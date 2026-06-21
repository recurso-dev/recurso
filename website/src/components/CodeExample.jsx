const codeSnippet = `import { Recurso } from '@recurso/sdk';

const recurso = new Recurso({
  apiKey: process.env.RECURSO_API_KEY
});

// Create a subscription
const subscription = await recurso.subscriptions.create({
  customerId: 'cus_123',
  planId: 'plan_pro_monthly',
  couponCode: 'LAUNCH20'
});

// Generate invoice
const invoice = await subscription.generateInvoice();

// Track usage
await recurso.usage.record({
  subscriptionId: subscription.id,
  metric: 'api_calls',
  value: 1000
});`;

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
                                <span className="text-xs text-gray-500 ml-2 font-mono">billing.ts</span>
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
                            Our TypeScript SDK makes billing integration a breeze. Create subscriptions,
                            generate invoices, and track usage with just a few lines of code.
                        </p>

                        <ul className="space-y-4">
                            {[
                                'Full TypeScript support with autocomplete',
                                'Promise-based async/await API',
                                'Built-in error handling and retries',
                                'Webhook signature verification included'
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
        .replace(/(import|from|const|await|async|process)/g, '<span class="text-purple-400">$1</span>')
        .replace(/('.*?')/g, '<span class="text-emerald-400">$1</span>')
        .replace(/(\/\/.*)/g, '<span class="text-gray-500">$1</span>')
        .replace(/(\{|\}|\(|\))/g, '<span class="text-gray-500">$1</span>')
        .replace(/(\.create|\.record|\.generateInvoice)/g, '<span class="text-blue-400">$1</span>')
}

export default CodeExample
