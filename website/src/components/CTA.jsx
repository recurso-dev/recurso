import { Github, Mail } from 'lucide-react'

const CTA = () => (
    <section className="relative overflow-hidden border-t border-line">
        <div className="absolute inset-0 bg-dots" />
        <div className="hero-glow absolute inset-x-0 bottom-0 h-full rotate-180" />

        <div className="relative mx-auto max-w-site px-4 py-24 text-center sm:px-6 sm:py-28 lg:px-8">
            <h2 className="mx-auto max-w-2xl text-3xl font-bold tracking-tight text-fg sm:text-5xl">
                Start owning your billing <span className="gradient-brand">tonight</span>
            </h2>
            <p className="mx-auto mt-5 max-w-xl text-base leading-relaxed text-fg-muted">
                One command from clone to a seeded dashboard. Free forever on your own
                infrastructure — or get in line for Recurso Cloud.
            </p>
            <div className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
                <a
                    href="https://github.com/swapnull-in/recur-so"
                    target="_blank"
                    rel="noreferrer"
                    className="btn-primary w-full sm:w-auto"
                >
                    <Github className="h-4 w-4" /> Start self-hosting
                </a>
                <a href="mailto:cloud@recurso.dev" className="btn-secondary w-full sm:w-auto">
                    <Mail className="h-4 w-4" /> Join the Cloud waitlist
                </a>
            </div>
            <p className="mt-6 font-mono text-xs text-fg-subtle">
                git clone https://github.com/swapnull-in/recur-so.git && make demo
            </p>
        </div>
    </section>
)

export default CTA
