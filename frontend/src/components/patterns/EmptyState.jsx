import { Inbox } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * EmptyState — shown when a list/section has no data.
 *
 * Props:
 *  - icon:        lucide icon component (defaults to Inbox)
 *  - title:       string
 *  - description: string
 *  - action:      ReactNode (e.g. a <Button>)
 */
export function EmptyState({
  icon: Icon = Inbox,
  title = "Nothing here yet",
  description,
  action,
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
        <Icon className="h-5 w-5 text-stone-400" />
      </div>
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      {description && (
        <p className="mt-1 max-w-sm text-sm text-muted-foreground">{description}</p>
      )}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}

export default EmptyState;
