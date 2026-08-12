import { cva } from "class-variance-authority";

import { cn } from "@/lib/utils";

// Status pills. All variants are token-sourced (DASHBOARD_UI_AUDIT §11) —
// this file is copied from more than any other, so it must never teach raw
// palette. Text-on-its-tint contrast (computed): success 4.85 · warning 5.00
// · info 5.62 · destructive 4.94 · primary 4.85 — all AA for the 12px label.
const badgeVariants = cva(
  "inline-flex items-center gap-1 whitespace-nowrap rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors",
  {
    variants: {
      variant: {
        default: "border-transparent bg-primary/10 text-primary",
        secondary: "border-transparent bg-secondary text-secondary-foreground",
        success:
          "border-transparent bg-success/10 text-success ring-1 ring-inset ring-success/20",
        warning:
          "border-transparent bg-warning/10 text-warning ring-1 ring-inset ring-warning/20",
        destructive:
          "border-transparent bg-destructive/10 text-destructive ring-1 ring-inset ring-destructive/20",
        info: "border-transparent bg-info/10 text-info ring-1 ring-inset ring-info/20",
        neutral:
          "border-transparent bg-muted text-muted-foreground ring-1 ring-inset ring-border",
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
