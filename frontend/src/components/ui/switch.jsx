import * as React from "react";

import { cn } from "@/lib/utils";

/**
 * Switch — an accessible on/off toggle for settings (dependency-free).
 *
 * Controlled: pass `checked` and `onCheckedChange(next)`. Native <button> gives
 * Enter/Space + focus for free; supply an accessible name via `aria-label` or
 * `aria-labelledby` (point it at the visible setting label). Use for on/off
 * settings; use a checkbox for multi-select lists.
 */
const Switch = React.forwardRef(
  ({ className, checked = false, onCheckedChange, disabled = false, ...props }, ref) => (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      ref={ref}
      onClick={() => onCheckedChange?.(!checked)}
      className={cn(
        "peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
        checked ? "bg-primary" : "bg-input",
        className
      )}
      {...props}
    >
      <span
        aria-hidden="true"
        className={cn(
          "pointer-events-none block h-4 w-4 rounded-full bg-background shadow-sm ring-0 transition-transform",
          checked ? "translate-x-4" : "translate-x-0"
        )}
      />
    </button>
  )
);
Switch.displayName = "Switch";

export { Switch };
