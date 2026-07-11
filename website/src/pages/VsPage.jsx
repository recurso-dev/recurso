import { Head } from 'vite-react-ssg'
import { Check, X, Minus, Github } from 'lucide-react'
import Navbar from '../components/Navbar'
import CTA from '../components/CTA'
import Footer from '../components/Footer'
import { competitorList } from '../data/competitors'

const Cell = ({ value }) => {
    if (value === true) {
        return (
            <span className="mx-auto flex h-5 w-5 items-center justify-center rounded-full bg-brand/10">
                <Check className="h-3 w-3 text-brand" />
            </span>
        )
    }
    if (value === false) {
        return <X className="mx-auto h-3.5 w-3.5 text-fg-subtle/60" />
    }
    return (
        <span className="mx-auto flex h-5 w-5 items-center justify-center rounded-full bg-amber-500/10">
            <Minus className="h-3 w-3 text-amber-400" />
        </span>
    )
}

const VsPage = ({ data }) => (
    <div className="min-h-screen bg-surface">
        <Head>
            <title>{data.title}</title>
            <meta name="description" content={data.description} />
            <meta property="og:title" content={data.title} />
            <meta property="og:description" content={data.description} />
            <meta property="og:type" content="website" />
        </Head>

        <Navbar />

        <main>
            {/* Hero */}
            <section className="relative overflow-hidden border-b border-line">
                <div className="absolute inset-0 bg-dots" />
                <div className="hero-glow absolute inset-x-0 top-0 h-full" />
                <div className="relative mx-auto max-w-3xl px-4 py-24 text-center sm:px-6 sm:py-28 lg:px-8">
                    <p className="section-label">Compare</p>
                    <h1 className="mx-auto mt-3 max-w-2xl text-4xl font-bold tracking-tight text-fg sm:text-5xl">
                        {data.headline}
                    </h1>
                    <p className="mx-auto mt-5 max-w-2xl text-base leading-relaxed text-fg-muted">
                        {data.lede}
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
                        <a
                            href="https://docs.recurso.dev/docs/quickstart"
                            target="_blank"
                            rel="noreferrer"
                            className="btn-secondary w-full sm:w-auto"
                        >
                            Read the quickstart
                        </a>
                    </div>
                    <p className="mt-6 font-mono text-xs text-fg-subtle">{data.note}</p>
                </div>
            </section>

            {/* Differentiators */}
            <section className="border-t border-line py-20 sm:py-24">
                <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
                    <div className="mx-auto max-w-2xl text-center">
                        <p className="section-label">Why Recurso</p>
                        <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                            What Recurso brings
                        </h2>
                    </div>
                    <div className="mt-12 grid gap-5 sm:grid-cols-2">
                        {data.bullets.map((b) => (
                            <div
                                key={b.title}
                                className="rounded-xl border border-line bg-surface-100 p-6"
                            >
                                <div className="flex items-start gap-3">
                                    <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-brand/10">
                                        <Check className="h-3 w-3 text-brand" />
                                    </span>
                                    <div>
                                        <h3 className="text-sm font-semibold text-fg">{b.title}</h3>
                                        <p className="mt-1.5 text-[13px] leading-relaxed text-fg-muted sm:text-sm">
                                            {b.body}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </section>

            {/* Capability table */}
            <section className="border-t border-line py-20 sm:py-24">
                <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
                    <div className="mx-auto max-w-2xl text-center">
                        <p className="section-label">Capabilities</p>
                        <h2 className="mt-3 text-3xl font-bold tracking-tight text-fg sm:text-4xl">
                            Recurso vs {data.name}, feature by feature
                        </h2>
                    </div>

                    <div className="mt-12 overflow-hidden rounded-xl border border-line bg-surface-100">
                        <div className="grid grid-cols-[minmax(0,2fr)_repeat(2,minmax(0,1fr))] gap-3 border-b border-line bg-surface-200 px-4 py-4 sm:px-6">
                            <div className="text-xs font-medium uppercase tracking-wider text-fg-subtle">
                                Capability
                            </div>
                            <div className="text-center text-sm font-semibold text-brand">Recurso</div>
                            <div className="text-center text-xs font-medium text-fg-muted sm:text-sm">
                                {data.name}
                            </div>
                        </div>
                        {data.rows.map((f, idx) => (
                            <div
                                key={f.name}
                                className={`grid grid-cols-[minmax(0,2fr)_repeat(2,minmax(0,1fr))] items-center gap-3 px-4 py-3 transition-colors hover:bg-surface-200/50 sm:px-6 ${
                                    idx !== data.rows.length - 1
                                        ? 'border-b border-line/60'
                                        : ''
                                }`}
                            >
                                <div className="text-[13px] text-fg-muted sm:text-sm">{f.name}</div>
                                <Cell value={f.recurso} />
                                <Cell value={f.competitor} />
                            </div>
                        ))}
                    </div>

                    {/* Legend */}
                    <div className="mt-6 flex justify-center gap-6 text-xs text-fg-subtle">
                        <span className="flex items-center gap-1.5">
                            <Check className="h-3 w-3 text-brand" /> Full support
                        </span>
                        <span className="flex items-center gap-1.5">
                            <Minus className="h-3 w-3 text-amber-400" /> Partial
                        </span>
                        <span className="flex items-center gap-1.5">
                            <X className="h-3 w-3 text-fg-subtle" /> Not available
                        </span>
                    </div>

                    {/* Other comparisons */}
                    <div className="mt-10 flex flex-wrap justify-center gap-x-5 gap-y-2 text-sm">
                        {competitorList
                            .filter((c) => c.slug !== data.slug)
                            .map((c) => (
                                <a
                                    key={c.slug}
                                    href={`/vs/${c.slug}`}
                                    className="text-fg-muted transition-colors hover:text-fg"
                                >
                                    vs {c.name}
                                </a>
                            ))}
                    </div>
                </div>
            </section>

            <CTA />
        </main>

        <Footer />
    </div>
)

export default VsPage
