import { useEffect, useRef, useState } from "react";

import { cn } from "@/lib/utils";
import { useReducedMotion } from "@/lib/useReducedMotion";

/**
 * MotionState — briefly highlights its content when `motionKey` changes, to say
 * "something actually happened" as a status or value transitions (e.g. a
 * payment badge going Pending → Succeeded). It does NOT flash on first mount,
 * and never pulses continuously — one restrained highlight, then normal state.
 *
 * Under reduced motion the highlight is skipped; the changed content still
 * renders, so no information depends on the animation.
 */
export function MotionState({ motionKey, className, children, ...rest }) {
  const reduced = useReducedMotion();
  const [flash, setFlash] = useState(false);
  const prevKey = useRef(motionKey);
  const mounted = useRef(false);

  useEffect(() => {
    if (!mounted.current) {
      mounted.current = true;
      prevKey.current = motionKey;
      return;
    }
    if (prevKey.current !== motionKey) {
      prevKey.current = motionKey;
      if (!reduced) setFlash(true);
    }
  }, [motionKey, reduced]);

  return (
    <span
      className={cn("rounded-md", flash && "animate-motion-flash", className)}
      onAnimationEnd={(e) => {
        // Ignore animations bubbling up from children.
        if (e.target === e.currentTarget) setFlash(false);
      }}
      {...rest}
    >
      {children}
    </span>
  );
}
