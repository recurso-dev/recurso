import { Github } from 'lucide-react'

const columns = [
    {
        title: 'Product',
        links: [
            { label: 'Product modules', href: '#products' },
            { label: 'Pricing', href: '#pricing' },
            { label: 'Compare', href: '#compare' },
            { label: 'Recurso Cloud waitlist', href: 'mailto:cloud@recurso.dev' },
        ],
    },
    {
        title: 'Developers',
        links: [
            { label: 'Documentation', href: 'https://docs.recurso.dev', external: true },
            { label: 'Quickstart', href: 'https://docs.recurso.dev/quickstart', external: true },
            { label: 'API reference', href: 'https://docs.recurso.dev/api-reference/introduction', external: true },
            { label: 'Core concepts', href: 'https://docs.recurso.dev/concepts', external: true },
            { label: 'Going to production', href: 'https://docs.recurso.dev/going-to-production', external: true },
            { label: 'Node SDK', href: 'https://github.com/swapnull-in/recur-so/tree/main/sdk/node', external: true },
            { label: 'Roadmap', href: 'https://github.com/swapnull-in/recur-so/blob/main/ROADMAP.md', external: true },
            { label: 'Changelog', href: 'https://github.com/swapnull-in/recur-so/blob/main/CHANGELOG.md', external: true },
        ],
    },
    {
        title: 'Company',
        links: [
            { label: 'GitHub', href: 'https://github.com/swapnull-in/recur-so', external: true },
            { label: 'Community discussions', href: 'https://github.com/swapnull-in/recur-so/discussions', external: true },
            { label: 'Security', href: 'https://github.com/swapnull-in/recur-so/blob/main/docs/security.md', external: true },
            { label: 'License (MIT)', href: 'https://github.com/swapnull-in/recur-so/blob/main/LICENSE', external: true },
            { label: 'Contact', href: 'mailto:sales@recurso.dev' },
        ],
    },
]

const Footer = () => (
    <footer className="border-t border-line bg-surface-75">
        <div className="mx-auto max-w-site px-4 py-16 sm:px-6 lg:px-8">
            <div className="grid gap-12 lg:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
                {/* Brand */}
                <div>
                    <a href="#" className="flex items-center gap-2.5">
                        <img src="/logo.svg" alt="Recurso" className="h-7 w-7" />
                        <span className="text-[17px] font-semibold tracking-tight text-fg">recurso</span>
                    </a>
                    <p className="mt-4 max-w-xs text-sm leading-relaxed text-fg-muted">
                        The open-source billing engine for SaaS. Built with Go, PostgreSQL, and
                        TigerBeetle. MIT licensed.
                    </p>
                    <a
                        href="https://github.com/swapnull-in/recur-so"
                        target="_blank"
                        rel="noreferrer"
                        aria-label="Recurso on GitHub"
                        className="mt-6 inline-flex h-9 w-9 items-center justify-center rounded-md border border-line text-fg-muted transition-colors hover:border-line-strong hover:text-fg"
                    >
                        <Github className="h-4 w-4" />
                    </a>
                </div>

                {/* Link columns */}
                <div className="grid grid-cols-2 gap-8 sm:grid-cols-3">
                    {columns.map((col) => (
                        <div key={col.title}>
                            <h4 className="text-sm font-semibold text-fg">{col.title}</h4>
                            <ul className="mt-4 space-y-3">
                                {col.links.map((l) => (
                                    <li key={l.label}>
                                        <a
                                            href={l.href}
                                            {...(l.external ? { target: '_blank', rel: 'noreferrer' } : {})}
                                            className="text-sm text-fg-muted transition-colors hover:text-fg"
                                        >
                                            {l.label}
                                        </a>
                                    </li>
                                ))}
                            </ul>
                        </div>
                    ))}
                </div>
            </div>

            <div className="mt-14 flex flex-col items-start justify-between gap-4 border-t border-line pt-8 sm:flex-row sm:items-center">
                <p className="text-xs text-fg-subtle">
                    © {new Date().getFullYear()} Recurso. Open source under the MIT license.
                </p>
                <p className="font-mono text-xs text-fg-subtle">Go · PostgreSQL · TigerBeetle</p>
            </div>
        </div>
    </footer>
)

export default Footer
