import { useEffect, useState } from "react";
import { Check, Clock, CreditCard, Loader2 } from "lucide-react";

import { endpoints } from "@/lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

const PRICING_URL = "https://recurso.dev/pricing";
const CONTACT_URL = "mailto:cloud@recurso.dev";

// StatusPill summarizes the tenant's current billing lifecycle.
function CurrentPlan({ status }) {
  if (!status) return null;
  const trialing = status.billing_status === "trialing";
  const days = status.trial_days_left ?? 0;

  return (
    <Card>
      <CardContent className="flex flex-wrap items-center justify-between gap-4 p-6">
        <div>
          <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Current plan</div>
          <div className="mt-1 flex items-center gap-2 text-lg font-semibold capitalize text-foreground">
            {trialing ? "Free trial" : status.plan_tier || "—"}
            {trialing && (
              <span className="inline-flex items-center gap-1 rounded-full border border-sky-200 bg-sky-50 px-2 py-0.5 text-xs font-medium text-sky-700">
                <Clock className="h-3 w-3" />
                {status.trial_expired ? "ended" : `${days} day${days === 1 ? "" : "s"} left`}
              </span>
            )}
          </div>
        </div>
        <Button asChild>
          <a href={PRICING_URL} target="_blank" rel="noopener noreferrer">
            <CreditCard className="h-4 w-4" />
            {status.trial_expired ? "Choose a plan to continue" : "Choose a plan"}
          </a>
        </Button>
      </CardContent>
    </Card>
  );
}

function PlanCard({ plan }) {
  const href = plan.tier === "enterprise" ? CONTACT_URL : PRICING_URL;
  return (
    <Card className={plan.recommended ? "border-primary shadow-sm" : ""}>
      <CardContent className="flex h-full flex-col p-6">
        <div className="flex items-center justify-between">
          <h3 className="text-base font-semibold text-foreground">{plan.name}</h3>
          {plan.recommended && (
            <span className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">Popular</span>
          )}
        </div>
        <div className="mt-2">
          <span className="text-2xl font-semibold text-foreground">{plan.price}</span>{" "}
          {plan.period && <span className="text-sm text-muted-foreground">{plan.period}</span>}
        </div>
        {plan.free_note && <p className="mt-1 text-xs text-muted-foreground">{plan.free_note}</p>}
        <ul className="mt-4 flex-1 space-y-2">
          {plan.features.map((f) => (
            <li key={f} className="flex items-start gap-2 text-sm text-foreground">
              <Check className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
              {f}
            </li>
          ))}
        </ul>
        <Button variant={plan.recommended ? "default" : "outline"} className="mt-5 w-full" asChild>
          <a href={href} target="_blank" rel="noopener noreferrer">{plan.cta}</a>
        </Button>
      </CardContent>
    </Card>
  );
}

export default function BillingSettings() {
  const [status, setStatus] = useState(null);
  const [plans, setPlans] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError(null);
    Promise.allSettled([endpoints.getBillingStatus(), endpoints.getBillingPlans()]).then(
      ([s, p]) => {
        if (!active) return;
        if (s.status === "fulfilled") setStatus(s.value.data || null);
        if (p.status === "fulfilled") setPlans(p.value.data?.plans || []);
        // The plans catalog is the load-bearing content; billing status is
        // supplementary (it's legitimately absent on self-host). Only surface a
        // retryable error when the catalog itself failed — a status-only failure
        // degrades gracefully (the trial pill just doesn't render).
        if (p.status === "rejected") {
          setError(
            p.reason?.response?.data?.error?.message ||
              "We couldn't load the available plans."
          );
        }
        setLoading(false);
      }
    );
    return () => {
      active = false;
    };
  }, [reloadKey]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Billing & plan"
        description="Your managed-cloud plan and the options available. Prices mirror recurso.dev/pricing."
      />

      {loading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground" role="status">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading your billing status…
        </div>
      ) : error ? (
        <ErrorState
          title="Couldn't load billing"
          message={error}
          onRetry={() => setReloadKey((k) => k + 1)}
        />
      ) : (
        <>
          <CurrentPlan status={status} />

          <div>
            <h2 className="mb-3 text-sm font-semibold text-foreground">Plans</h2>
            <div className="grid gap-4 md:grid-cols-3">
              {plans.map((p) => (
                <PlanCard key={p.tier} plan={p} />
              ))}
            </div>
            <p className="mt-4 text-xs text-muted-foreground">
              Self-serve checkout is coming soon. For now, choosing a plan takes you to pricing or
              starts a conversation — we'll get you set up.
            </p>
          </div>
        </>
      )}
    </div>
  );
}
