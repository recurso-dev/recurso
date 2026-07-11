import { Github, Map, GitPullRequest, ArrowUpRight, ScrollText } from 'lucide-react'

const cards = [
    {
        icon: ScrollText,
        title: 'MIT licensed',
        body: 'Fork it, extend it, ship it. No open-core bait — the whole billing engine is in the repo.',
        link: { label: 'Read the license', href: 'https://github.com/recurso-dev/recurso/blob/main/LICENSE' },
    },
    {
        icon: Map,
        title: 'Public roadmap',
        body: 'ROADMAP.md is a living document — what shipped, what is next, and in what order.',
        link: { label: 'View the roadmap', href: 'https://github.com/recurso-dev/recurso/blob/main/ROADMAP.md' },
    },
    {
        icon: GitPullRequest,
        title: 'Contributions welcome',
        body: 'Good first issues, contributor docs, and an e2e suite so you can change billing code with confidence.',
        link: { label: 'CONTRIBUTING.md', href: 'https://github.com/recurso-dev/recurso/blob/main/CONTRIBUTING.md' },
    },
]

const OpenSource = () => (
    <section className="border-t border-line bg-surface-75 py-24 sm:py-28">
        <div className="mx-auto max-w-site px-4 sm:px-6 lg:px-8">
            <div className="mx-auto max-w-2xl text-center">
                <p className="section-label">Open source</p>
                <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                    Your billing system should not be a black box
                </h2>
                <p className="mt-4 text-base leading-relaxed text-fg-muted">
                    Billing touches every rupee and dollar you earn. Recurso keeps the code, the
                    data, and the roadmap in the open — on your infrastructure, under your control.
                </p>
                <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
                    <a
                        href="https://github.com/recurso-dev/recurso"
                        target="_blank"
                        rel="noreferrer"
                        className="btn-primary"
                    >
                        <Github className="h-4 w-4" /> Explore the repository
                    </a>
                    <a
                        href="https://github.com/recurso-dev/recurso/discussions"
                        target="_blank"
                        rel="noreferrer"
                        className="btn-secondary"
                    >
                        Join the discussion
                    </a>
                </div>
            </div>

            <div className="mt-14 grid gap-4 md:grid-cols-3">
                {cards.map((c) => (
                    <div key={c.title} className="card flex flex-col p-6">
                        <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-line bg-surface-200">
                            <c.icon className="h-[18px] w-[18px] text-brand" />
                        </div>
                        <h3 className="mt-4 text-[15px] font-semibold text-fg">{c.title}</h3>
                        <p className="mt-2 flex-1 text-sm leading-relaxed text-fg-muted">{c.body}</p>
                        <a
                            href={c.link.href}
                            target="_blank"
                            rel="noreferrer"
                            className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-brand hover:text-brand-light"
                        >
                            {c.link.label} <ArrowUpRight className="h-3.5 w-3.5" />
                        </a>
                    </div>
                ))}
            </div>
        </div>
    </section>
)

export default OpenSource
