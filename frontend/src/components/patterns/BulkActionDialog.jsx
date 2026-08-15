import { Loader2, CheckCircle2, AlertTriangle } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

/**
 * BulkActionDialog — the confirm → progress → result surface for a bulk
 * operation, driven by a useBulkAction() `state`.
 *
 * Scope is always stated in words on the confirm view (e.g. "Send 24 invoices").
 * A partial failure is a FIRST-CLASS state, never reported as success — the
 * failed records stay listed and retryable, and retry re-runs only those.
 *
 * Props:
 *  - open, onOpenChange
 *  - title, description, confirmLabel   (confirm view; shown when state == null)
 *  - irreversible?                       adds an explicit "can't be undone" note
 *  - destructive?                        destructive styling on the confirm CTA
 *  - noun                                singular label, e.g. "invoice"
 *  - state                               from useBulkAction (null | {...})
 *  - onConfirm()                         starts the run
 *  - onRetry(failedIds)                  retries the still-failed records
 *  - labelForId(id)?                     pretty label for a failed row (default: id)
 */
export function BulkActionDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  irreversible = false,
  destructive = false,
  noun = "record",
  state,
  onConfirm,
  onRetry,
  labelForId = (id) => id,
}) {
  const running = state?.status === "running";
  const done = state ? state.succeeded.length + state.failed.length : 0;
  const plural = `${noun}s`;

  const close = () => onOpenChange(false);
  const failedIds = state?.failed.map((f) => f.id) ?? [];

  return (
    <Dialog open={open} onOpenChange={running ? undefined : onOpenChange}>
      <DialogContent
        className="sm:max-w-md"
        // Don't let a click-away or Esc abandon an in-flight run.
        onEscapeKeyDown={running ? (e) => e.preventDefault() : undefined}
        onInteractOutside={running ? (e) => e.preventDefault() : undefined}
      >
        {/* CONFIRM */}
        {state == null && (
          <>
            <DialogHeader>
              <DialogTitle>{title}</DialogTitle>
              <DialogDescription>
                {description}
                {irreversible && (
                  <span className="mt-2 block font-medium text-foreground">
                    This can’t be undone.
                  </span>
                )}
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={close}>
                Cancel
              </Button>
              <Button variant={destructive ? "destructive" : "default"} onClick={onConfirm}>
                {confirmLabel}
              </Button>
            </DialogFooter>
          </>
        )}

        {/* PROGRESS */}
        {running && (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin text-primary" aria-hidden="true" />
                Working…
              </DialogTitle>
              <DialogDescription>
                {done} of {state.total} {plural} processed
                {state.failed.length > 0 ? ` · ${state.failed.length} failed so far` : ""}.
              </DialogDescription>
            </DialogHeader>
            <div
              className="h-2 w-full overflow-hidden rounded-full bg-muted"
              role="progressbar"
              aria-valuenow={done}
              aria-valuemin={0}
              aria-valuemax={state.total}
            >
              <div
                className="h-full rounded-full bg-primary transition-[width] duration-normal"
                style={{ width: `${state.total ? (done / state.total) * 100 : 0}%` }}
              />
            </div>
          </>
        )}

        {/* RESULT */}
        {state != null && !running && (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                {state.status === "all_succeeded" ? (
                  <>
                    <CheckCircle2 className="h-4 w-4 text-success" aria-hidden="true" />
                    All {state.total} {state.total === 1 ? noun : plural} done
                  </>
                ) : state.status === "all_failed" ? (
                  <>
                    <AlertTriangle className="h-4 w-4 text-destructive" aria-hidden="true" />
                    All {state.total} failed
                  </>
                ) : (
                  <>
                    <AlertTriangle className="h-4 w-4 text-warning" aria-hidden="true" />
                    Partially failed
                  </>
                )}
              </DialogTitle>
              <DialogDescription>
                {state.succeeded.length.toLocaleString()} succeeded,{" "}
                {state.failed.length.toLocaleString()} failed.
              </DialogDescription>
            </DialogHeader>

            {state.failed.length > 0 && (
              <ul className="max-h-48 space-y-1.5 overflow-y-auto rounded-md border border-border bg-muted/20 p-3 text-sm">
                {state.failed.map((f) => (
                  <li key={f.id} className="flex items-baseline justify-between gap-3">
                    <span className="min-w-0 truncate font-medium text-foreground">
                      {labelForId(f.id)}
                    </span>
                    <span className="shrink-0 text-xs text-destructive">{f.error}</span>
                  </li>
                ))}
              </ul>
            )}

            <DialogFooter>
              <Button variant="outline" onClick={close}>
                {state.status === "all_succeeded" ? "Done" : "Close"}
              </Button>
              {state.failed.length > 0 && onRetry && (
                <Button onClick={() => onRetry(failedIds)}>
                  Retry {state.failed.length} failed
                </Button>
              )}
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

export default BulkActionDialog;
