import { ArrowRight, Github, BookOpen, MessageCircle, FileCode } from 'lucide-react'
import { useScrollAnimation } from '../hooks/useScrollAnimation'

const CTA = () => {
    const sectionRef = useScrollAnimation()

    const links = [
        { icon: BookOpen, label: 'Documentation', href: '#', color: 'bg-blue-500/10 text-blue-400', hoverBg: 'hover:bg-blue-500/15' },
        { icon: Github, label: 'GitHub', href: 'https://github.com/recur-so/recurso', color: 'bg-gray-500/10 text-gray-400', hoverBg: 'hover:bg-gray-500/15' },
        { icon: MessageCircle, label: 'Discord', href: '#', color: 'bg-violet-500/10 text-violet-400', hoverBg: 'hover:bg-violet-500/15' },
        { icon: FileCode, label: 'API Reference', href: '#', color: 'bg-emerald-500/10 text-emerald-400', hoverBg: 'hover:bg-emerald-500/15' },
    ]

    return (
        <section className="relative py-32 overflow-hidden aurora">
            {/* Background effects */}
            <div className="absolute inset-0 bg-grid opacity-20" />
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[500px] bg-emerald-500/[0.06] rounded-full blur-[150px]" />
            <div className="absolute bottom-0 right-1/4 w-[300px] h-[300px] bg-cyan-500/[0.03] rounded-full blur-[80px]" />

            <div ref={sectionRef} className="relative z-10 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
                {/* Trust counter */}
                <div className="flex items-center justify-center gap-8 mb-12">
                    {[
                        { value: '500+', label: 'Companies' },
                        { value: '50K+', label: 'Subscriptions' },
                        { value: '$2M+', label: 'Revenue tracked' },
                    ].map((stat) => (
                        <div key={stat.label} className="text-center">
                            <div className="text-2xl font-bold text-white">{stat.value}</div>
                            <div className="text-[10px] text-gray-500 uppercase tracking-wider mt-1">{stat.label}</div>
                        </div>
                    ))}
                </div>

                {/* Main CTA */}
                <h2 className="text-5xl sm:text-6xl lg:text-7xl font-extrabold text-white mb-6 leading-[0.95] tracking-tight">
                    Start billing
                    <br />
                    <span className="gradient-text">in minutes</span>
                </h2>

                <p className="text-xl text-gray-400 mb-12 max-w-xl mx-auto leading-relaxed">
                    Open source. Self-hosted or cloud.
                    Complete control over your billing infrastructure.
                </p>

                {/* CTA Buttons */}
                <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-20">
                    <a
                        href="#"
                        className="group flex items-center gap-2 px-10 py-4 bg-emerald-500 text-black font-bold rounded-xl hover:bg-emerald-400 transition-all duration-300 glow-green-intense glow-ring text-base"
                    >
                        Get Started Free
                        <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
                    </a>

                    <a
                        href="https://github.com/recur-so/recurso"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-2 px-8 py-4 glass-strong text-white font-semibold rounded-xl hover:bg-white/[0.06] transition-all duration-300"
                    >
                        <Github className="w-5 h-5" />
                        View on GitHub
                    </a>
                </div>

                {/* Quick links */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    {links.map((link) => {
                        const Icon = link.icon
                        return (
                            <a
                                key={link.label}
                                href={link.href}
                                className={`group glass rounded-xl p-5 ${link.hoverBg} transition-all duration-300 hover:-translate-y-0.5`}
                            >
                                <div className={`w-10 h-10 rounded-lg ${link.color} flex items-center justify-center mx-auto mb-3`}>
                                    <Icon className="w-5 h-5" />
                                </div>
                                <span className="text-sm font-medium text-gray-400 group-hover:text-white transition-colors">
                                    {link.label}
                                </span>
                            </a>
                        )
                    })}
                </div>
            </div>
        </section>
    )
}

export default CTA
