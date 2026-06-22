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
            code: `curl -X POST http://localhost:3000/v1/subscriptions \\
  -H "Authorization: Bearer your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "customer_id": "cust_123",
    "plan_id": "plan_pro_monthly",
    "billing_cycle_anchor": "now",
    "payment_method": "pm_card_visa"
  }'`,
            output: `{
  "id": "sub_1234567890",
  "status": "active",
  "customer_id": "cust_123",
  "plan_id": "plan_pro_monthly",
  "current_period_start": "2024-01-15T00:00:00Z",
  "current_period_end": "2024-02-15T00:00:00Z",
  "amount": 2900,
  "currency": "USD"
}`,
        },
        invoice: {
            title: 'Generate an Invoice',
            code: `curl -X POST http://localhost:3000/v1/subscriptions/sub_1234567890/advance \\
  -H "Authorization: Bearer your-api-key" \\
  -H "Content-Type: application/json"`,
            output: `{
  "id": "inv_abc123",
  "number": "INV-2024-0042",
  "status": "issued",
  "subscription_id": "sub_1234567890",
  "subtotal": 2900,
  "tax": 0,
  "total": 2900,
  "due_date": "2024-01-29T00:00:00Z"
}`,
        },
        usage: {
            title: 'Track Usage',
            code: `curl -X POST http://localhost:3000/v1/usage/events \\
  -H "Authorization: Bearer your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "subscription_id": "sub_123",
    "metric": "api_calls",
    "quantity": 1500,
    "timestamp": "2024-01-15T10:30:00Z"
  }'`,
            output: `{
  "id": "usage_evt_abc123",
  "subscription_id": "sub_123",
  "metric": "api_calls",
  "quantity": 1500,
  "timestamp": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-15T10:30:01Z"
}`,
        },
        webhook: {
            title: 'Handle Webhooks',
            code: `import express from 'express';
import crypto from 'crypto';

const app = express();

app.post('/webhooks/recurso', express.json(), (req, res) => {
  // Verify webhook signature
  const signature = req.headers['recurso-signature'];
  const expected = crypto
    .createHmac('sha256', 'your-webhook-secret')
    .update(JSON.stringify(req.body))
    .digest('hex');

  if (signature !== expected) {
    return res.status(401).json({ error: 'Invalid signature' });
  }

  switch (req.body.type) {
    case 'invoice.paid':
      console.log('Invoice paid:', req.body.data.invoice_id);
      break;
    case 'subscription.canceled':
      console.log('Canceled:', req.body.data.id);
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
                                <span className="text-sm text-gray-300">{activeTab === 'webhook' ? 'server.js' : 'terminal'}</span>
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
