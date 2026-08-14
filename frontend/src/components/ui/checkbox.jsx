import * as React from "react";

import { cn } from "@/lib/utils";

/**
 * Checkbox — the shared design-system checkbox. A native `<input>` styled with
 * tokens (`accent-primary` for the checked fill, `border-input` for the box),
 * so it themes correctly and stays consistent. Codifies the pattern that was
 * hand-rolled across the app (`h-4 w-4 rounded border-input accent-primary`)
 * and adds a visible focus ring for keyboard users.
 *
 * Forwards all native input props (checked, onChange, disabled, required, …).
 */
export const Checkbox = React.forwardRef(function Checkbox(
  { className, ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      type="checkbox"
      className={cn(
        "h-4 w-4 shrink-0 rounded border-input accent-primary",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
});

export default Checkbox;
