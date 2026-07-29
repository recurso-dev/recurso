import { useEffect, useState } from "react";
import { Clock, X } from "lucide-react";

import { endpoints } from "@/lib/api";

const PRICING_URL = "https://recurso.dev/pricing";

// TrialBanner surfaces the managed-cloud trial: how many days are left, turning
// urgent as it nears (or passes) expiry, with a path to choose a plan. It fetches
// billing status on mount and renders nothing unless the tenant is trialing.
// Dismissal is session-local; it returns on reload until the tenant upgrades.
export default function TrialBanner() {
  const [status, setStatus] = useState(null);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    let active = true;
    endpoints
      .getBillingStatus()
      .then((res) => {
        if (active) setStatus(res.data || null);
      })
      .catch(() => {
        // Legacy/self-host or an error: no banner. Never block the app on this.
        if (active) setStatus(null);
      });
    return () => {
      active = false;
    };
  }, []);

  if (!status || status.billing_status !== "trialing" || dismissed) return null;

  const expired = status.trial_expired;
  const days = status.trial_days_left ?? 0;
  const urgent = expired || days <= 3;

  const tone = urgent
    ? "border-amber-200 bg-amber-50 text-amber-900"
    : "border-sky-200 bg-sky-50 text-sky-900";

  return (
    <div
      role="status"
      className={`flex shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b px-4 py-2 text-xs sm:px-6 ${tone}`}
    >
      <Clock className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      <span className="min-w-0 font-medium">
        {expired
          ? "Your free trial has ended."
          : days <= 1
            ? "Your free trial ends today."
            : `${days} days left in your free trial.`}
      </span>
      <div className="ml-auto flex items-center gap-1">
        <a
          href={PRICING_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center rounded-md border border-black/10 bg-white/60 px-2.5 py-1 font-semibold transition-colors hover:bg-white"
        >
          {expired ? "Choose a plan to continue" : "Choose a plan"}
        </a>
        {!expired && (
          <button
            type="button"
            onClick={() => setDismissed(true)}
            aria-label="Dismiss"
            className="flex h-6 w-6 items-center justify-center rounded-md transition-colors hover:bg-black/5"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
    </div>
  );
}
