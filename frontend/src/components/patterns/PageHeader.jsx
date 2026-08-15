import { cn } from "@/lib/utils";

/**
 * PageHeader — the standard header for every page.
 *
 * Props:
 *  - title:       string (required)
 *  - description: string
 *  - actions:     ReactNode (right-aligned buttons, e.g. <Button>New</Button>)
 *
 * (No breadcrumbs: the active-nav rail answers "where am I", and object pages use
 * context-preserving back navigation — see useListBackDestination. The old dead
 * `breadcrumbs` prop was retired in Batch F1.)
 */
export function PageHeader({ title, description, actions, className, titleId = "page-title" }) {
  return (
    <div className={cn("mb-6", className)}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          {/* Stable id so a page's DataTable can name itself from this visible
              heading via aria-labelledby (Batch D — accessible table names). */}
          <h1
            id={titleId}
            className="truncate text-2xl font-semibold tracking-tight text-foreground"
          >
            {title}
          </h1>
          {description && (
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          )}
        </div>
        {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}

export default PageHeader;
