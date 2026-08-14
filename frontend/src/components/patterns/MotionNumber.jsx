import { useEffect, useRef, useState } from "react";

import { cn } from "@/lib/utils";
import { useReducedMotion } from "@/lib/useReducedMotion";

/**
 * MotionNumber — a financial value that interpolates to its new figure when it
 * changes, like a terminal updating. No bounce, no marketing count-up: it holds
 * its value on mount and only animates on a real change.
 *
 * Props:
 *  - value:    number (integer, e.g. minor units). Non-numbers render as-is.
 *  - format:   (n:number) => string — how to display the (rounded) number.
 *  - duration: ms for the interpolation (default 450).
 *
 * Reduced motion / non-numeric value → snap straight to the final value.
 * Renders tabular-nums so digits don't jitter as they change.
 */
export function MotionNumber({
  value,
  format = (n) => n.toLocaleString(),
  duration = 450,
  className,
  ...rest
}) {
  const reduced = useReducedMotion();
  const numeric = typeof value === "number" && Number.isFinite(value);

  const [display, setDisplay] = useState(value);
  const displayRef = useRef(value); // current shown number, survives re-renders
  const rafRef = useRef(null);

  useEffect(() => {
    // Nothing to interpolate: reduced motion, non-numeric, no duration, or the
    // shown value already matches — snap and record it.
    if (
      reduced ||
      !numeric ||
      duration <= 0 ||
      typeof displayRef.current !== "number" ||
      displayRef.current === value
    ) {
      displayRef.current = value;
      setDisplay(value);
      return undefined;
    }

    const from = displayRef.current;
    const to = value;
    let start = null;
    const tick = (ts) => {
      if (start === null) start = ts;
      const t = Math.min(1, (ts - start) / duration);
      const eased = 1 - Math.pow(1 - t, 3); // easeOutCubic — decelerate, settle
      const current = Math.round(from + (to - from) * eased);
      displayRef.current = current;
      setDisplay(current);
      if (t < 1) {
        rafRef.current = requestAnimationFrame(tick);
      } else {
        displayRef.current = to;
        setDisplay(to);
      }
    };
    rafRef.current = requestAnimationFrame(tick);
    return () => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [value, duration, reduced, numeric]);

  return (
    <span className={cn("tabular-nums", className)} {...rest}>
      {numeric && typeof display === "number" ? format(display) : format?.(value) ?? value}
    </span>
  );
}
