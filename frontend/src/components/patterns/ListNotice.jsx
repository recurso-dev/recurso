import { cn } from "@/lib/utils";

/**
 * ListNotice — a quiet, honest warning above a list when the rows shown aren't
 * the complete set (a page-through hit its safety cap, or an endpoint that can't
 * paginate returned its hard limit). Financial lists must never *silently* drop
 * records (ANTI_PATTERNS "never hide accounting numbers"), so when we can't show
 * everything we say so, in words, with the count. Mirrors the Invoices banner.
 */
export function ListNotice({ children, className }) {
  return (
    <p
      role="status"
      className={cn(
        "mb-4 rounded-md border border-warning/20 bg-warning/5 px-3 py-2 text-sm text-warning",
        className,
      )}
    >
      {children}
    </p>
  );
}

export default ListNotice;
