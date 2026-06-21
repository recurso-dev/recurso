import { Github, Twitter, MessageCircle, Youtube } from 'lucide-react'

const footerLinks = {
    Product: [
        { name: 'Features', href: '#features' },
        { name: 'Pricing', href: '#pricing' },
        { name: 'Changelog', href: '#' },
        { name: 'Roadmap', href: '#' },
    ],
    Developers: [
        { name: 'Documentation', href: '#' },
        { name: 'API Reference', href: '#' },
        { name: 'SDKs', href: '#' },
        { name: 'Examples', href: '#' },
    ],
    Company: [
        { name: 'About', href: '#' },
        { name: 'Blog', href: '#' },
        { name: 'Careers', href: '#' },
        { name: 'Open Source', href: 'https://github.com/recur-so/recurso' },
    ],
    Legal: [
        { name: 'Privacy', href: '#' },
        { name: 'Terms', href: '#' },
        { name: 'Security', href: '#' },
    ],
}

const socialLinks = [
    { icon: Github, href: 'https://github.com/recur-so/recurso', label: 'GitHub' },
    { icon: Twitter, href: '#', label: 'Twitter' },
    { icon: MessageCircle, href: '#', label: 'Discord' },
    { icon: Youtube, href: '#', label: 'YouTube' },
]

const Footer = () => {
    return (
        <footer className="border-t border-white/[0.04] bg-[#030303]">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
                <div className="grid grid-cols-2 md:grid-cols-6 gap-8 lg:gap-12">
                    {/* Brand column */}
                    <div className="col-span-2">
                        <a href="#" className="flex items-center gap-2.5 mb-5 group">
                            <div className="w-8 h-8 rounded-lg bg-emerald-500 flex items-center justify-center">
                                <span className="text-black font-bold text-lg">R</span>
                            </div>
                            <span className="text-xl font-bold text-white">Recurso</span>
                        </a>
                        <p className="text-gray-500 text-sm mb-6 max-w-xs leading-relaxed">
                            The open-source billing engine for SaaS. Built by developers, for developers.
                        </p>

                        {/* Social links */}
                        <div className="flex items-center gap-2">
                            {socialLinks.map((social) => {
                                const Icon = social.icon
                                return (
                                    <a
                                        key={social.label}
                                        href={social.href}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="w-9 h-9 rounded-lg bg-white/[0.04] flex items-center justify-center text-gray-500 hover:text-white hover:bg-white/[0.08] transition-all duration-200"
                                        aria-label={social.label}
                                    >
                                        <Icon className="w-4 h-4" />
                                    </a>
                                )
                            })}
                        </div>
                    </div>

                    {/* Link columns */}
                    {Object.entries(footerLinks).map(([category, links]) => (
                        <div key={category}>
                            <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-4">{category}</h3>
                            <ul className="space-y-3">
                                {links.map((link) => (
                                    <li key={link.name}>
                                        <a
                                            href={link.href}
                                            className="text-sm text-gray-500 hover:text-white transition-colors duration-200"
                                        >
                                            {link.name}
                                        </a>
                                    </li>
                                ))}
                            </ul>
                        </div>
                    ))}
                </div>

                {/* Bottom bar */}
                <div className="mt-16 pt-8 border-t border-white/[0.04] flex flex-col sm:flex-row items-center justify-between gap-4">
                    <p className="text-xs text-gray-600">
                        © {new Date().getFullYear()} Recurso. Open source under MIT License.
                    </p>
                    <div className="flex items-center gap-2 text-xs text-gray-600">
                        <span className="relative flex h-2 w-2">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-40" />
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
                        </span>
                        All systems operational
                    </div>
                </div>
            </div>
        </footer>
    )
}

export default Footer
