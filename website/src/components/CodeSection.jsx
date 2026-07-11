import { useState } from 'react'
import { ArrowUpRight, Terminal, Braces } from 'lucide-react'

/* Real API usage — mirrors the docs quickstart and the SDK READMEs */

const curlCode = (
    <>
        <span className="tok-dim"># 1. Create a plan</span>{'\n'}
        <span className="tok-cmd">curl -X POST http://localhost:8080/v1/plans \</span>{'\n'}
        <span className="tok-cmd">{'  '}-H </span><span className="tok-str">"Authorization: Bearer your-api-key"</span><span className="tok-cmd"> \</span>{'\n'}
        <span className="tok-cmd">{'  '}-H </span><span className="tok-str">"Content-Type: application/json"</span><span className="tok-cmd"> \</span>{'\n'}
        <span className="tok-cmd">{'  '}-d </span><span className="tok-cmd">{"'{"}</span>{'\n'}
        <span className="tok-key">{'    "name"'}</span><span className="tok-cmd">: </span><span className="tok-str">"Pro Plan"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-key">{'    "interval"'}</span><span className="tok-cmd">: </span><span className="tok-str">"month"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-key">{'    "prices"'}</span><span className="tok-cmd">: [</span>{'\n'}
        <span className="tok-cmd">{'      { '}</span><span className="tok-key">"amount"</span><span className="tok-cmd">: 4999, </span><span className="tok-key">"currency"</span><span className="tok-cmd">: </span><span className="tok-str">"INR"</span><span className="tok-cmd">{' },'}</span>{'\n'}
        <span className="tok-cmd">{'      { '}</span><span className="tok-key">"amount"</span><span className="tok-cmd">: 999, </span><span className="tok-key">"currency"</span><span className="tok-cmd">: </span><span className="tok-str">"USD"</span><span className="tok-cmd">{' }'}</span>{'\n'}
        <span className="tok-cmd">{'    ]'}</span>{'\n'}
        <span className="tok-cmd">{"  }'"}</span>{'\n\n'}
        <span className="tok-dim"># 2. Add a customer</span>{'\n'}
        <span className="tok-cmd">curl -X POST http://localhost:8080/v1/customers \</span>{'\n'}
        <span className="tok-cmd">{'  '}-H </span><span className="tok-str">"Authorization: Bearer your-api-key"</span><span className="tok-cmd"> \</span>{'\n'}
        <span className="tok-cmd">{'  '}-d </span><span className="tok-cmd">{"'{ "}</span><span className="tok-key">"name"</span><span className="tok-cmd">: </span><span className="tok-str">"Acme Corp"</span><span className="tok-cmd">, </span><span className="tok-key">"email"</span><span className="tok-cmd">: </span><span className="tok-str">"billing@acme.com"</span><span className="tok-cmd">{" }'"}</span>{'\n\n'}
        <span className="tok-dim"># 3. Start a subscription</span>{'\n'}
        <span className="tok-cmd">curl -X POST http://localhost:8080/v1/subscriptions \</span>{'\n'}
        <span className="tok-cmd">{'  '}-H </span><span className="tok-str">"Authorization: Bearer your-api-key"</span><span className="tok-cmd"> \</span>{'\n'}
        <span className="tok-cmd">{'  '}-d </span><span className="tok-cmd">{"'{ "}</span><span className="tok-key">"customer_id"</span><span className="tok-cmd">: </span><span className="tok-str">"cust_abc"</span><span className="tok-cmd">, </span><span className="tok-key">"plan_id"</span><span className="tok-cmd">: </span><span className="tok-str">"plan_pro"</span><span className="tok-cmd">, </span><span className="tok-key">"payment_gateway"</span><span className="tok-cmd">: </span><span className="tok-str">"razorpay"</span><span className="tok-cmd">{" }'"}</span>
    </>
)

const nodeCode = (
    <>
        <span className="tok-flag">import</span><span className="tok-cmd">{' { Recurso } '}</span><span className="tok-flag">from</span><span className="tok-cmd"> </span><span className="tok-str">'recurso-node'</span><span className="tok-cmd">;</span>{'\n\n'}
        <span className="tok-flag">const</span><span className="tok-cmd"> recurso = </span><span className="tok-flag">new</span><span className="tok-cmd"> </span><span className="tok-key">Recurso</span><span className="tok-cmd">(</span><span className="tok-str">'rsk_live_your_api_key'</span><span className="tok-cmd">, </span><span className="tok-str">'https://billing.example.com'</span><span className="tok-cmd">);</span>{'\n\n'}
        <span className="tok-dim">{'// 1. Create a plan'}</span>{'\n'}
        <span className="tok-flag">const</span><span className="tok-cmd"> plan = </span><span className="tok-flag">await</span><span className="tok-cmd"> recurso.plans.</span><span className="tok-key">create</span><span className="tok-cmd">{'({'}</span>{'\n'}
        <span className="tok-cmd">{'  name: '}</span><span className="tok-str">'Pro Plan'</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  code: '}</span><span className="tok-str">'PRO-USD'</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  amount: '}</span><span className="tok-brand">2900</span><span className="tok-cmd">,          </span><span className="tok-dim">{'// minor units'}</span>{'\n'}
        <span className="tok-cmd">{'  currency: '}</span><span className="tok-str">'USD'</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  interval_unit: '}</span><span className="tok-str">'month'</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'});'}</span>{'\n\n'}
        <span className="tok-dim">{'// 2. Add a customer'}</span>{'\n'}
        <span className="tok-flag">const</span><span className="tok-cmd"> customer = </span><span className="tok-flag">await</span><span className="tok-cmd"> recurso.customers.</span><span className="tok-key">create</span><span className="tok-cmd">{'({'}</span>{'\n'}
        <span className="tok-cmd">{'  name: '}</span><span className="tok-str">'Jane User'</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  email: '}</span><span className="tok-str">'jane@example.com'</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  country: '}</span><span className="tok-str">'US'</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'});'}</span>{'\n\n'}
        <span className="tok-dim">{'// 3. Start a subscription'}</span>{'\n'}
        <span className="tok-flag">await</span><span className="tok-cmd"> recurso.subscriptions.</span><span className="tok-key">create</span><span className="tok-cmd">{'({'}</span>{'\n'}
        <span className="tok-cmd">{'  customer_id: customer.id,'}</span>{'\n'}
        <span className="tok-cmd">{'  plan_id: plan.id,'}</span>{'\n'}
        <span className="tok-cmd">{'});'}</span>
    </>
)

const goCode = (
    <>
        <span className="tok-flag">import</span><span className="tok-cmd"> recurso </span><span className="tok-str">"github.com/swapnull-in/recurso-go"</span>{'\n\n'}
        <span className="tok-cmd">client := recurso.</span><span className="tok-key">NewClient</span><span className="tok-cmd">(</span><span className="tok-str">"sk_live_your_api_key"</span><span className="tok-cmd">)</span>{'\n\n'}
        <span className="tok-dim">{'// 1. Create a plan'}</span>{'\n'}
        <span className="tok-cmd">plan, _ := client.Plans.</span><span className="tok-key">Create</span><span className="tok-cmd">{'(ctx, &recurso.PlanCreateParams{'}</span>{'\n'}
        <span className="tok-cmd">{'  Name:         '}</span><span className="tok-str">"Pro Plan"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  Amount:       '}</span><span className="tok-brand">2900</span><span className="tok-cmd">,        </span><span className="tok-dim">{'// minor units'}</span>{'\n'}
        <span className="tok-cmd">{'  Currency:     '}</span><span className="tok-str">"USD"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  IntervalUnit: '}</span><span className="tok-str">"month"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'})'}</span>{'\n\n'}
        <span className="tok-dim">{'// 2. Add a customer'}</span>{'\n'}
        <span className="tok-cmd">customer, _ := client.Customers.</span><span className="tok-key">Create</span><span className="tok-cmd">{'(ctx, &recurso.CustomerCreateParams{'}</span>{'\n'}
        <span className="tok-cmd">{'  Name:  '}</span><span className="tok-str">"Jane User"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  Email: '}</span><span className="tok-str">"jane@example.com"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'})'}</span>{'\n\n'}
        <span className="tok-dim">{'// 3. Start a subscription'}</span>{'\n'}
        <span className="tok-cmd">client.Subscriptions.</span><span className="tok-key">Create</span><span className="tok-cmd">{'(ctx, &recurso.SubscriptionCreateParams{'}</span>{'\n'}
        <span className="tok-cmd">{'  CustomerID: customer.ID,'}</span>{'\n'}
        <span className="tok-cmd">{'  PlanID:     plan.ID,'}</span>{'\n'}
        <span className="tok-cmd">{'})'}</span>
    </>
)

const pythonCode = (
    <>
        <span className="tok-flag">from</span><span className="tok-cmd"> recurso </span><span className="tok-flag">import</span><span className="tok-cmd"> AuthenticatedClient</span>{'\n'}
        <span className="tok-flag">from</span><span className="tok-cmd"> recurso.api.plans </span><span className="tok-flag">import</span><span className="tok-cmd"> create_plan</span>{'\n'}
        <span className="tok-flag">from</span><span className="tok-cmd"> recurso.models </span><span className="tok-flag">import</span><span className="tok-cmd"> CreatePlanRequest</span>{'\n\n'}
        <span className="tok-cmd">client = </span><span className="tok-key">AuthenticatedClient</span><span className="tok-cmd">(</span>{'\n'}
        <span className="tok-cmd">{'  base_url='}</span><span className="tok-str">"https://billing.example.com"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  token='}</span><span className="tok-str">"sk_live_your_api_key"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">)</span>{'\n\n'}
        <span className="tok-dim">{'# 1. Create a plan'}</span>{'\n'}
        <span className="tok-cmd">plan = create_plan.</span><span className="tok-key">sync</span><span className="tok-cmd">(client=client, body=CreatePlanRequest(</span>{'\n'}
        <span className="tok-cmd">{'  name='}</span><span className="tok-str">"Pro Plan"</span><span className="tok-cmd">, code=</span><span className="tok-str">"PRO-USD"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">{'  amount='}</span><span className="tok-brand">2900</span><span className="tok-cmd">, currency=</span><span className="tok-str">"USD"</span><span className="tok-cmd">,</span>{'\n'}
        <span className="tok-cmd">))</span>{'\n\n'}
        <span className="tok-dim">{'# 3. Subscribe (after create_customer)'}</span>{'\n'}
        <span className="tok-cmd">create_subscription.</span><span className="tok-key">sync</span><span className="tok-cmd">(client=client, body=CreateSubscriptionRequest(</span>{'\n'}
        <span className="tok-cmd">{'  customer_id=customer.id, plan_id=plan.id,'}</span>{'\n'}
        <span className="tok-cmd">))</span>
    </>
)

const tabs = [
    { id: 'curl', label: 'curl', icon: Terminal, code: curlCode, file: 'quickstart.sh' },
    { id: 'node', label: 'Node', icon: Braces, code: nodeCode, file: 'billing.ts' },
    { id: 'go', label: 'Go', icon: Braces, code: goCode, file: 'main.go' },
    { id: 'python', label: 'Python', icon: Braces, code: pythonCode, file: 'billing.py' },
]

const CodeSection = () => {
    const [active, setActive] = useState('curl')
    const tab = tabs.find((t) => t.id === active)

    return (
        <section className="border-t border-line py-24 sm:py-28">
            <div className="mx-auto max-w-site px-4 sm:px-6 lg:px-8">
                <div className="grid items-start gap-12 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)]">
                    {/* Left copy */}
                    <div className="lg:sticky lg:top-28">
                        <p className="section-label">Developer experience</p>
                        <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                            Plan → customer → subscription in three calls
                        </h2>
                        <p className="mt-4 text-base leading-relaxed text-fg-muted">
                            A predictable REST API with an OpenAPI 3.1 spec served straight from your
                            instance, plus official typed SDKs for Go, Node, and Python. Amounts are in
                            minor units, every list is paginated, and webhooks are HMAC-signed.
                        </p>
                        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
                            <a
                                href="https://swapnull.mintlify.site/api-reference/introduction"
                                target="_blank"
                                rel="noreferrer"
                                className="btn-secondary"
                            >
                                API reference <ArrowUpRight className="h-4 w-4" />
                            </a>
                            <a
                                href="https://github.com/swapnull-in/recur-so/tree/main/sdk"
                                target="_blank"
                                rel="noreferrer"
                                className="inline-flex items-center gap-1.5 px-1 py-2.5 text-sm font-medium text-brand hover:text-brand-light"
                            >
                                SDKs on GitHub <ArrowUpRight className="h-3.5 w-3.5" />
                            </a>
                        </div>
                    </div>

                    {/* Code panel */}
                    <div className="terminal">
                        <div className="flex items-center justify-between border-b border-line bg-surface-100 pr-4">
                            <div className="flex">
                                {tabs.map((t) => (
                                    <button
                                        key={t.id}
                                        onClick={() => setActive(t.id)}
                                        className={`flex items-center gap-2 border-b-2 px-5 py-3 font-mono text-xs transition-colors ${
                                            active === t.id
                                                ? 'border-brand text-fg'
                                                : 'border-transparent text-fg-subtle hover:text-fg-muted'
                                        }`}
                                    >
                                        <t.icon className="h-3.5 w-3.5" />
                                        {t.label}
                                    </button>
                                ))}
                            </div>
                            <span className="hidden font-mono text-[11px] text-fg-subtle sm:block">{tab.file}</span>
                        </div>
                        <pre className="overflow-x-auto p-5 font-mono text-[12.5px] leading-[1.7]">
                            <code>{tab.code}</code>
                        </pre>
                    </div>
                </div>
            </div>
        </section>
    )
}

export default CodeSection
