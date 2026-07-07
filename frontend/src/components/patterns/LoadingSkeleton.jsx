import { cn } from "@/lib/utils";

/**
 * Skeleton — the base shimmer block (shadcn-style). Compose these to mimic
 * the shape of loading content.
 *   <Skeleton className="h-4 w-32" />
 */
export function Skeleton({ className, ...props }) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-zinc-100", className)}
      {...props}
    />
  );
}

/**
 * TableSkeleton — placeholder rows for a DataTable while loading.
 */
export function TableSkeleton({ rows = 6, columns = 4 }) {
  return (
    <div className="divide-y divide-border">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex items-center gap-4 px-4 py-3.5">
          {Array.from({ length: columns }).map((_, c) => (
            <Skeleton
              key={c}
              className={cn("h-4", c === 0 ? "w-40" : "flex-1 max-w-[8rem]")}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

/**
 * CardGridSkeleton — placeholder for a row of StatCards.
 */
export function CardGridSkeleton({ count = 4 }) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="rounded-xl border border-border bg-card p-5">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="mt-4 h-8 w-28" />
        </div>
      ))}
    </div>
  );
}

export default Skeleton;
