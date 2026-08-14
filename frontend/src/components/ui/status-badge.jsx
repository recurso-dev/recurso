import { Badge } from "@/components/ui/badge";
import { MotionState } from "@/components/patterns/MotionState";

/**
 * StatusBadge — THE only sanctioned status rendering (DASHBOARD_REDESIGN.md).
 *
 *   status string → semantic meaning → canonical variant → canonical treatment
 *
 * Replaces the 13 per-page status→variant maps the audit found (and the
 * `past_due`-renders-raw bug: labels are always humanized). Do not map a
 * status to a Badge variant anywhere else — extend REGISTRY here instead.
 *
 * - Case- and snake_case-insensitive ("GENERATED", "past_due").
 * - Unknown statuses render neutral with a humanized label, never raw.
 * - Domain collisions are resolved with a `kind` prefix entry
 *   (e.g. a dispute's "open" is attention-needed; an invoice's is normal).
 *
 * Props: status (string) · kind? (collision namespace) · label? (override,
 * e.g. computed counts) · className? (layout only — never visual overrides).
 */

const REGISTRY = {
  // lifecycle
  active: "success",
  trialing: "info",
  paused: "warning",
  canceled: "neutral",
  archived: "neutral",
  inactive: "neutral",
  draft: "neutral",
  expired: "warning",
  pending: "warning",
  processing: "info",
  completed: "success",
  // invoices / money
  paid: "success",
  open: "info",
  past_due: "destructive",
  overdue: "destructive",
  uncollectible: "destructive",
  void: "neutral",
  refunded: "neutral",
  // quotes
  sent: "info",
  accepted: "success",
  declined: "destructive",
  converted: "success",
  // credit notes
  issued: "info",
  used: "success",
  pending_approval: "warning",
  rejected: "destructive",
  // gifts / referrals
  redeemed: "success",
  purchased: "warning",
  rewarded: "success",
  qualified: "info",
  // mandates
  authorized: "info",
  created: "neutral",
  revoked: "destructive",
  // sync / delivery / jobs
  success: "success",
  succeeded: "success",
  synced: "success",
  failed: "destructive",
  error: "destructive",
  skipped: "neutral",
  // e-invoicing (IRP/EU) — arrives uppercase from the API
  generated: "success",
  cancelled: "warning", // statutory cancellation (distinct spelling from "canceled")
  submitted: "info",
  na: "neutral",
  // risk levels
  critical: "destructive",
  high: "destructive",
  medium: "warning",
  low: "neutral",
  // disputes (kind-scoped: "open" here means action needed)
  "dispute:open": "warning",
  "dispute:resolved": "success",
};

// Labels the plain humanizer would get wrong.
const LABELS = {
  na: "N/A",
};

const humanize = (s) => {
  const t = s.replace(/_/g, " ").toLowerCase();
  return t.charAt(0).toUpperCase() + t.slice(1);
};

export function StatusBadge({ status, kind, label, className, flashOnChange = false }) {
  if (status == null || status === "") return null;
  const key = String(status).toLowerCase();
  const variant =
    (kind && REGISTRY[`${kind}:${key}`]) || REGISTRY[key] || "neutral";
  const badge = (
    <Badge variant={variant} className={className}>
      {label ?? LABELS[key] ?? humanize(key)}
    </Badge>
  );
  // On a detail page a status can advance while you watch (after an action +
  // refetch). Opt in with flashOnChange to briefly highlight the transition —
  // "something happened". No flash on first mount, and never in lists (default
  // off) where a badge just scrolls into view.
  if (!flashOnChange) return badge;
  return <MotionState motionKey={key}>{badge}</MotionState>;
}

export default StatusBadge;
