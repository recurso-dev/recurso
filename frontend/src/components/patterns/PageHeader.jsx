import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * PageHeader — the standard header for every page.
 *
 * Props:
 *  - title:       string (required)
 *  - description: string
 *  - breadcrumbs: [{ label, to? }]  (last item is rendered as current page)
 *  - actions:     ReactNode (right-aligned buttons, e.g. <Button>New</Button>)
 */
export function PageHeader({ title, description, breadcrumbs, actions, className }) {
  return (
    <div className={cn("mb-6", className)}>
      {breadcrumbs?.length > 0 && (
        <nav className="mb-2 flex items-center gap-1.5 text-sm text-muted-foreground">
          {breadcrumbs.map((crumb, i) => {
            const isLast = i === breadcrumbs.length - 1;
            return (
              <span key={i} className="flex items-center gap-1.5">
                {i > 0 && <ChevronRight className="h-3.5 w-3.5 text-zinc-300" />}
                {crumb.to && !isLast ? (
                  <Link to={crumb.to} className="hover:text-foreground transition-colors">
                    {crumb.label}
                  </Link>
                ) : (
                  <span className={cn(isLast && "text-foreground font-medium")}>
                    {crumb.label}
                  </span>
                )}
              </span>
            );
          })}
        </nav>
      )}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <h1 className="truncate text-2xl font-semibold tracking-tight text-foreground">
            {title}
          </h1>
          {description && (
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          )}
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}

export default PageHeader;
