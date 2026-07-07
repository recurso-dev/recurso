import { AlertTriangle } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

/**
 * ErrorState — shown when a fetch fails. Offers a retry.
 *
 * Props:
 *  - title:   string
 *  - message: string (the error detail)
 *  - onRetry: () => void
 */
export function ErrorState({
  title = "Something went wrong",
  message = "We couldn't load this data. Please try again.",
  onRetry,
  className,
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center px-6 py-16 text-center",
        className
      )}
    >
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-red-100 bg-red-50">
        <AlertTriangle className="h-5 w-5 text-red-500" />
      </div>
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">{message}</p>
      {onRetry && (
        <Button variant="outline" size="sm" className="mt-5" onClick={onRetry}>
          Retry
        </Button>
      )}
    </div>
  );
}

export default ErrorState;
