import { ArrowRight } from 'lucide-react'

const AnnouncementBar = () => (
    <a
        href="https://github.com/swapnull-in/recur-so/releases"
        target="_blank"
        rel="noreferrer"
        className="group relative z-50 flex items-center justify-center gap-2 border-b border-line bg-surface-100 px-4 py-2 text-center text-xs text-fg-muted transition-colors hover:text-fg"
    >
        <span className="inline-block h-1.5 w-1.5 rounded-full bg-brand" />
        <span>
            <span className="font-medium text-fg">Recurso v0.1.1 is out</span>
            <span className="hidden sm:inline"> — subscriber migration importer and one-command demo data</span>
        </span>
        <ArrowRight className="h-3 w-3 text-brand transition-transform group-hover:translate-x-0.5" />
    </a>
)

export default AnnouncementBar
