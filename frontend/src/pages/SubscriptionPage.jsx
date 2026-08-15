import { useEffect, useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Pause, Play, RotateCw, ArrowLeftRight, Plus, X } from "lucide-react";

import { endpoints } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { usePlans } from "../lib/useCustomers";
import { useObjectQuery } from "@/lib/useObjectQuery";
import { cn, formatCurrency, formatDate, formatDateTime, toMinorUnits } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedRow,
  RelatedEmpty,
  ObjectPageSkeleton,
  ObjectNotFound,
  ObjectPageError,
} from "@/components/patterns/ObjectPage";
import { AuditTrail } from "@/components/patterns/AuditTrail";
import { FinancialSummary } from "@/components/patterns/FinancialSummary";
import { ObjectTimeline } from "@/components/patterns/ObjectTimeline";
import { AttentionBanner } from "@/components/patterns/AttentionBanner";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Money } from "@/components/ui/money";
import { StatusBadge } from "@/components/ui/status-badge";
import { CopyableId } from "@/components/ui/copyable-id";
import { Overline } from "@/components/ui/overline";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

function prorationRow(label, valueMinor, cur, muted) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className={muted ? "text-muted-foreground" : "text-foreground"}>{label}</span>
      <span
        className={cn(
          "tabular-nums",
          muted ? "text-muted-foreground" : "font-medium text-foreground"
        )}
      >
        {formatCurrency(valueMinor, cur)}
      </span>
    </div>
  );
}

/**
 * SubscriptionPage — the subscription's full object page at
 * /subscriptions/:id (DASHBOARD_REDESIGN.md Phase 5). Decomposes the former
 * 1000-line detail sheet: identity header with lifecycle actions, overview,
 * billing controls (advance / commitment / one-off charge / bill-usage-now),
 * plan change with proration preview, usage, add-ons, subscription-scoped
 * invoices, metadata + audit rail.
 */
export default function SubscriptionPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();

  const {
    object: subscription,
    loading: objectLoading,
    notFound,
    isError,
    error,
    refetch,
  } = useObjectQuery(
    ["subscription", id],
    async () => (await endpoints.getSubscription(id)).data.data,
    { enabled: Boolean(id) }
  );

  const { data: customer } = useQuery({
    queryKey: ["customer", subscription?.customer_id],
    queryFn: async () =>
      (await endpoints.getCustomer(subscription.customer_id)).data.data,
    enabled: Boolean(subscription?.customer_id),
  });

  const { plans } = usePlans();
  const plan = plans.find((p) => p.id === subscription?.plan_id);

  // The lifecycle: every recorded status transition and plan switch.
  const { data: lifecycleHistory = [] } = useQuery({
    queryKey: ["subscriptionHistory", id],
    queryFn: async () => (await endpoints.getSubscriptionHistory(id)).data.data?.history || [],
    enabled: Boolean(id),
  });

  const { data: addons = [], refetch: refreshAddons } = useQuery({
    queryKey: ["subscriptionAddons", id],
    queryFn: async () => (await endpoints.getSubscriptionAddons(id)).data.data || [],
    enabled: Boolean(id),
  });

  // Accrued metered charges for the running period; absent → section hides.
  const { data: usageAmount, refetch: refreshUsageAmount } = useQuery({
    queryKey: ["usageAmount", id],
    queryFn: async () => {
      try {
        return (await endpoints.getUsageAmount(id)).data.data || null;
      } catch {
        return null;
      }
    },
    enabled: Boolean(id),
  });

  const { data: subUsage } = useQuery({
    queryKey: ["subscriptionUsage", id],
    queryFn: async () => {
      try {
        return (await endpoints.getSubscriptionUsage(id)).data.data || null;
      } catch {
        return null;
      }
    },
    enabled: Boolean(id),
  });

  const { data: pendingCharges, refetch: refreshCharges } = useQuery({
    queryKey: ["subscriptionCharges", id],
    queryFn: async () => (await endpoints.getSubscriptionCharges(id)).data.data || [],
    enabled: Boolean(id),
  });

  const { data: invoicesPage } = useQuery({
    queryKey: ["invoices", { subscription_id: id }],
    queryFn: async () =>
      (await endpoints.getInvoices({ subscription_id: id, per_page: 5 })).data,
    enabled: Boolean(id),
  });
  const invoices = invoicesPage?.data || [];

  // The financial position: MRR (canonical backend definition), recurring
  // amount, next invoice, and the invoice-derived outstanding position.
  const { data: finSummary } = useQuery({
    queryKey: ["subscriptionFinancialSummary", id],
    queryFn: async () => (await endpoints.getSubscriptionFinancialSummary(id)).data.data,
    enabled: Boolean(id),
  });
  const invoiceTotal = invoicesPage?.pagination?.total ?? invoices.length;

  // ---- action state (ported from the detail sheet) ----
  const [loading, setLoading] = useState(false);
  const [changing, setChanging] = useState(false);
  const [newPlanId, setNewPlanId] = useState("");
  const [preview, setPreview] = useState(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [applying, setApplying] = useState(false);
  const [addonPlanId, setAddonPlanId] = useState("");
  const [addonQty, setAddonQty] = useState("1");
  const [addonBusy, setAddonBusy] = useState(false);
  const [confirmAction, setConfirmAction] = useState(null); // pause | resume | reactivate
  const [billingPanel, setBillingPanel] = useState(null); // advance | commitment | charge
  const [advPeriods, setAdvPeriods] = useState("1");
  const [commitAmount, setCommitAmount] = useState("");
  const [billingBusy, setBillingBusy] = useState(false);
  const [chargeAmount, setChargeAmount] = useState("");
  const [chargeDesc, setChargeDesc] = useState("");
  const [billingUsage, setBillingUsage] = useState(false);
  const [confirmBillUsage, setConfirmBillUsage] = useState(false);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [cancelReason, setCancelReason] = useState("");
  const [cancelFeedback, setCancelFeedback] = useState("");
  const [cancelAtPeriodEnd, setCancelAtPeriodEnd] = useState(true);
  const [reasons, setReasons] = useState([]);
  const [cancelBusy, setCancelBusy] = useState(false);
  // Cancel is a two-step flow: pick reason → preview the financial impact →
  // confirm. cancelStep gates which the dialog shows.
  const [cancelStep, setCancelStep] = useState("form"); // "form" | "preview"
  const [cancelPreview, setCancelPreview] = useState(null);
  const [cancelPreviewLoading, setCancelPreviewLoading] = useState(false);
  const [cancelPreviewError, setCancelPreviewError] = useState(false);

  // Refresh everything this page derives from the subscription.
  const refreshSubscription = () => {
    queryClient.invalidateQueries({ queryKey: ["subscription", id] });
    queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
    queryClient.invalidateQueries({ queryKey: ["invoices", { subscription_id: id }] });
  };

  // Proration preview whenever a target plan is picked.
  useEffect(() => {
    if (!newPlanId || !id) return;
    setPreviewLoading(true);
    setPreview(null);
    endpoints
      .previewPlanChange(id, newPlanId)
      .then((res) => setPreview(res.data.data))
      .catch((err) =>
        toast.error(err?.response?.data?.error?.message || "Failed to preview plan change")
      )
      .finally(() => setPreviewLoading(false));
  }, [newPlanId, id]);

  // Cancellation-reason catalog, loaded when the cancel dialog first opens.
  useEffect(() => {
    if (!cancelOpen || reasons.length > 0) return;
    endpoints
      .getCancellationReasons()
      .then((res) => setReasons(res.data?.data || []))
      .catch(() => setReasons([]));
  }, [cancelOpen, reasons.length]);

  if (objectLoading) return <ObjectPageSkeleton />;
  if (notFound) {
    return (
      <ObjectNotFound
        objectLabel="subscription"
        identifier={id ? String(id).slice(0, 8) : undefined}
        backTo="/subscriptions"
        backLabel="Subscriptions"
      />
    );
  }
  if (isError) {
    return (
      <ObjectPageError objectLabel="subscription" error={error} onRetry={refetch} backTo="/subscriptions" backLabel="Subscriptions" />
    );
  }

  const price = plan?.prices?.[0];
  const currency = price ? price.currency : "USD";
  const planName = plan?.name || subscription.plan_id?.slice(0, 8);
  const planNameOf = (pid) => plans.find((p) => p.id === pid)?.name || pid?.slice(0, 8);
  const isActive = subscription.status === "active";

  // Layer 3 — why is it in this state, and what happens next? Surfaced above the
  // fold via the shared AttentionBanner. Every line is grounded in a real field;
  // the past-due decline reason comes from the subscription's own past-due
  // invoice (already fetched), so the operator sees the cause, not just "failed".
  const pastDueInvoice = invoices.find((inv) => inv.status === "past_due");
  const attention = [];
  if (subscription.status === "past_due") {
    attention.push({
      tone: "danger",
      to: pastDueInvoice ? `/invoices/${pastDueInvoice.id}` : undefined,
      text: (
        <>
          Renewal payment failed
          {pastDueInvoice?.last_payment_error ? `: ${pastDueInvoice.last_payment_error}` : ""}
          {pastDueInvoice?.next_retry_at
            ? ` — next retry ${formatDate(pastDueInvoice.next_retry_at)}`
            : ""}
          .
        </>
      ),
    });
  } else if (subscription.status === "unpaid") {
    attention.push({
      tone: "danger",
      text: "Dunning exhausted without recovery — this subscription is unpaid.",
    });
  } else if (subscription.status === "paused") {
    attention.push({
      tone: "warning",
      text: subscription.resume_at ? (
        <>Billing is paused — resumes {formatDate(subscription.resume_at)}.</>
      ) : (
        "Billing is paused. Resume to restart the cycle."
      ),
    });
  }
  if (isActive && subscription.cancel_at_period_end) {
    attention.push({
      tone: "warning",
      text: <>Scheduled to cancel at period end — {formatDate(subscription.current_period_end)}.</>,
    });
  }
  if (subscription.status === "trialing" && subscription.trial_end) {
    attention.push({
      tone: "warning",
      text: <>Trial ends {formatDate(subscription.trial_end)} — the first invoice issues on conversion.</>,
    });
  }

  const lifecycle = {
    pause: {
      title: "Pause this subscription?",
      description:
        "Billing stops until it is resumed. The customer keeps access per your pause policy.",
      confirmLabel: "Pause subscription",
      run: () => endpoints.pauseSubscription(subscription.id),
      failure: "Failed to pause subscription",
    },
    resume: {
      title: "Resume this subscription?",
      description: "Billing restarts from the current period.",
      confirmLabel: "Resume subscription",
      run: () => endpoints.resumeSubscription(subscription.id),
      failure: "Failed to resume subscription",
    },
    reactivate: {
      title: "Reactivate this subscription?",
      description: "Billing restarts on the current plan from the next cycle.",
      confirmLabel: "Reactivate subscription",
      run: () => endpoints.reactivateSubscription(subscription.id),
      failure: "Failed to reactivate subscription",
    },
  };

  const runLifecycleAction = async () => {
    const action = lifecycle[confirmAction];
    if (!action) return;
    setLoading(true);
    try {
      await action.run();
      setConfirmAction(null);
      toast.success(`${action.confirmLabel.split(" ")[0]}d`);
      refreshSubscription();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || action.failure);
    } finally {
      setLoading(false);
    }
  };

  const selectedReason = reasons.find((r) => r.id === cancelReason);

  // Reset the cancel flow whenever the dialog opens, so a re-open never shows a
  // stale preview.
  const openCancel = () => {
    setCancelStep("form");
    setCancelPreview(null);
    setCancelPreviewError(false);
    setCancelOpen(true);
  };

  // Step 1 → 2: fetch the backend's financial forecast for THIS cancel choice
  // (immediate vs period-end). The amounts shown at confirm come only from here
  // — never computed on the client. On failure we surface the error and make it
  // explicit that no preview is available (the confirm must not imply one).
  const loadCancelPreview = async () => {
    if (!cancelReason) return;
    setCancelPreviewLoading(true);
    setCancelPreviewError(false);
    setCancelPreview(null);
    try {
      const res = await endpoints.getSubscriptionCancelPreview(subscription.id, {
        immediately: !cancelAtPeriodEnd,
      });
      setCancelPreview(res.data.data);
    } catch {
      setCancelPreviewError(true);
    } finally {
      setCancelPreviewLoading(false);
      setCancelStep("preview");
    }
  };

  const submitCancel = async () => {
    if (!cancelReason) return;
    setCancelBusy(true);
    try {
      await endpoints.cancelSubscription(subscription.id, {
        reason: cancelReason,
        feedback: cancelFeedback.trim() || undefined,
        cancel_at_period_end: cancelAtPeriodEnd,
        immediately: !cancelAtPeriodEnd,
      });
      setCancelOpen(false);
      setCancelReason("");
      setCancelFeedback("");
      setCancelStep("form");
      setCancelPreview(null);
      toast.success(
        cancelAtPeriodEnd ? "Subscription set to cancel at period end." : "Subscription canceled."
      );
      refreshSubscription();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to cancel subscription");
    } finally {
      setCancelBusy(false);
    }
  };

  const applyChange = async () => {
    if (!newPlanId) return;
    setApplying(true);
    try {
      await endpoints.updateSubscription(subscription.id, { plan_id: newPlanId });
      toast.success("Plan changed");
      setChanging(false);
      setNewPlanId("");
      setPreview(null);
      refreshSubscription();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to change plan");
    } finally {
      setApplying(false);
    }
  };

  const addAddon = async () => {
    if (!addonPlanId) return;
    setAddonBusy(true);
    try {
      await endpoints.addSubscriptionAddon(subscription.id, {
        plan_id: addonPlanId,
        quantity: parseInt(addonQty, 10) || 1,
      });
      toast.success("Add-on added");
      setAddonPlanId("");
      setAddonQty("1");
      refreshAddons();
      refreshSubscription();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to add add-on");
    } finally {
      setAddonBusy(false);
    }
  };

  const removeAddon = async (addonId) => {
    setAddonBusy(true);
    try {
      await endpoints.removeSubscriptionAddon(subscription.id, addonId);
      toast.success("Add-on removed");
      refreshAddons();
      refreshSubscription();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to remove add-on");
    } finally {
      setAddonBusy(false);
    }
  };

  const billUsageNow = async () => {
    setConfirmBillUsage(false);
    setBillingUsage(true);
    try {
      const res = await endpoints.billUsageNow(subscription.id);
      if (res?.data?.data?.id) {
        toast.success("Interim usage invoice generated");
      } else {
        toast.info("No usage past the threshold to bill yet");
      }
      refreshUsageAmount();
      refreshSubscription();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to bill usage");
    } finally {
      setBillingUsage(false);
    }
  };

  return (
    <div>
      <ObjectHeader
        backTo="/subscriptions"
        backLabel="Subscriptions"
        kicker="Subscription"
        title={
          customer?.name ? `${customer.name} — ${planName}` : planName || "Subscription"
        }
        badge={<StatusBadge status={subscription.status || "unknown"} flashOnChange />}
        amount={
          finSummary ? (
            <Money amountMinor={finSummary.mrr} currency={finSummary.currency} size="hero" />
          ) : undefined
        }
        amountLabel={finSummary ? "MRR" : undefined}
        meta={
          <>
            <CopyableId value={subscription.id} />
            <span>
              {formatDate(subscription.current_period_start)} –{" "}
              {formatDate(subscription.current_period_end)}
            </span>
          </>
        }
        actions={
          <>
            <Button variant="outline" onClick={() => setChanging((c) => !c)}>
              <ArrowLeftRight className="h-4 w-4" />
              Change plan
            </Button>
            {isActive && (
              <Button
                variant="outline"
                onClick={() => setConfirmAction("pause")}
                disabled={loading}
              >
                <Pause className="h-4 w-4" />
                Pause
              </Button>
            )}
            {subscription.status === "paused" && (
              <Button
                variant="outline"
                onClick={() => setConfirmAction("resume")}
                disabled={loading}
              >
                <Play className="h-4 w-4" />
                Resume
              </Button>
            )}
            {subscription.status === "canceled" && (
              <Button
                variant="outline"
                onClick={() => setConfirmAction("reactivate")}
                disabled={loading}
              >
                <RotateCw className="h-4 w-4" />
                Reactivate
              </Button>
            )}
            {(isActive || subscription.status === "paused") && (
              <Button
                variant="outline"
                onClick={openCancel}
                disabled={loading}
                className="text-destructive hover:text-destructive"
              >
                Cancel
              </Button>
            )}
          </>
        }
      />

      <AttentionBanner items={attention} />

      <ObjectPageLayout
        rail={
          <>
            <ObjectSection title="Metadata">
              <AttributeList
                columns={1}
                items={[
                  { label: "Subscription ID", value: <CopyableId value={subscription.id} /> },
                  {
                    label: "Customer ID",
                    value: subscription.customer_id ? (
                      <CopyableId value={subscription.customer_id} />
                    ) : null,
                  },
                  {
                    label: "Plan ID",
                    value: subscription.plan_id ? (
                      <CopyableId value={subscription.plan_id} />
                    ) : null,
                  },
                  { label: "Created", value: formatDate(subscription.created_at) },
                ]}
              />
            </ObjectSection>
            <ObjectSection title="Timeline">
              <ObjectTimeline objectId={subscription.id} />
            </ObjectSection>
            <ObjectSection title="Audit trail">
              <AuditTrail entityType="subscriptions" entityId={subscription.id} />
            </ObjectSection>
          </>
        }
      >
        {/* Change-plan flow with proration preview */}
        {changing && (
          <ObjectSection
            title="Change plan"
            action={
              <button
                type="button"
                onClick={() => setChanging(false)}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Close
              </button>
            }
          >
            <div className="space-y-4">
              <Select value={newPlanId} onValueChange={setNewPlanId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a new plan" />
                </SelectTrigger>
                <SelectContent>
                  {plans
                    .filter((p) => p.id !== subscription.plan_id)
                    .map((p) => (
                      <SelectItem key={p.id} value={p.id}>
                        {p.name}
                        {p.prices?.[0]
                          ? ` — ${formatCurrency(p.prices[0].amount, p.prices[0].currency)}/${p.interval_unit || "mo"}`
                          : ""}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>

              {previewLoading && (
                <p className="text-sm text-muted-foreground">Calculating proration…</p>
              )}

              {preview && !previewLoading && (
                <div className="space-y-2 border-t border-border pt-3">
                  {prorationRow("Credit for unused time", -preview.credit_amount, preview.currency, true)}
                  {prorationRow("Prorated charge for new plan", preview.charge_amount, preview.currency, true)}
                  {preview.tax_amount > 0 &&
                    prorationRow("Tax", preview.tax_amount, preview.currency, true)}
                  <div className="border-t border-border pt-2">
                    {prorationRow(
                      preview.total_amount >= 0 ? "Due now" : "Credited to account",
                      Math.abs(preview.total_amount),
                      preview.currency
                    )}
                  </div>
                  <p className="pt-1 text-xs text-muted-foreground">
                    Next invoice: {formatCurrency(preview.next_invoice_amount, preview.currency)} on{" "}
                    {formatDate(subscription.current_period_end)}
                  </p>
                  <Button size="sm" className="mt-2" onClick={applyChange} disabled={applying}>
                    {applying ? "Applying…" : "Confirm plan change"}
                  </Button>
                </div>
              )}
            </div>
          </ObjectSection>
        )}

        <ObjectSection title="Overview">
          <AttributeList
            items={[
              {
                label: "Customer",
                value: customer ? (
                  <Link
                    to={`/customers/${subscription.customer_id}`}
                    className="text-primary underline-offset-2 hover:underline"
                  >
                    {customer.name || customer.email}
                  </Link>
                ) : null,
              },
              {
                label: "Plan",
                value: plan ? (
                  <Link
                    to={`/plans/${plan.id}`}
                    className="text-primary underline-offset-2 hover:underline"
                  >
                    {planName}
                  </Link>
                ) : (
                  planName
                ),
              },
              { label: "Created", value: formatDate(subscription.created_at) },
              {
                label: "Current period",
                value: `${formatDate(subscription.current_period_start)} – ${formatDate(subscription.current_period_end)}`,
              },
            ]}
          />
        </ObjectSection>

        {/* Financial summary — MRR, current billing, and next invoice kept
            distinct (never conflated), plus the invoice-derived position. All
            amounts come from the backend's canonical summary. */}
        {finSummary && (
          <ObjectSection title="Financial summary">
            <dl className="grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-3">
              <div className="min-w-0">
                <Overline as="dt" title="Monthly Recurring Revenue — the plan's list price normalized to a month. 0 unless the subscription is active. Excludes tax, usage, add-ons and one-off charges.">
                  MRR
                </Overline>
                <dd className="mt-1 text-lg font-semibold tabular-nums text-foreground">
                  <Money amountMinor={finSummary.mrr} currency={finSummary.currency} />
                </dd>
              </div>
              <div className="min-w-0">
                <Overline as="dt">
                  Billed each period
                </Overline>
                <dd className="mt-1 text-lg font-semibold tabular-nums text-foreground">
                  <Money amountMinor={finSummary.recurring_amount} currency={finSummary.currency} />
                  <span className="ml-1 text-xs font-normal text-muted-foreground">
                    / {finSummary.interval_count > 1 ? `${finSummary.interval_count} ` : ""}
                    {finSummary.interval_unit}
                  </span>
                </dd>
              </div>
              <div className="min-w-0">
                <Overline as="dt">
                  Next invoice
                </Overline>
                <dd className="mt-1 text-lg font-semibold tabular-nums text-foreground">
                  {finSummary.next_invoice_date ? (
                    <>
                      <Money
                        amountMinor={finSummary.next_invoice_base_amount}
                        currency={finSummary.currency}
                      />
                      <span className="ml-1 text-xs font-normal text-muted-foreground">
                        base · {formatDate(finSummary.next_invoice_date)}
                      </span>
                    </>
                  ) : (
                    <span className="text-sm font-normal text-muted-foreground">
                      No further invoices
                    </span>
                  )}
                </dd>
              </div>
            </dl>
            {finSummary.next_invoice_date && (
              <p className="mt-3 text-xs text-muted-foreground">
                Next-invoice amount is the plan’s base list price — it excludes tax, metered usage,
                add-ons and one-off charges, which are computed at billing time.
              </p>
            )}
            {finSummary.outstanding?.length > 0 && (
              <div className="mt-5 border-t border-border pt-5">
                <FinancialSummary currencies={finSummary.outstanding} />
              </div>
            )}
          </ObjectSection>
        )}

        {isActive && (
          <ObjectSection title="Billing controls">
            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setBillingPanel((p) => (p === "advance" ? null : "advance"))}
              >
                Advance invoice
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setBillingPanel((p) => (p === "commitment" ? null : "commitment"))}
              >
                Commitment
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setBillingPanel((p) => (p === "charge" ? null : "charge"))}
              >
                One-off charge
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={billingUsage}
                onClick={() => setConfirmBillUsage(true)}
              >
                {billingUsage ? "Billing…" : "Bill usage now"}
              </Button>
            </div>

            {billingPanel === "advance" && (
              <div className="mt-4 space-y-3 rounded-lg border border-border bg-muted/30 p-4">
                <p className="text-sm text-muted-foreground">
                  Generate one invoice now covering the next N billing periods.
                </p>
                <div className="flex items-end gap-2">
                  <div>
                    <Label className="text-xs" htmlFor="periods-1-60">Periods (1–60)</Label>
                    <Input id="periods-1-60"
                      type="number"
                      min="1"
                      max="60"
                      value={advPeriods}
                      onChange={(e) => setAdvPeriods(e.target.value)}
                      className="w-24"
                    />
                  </div>
                  <Button
                    size="sm"
                    disabled={billingBusy || !advPeriods || Number(advPeriods) < 1}
                    onClick={async () => {
                      setBillingBusy(true);
                      try {
                        await endpoints.advanceSubscription(subscription.id, Number(advPeriods));
                        toast.success("Advance invoice generated.");
                        setBillingPanel(null);
                        refreshSubscription();
                      } catch (err) {
                        toast.error(
                          err?.response?.data?.error?.message ||
                            "Failed to generate advance invoice"
                        );
                      } finally {
                        setBillingBusy(false);
                      }
                    }}
                  >
                    {billingBusy ? "Generating…" : "Generate"}
                  </Button>
                </div>
              </div>
            )}

            {billingPanel === "commitment" && (
              <div className="mt-4 space-y-3 rounded-lg border border-border bg-muted/30 p-4">
                <p className="text-sm text-muted-foreground">
                  Minimum billed per period regardless of usage ({currency}). Set 0 to clear.
                </p>
                <div className="flex items-end gap-2">
                  <div>
                    <Label className="text-xs" htmlFor="amount">Amount ({currency})</Label>
                    <Input id="amount"
                      type="number"
                      min="0"
                      step="0.01"
                      value={commitAmount}
                      onChange={(e) => setCommitAmount(e.target.value)}
                      className="w-32"
                    />
                  </div>
                  <Button
                    size="sm"
                    disabled={billingBusy || commitAmount === ""}
                    onClick={async () => {
                      setBillingBusy(true);
                      try {
                        await endpoints.setSubscriptionCommitment(
                          subscription.id,
                          toMinorUnits(commitAmount, currency)
                        );
                        toast.success("Commitment updated.");
                        setBillingPanel(null);
                        refreshSubscription();
                      } catch (err) {
                        toast.error(
                          err?.response?.data?.error?.message || "Failed to set commitment"
                        );
                      } finally {
                        setBillingBusy(false);
                      }
                    }}
                  >
                    {billingBusy ? "Saving…" : "Save"}
                  </Button>
                </div>
              </div>
            )}

            {billingPanel === "charge" && (
              <div className="mt-4 space-y-3 rounded-lg border border-border bg-muted/30 p-4">
                <p className="text-sm text-muted-foreground">
                  Add a one-off charge (e.g. a manual adjustment or professional services) to this
                  subscription&apos;s next invoice.
                </p>
                <div className="flex flex-wrap items-end gap-2">
                  <div>
                    <Label className="text-xs" htmlFor="amount-2">Amount ({currency})</Label>
                    <Input id="amount-2"
                      type="number"
                      min="0.01"
                      step="0.01"
                      value={chargeAmount}
                      onChange={(e) => setChargeAmount(e.target.value)}
                      className="w-32"
                    />
                  </div>
                  <div className="min-w-[10rem] flex-1">
                    <Label className="text-xs" htmlFor="description">Description</Label>
                    <Input id="description"
                      value={chargeDesc}
                      onChange={(e) => setChargeDesc(e.target.value)}
                      placeholder="e.g. Onboarding services"
                    />
                  </div>
                  <Button
                    size="sm"
                    disabled={billingBusy || !chargeDesc.trim() || !(parseFloat(chargeAmount) > 0)}
                    onClick={async () => {
                      setBillingBusy(true);
                      try {
                        await endpoints.addSubscriptionCharge(subscription.id, {
                          amount: toMinorUnits(chargeAmount, currency),
                          currency,
                          description: chargeDesc.trim(),
                        });
                        toast.success("One-off charge added to the next invoice.");
                        setChargeAmount("");
                        setChargeDesc("");
                        refreshCharges();
                        refreshSubscription();
                      } catch (err) {
                        toast.error(err?.response?.data?.error?.message || "Failed to add charge");
                      } finally {
                        setBillingBusy(false);
                      }
                    }}
                  >
                    {billingBusy ? "Adding…" : "Add charge"}
                  </Button>
                </div>

                {pendingCharges != null && (
                  <div className="border-t border-border pt-3">
                    <Overline as="p" className="mb-2">
                      Pending on next invoice
                    </Overline>
                    {pendingCharges.length === 0 ? (
                      <p className="text-sm text-muted-foreground">No pending one-off charges.</p>
                    ) : (
                      <ul className="space-y-1.5">
                        {pendingCharges.map((ch, i) => (
                          <li
                            key={ch.id || i}
                            className="flex items-center justify-between gap-3 text-sm"
                          >
                            <span className="truncate text-foreground">
                              {ch.description || "One-off charge"}
                            </span>
                            <span className="shrink-0 tabular-nums font-medium">
                              {formatCurrency(ch.amount, ch.currency || currency)}
                            </span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                )}
              </div>
            )}
          </ObjectSection>
        )}

        {(usageAmount?.charges?.length > 0 || subUsage?.dimensions?.length > 0) && (
          <ObjectSection title="Usage">
            {isActive && usageAmount?.charges?.length > 0 && (
              <div className="mb-4">
                <div className="mb-2 flex items-center justify-between">
                  <h3 className="text-sm font-medium text-foreground">This period</h3>
                  <span className="text-xs text-muted-foreground">
                    accrued, pre-tax · if invoiced now
                  </span>
                </div>
                <div className="flex flex-col gap-1.5">
                  {usageAmount.charges.map((c) => (
                    <div
                      key={c.metric_code}
                      className="flex items-center justify-between gap-3 text-sm"
                    >
                      <span className="min-w-0 truncate text-foreground">
                        {c.metric_name || c.metric_code}
                        <span className="ml-1.5 font-mono text-xs text-muted-foreground">
                          {c.quantity} × {c.charge_model}
                        </span>
                      </span>
                      <span className="tabular-nums text-foreground">
                        {formatCurrency(c.amount, usageAmount.currency)}
                      </span>
                    </div>
                  ))}
                  <div className="mt-1 flex items-center justify-between border-t border-border pt-1.5 text-sm font-medium">
                    <span className="text-foreground">Total accrued</span>
                    <span className="tabular-nums text-foreground">
                      {formatCurrency(usageAmount.total_amount, usageAmount.currency)}
                    </span>
                  </div>
                </div>
              </div>
            )}

            {subUsage?.dimensions?.length > 0 && (
              <div>
                <div className="mb-2 flex items-center justify-between">
                  <h3 className="text-sm font-medium text-foreground">Consumption</h3>
                  <span className="text-xs text-muted-foreground">this period · lifetime</span>
                </div>
                <div className="flex flex-col gap-3">
                  {subUsage.dimensions.map((d) => {
                    const hasLimit = d.limit_value != null;
                    const pct =
                      hasLimit && d.limit_value > 0
                        ? Math.min(100, Math.round((d.period_quantity / d.limit_value) * 100))
                        : 0;
                    const over = hasLimit && d.remaining != null && d.remaining < 0;
                    return (
                      <div key={d.dimension} className="text-sm">
                        <div className="flex items-center justify-between gap-3">
                          <span className="min-w-0 truncate font-mono text-xs text-foreground">
                            {d.dimension}
                          </span>
                          <span className="tabular-nums text-foreground">
                            {d.period_quantity.toLocaleString()}
                            {hasLimit && (
                              <span className="text-muted-foreground">
                                {" "}
                                / {d.limit_value.toLocaleString()}
                              </span>
                            )}
                            <span className="ml-1.5 text-xs text-muted-foreground">
                              · {d.lifetime_quantity.toLocaleString()} lifetime
                            </span>
                          </span>
                        </div>
                        {hasLimit && (
                          <div className="mt-1.5 flex items-center gap-2">
                            <span className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                              <span
                                className={cn(
                                  "block h-full rounded-full",
                                  over ? "bg-destructive/50" : "bg-primary"
                                )}
                                style={{ width: `${over ? 100 : pct}%` }}
                              />
                            </span>
                            <span
                              className={cn(
                                "text-xs tabular-nums",
                                over ? "text-destructive" : "text-muted-foreground"
                              )}
                            >
                              {over
                                ? `${Math.abs(d.remaining).toLocaleString()} over`
                                : `${d.remaining.toLocaleString()} left`}
                            </span>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </ObjectSection>
        )}

        <ObjectSection
          title="Add-ons"
          action={
            <span className="text-xs text-muted-foreground">Billed from the next invoice</span>
          }
        >
          {addons.length > 0 ? (
            <ul className="mb-4 divide-y divide-border rounded-lg border border-border">
              {addons.map((a) => (
                <li key={a.id} className="flex items-center justify-between px-3 py-2 text-sm">
                  <span className="text-foreground">
                    {planNameOf(a.plan_id)}
                    {a.quantity > 1 && (
                      <span className="text-muted-foreground"> × {a.quantity}</span>
                    )}
                  </span>
                  <button
                    type="button"
                    onClick={() => removeAddon(a.id)}
                    disabled={addonBusy}
                    className="text-muted-foreground hover:text-destructive disabled:opacity-50"
                    aria-label="Remove add-on"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="mb-4 text-sm text-muted-foreground">No add-ons attached.</p>
          )}

          <div className="flex items-end gap-2">
            <div className="flex-1">
              <Select value={addonPlanId} onValueChange={setAddonPlanId}>
                <SelectTrigger>
                  <SelectValue placeholder="Add an add-on plan" />
                </SelectTrigger>
                <SelectContent>
                  {plans
                    .filter(
                      (p) =>
                        p.id !== subscription.plan_id && !addons.some((a) => a.plan_id === p.id)
                    )
                    .map((p) => (
                      <SelectItem key={p.id} value={p.id}>
                        {p.name}
                        {p.prices?.[0]
                          ? ` — ${formatCurrency(p.prices[0].amount, p.prices[0].currency)}`
                          : ""}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>
            <Input
              type="number"
              min="1"
              value={addonQty}
              onChange={(e) => setAddonQty(e.target.value)}
              className="w-16"
              aria-label="Quantity"
            />
            <Button size="sm" variant="outline" onClick={addAddon} disabled={addonBusy || !addonPlanId}>
              <Plus className="h-3.5 w-3.5" />
              Add
            </Button>
          </div>
        </ObjectSection>

        <ObjectSection
          title={`Invoices${invoiceTotal ? ` (${invoiceTotal})` : ""}`}
          flush
          action={
            invoiceTotal > invoices.length ? (
              <Link
                to="/invoices"
                className="text-xs font-medium text-primary underline-offset-2 hover:underline"
              >
                View all
              </Link>
            ) : null
          }
        >
          {invoices.length === 0 ? (
            <RelatedEmpty>No invoices for this subscription yet.</RelatedEmpty>
          ) : (
            <div className="divide-y divide-border">
              {invoices.map((inv) => (
                <RelatedRow key={inv.id} to={`/invoices/${inv.id}`}>
                  <span className="min-w-0 truncate font-medium text-foreground">
                    {inv.invoice_number || inv.id.slice(0, 8)}
                  </span>
                  <span className="flex shrink-0 items-center gap-3">
                    <span className="hidden text-muted-foreground sm:inline">
                      {formatDate(inv.created_at)}
                    </span>
                    <Money amountMinor={inv.total} currency={inv.currency} />
                    <StatusBadge status={inv.status || "unknown"} />
                  </span>
                </RelatedRow>
              ))}
            </div>
          )}
        </ObjectSection>

        {/* The lifecycle: every recorded status transition and plan switch,
            captured at the source (subscriptions predating the trigger have none). */}
        {lifecycleHistory.length > 0 && (
          <ObjectSection
            title="Lifecycle"
            action={
              <span className="text-xs text-muted-foreground">Status &amp; plan changes over time</span>
            }
          >
            <ol className="space-y-0">
              {lifecycleHistory.map((h, i) => {
                // Bounds-check against the array being mapped — `lifecycle` is
                // the actions config object (no `.length`), so the old
                // `lifecycle.length - 1` was NaN and the connector line drew on
                // the final item too.
                const last = i === lifecycleHistory.length - 1;
                return (
                  <li key={h.id} className="relative flex gap-3 pb-4 last:pb-0">
                    {!last && (
                      <span
                        data-testid="lifecycle-connector"
                        className="absolute left-[4px] top-4 h-full w-px bg-border"
                        aria-hidden
                      />
                    )}
                    <span className="relative z-10 mt-1.5 size-2 shrink-0 rounded-full bg-primary/70" />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2 text-sm">
                        {h.change_type === "plan" ? (
                          <>
                            <Overline as="span">
                              Plan
                            </Overline>
                            <span className="text-foreground">{planNameOf(h.from_value)}</span>
                            <span className="text-muted-foreground" aria-label="became">→</span>
                            <span className="text-foreground">{planNameOf(h.to_value)}</span>
                          </>
                        ) : h.from_value ? (
                          <>
                            <StatusBadge status={h.from_value} />
                            <span className="text-muted-foreground" aria-label="became">→</span>
                            <StatusBadge status={h.to_value} />
                          </>
                        ) : (
                          <>
                            <span className="text-xs text-muted-foreground">Created as</span>
                            <StatusBadge status={h.to_value} />
                          </>
                        )}
                      </div>
                      <p className="mt-0.5 text-xs text-muted-foreground">
                        {formatDateTime(h.changed_at)}
                      </p>
                    </div>
                  </li>
                );
              })}
            </ol>
          </ObjectSection>
        )}
      </ObjectPageLayout>

      <ConfirmDialog
        open={confirmBillUsage}
        onOpenChange={setConfirmBillUsage}
        title="Bill accrued usage now?"
        description="An interim invoice is generated for the usage accrued this period (if any is past the billing threshold). It is charged like any other invoice."
        confirmLabel="Bill usage now"
        busy={billingUsage}
        onConfirm={billUsageNow}
      />
      <ConfirmDialog
        open={!!confirmAction}
        onOpenChange={(open) => !open && setConfirmAction(null)}
        busy={loading}
        onConfirm={runLifecycleAction}
        {...(lifecycle[confirmAction] || {})}
      />

      <Dialog open={cancelOpen} onOpenChange={(o) => !o && setCancelOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {cancelStep === "preview" ? "Confirm cancellation" : "Cancel subscription"}
            </DialogTitle>
          </DialogHeader>

          {cancelStep === "form" ? (
            <>
              <div className="space-y-4">
                <div>
                  <Label htmlFor="reason-for-cancellation">Reason for cancellation</Label>
                  <Select value={cancelReason} onValueChange={setCancelReason}>
                    <SelectTrigger id="reason-for-cancellation">
                      <SelectValue placeholder="Select a reason…" />
                    </SelectTrigger>
                    <SelectContent>
                      {reasons.map((r) => (
                        <SelectItem key={r.id} value={r.id}>
                          {r.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                {selectedReason?.allows_feedback && (
                  <div>
                    <Label htmlFor="feedback-optional">Feedback (optional)</Label>
                    <Input id="feedback-optional"
                      value={cancelFeedback}
                      onChange={(e) => setCancelFeedback(e.target.value)}
                      placeholder="What could we have done better?"
                    />
                  </div>
                )}
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    className="h-4 w-4 accent-primary"
                    checked={cancelAtPeriodEnd}
                    onChange={(e) => setCancelAtPeriodEnd(e.target.checked)}
                  />
                  Cancel at period end (uncheck to cancel immediately)
                </label>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setCancelOpen(false)} disabled={cancelPreviewLoading}>
                  Keep subscription
                </Button>
                <Button
                  onClick={loadCancelPreview}
                  disabled={cancelPreviewLoading || !cancelReason}
                >
                  {cancelPreviewLoading ? "Loading preview…" : "Review impact"}
                </Button>
              </DialogFooter>
            </>
          ) : (
            <>
              {cancelPreviewError ? (
                <div className="rounded-lg border border-warning/30 bg-warning/5 px-4 py-3 text-sm text-warning">
                  Couldn’t load the financial preview for this cancellation. The impact below is
                  <strong> unavailable</strong> — proceed only if you’re certain. Nothing has been
                  canceled yet.
                </div>
              ) : cancelPreview ? (
                <div className="space-y-4">
                  {/* WHAT + WHEN */}
                  <div className="rounded-lg border border-border bg-muted/30 px-4 py-3">
                    <p className="text-sm font-medium text-foreground">
                      {cancelPreview.immediately
                        ? "Cancels immediately."
                        : "Cancels at the end of the current period."}
                    </p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      Effective {formatDate(cancelPreview.effective_date)} · resulting status:{" "}
                      {cancelPreview.resulting_status}
                    </p>
                  </div>
                  {/* FINANCIAL IMPACT — amount-anchored, all from the backend. */}
                  <dl className="space-y-2 text-sm">
                    {cancelPreview.deferred_revenue_forfeited > 0 && (
                      <div className="flex items-center justify-between gap-3">
                        <dt className="text-muted-foreground">
                          Deferred revenue forfeited
                          <span className="block text-xs">recognized immediately as breakage</span>
                        </dt>
                        <dd className="font-mono tabular-nums text-foreground">
                          <Money
                            amountMinor={cancelPreview.deferred_revenue_forfeited}
                            currency={cancelPreview.currency}
                          />
                        </dd>
                      </div>
                    )}
                    {cancelPreview.avoided_future_recurring > 0 && (
                      <div className="flex items-center justify-between gap-3">
                        <dt className="text-muted-foreground">Future recurring no longer billed</dt>
                        <dd className="font-mono tabular-nums text-foreground">
                          <Money
                            amountMinor={cancelPreview.avoided_future_recurring}
                            currency={cancelPreview.currency}
                          />
                        </dd>
                      </div>
                    )}
                    <div className="flex items-center justify-between gap-3">
                      <dt className="text-muted-foreground">Refund</dt>
                      <dd className="text-foreground">
                        {cancelPreview.flat_fee_refund > 0 ? (
                          <span className="font-mono tabular-nums">
                            <Money
                              amountMinor={cancelPreview.flat_fee_refund}
                              currency={cancelPreview.currency}
                            />
                          </span>
                        ) : (
                          <span className="text-muted-foreground">
                            None — the current period was paid in advance.
                          </span>
                        )}
                      </dd>
                    </div>
                  </dl>
                </div>
              ) : null}
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setCancelStep("form")}
                  disabled={cancelBusy}
                >
                  Back
                </Button>
                <Button variant="destructive" onClick={submitCancel} disabled={cancelBusy}>
                  {cancelBusy
                    ? "Canceling…"
                    : cancelPreviewError
                      ? "Cancel without preview"
                      : "Confirm cancellation"}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
