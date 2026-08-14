import { Link } from "react-router";

import { cn } from "@/lib/utils";
import { useSubscriptions, usePlans } from "@/lib/useCustomers";

/**
 * SubscriptionRef — a link to a subscription labeled by its **plan name** so
 * the reference reads as something meaningful ("Scale plan") instead of a raw
 * UUID fragment. Subscriptions have no name of their own, so the plan is the
 * most useful human handle; the customer is usually already on-screen where a
 * subscription is referenced.
 *
 * Resolves via the shared react-query caches (usePlans / useSubscriptions,
 * ADR-005), the same bulk name-resolution the rest of the app uses. Falls back
 * to a short id until the caches populate, so it never blocks and never shows
 * an empty reference.
 *
 * Props:
 *  - subscriptionId: the subscription uuid
 *  - className:      applied to the link
 */
export function SubscriptionRef({ subscriptionId, className }) {
  const subscriptions = useSubscriptions();
  const { names: planNames } = usePlans();

  if (!subscriptionId) {
    return <span className="text-muted-foreground">—</span>;
  }

  const sub = subscriptions.find((s) => s.id === subscriptionId);
  const planName = sub ? planNames[sub.plan_id] : undefined;
  const label = planName || `${String(subscriptionId).slice(0, 8)}…`;

  return (
    <Link
      to={`/subscriptions/${subscriptionId}`}
      className={cn(
        "text-primary underline-offset-2 hover:underline",
        // Fall back to mono while it's still a raw id so the fragment reads
        // as an identifier, not a truncated word.
        planName ? "font-medium" : "font-mono text-xs",
        className,
      )}
      title={planName ? `Subscription · ${planName}` : subscriptionId}
    >
      {label}
    </Link>
  );
}

export default SubscriptionRef;
