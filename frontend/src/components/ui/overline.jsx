import * as React from "react";

import { cn } from "@/lib/utils";

/**
 * Overline — the canonical uppercase micro-label. It is the ONE home for the
 * small, tracked, uppercase text role that recurs as:
 *   - page / object kickers (the eyebrow above a title)
 *   - metadata / attribute terms (the label half of a key→value pair)
 *   - stat / KPI labels (the label above a metric)
 *
 * One deliberate visual role, one token set. There are intentionally NO
 * size / tone / weight variants — that proliferation is the anti-pattern this
 * primitive exists to prevent. The only knob is `as`, which sets the rendered
 * element so callers keep correct semantics (a `dt` inside a `<dl>`, a `th` in a
 * table header, a `span` inline) without ever restyling the role. A caller
 * `className` is for layout-only tweaks (margins), never for changing the look.
 *
 * Not this role: form field labels (sentence-case, use `ui/label.jsx`),
 * badges/chips, mono identifiers/codes, section titles, and secondary/context
 * text — see docs/quality/DASHBOARD_POLISH_BATCH_C_DESIGN.md.
 */
const OVERLINE_CLASS = "text-xs font-medium uppercase tracking-wide text-subtle";

const Overline = React.forwardRef(function Overline(
  { as: Component = "div", className, ...props },
  ref
) {
  return <Component ref={ref} className={cn(OVERLINE_CLASS, className)} {...props} />;
});
Overline.displayName = "Overline";

export { Overline, OVERLINE_CLASS };
