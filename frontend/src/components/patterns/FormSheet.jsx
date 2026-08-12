import { useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";

/**
 * FormSheet — the canonical create/edit sheet (DASHBOARD_REDESIGN.md Phase 6).
 *
 * Fixes the three defects every hand-rolled sheet form shared (audit §7):
 * no <form> (Enter dead), no autofocus, and Esc/backdrop silently discarding
 * a half-built form.
 *
 *  - Children render inside a real <form>: Enter submits, the footer's
 *    submit button is type="submit".
 *  - The first enabled field autofocuses when the sheet opens.
 *  - When `dirty` is true, closing (Esc, backdrop, Cancel) asks before
 *    discarding.
 *
 * Props:
 *  - open / onOpenChange: standard sheet control
 *  - title / description
 *  - onSubmit: called with the submit event (preventDefault already done)
 *  - submitLabel: footer button text; busyLabel while busy
 *  - busy: disables submit and shows busyLabel
 *  - canSubmit: extra gate for the submit button (default true)
 *  - dirty: enables the discard guard
 *  - error: inline error line above the footer
 *  - wide: sm:max-w-lg instead of sm:max-w-md
 */
export function FormSheet({
  open,
  onOpenChange,
  title,
  description,
  onSubmit,
  submitLabel = "Save",
  busyLabel,
  busy = false,
  canSubmit = true,
  dirty = false,
  error,
  wide = false,
  children,
}) {
  const bodyRef = useRef(null);
  const [confirmDiscard, setConfirmDiscard] = useState(false);

  // Autofocus the first enabled field when the sheet opens. Radix's own
  // focus lands on the close button; retarget after it settles.
  useEffect(() => {
    if (!open) return;
    const t = setTimeout(() => {
      const first = bodyRef.current?.querySelector(
        'input:not([disabled]):not([type="hidden"]), textarea:not([disabled]), select:not([disabled]), [role="combobox"]:not([disabled])'
      );
      first?.focus();
    }, 50);
    return () => clearTimeout(t);
  }, [open]);

  const requestClose = (next) => {
    if (next) return onOpenChange(true);
    if (dirty) {
      setConfirmDiscard(true);
      return;
    }
    onOpenChange(false);
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    if (busy || !canSubmit) return;
    onSubmit(e);
  };

  return (
    <>
      <Sheet open={open} onOpenChange={requestClose}>
        <SheetContent side="right" className={wide ? "w-full sm:max-w-lg" : "w-full sm:max-w-md"}>
          <form onSubmit={handleSubmit} className="flex h-full flex-col">
            <SheetHeader>
              <SheetTitle>{title}</SheetTitle>
              {description && <SheetDescription>{description}</SheetDescription>}
            </SheetHeader>
            <div ref={bodyRef} className="flex-1 space-y-4 overflow-y-auto px-6 py-4">
              {children}
              {error && (
                <p role="alert" className="text-sm text-destructive">
                  {error}
                </p>
              )}
            </div>
            <SheetFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => requestClose(false)}
                disabled={busy}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={busy || !canSubmit}>
                {busy ? busyLabel || `${submitLabel}…` : submitLabel}
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={confirmDiscard}
        onOpenChange={setConfirmDiscard}
        title="Discard changes?"
        description="This form has unsaved changes. Closing discards them."
        confirmLabel="Discard"
        destructive
        onConfirm={() => {
          setConfirmDiscard(false);
          onOpenChange(false);
        }}
      />
    </>
  );
}

export default FormSheet;
