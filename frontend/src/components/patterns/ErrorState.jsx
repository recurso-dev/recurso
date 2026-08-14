import { AlertTriangle } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

/**
 * ErrorState — shown when a fetch fails. Offers a retry.
 *
 * Props:
 *  - title:      string
 *  - message:    string (the error detail)
 *  - onRetry:    () => void
 *  - retryLabel: label for the retry button (default "Retry")
 */
export function ErrorState({
  title = "Couldn't load this",
  message = "We couldn't load this data. Check your connection and try again.",
  onRetry,
  retryLabel = "Retry",
  className,
}) {
  return (
    <div
      role="alert"
      className={cn(
        "flex flex-col items-center justify-center px-6 py-16 text-center",
        className
      )}
    >
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-destructive/10 bg-destructive/5">
        <AlertTriangle className="h-5 w-5 text-destructive" />
      </div>
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">{message}</p>
      {onRetry && (
        <Button variant="outline" size="sm" className="mt-5" onClick={onRetry}>
          {retryLabel}
        </Button>
      )}
    </div>
  );
}

export default ErrorState;
