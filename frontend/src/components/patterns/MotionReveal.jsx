import { Children, cloneElement, isValidElement } from "react";

import { cn } from "@/lib/utils";
import { useReducedMotion } from "@/lib/useReducedMotion";

/**
 * MotionReveal — reveals its content on mount with a fade + small rise, with an
 * optional delay. transform/opacity only. Under reduced motion it renders
 * immediately with no delay.
 *
 * On dashboard load, reveal in hierarchy order: header first, then primary
 * content, then secondary — a small stagger, never a cinematic sequence.
 */
export function MotionReveal({
  as: Tag = "div",
  delay = 0,
  className,
  style,
  children,
  ...rest
}) {
  const reduced = useReducedMotion();
  if (reduced) {
    return (
      <Tag className={className} style={style} {...rest}>
        {children}
      </Tag>
    );
  }
  return (
    <Tag
      className={cn("animate-motion-reveal", className)}
      style={{ ...(delay ? { animationDelay: `${delay}ms` } : null), ...style }}
      {...rest}
    >
      {children}
    </Tag>
  );
}

/**
 * MotionStagger — reveals its direct children in sequence, each delayed a step
 * more than the last. Clones the reveal onto each child (no extra wrapper), so
 * children must forward `className` and `style` — all Recurso patterns do. Under
 * reduced motion, children render as-is.
 *
 *   <MotionStagger>
 *     <StatCard .../> <StatCard .../> <StatCard .../>
 *   </MotionStagger>
 */
export function MotionStagger({ step = 60, initialDelay = 0, children }) {
  const reduced = useReducedMotion();
  return Children.map(children, (child, i) => {
    if (reduced || !isValidElement(child)) return child;
    const delay = initialDelay + i * step;
    return cloneElement(child, {
      className: cn("animate-motion-reveal", child.props.className),
      style: { animationDelay: `${delay}ms`, ...(child.props.style || null) },
    });
  });
}
