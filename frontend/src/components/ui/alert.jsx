import { Info, CheckCircle2, AlertTriangle, AlertCircle } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * Alert — THE canonical tinted status panel (DASHBOARD_REDESIGN.md Stage 2).
 *
 * Replaces the ~107 hand-rolled `border-*-200 bg-*-50 text-*-700` panels the
 * audit found (84 distinct class strings; 4 radii; 4 paddings). One radius,
 * one padding, one type treatment, a fixed icon per variant. There are no
 * visual override props by design — `className` is for LAYOUT only (margins,
 * width, grid placement). If a panel needs to look different, it isn't an
 * Alert.
 *
 * Variants: info | success | warning | danger.
 * A danger Alert announces (`role="alert"`); the rest are polite status.
 *
 *   <Alert variant="warning" title="Trial ends soon">
 *     Add a payment method to keep billing running.
 *   </Alert>
 */

const VARIANTS = {
  info: {
    icon: Info,
    classes: "border-info/25 bg-info/5 text-info",
    role: "status",
  },
  success: {
    icon: CheckCircle2,
    classes: "border-success/25 bg-success/5 text-success",
    role: "status",
  },
  warning: {
    icon: AlertTriangle,
    classes: "border-warning/25 bg-warning/5 text-warning",
    role: "status",
  },
  danger: {
    icon: AlertCircle,
    classes: "border-destructive/25 bg-destructive/5 text-destructive",
    role: "alert",
  },
};

export function Alert({ variant = "info", title, children, className, ...props }) {
  const v = VARIANTS[variant] ?? VARIANTS.info;
  const Icon = v.icon;
  return (
    <div
      role={v.role}
      className={cn(
        "flex items-start gap-3 rounded-lg border px-4 py-3 text-sm",
        v.classes,
        className
      )}
      {...props}
    >
      <Icon className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        {title && <p className="font-semibold">{title}</p>}
        {children && <div className={cn(title && "mt-0.5")}>{children}</div>}
      </div>
    </div>
  );
}

export default Alert;
