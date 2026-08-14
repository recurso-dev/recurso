import { Link } from "react-router";
import { ArrowDownRight, ArrowUpRight, Info } from "lucide-react";

import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import { MotionNumber } from "@/components/patterns/MotionNumber";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

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
 *  - definition: string — explains what the metric measures, via an info tooltip
 *                next to the label (native title fallback on linked tiles, to
 *                avoid nesting an interactive trigger inside the tile's <a>)
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
  definition,
  className,
  style,
  format,
}) {
  const deltaStyles = {
    positive: "text-success",
    negative: "text-destructive",
    neutral: "text-muted-foreground",
  };
  const toneStyles = {
    danger: "text-destructive",
    warning: "text-warning",
  };
  const DeltaArrow = deltaType === "negative" ? ArrowDownRight : ArrowUpRight;

  const card = (
    <Card
      style={style}
      className={cn(
        "p-5",
        to && "transition-shadow hover:shadow-md",
        className
      )}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <p
            className="text-xs font-medium uppercase tracking-wide text-muted-foreground"
            // On a linked tile we can't nest an interactive tooltip trigger in
            // the <a>, so the definition rides along as a native tooltip.
            title={definition && to ? definition : undefined}
          >
            {label}
          </p>
          {definition && !to && (
            <TooltipProvider delayDuration={150}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    aria-label={`What does ${label} mean?`}
                    className="text-subtle transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-full"
                  >
                    <Info className="h-3.5 w-3.5" />
                  </button>
                </TooltipTrigger>
                <TooltipContent className="max-w-xs text-left font-normal normal-case tracking-normal">
                  {definition}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}
        </div>
        {Icon && <Icon className="h-4 w-4 text-subtle" />}
      </div>
      <div className="mt-3 flex items-end justify-between gap-2">
        {loading ? (
          <div className="h-8 w-24 animate-pulse rounded bg-muted" />
        ) : (
          <p
            className={cn(
              "text-3xl font-semibold tracking-tight tabular-nums",
              toneStyles[tone] || "text-foreground"
            )}
          >
            {/* A numeric value interpolates to its new figure when the metric
                changes (integer domain: counts, minor-unit money). Pre-formatted
                string values render as-is. */}
            {typeof value === "number" ? (
              <MotionNumber value={value} format={format} />
            ) : (
              value
            )}
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
