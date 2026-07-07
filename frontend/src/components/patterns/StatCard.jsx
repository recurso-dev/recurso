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
 */
export function StatCard({
  label,
  value,
  delta,
  deltaType = "neutral",
  icon: Icon,
  hint,
  loading = false,
  className,
}) {
  const deltaStyles = {
    positive: "text-emerald-600",
    negative: "text-red-600",
    neutral: "text-muted-foreground",
  };
  const DeltaArrow = deltaType === "negative" ? ArrowDownRight : ArrowUpRight;

  return (
    <Card className={cn("p-5", className)}>
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </p>
        {Icon && <Icon className="h-4 w-4 text-zinc-400" />}
      </div>
      <div className="mt-3 flex items-end justify-between gap-2">
        {loading ? (
          <div className="h-8 w-24 animate-pulse rounded bg-zinc-100" />
        ) : (
          <p className="text-3xl font-semibold tracking-tight tabular-nums text-foreground">
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
}

export default StatCard;
