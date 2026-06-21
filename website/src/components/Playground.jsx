import { useState } from 'react'
import { Play, Copy, Check, Code, Terminal } from 'lucide-react'

const Playground = () => {
    const [activeTab, setActiveTab] = useState('subscription')
    const [copied, setCopied] = useState(false)
    const [output, setOutput] = useState('')
    const [isRunning, setIsRunning] = useState(false)

    const examples = {
        subscription: {
            title: 'Create a Subscription',
            code: `import Recurso from '@recurso/sdk';

const recurso = new Recurso('your-api-key');

// Create a subscription for a customer
const subscription = await recurso.subscriptions.create({
  customer_id: 'cust_123',
  plan_id: 'plan_pro_monthly',
  billing_cycle_anchor: 'now',
  payment_method: 'pm_card_visa',
});

console.log('Subscription created:', subscription.id);
console.log('Status:', subscription.status);
console.log('Next billing:', subscription.current_period_end);`,
            output: `{
  "id": "sub_1234567890",
  "status": "active",
  "plan_id": "plan_pro_monthly",
  "current_period_start": "2024-01-15T00:00:00Z",
  "current_period_end": "2024-02-15T00:00:00Z",
  "amount": 2900,
  "currency": "USD"
}`,
        },
        invoice: {
            title: 'Generate an Invoice',
            code: `import Recurso from '@recurso/sdk';

const recurso = new Recurso('your-api-key');

// Create an invoice with line items
const invoice = await recurso.invoices.create({
  customer_id: 'cust_123',
  line_items: [
    { description: 'Pro Plan', amount: 2900 },
    { description: 'API Usage (10K calls)', amount: 500 },
  ],
  auto_send: true,
  due_days: 14,
});

console.log('Invoice:', invoice.number);
console.log('Total:', invoice.total);`,
            output: `{
  "id": "inv_abc123",
  "number": "INV-2024-0042",
  "status": "sent",
  "subtotal": 3400,
  "tax": 0,
  "total": 3400,
  "due_date": "2024-01-29T00:00:00Z",
  "pdf_url": "https://api.recurso.io/invoices/inv_abc123.pdf"
}`,
        },
        usage: {
            title: 'Track Usage',
            code: `import Recurso from '@recurso/sdk';

const recurso = new Recurso('your-api-key');

// Record usage for a customer
await recurso.usage.record({
  subscription_id: 'sub_123',
  metric: 'api_calls',
  quantity: 1500,
  timestamp: new Date(),
});

// Get usage summary
const summary = await recurso.usage.summary({
  subscription_id: 'sub_123',
  metric: 'api_calls',
  period: 'current_billing_period',
});

console.log('Total usage:', summary.total);
console.log('Billable amount:', summary.amount);`,
            output: `{
  "subscription_id": "sub_123",
  "metric": "api_calls",
  "period_start": "2024-01-01T00:00:00Z",
  "period_end": "2024-01-31T23:59:59Z",
  "total_usage": 45000,
  "included_usage": 10000,
  "overage_usage": 35000,
  "unit_price": 0.001,
  "billable_amount": 3500
}`,
        },
        webhook: {
            title: 'Handle Webhooks',
            code: `import express from 'express';
import Recurso from '@recurso/sdk';

const app = express();
const recurso = new Recurso('your-api-key');

app.post('/webhooks/recurso', async (req, res) => {
  const event = recurso.webhooks.verify(
    req.body,
    req.headers['recurso-signature'],
    'your-webhook-secret'
  );

  switch (event.type) {
    case 'invoice.paid':
      console.log('Invoice paid:', event.data.invoice_id);
      // Provision access, send receipt, etc.
      break;
    case 'subscription.canceled':
      console.log('Subscription canceled:', event.data.id);
      // Revoke access, send survey, etc.
      break;
  }

  res.json({ received: true });
});`,
            output: `Webhook received:
{
  "id": "evt_123abc",
  "type": "invoice.paid",
  "created_at": "2024-01-15T10:30:00Z",
  "data": {
    "invoice_id": "inv_abc123",
    "amount_paid": 3400,
    "payment_method": "card"
  }
}
✓ Webhook processed successfully`,
        },
    }

    const handleCopy = () => {
        navigator.clipboard.writeText(examples[activeTab].code)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    const handleRun = () => {
        setIsRunning(true)
        setOutput('Running...')
        setTimeout(() => {
            setOutput(examples[activeTab].output)
            setIsRunning(false)
        }, 800)
    }

    return (
        <section id="playground" className="py-24 relative overflow-hidden">
            {/* Background */}
            <div className="absolute inset-0 bg-grid opacity-20" />
            <div className="absolute top-1/3 right-0 w-[500px] h-[500px] bg-primary/5 rounded-full blur-3xl" />

            <div className="relative z-10 max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
                {/* Header */}
                <div className="text-center mb-16">
                    <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full glass border border-primary/20 mb-6">
                        <Code className="w-4 h-4 text-primary" />
                        <span className="text-sm text-gray-300">Interactive Playground</span>
                    </div>
                    <h2 className="text-4xl sm:text-5xl font-bold text-white mb-4">
                        Try it yourself
                    </h2>
                    <p className="text-xl text-gray-400 max-w-2xl mx-auto">
                        Explore the API with live examples. Copy, modify, and run.
                    </p>
                </div>

                {/* Tabs */}
                <div className="flex flex-wrap justify-center gap-2 mb-8">
                    {Object.entries(examples).map(([key, example]) => (
                        <button
                            key={key}
                            onClick={() => {
                                setActiveTab(key)
                                setOutput('')
                            }}
                            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${activeTab === key
                                    ? 'bg-primary text-black'
                                    : 'bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white'
                                }`}
                        >
                            {example.title}
                        </button>
                    ))}
                </div>

                {/* Code editor */}
                <div className="grid lg:grid-cols-2 gap-6">
                    {/* Code panel */}
                    <div className="glass border border-white/10 rounded-2xl overflow-hidden">
                        <div className="flex items-center justify-between px-4 py-3 border-b border-white/10 bg-white/5">
                            <div className="flex items-center gap-2">
                                <Terminal className="w-4 h-4 text-primary" />
                                <span className="text-sm text-gray-300">index.js</span>
                            </div>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={handleCopy}
                                    className="p-2 rounded-lg hover:bg-white/10 transition-colors"
                                    title="Copy code"
                                >
                                    {copied ? (
                                        <Check className="w-4 h-4 text-primary" />
                                    ) : (
                                        <Copy className="w-4 h-4 text-gray-400" />
                                    )}
                                </button>
                                <button
                                    onClick={handleRun}
                                    disabled={isRunning}
                                    className="flex items-center gap-2 px-3 py-1.5 bg-primary text-black text-sm font-medium rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
                                >
                                    <Play className="w-3 h-3" />
                                    Run
                                </button>
                            </div>
                        </div>
                        <pre className="p-4 text-sm overflow-x-auto h-[400px]">
                            <code className="text-gray-300 font-mono whitespace-pre">
                                {examples[activeTab].code}
                            </code>
                        </pre>
                    </div>

                    {/* Output panel */}
                    <div className="glass border border-white/10 rounded-2xl overflow-hidden">
                        <div className="flex items-center gap-2 px-4 py-3 border-b border-white/10 bg-white/5">
                            <div className="w-3 h-3 rounded-full bg-primary/50 animate-pulse" />
                            <span className="text-sm text-gray-300">Output</span>
                        </div>
                        <pre className="p-4 text-sm overflow-x-auto h-[400px]">
                            <code className={`font-mono whitespace-pre ${isRunning ? 'text-gray-500' : 'text-primary'}`}>
                                {output || 'Click "Run" to see the output'}
                            </code>
                        </pre>
                    </div>
                </div>
            </div>
        </section>
    )
}

export default Playground
