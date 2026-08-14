import { useEffect, useState } from "react";

const QUERY = "(prefers-reduced-motion: reduce)";

/**
 * useReducedMotion — true when the OS "reduce motion" setting is on.
 *
 * CSS already honors the preference globally (see index.css), but JS-driven
 * motion (rAF number interpolation, staggered reveals) can't — it must gate on
 * this hook and fall back to the final, static state. SSR-safe (defaults to
 * "reduce" so nothing animates before hydration) and updates live if the user
 * flips the setting.
 */
export function useReducedMotion() {
  const [reduced, setReduced] = useState(() => {
    if (typeof window === "undefined" || !window.matchMedia) return true;
    return window.matchMedia(QUERY).matches;
  });

  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) return undefined;
    const mql = window.matchMedia(QUERY);
    const onChange = (e) => setReduced(e.matches);
    // addEventListener is the modern API; older Safari only has addListener.
    if (mql.addEventListener) mql.addEventListener("change", onChange);
    else mql.addListener(onChange);
    setReduced(mql.matches);
    return () => {
      if (mql.removeEventListener) mql.removeEventListener("change", onChange);
      else mql.removeListener(onChange);
    };
  }, []);

  return reduced;
}
