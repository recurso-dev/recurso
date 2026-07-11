import { useState } from 'react'
import { Github, Menu, Star, X } from 'lucide-react'

const links = [
    { label: 'Product', href: '#products' },
    { label: 'Docs', href: 'https://docs.recurso.dev/docs', external: true },
    { label: 'Pricing', href: '#pricing' },
    { label: 'Compare', href: '#compare' },
]

const Navbar = () => {
    const [open, setOpen] = useState(false)

    return (
        <header className="nav-blur sticky top-0 z-40 border-b border-line">
            <nav className="mx-auto flex h-16 max-w-site items-center justify-between px-4 sm:px-6 lg:px-8">
                {/* Logo */}
                <a href="#" className="flex items-center gap-2.5">
                    <img src="/logo.svg" alt="Recurso" className="h-7 w-7" />
                    <span className="text-[17px] font-semibold tracking-tight text-fg">recurso</span>
                </a>

                {/* Desktop links */}
                <div className="hidden items-center gap-1 md:flex">
                    {links.map((l) => (
                        <a
                            key={l.label}
                            href={l.href}
                            {...(l.external ? { target: '_blank', rel: 'noreferrer' } : {})}
                            className="rounded-md px-3 py-2 text-sm text-fg-muted transition-colors hover:text-fg"
                        >
                            {l.label}
                        </a>
                    ))}
                </div>

                {/* Right side */}
                <div className="hidden items-center gap-3 md:flex">
                    <a
                        href="https://github.com/swapnull-in/recur-so"
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-2 rounded-md border border-line-strong bg-surface-200 px-3.5 py-2 text-sm font-medium text-fg transition-colors hover:border-[#3a4157] hover:bg-surface-300"
                    >
                        <Github className="h-4 w-4" />
                        <span>Star on GitHub</span>
                        <Star className="h-3.5 w-3.5 text-fg-subtle" />
                    </a>
                    <a href="https://docs.recurso.dev/docs/quickstart" target="_blank" rel="noreferrer" className="btn-primary !px-4 !py-2">
                        Get started
                    </a>
                </div>

                {/* Mobile toggle */}
                <button
                    className="p-2 text-fg-muted md:hidden"
                    onClick={() => setOpen(!open)}
                    aria-label="Toggle menu"
                >
                    {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
                </button>
            </nav>

            {/* Mobile menu */}
            {open && (
                <div className="border-t border-line bg-surface-100 px-4 py-4 md:hidden">
                    <div className="flex flex-col gap-1">
                        {links.map((l) => (
                            <a
                                key={l.label}
                                href={l.href}
                                {...(l.external ? { target: '_blank', rel: 'noreferrer' } : {})}
                                onClick={() => setOpen(false)}
                                className="rounded-md px-3 py-2.5 text-sm text-fg-muted hover:bg-surface-200 hover:text-fg"
                            >
                                {l.label}
                            </a>
                        ))}
                        <a
                            href="https://github.com/swapnull-in/recur-so"
                            target="_blank"
                            rel="noreferrer"
                            className="mt-2 inline-flex items-center gap-2 rounded-md border border-line-strong px-3 py-2.5 text-sm font-medium text-fg"
                        >
                            <Github className="h-4 w-4" /> Star on GitHub
                        </a>
                    </div>
                </div>
            )}
        </header>
    )
}

export default Navbar
