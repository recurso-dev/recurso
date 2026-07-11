import { Github, Mail } from 'lucide-react'

const terminalLines = [
    { prompt: true, text: 'git clone https://github.com/recurso-dev/recurso.git && cd recur-so' },
    { prompt: true, text: 'make demo' },
    { cls: 'tok-dim', text: 'Starting postgres, tigerbeetle, mailhog… done' },
    { cls: 'tok-dim', text: 'Running migrations… 42 applied' },
    { cls: 'tok-brand', text: '🌱 Seeding demo data…' },
    { cls: 'tok-dim', text: '   Tenant: Demo Tenant  ·  Plans: Starter, Pro, Scale' },
    { cls: 'tok-dim', text: '   12 customers · 9 subscriptions · 14 invoices · 2 coupons' },
    { cls: 'tok-cmd', text: 'API ready      → http://localhost:8080' },
    { cls: 'tok-cmd', text: 'Dashboard      → http://localhost:5173' },
    {
        cls: 'tok-cmd',
        text: 'Demo API key   → ',
        suffix: { cls: 'tok-str', text: 'sk_test_12345' },
    },
]

const Hero = () => (
    <section className="relative overflow-hidden">
        <div className="absolute inset-0 bg-grid" />
        <div className="hero-glow absolute inset-x-0 top-0 h-[560px]" />

        <div className="relative mx-auto max-w-site px-4 pb-24 pt-20 sm:px-6 sm:pt-28 lg:px-8">
            <div className="mx-auto max-w-3xl text-center">
                <h1 className="rise-in text-4xl font-bold tracking-tight text-fg sm:text-6xl">
                    Bill your customers.
                    <br />
                    <span className="gradient-brand">Own your billing.</span>
                </h1>

                <p className="rise-in rise-in-1 mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-fg-muted">
                    Recurso is the open-source billing engine for SaaS — subscriptions, invoicing,
                    GST-native India compliance, and an immutable financial ledger. MIT licensed,
                    self-hosted, and no percentage-of-revenue tax as you grow.
                </p>

                <div className="rise-in rise-in-2 mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
                    <a
                        href="https://github.com/recurso-dev/recurso"
                        target="_blank"
                        rel="noreferrer"
                        className="btn-primary w-full sm:w-auto"
                    >
                        <Github className="h-4 w-4" />
                        Start self-hosting
                    </a>
                    <a href="mailto:swapnil.go20@gmail.com" className="btn-secondary w-full sm:w-auto">
                        <Mail className="h-4 w-4" />
                        Join the Cloud waitlist
                    </a>
                </div>

                <p className="rise-in rise-in-3 mt-5 text-xs text-fg-subtle">
                    MIT licensed · Go + PostgreSQL + TigerBeetle · Stripe & Razorpay
                </p>
            </div>

            {/* Terminal */}
            <div className="rise-in rise-in-4 mx-auto mt-16 max-w-3xl">
                <div className="terminal">
                    <div className="terminal-bar">
                        <span className="terminal-dot bg-[#ff5f57]" />
                        <span className="terminal-dot bg-[#febc2e]" />
                        <span className="terminal-dot bg-[#28c840]" />
                        <span className="ml-3 font-mono text-xs text-fg-subtle">
                            zero to seeded dashboard — one command
                        </span>
                    </div>
                    <div className="overflow-x-auto p-5 font-mono text-[13px] leading-7">
                        {terminalLines.map((line, i) => (
                            <div key={i} className="whitespace-pre">
                                {line.prompt ? (
                                    <>
                                        <span className="tok-brand">$ </span>
                                        <span className="tok-cmd">{line.text}</span>
                                    </>
                                ) : (
                                    <>
                                        <span className={line.cls}>{line.text}</span>
                                        {line.suffix && <span className={line.suffix.cls}>{line.suffix.text}</span>}
                                    </>
                                )}
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        </div>
    </section>
)

export default Hero
