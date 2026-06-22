import { useState, useEffect } from 'react'
import { Menu, X, Github, ExternalLink } from 'lucide-react'

const DOCS_URL = 'https://docs.recurso.dev'

const Navbar = () => {
    const [isOpen, setIsOpen] = useState(false)
    const [scrolled, setScrolled] = useState(false)

    useEffect(() => {
        const onScroll = () => setScrolled(window.scrollY > 20)
        window.addEventListener('scroll', onScroll, { passive: true })
        return () => window.removeEventListener('scroll', onScroll)
    }, [])

    const navLinks = [
        { name: 'Features', href: '#features', external: false },
        { name: 'Playground', href: '#playground', external: false },
        { name: 'Pricing', href: '#pricing', external: false },
        { name: 'Docs', href: DOCS_URL, external: true },
    ]

    return (
        <header className="fixed top-0 left-0 right-0 z-50">
            <div className={`transition-all duration-500 ${scrolled
                    ? 'bg-[#050505]/80 backdrop-blur-xl border-b border-white/[0.06] shadow-lg shadow-black/20'
                    : 'bg-transparent border-b border-transparent'
                }`}>
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex items-center justify-between h-16">
                        {/* Logo */}
                        <a href="#top" className="flex items-center gap-2.5 group">
                            <div className="w-8 h-8 rounded-lg bg-emerald-500 flex items-center justify-center group-hover:shadow-lg group-hover:shadow-emerald-500/25 transition-shadow">
                                <span className="text-black font-bold text-lg">R</span>
                            </div>
                            <span className="text-xl font-bold text-white">Recurso</span>
                        </a>

                        {/* Desktop Nav */}
                        <nav className="hidden md:flex items-center gap-1">
                            {navLinks.map((link) => (
                                <a
                                    key={link.name}
                                    href={link.href}
                                    {...(link.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
                                    className="flex items-center gap-1 px-4 py-2 text-gray-400 hover:text-white transition-colors text-sm font-medium rounded-lg hover:bg-white/[0.04]"
                                >
                                    {link.name}
                                    {link.external && <ExternalLink className="w-3 h-3 opacity-50" />}
                                </a>
                            ))}
                        </nav>

                        {/* Right side */}
                        <div className="hidden md:flex items-center gap-3">
                            <a
                                href="https://github.com/recur-so/recurso"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="flex items-center gap-2 px-3 py-2 text-gray-400 hover:text-white transition-colors text-sm rounded-lg hover:bg-white/[0.04]"
                            >
                                <Github className="w-4 h-4" />
                            </a>
                            <a
                                href="https://github.com/recur-so/recurso"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="px-5 py-2 text-sm font-semibold text-black bg-emerald-500 rounded-lg hover:bg-emerald-400 transition-all duration-200 glow-ring"
                            >
                                Get Started
                            </a>
                        </div>

                        {/* Mobile menu button */}
                        <button
                            onClick={() => setIsOpen(!isOpen)}
                            className="md:hidden p-2 text-gray-400 hover:text-white rounded-lg hover:bg-white/[0.04]"
                        >
                            {isOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
                        </button>
                    </div>
                </div>

                {/* Mobile menu */}
                <div className={`md:hidden overflow-hidden transition-all duration-300 ease-in-out ${isOpen ? 'max-h-72 opacity-100' : 'max-h-0 opacity-0'
                    }`}>
                    <div className="px-4 py-4 space-y-1 border-t border-white/[0.06] bg-[#050505]/95 backdrop-blur-xl">
                        {navLinks.map((link) => (
                            <a
                                key={link.name}
                                href={link.href}
                                {...(link.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
                                className="flex items-center gap-1 text-gray-400 hover:text-white transition-colors text-sm font-medium py-2.5 px-3 rounded-lg hover:bg-white/[0.04]"
                                onClick={() => !link.external && setIsOpen(false)}
                            >
                                {link.name}
                                {link.external && <ExternalLink className="w-3 h-3 opacity-50" />}
                            </a>
                        ))}
                        <a
                            href="https://github.com/recur-so/recurso"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="block w-full px-4 py-2.5 text-sm font-semibold text-center text-black bg-emerald-500 rounded-lg mt-2"
                        >
                            Get Started
                        </a>
                    </div>
                </div>
            </div>
        </header>
    )
}

export default Navbar
