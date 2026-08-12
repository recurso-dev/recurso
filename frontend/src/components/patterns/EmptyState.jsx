import { Inbox, ArrowUpRight } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * EmptyState — shown when a list/section has no data.
 *
 * Props:
 *  - icon:           lucide icon component (defaults to Inbox)
 *  - title:          string
 *  - description:    string
 *  - action:         ReactNode (e.g. a <Button>)
 *  - learnMoreHref:  docs.recurso.dev URL — renders a "Read the guide" link so
 *                    users get help right where they hit an empty screen.
 *  - learnMoreLabel: label for that link (defaults to "Read the guide")
 */
export function EmptyState({
  icon: Icon = Inbox,
  title = "Nothing here yet",
  description,
  action,
  learnMoreHref,
  learnMoreLabel = "Read the guide",
  className,
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center px-6 py-16 text-center",
        className
      )}
    >
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-border bg-muted">
        <Icon className="h-5 w-5 text-subtle" />
      </div>
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      {description && (
        <p className="mt-1 max-w-sm text-sm text-muted-foreground">{description}</p>
      )}
      {action && <div className="mt-5">{action}</div>}
      {learnMoreHref && (
        <a
          href={learnMoreHref}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-success transition-colors hover:text-primary"
        >
          {learnMoreLabel}
          <ArrowUpRight className="h-3.5 w-3.5" />
        </a>
      )}
    </div>
  );
}

export default EmptyState;
