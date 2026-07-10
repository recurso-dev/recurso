import { useState } from 'react'
import { Github, Mail, Check } from 'lucide-react'

// The waitlist posts to the deployed Recurso API (ENG-12). Override the origin
// at build time with VITE_API_URL for staging.
const API_URL = import.meta.env.VITE_API_URL || 'https://api.recurso.dev'

const WaitlistForm = () => {
    const [email, setEmail] = useState('')
    const [state, setState] = useState('idle') // idle | busy | done | error

    const submit = async (e) => {
        e.preventDefault()
        if (!email || state === 'busy') return
        setState('busy')
        try {
            const res = await fetch(`${API_URL}/waitlist`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, source: 'website-cta' }),
            })
            if (!res.ok) throw new Error('bad status')
            setState('done')
        } catch {
            setState('error')
        }
    }

    if (state === 'done') {
        return (
            <p className="inline-flex items-center gap-2 rounded-lg border border-line px-4 py-3 text-sm text-fg">
                <Check className="h-4 w-4 text-brand" /> You&apos;re on the list — we&apos;ll be in touch.
            </p>
        )
    }

    return (
        <form onSubmit={submit} className="flex w-full max-w-md flex-col gap-3 sm:flex-row" id="waitlist-form">
            {/* honeypot: hidden from humans, bots fill it and get silently dropped */}
            <input type="text" name="website" tabIndex="-1" autoComplete="off" className="hidden" aria-hidden="true" />
            <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@company.com"
                aria-label="Work email"
                className="w-full flex-1 rounded-lg border border-line bg-transparent px-4 py-3 text-sm text-fg placeholder:text-fg-subtle focus:border-brand focus:outline-none"
            />
            <button type="submit" disabled={state === 'busy'} className="btn-secondary w-full sm:w-auto">
                <Mail className="h-4 w-4" /> {state === 'busy' ? 'Joining…' : 'Join the Cloud waitlist'}
            </button>
            {state === 'error' && (
                <p className="text-xs text-red-400 sm:self-center">
                    Something went wrong — email cloud@recurso.dev instead.
                </p>
            )}
        </form>
    )
}

const CTA = () => (
    <section id="waitlist" className="relative overflow-hidden border-t border-line">
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
            <div className="mt-9 flex flex-col items-center justify-center gap-3">
                <a
                    href="https://github.com/swapnull-in/recur-so"
                    target="_blank"
                    rel="noreferrer"
                    className="btn-primary w-full sm:w-auto"
                >
                    <Github className="h-4 w-4" /> Start self-hosting
                </a>
                <WaitlistForm />
            </div>
            <p className="mt-6 font-mono text-xs text-fg-subtle">
                git clone https://github.com/swapnull-in/recur-so.git && make demo
            </p>
        </div>
    </section>
)

export default CTA
