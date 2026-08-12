import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";
import { ArrowRight, CheckCircle2, Circle, X } from "lucide-react";

import { endpoints } from "../../lib/api";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useState } from "react";

// Time-to-first-invoice is the product's north-star metric. A brand-new tenant
// used to land on a dashboard of zeros with no path forward; this checklist is
// that path. Each step is STATE-DRIVEN from real data (never a stored flag
// that can drift) and deep-links to where the step is done. It disappears on
// its own the moment the journey is complete, and can be dismissed early
// (per-tenant, localStorage) by teams who imported instead.
const DISMISS_KEY = "recurso.onboarding.dismissed";

export default function OnboardingChecklist() {
  // Storage-safe: private-mode browsers and some test environments have no
  // localStorage; the checklist must degrade to "shown", never crash Home.
  const [dismissed, setDismissed] = useState(() => {
    try {
      return window.localStorage?.getItem(DISMISS_KEY) === "1";
    } catch {
      return false;
    }
  });

  // One probe per step, limit 1 — existence checks, not lists.
  const { data: gateways } = useQuery({
    queryKey: ["onboarding", "gateways"],
    queryFn: () => endpoints.getGatewayConnections(),
    staleTime: 30_000,
    retry: false,
  });
  const { data: plans } = useQuery({
    queryKey: ["onboarding", "plans"],
    queryFn: () => endpoints.getPlans({ limit: 1 }),
    staleTime: 30_000,
    retry: false,
  });
  const { data: customers } = useQuery({
    queryKey: ["onboarding", "customers"],
    queryFn: () => endpoints.getCustomers({ limit: 1 }),
    staleTime: 30_000,
    retry: false,
  });
  const { data: subs } = useQuery({
    queryKey: ["onboarding", "subscriptions"],
    queryFn: () => endpoints.getSubscriptions({ limit: 1 }),
    staleTime: 30_000,
    retry: false,
  });

  const count = (res) => (res?.data?.data || res?.data || []).length;

  const steps = [
    {
      label: "Connect a payment gateway",
      hint: "Stripe, Razorpay, or GoCardless — your keys, your money flow.",
      to: "/integrations",
      done: count(gateways) > 0,
    },
    {
      label: "Create your first plan",
      hint: "A price to subscribe customers to.",
      to: "/plans",
      done: count(plans) > 0,
    },
    {
      label: "Add your first customer",
      hint: "Who you'll be billing.",
      to: "/customers",
      done: count(customers) > 0,
    },
    {
      label: "Start a subscription",
      hint: "This generates your first invoice — the whole point.",
      to: "/subscriptions",
      done: count(subs) > 0,
    },
  ];

  const doneCount = steps.filter((s) => s.done).length;
  const allDone = doneCount === steps.length;
  // Auto-hide when complete; honor early dismissal. Data still loading for
  // every probe -> render nothing rather than flash an empty checklist.
  const anyLoaded = [gateways, plans, customers, subs].some(Boolean);
  if (dismissed || allDone || !anyLoaded) return null;

  const dismiss = () => {
    try {
      window.localStorage?.setItem(DISMISS_KEY, "1");
    } catch {
      /* storage unavailable — session-only dismissal */
    }
    setDismissed(true);
  };

  const next = steps.find((s) => !s.done);

  return (
    <Card className="mb-6 border-success/20 bg-success/5/50 p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-foreground">
            Get to your first invoice
          </h2>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {doneCount} of {steps.length} steps done
            {next ? <> — next: <span className="font-medium text-foreground">{next.label}</span></> : null}
          </p>
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-muted-foreground"
          onClick={dismiss}
          title="Dismiss checklist"
          aria-label="Dismiss onboarding checklist"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Progress */}
      <div className="mt-3 h-1.5 w-full overflow-hidden rounded-full bg-success/15" role="progressbar" aria-valuenow={doneCount} aria-valuemin={0} aria-valuemax={steps.length} aria-label="Onboarding progress">
        <div
          className="h-full rounded-full bg-success/50 transition-all"
          style={{ width: `${(doneCount / steps.length) * 100}%` }}
        />
      </div>

      <ul className="mt-4 space-y-2">
        {steps.map((step) => (
          <li key={step.label}>
            <Link
              to={step.to}
              className={cn(
                "group flex items-center gap-3 rounded-md px-2 py-1.5 transition-colors hover:bg-primary/15/60",
                step.done && "opacity-60"
              )}
            >
              {step.done ? (
                <CheckCircle2 className="h-5 w-5 shrink-0 text-success" aria-hidden />
              ) : (
                <Circle className="h-5 w-5 shrink-0 text-primary/40" aria-hidden />
              )}
              <span className={cn("text-sm font-medium", step.done && "line-through")}>
                {step.label}
              </span>
              <span className="hidden text-xs text-muted-foreground sm:inline">
                {step.hint}
              </span>
              {!step.done && (
                <ArrowRight className="ml-auto h-4 w-4 shrink-0 text-success opacity-0 transition-opacity group-hover:opacity-100" aria-hidden />
              )}
            </Link>
          </li>
        ))}
      </ul>
    </Card>
  );
}
