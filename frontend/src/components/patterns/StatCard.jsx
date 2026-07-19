import { Link } from "react-router-dom";
import { ArrowDownRight, ArrowUpRight } from "lucide-react";

import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";

/**
 * StatCard — a KPI tile: small uppercase label, large numeral, optional delta.
 *
 * Props:
 *  - label:      string (rendered small + uppercase)
 *  - value:      string | number (the big numeral; pre-format currency yourself)
 *  - delta:      string (e.g. "+12.5%")  — omit to hide
 *  - deltaType:  "positive" | "negative" | "neutral" (drives color + arrow)
 *  - icon:       lucide icon component
 *  - hint:       string (muted helper under the value, e.g. "vs. last month")
 *  - loading:    boolean (renders a skeleton value)
 *  - to:         route path — makes the whole tile a link (hover + focus ring)
 *  - tone:       "danger" | "warning" — tints the value when the number needs attention
 */
export function StatCard({
  label,
  value,
  delta,
  deltaType = "neutral",
  icon: Icon,
  hint,
  loading = false,
  to,
  tone,
  className,
}) {
  const deltaStyles = {
    positive: "text-emerald-600",
    negative: "text-red-600",
    neutral: "text-muted-foreground",
  };
  const toneStyles = {
    danger: "text-red-600",
    warning: "text-amber-600",
  };
  const DeltaArrow = deltaType === "negative" ? ArrowDownRight : ArrowUpRight;

  const card = (
    <Card
      className={cn(
        "p-5",
        to && "transition-shadow hover:shadow-md",
        className
      )}
    >
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </p>
        {Icon && <Icon className="h-4 w-4 text-stone-400" />}
      </div>
      <div className="mt-3 flex items-end justify-between gap-2">
        {loading ? (
          <div className="h-8 w-24 animate-pulse rounded bg-stone-100" />
        ) : (
          <p
            className={cn(
              "text-3xl font-semibold tracking-tight tabular-nums",
              toneStyles[tone] || "text-foreground"
            )}
          >
            {value}
          </p>
        )}
        {delta && !loading && (
          <span
            className={cn(
              "flex items-center gap-0.5 text-sm font-medium tabular-nums",
              deltaStyles[deltaType]
            )}
          >
            {deltaType !== "neutral" && <DeltaArrow className="h-3.5 w-3.5" />}
            {delta}
          </span>
        )}
      </div>
      {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
    </Card>
  );

  // A linked tile is a real <a>: keyboard-focusable, middle-clickable.
  if (to) {
    return (
      <Link
        to={to}
        aria-label={`${label}: view details`}
        className="block rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        {card}
      </Link>
    );
  }
  return card;
}

export default StatCard;
