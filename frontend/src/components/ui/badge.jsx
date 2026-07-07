import * as React from "react";
import { cva } from "class-variance-authority";

import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors focus:outline-none",
  {
    variants: {
      variant: {
        default: "border-transparent bg-primary/10 text-primary",
        secondary: "border-transparent bg-secondary text-secondary-foreground",
        success:
          "border-transparent bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-600/20",
        warning:
          "border-transparent bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20",
        destructive:
          "border-transparent bg-red-50 text-red-700 ring-1 ring-inset ring-red-600/20",
        info: "border-transparent bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-600/20",
        neutral:
          "border-transparent bg-zinc-100 text-zinc-600 ring-1 ring-inset ring-zinc-500/20",
        outline: "text-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

function Badge({ className, variant, ...props }) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
