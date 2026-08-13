import { Link } from "react-router";
import { AlertTriangle, ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * AttentionBanner — the exceptions-first signal for an object page
 * (DASHBOARD_OPERATIONAL_DEPTH.md: "what needs my attention?"). Renders nothing
 * when there's nothing wrong, so a healthy object stays calm; surfaces the few
 * things a financial operator must see when there is.
 *
 * Props:
 *  - items: [{ tone: "danger" | "warning", text, to? }]
 *    `text` is a ReactNode; `to` (optional) makes the row a link to the fix.
 */
export function AttentionBanner({ items = [], className }) {
  const live = items.filter(Boolean);
  if (live.length === 0) return null;

  const worst = live.some((i) => i.tone === "danger") ? "danger" : "warning";
  const shell =
    worst === "danger"
      ? "border-destructive/30 bg-destructive/5"
      : "border-warning/30 bg-warning/5";

  return (
    <div
      className={cn("mb-6 rounded-lg border px-4 py-3", shell, className)}
      role="status"
      aria-label="Needs attention"
    >
      <div className="flex items-start gap-3">
        <AlertTriangle
          className={cn(
            "mt-0.5 h-4 w-4 shrink-0",
            worst === "danger" ? "text-destructive" : "text-warning",
          )}
          aria-hidden="true"
        />
        <ul className="min-w-0 flex-1 space-y-1.5">
          {live.map((item, i) => {
            const body = (
              <span
                className={cn(
                  "text-sm",
                  item.tone === "danger" ? "text-destructive" : "text-warning",
                )}
              >
                {item.text}
              </span>
            );
            return (
              <li key={i} className="min-w-0">
                {item.to ? (
                  <Link
                    to={item.to}
                    className="group inline-flex items-center gap-1 rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {body}
                    <ChevronRight className="h-3.5 w-3.5 opacity-60 transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
                  </Link>
                ) : (
                  body
                )}
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}
