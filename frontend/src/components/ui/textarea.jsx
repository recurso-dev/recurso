import * as React from "react";

import { cn } from "@/lib/utils";

/**
 * Textarea — the multi-line sibling of ui/input. The audit found the Input
 * class string copy-pasted verbatim onto 10 raw <textarea>s, several of which
 * drifted (lost focus rings and the aria-invalid state). This is the one
 * source of that treatment; compose with FormField for label/error wiring.
 */
const Textarea = React.forwardRef(({ className, ...props }, ref) => (
  <textarea
    ref={ref}
    className={cn(
      "flex min-h-[72px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-raised transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-50 aria-[invalid=true]:border-destructive aria-[invalid=true]:focus-visible:ring-destructive",
      className
    )}
    {...props}
  />
));
Textarea.displayName = "Textarea";

export { Textarea };
export default Textarea;
