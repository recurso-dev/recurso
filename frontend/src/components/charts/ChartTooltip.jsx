/**
 * Premium chart tooltip shared by every Tremor chart in the dashboard.
 *
 * Tremor's built-in tooltip is functional but generic — a plain white box with
 * unformatted values. This one reads as a considered fintech surface: a soft
 * elevated card, a colored dot per series, the category label on the left, and
 * the value right-aligned in tabular figures so multi-series numbers line up.
 *
 * Tremor passes each chart's `customTooltip` the recharts payload, which carries
 * the resolved series `color` — so the swatch always matches the line/bar with
 * no extra color plumbing at the call site.
 *
 * Usage — wrap once (memoized) with the same formatter the chart uses so the
 * tooltip and axis agree:
 *   const tooltip = useMemo(() => makeChartTooltip(chartMoney), []);
 *   <AreaChart customTooltip={tooltip} valueFormatter={chartMoney} … />
 */
export function makeChartTooltip(valueFormatter = (v) => v) {
  function ChartTooltip({ active, payload, label }) {
    if (!active || !payload?.length) return null;
    return (
      <div className="min-w-[9rem] rounded-lg border border-border bg-white px-3 py-2 shadow-lg shadow-black/[0.06]">
        {label != null && (
          <p className="mb-1.5 text-xs font-medium text-muted-foreground">{label}</p>
        )}
        <div className="space-y-1">
          {payload.map((item, i) => (
            <div
              key={`${item.dataKey ?? item.name ?? i}`}
              className="flex items-center justify-between gap-6"
            >
              <span className="flex items-center gap-2 text-sm text-foreground">
                <span
                  className="h-2 w-2 shrink-0 rounded-full"
                  style={{ backgroundColor: item.color || "hsl(var(--primary))" }}
                />
                {item.name ?? item.dataKey}
              </span>
              <span className="text-sm font-semibold tabular-nums text-foreground">
                {valueFormatter(item.value)}
              </span>
            </div>
          ))}
        </div>
      </div>
    );
  }
  ChartTooltip.displayName = "ChartTooltip";
  return ChartTooltip;
}

/**
 * Shared multi-series palette, restricted to the Tremor color names that the
 * Tailwind safelist keeps alive (see tailwind.config.js). Ordered so the first
 * series is always the emerald brand accent, then cool→warm for contrast.
 */
export const chartCategoryColors = ["emerald", "blue", "amber", "violet", "red"];

/**
 * Baseline props every dashboard chart shares — one place to tune the house
 * style. Spread first, then set the height class and per-chart props after
 * (`className`, `colors`, `categories`, …). `showAnimation` + a longer
 * duration gives the deliberate, unhurried reveal that reads as premium
 * rather than the default snap-in.
 */
export const chartDefaults = {
  showAnimation: true,
  animationDuration: 900,
};
