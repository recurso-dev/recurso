import { useEffect, useState } from "react";
import { Pause, Play, Check, RotateCw, ArrowLeftRight, Plus, X } from "lucide-react";
import { toast } from "@/components/ui/sonner";

import { endpoints } from "../../lib/api";
import { cn, formatCurrency, formatDate, toMinorUnits } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";

export default function SubscriptionDetail({
  subscription,
  customer,
  plan,
  isOpen,
  onClose,
  onRefresh,
}) {
  const [loading, setLoading] = useState(false);
  const [changing, setChanging] = useState(false);
  const [plans, setPlans] = useState([]);
  const [newPlanId, setNewPlanId] = useState("");
  const [preview, setPreview] = useState(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [applying, setApplying] = useState(false);
  const [addons, setAddons] = useState([]);
  const [addonPlanId, setAddonPlanId] = useState("");
  const [addonQty, setAddonQty] = useState("1");
  const [addonBusy, setAddonBusy] = useState(false);
  // Which lifecycle action awaits confirmation: "pause" | "resume" | "cancel" | null.
  const [confirmAction, setConfirmAction] = useState(null);
  // Advance-invoice / minimum-commitment mini-forms.
  const [billingPanel, setBillingPanel] = useState(null); // 'advance' | 'commitment' | 'charge' | null
  const [advPeriods, setAdvPeriods] = useState("1");
  const [commitAmount, setCommitAmount] = useState("");
  const [billingBusy, setBillingBusy] = useState(false);
  const [chargeAmount, setChargeAmount] = useState("");
  const [chargeDesc, setChargeDesc] = useState("");
  // Live usage-amount preview (accrued metered charges for the running period).
  const [usageAmount, setUsageAmount] = useState(null);
  const [billingUsage, setBillingUsage] = useState(false);
  // Per-dimension usage report (period + lifetime quantity, entitlement limits).
  const [subUsage, setSubUsage] = useState(null);

  // Load plans + add-ons whenever the detail opens.
  useEffect(() => {
    if (!isOpen || !subscription) return;
    if (!plans.length) {
      endpoints
        .getPlans({ limit: 100 })
        .then((res) => setPlans(res.data?.data || []))
        .catch(() => toast.error("Failed to load plans"));
    }
    endpoints
      .getSubscriptionAddons(subscription.id)
      .then((res) => setAddons(res.data?.data || []))
      .catch(() => {});
    setUsageAmount(null);
    endpoints
      .getUsageAmount(subscription.id)
      .then((res) => setUsageAmount(res.data?.data || null))
      .catch(() => {}); // no metered charges / not applicable — section just hides
    setSubUsage(null);
    endpoints
      .getSubscriptionUsage(subscription.id)
      .then((res) => setSubUsage(res.data?.data || null))
      .catch(() => {}); // no metered usage — section just hides
  }, [isOpen, subscription, plans.length]);

  const refreshAddons = () =>
    endpoints
      .getSubscriptionAddons(subscription.id)
      .then((res) => setAddons(res.data?.data || []))
      .catch(() => {});

  const refreshUsageAmount = () =>
    endpoints
      .getUsageAmount(subscription.id)
      .then((res) => setUsageAmount(res.data?.data || null))
      .catch(() => {});

  const addAddon = async () => {
    if (!addonPlanId) return;
    setAddonBusy(true);
    try {
      await endpoints.addSubscriptionAddon(subscription.id, {
        plan_id: addonPlanId,
        quantity: parseInt(addonQty, 10) || 1,
      });
      toast.success("Add-on attached — bills from the next invoice");
      setAddonPlanId("");
      setAddonQty("1");
      await refreshAddons();
    } catch (err) {
      toast.error(
        err?.response?.data?.error?.message || "Failed to add add-on"
      );
    } finally {
      setAddonBusy(false);
    }
  };

  const removeAddon = async (addonId) => {
    setAddonBusy(true);
    try {
      await endpoints.removeSubscriptionAddon(subscription.id, addonId);
      await refreshAddons();
    } catch (err) {
      toast.error("Failed to remove add-on");
    } finally {
      setAddonBusy(false);
    }
  };

  const planName2 = (id) => plans.find((p) => p.id === id)?.name || id.slice(0, 8);

  useEffect(() => {
    if (!newPlanId || !subscription) return;
    setPreviewLoading(true);
    setPreview(null);
    endpoints
      .previewPlanChange(subscription.id, newPlanId)
      .then((res) => setPreview(res.data.data))
      .catch((err) =>
        toast.error(
          err?.response?.data?.error?.message || "Failed to preview plan change"
        )
      )
      .finally(() => setPreviewLoading(false));
  }, [newPlanId, subscription]);

  if (!subscription) return null;

  const startChange = () => {
    setChanging(true);
    setNewPlanId("");
    setPreview(null);
  };

  const applyChange = async () => {
    if (!newPlanId) return;
    setApplying(true);
    try {
      await endpoints.updateSubscription(subscription.id, { plan_id: newPlanId });
      toast.success("Plan changed");
      setChanging(false);
      if (onRefresh) onRefresh();
    } catch (err) {
      toast.error(
        err?.response?.data?.error?.message || "Failed to change plan"
      );
    } finally {
      setApplying(false);
    }
  };

  const prorationRow = (label, valueMinor, cur, muted) => (
    <div className="flex items-center justify-between text-sm">
      <span className={muted ? "text-muted-foreground" : "text-foreground"}>
        {label}
      </span>
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

  const price = plan?.prices?.[0];
  const amountMinor = price ? price.amount : 0;
  const currency = price ? price.currency : "USD";
  const planName = plan?.name || subscription.plan_id.slice(0, 8);
  const interval = plan?.interval_unit || "month";
  const isActive = subscription.status === "active";

  const lifecycle = {
    pause: {
      title: "Pause this subscription?",
      description: "Billing stops until it is resumed. The customer keeps access per your pause policy.",
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
    cancel: {
      title: "Cancel this subscription?",
      description: "The subscription ends and no further invoices are generated. This can't be undone from here.",
      confirmLabel: "Cancel subscription",
      destructive: true,
      run: () => endpoints.cancelSubscription(subscription.id),
      failure: "Failed to cancel subscription",
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
      if (onRefresh) onRefresh();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || action.failure);
    } finally {
      setLoading(false);
    }
  };

  const detail = (label, value) => (
    <div className="flex flex-col gap-1">
      <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
      <div className="text-sm font-medium text-foreground">{value}</div>
    </div>
  );

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="font-mono text-base">{subscription.id}</SheetTitle>
          <SheetDescription className="flex items-center gap-2">
            <span className="relative flex h-2 w-2">
              <span
                className={cn(
                  "absolute inline-flex h-full w-full animate-ping rounded-full opacity-75",
                  isActive ? "bg-emerald-400" : "bg-stone-400"
                )}
              />
              <span
                className={cn(
                  "relative inline-flex h-2 w-2 rounded-full",
                  isActive ? "bg-emerald-500" : "bg-stone-500"
                )}
              />
            </span>
            <span
              className={cn(
                "text-sm font-medium capitalize",
                isActive ? "text-emerald-600" : "text-muted-foreground"
              )}
            >
              {subscription.status}
            </span>
          </SheetDescription>
        </SheetHeader>

        <div className="flex-1 space-y-8 overflow-y-auto px-6 py-6">
          {/* Actions */}
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" size="sm" onClick={startChange}>
              <ArrowLeftRight className="h-3.5 w-3.5" />
              Change plan
            </Button>
            {isActive && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setConfirmAction("pause")}
                disabled={loading}
                className="text-amber-700 hover:text-amber-800"
              >
                <Pause className="h-3.5 w-3.5" />
                Pause
              </Button>
            )}
            {subscription.status === "paused" && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setConfirmAction("resume")}
                disabled={loading}
                className="text-emerald-700 hover:text-emerald-800"
              >
                <Play className="h-3.5 w-3.5" />
                Resume
              </Button>
            )}
            {(isActive || subscription.status === "paused") && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setConfirmAction("cancel")}
                disabled={loading}
                className="text-red-600 hover:text-red-700"
              >
                Cancel
              </Button>
            )}
            {subscription.status === "canceled" && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setConfirmAction("reactivate")}
                disabled={loading}
                className="text-emerald-700 hover:text-emerald-800"
              >
                <RotateCw className="h-3.5 w-3.5" />
                Reactivate
              </Button>
            )}
            {isActive && (
              <>
                <Button variant="outline" size="sm" onClick={() => setBillingPanel((p) => (p === "advance" ? null : "advance"))}>
                  Advance invoice
                </Button>
                <Button variant="outline" size="sm" onClick={() => setBillingPanel((p) => (p === "commitment" ? null : "commitment"))}>
                  Commitment
                </Button>
                <Button variant="outline" size="sm" onClick={() => setBillingPanel((p) => (p === "charge" ? null : "charge"))}>
                  One-off charge
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={billingUsage}
                  onClick={async () => {
                    setBillingUsage(true);
                    try {
                      const res = await endpoints.billUsageNow(subscription.id);
                      // 200 => interim invoice generated; 204/empty => nothing due yet.
                      if (res?.data?.data?.id) {
                        toast.success("Interim usage invoice generated");
                      } else {
                        toast.info("No usage past the threshold to bill yet");
                      }
                      refreshUsageAmount();
                    } catch (err) {
                      toast.error(
                        err?.response?.data?.error?.message || "Failed to bill usage",
                      );
                    } finally {
                      setBillingUsage(false);
                    }
                  }}
                >
                  {billingUsage ? "Billing…" : "Bill usage now"}
                </Button>
              </>
            )}
          </div>

          {/* Advance-invoice / minimum-commitment mini-forms */}
          {billingPanel === "advance" && (
            <div className="space-y-3 rounded-lg border border-border bg-muted/30 p-4">
              <p className="text-sm text-muted-foreground">
                Generate one invoice now covering the next N billing periods.
              </p>
              <div className="flex items-end gap-2">
                <div>
                  <Label className="text-xs">Periods (1–60)</Label>
                  <Input
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
                      onRefresh?.();
                    } catch (err) {
                      toast.error(err?.response?.data?.error?.message || "Failed to generate advance invoice");
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
            <div className="space-y-3 rounded-lg border border-border bg-muted/30 p-4">
              <p className="text-sm text-muted-foreground">
                Minimum billed per period regardless of usage ({currency}). Set 0 to clear.
              </p>
              <div className="flex items-end gap-2">
                <div>
                  <Label className="text-xs">Amount ({currency})</Label>
                  <Input
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
                      onRefresh?.();
                    } catch (err) {
                      toast.error(err?.response?.data?.error?.message || "Failed to set commitment");
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
            <div className="space-y-3 rounded-lg border border-border bg-muted/30 p-4">
              <p className="text-sm text-muted-foreground">
                Add a one-off charge (e.g. a manual adjustment or professional
                services) to this subscription's next invoice.
              </p>
              <div className="flex flex-wrap items-end gap-2">
                <div>
                  <Label className="text-xs">Amount ({currency})</Label>
                  <Input
                    type="number"
                    min="0.01"
                    step="0.01"
                    value={chargeAmount}
                    onChange={(e) => setChargeAmount(e.target.value)}
                    className="w-32"
                  />
                </div>
                <div className="flex-1 min-w-[10rem]">
                  <Label className="text-xs">Description</Label>
                  <Input
                    value={chargeDesc}
                    onChange={(e) => setChargeDesc(e.target.value)}
                    placeholder="e.g. Onboarding services"
                  />
                </div>
                <Button
                  size="sm"
                  disabled={
                    billingBusy ||
                    !chargeDesc.trim() ||
                    !(parseFloat(chargeAmount) > 0)
                  }
                  onClick={async () => {
                    setBillingBusy(true);
                    try {
                      await endpoints.addSubscriptionCharge(subscription.id, {
                        amount: toMinorUnits(chargeAmount, currency),
                        currency,
                        description: chargeDesc.trim(),
                      });
                      toast.success("One-off charge added to the next invoice.");
                      setBillingPanel(null);
                      setChargeAmount("");
                      setChargeDesc("");
                      onRefresh?.();
                    } catch (err) {
                      toast.error(
                        err?.response?.data?.error?.message || "Failed to add charge",
                      );
                    } finally {
                      setBillingBusy(false);
                    }
                  }}
                >
                  {billingBusy ? "Adding…" : "Add charge"}
                </Button>
              </div>
            </div>
          )}

          {/* Live usage preview — accrued metered charges for the running period */}
          {isActive && usageAmount?.charges?.length > 0 && (
            <div className="rounded-lg border border-border bg-muted/30 p-4">
              <div className="mb-2 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-foreground">Usage this period</h3>
                <span className="text-xs text-muted-foreground">
                  accrued, pre-tax · if invoiced now
                </span>
              </div>
              <div className="flex flex-col gap-1.5">
                {usageAmount.charges.map((c) => (
                  <div key={c.metric_code} className="flex items-center justify-between gap-3 text-sm">
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

          {/* Usage consumption — quantity per dimension + entitlement limits */}
          {subUsage?.dimensions?.length > 0 && (
            <div className="rounded-lg border border-border bg-muted/30 p-4">
              <div className="mb-2 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-foreground">Usage consumption</h3>
                <span className="text-xs text-muted-foreground">this period · lifetime</span>
              </div>
              <div className="flex flex-col gap-3">
                {subUsage.dimensions.map((d) => {
                  const hasLimit = d.limit_value != null;
                  const pct = hasLimit && d.limit_value > 0
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
                            <span className="text-muted-foreground"> / {d.limit_value.toLocaleString()}</span>
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
                              className={`block h-full rounded-full ${over ? "bg-red-500" : "bg-primary"}`}
                              style={{ width: `${over ? 100 : pct}%` }}
                            />
                          </span>
                          <span className={`text-xs tabular-nums ${over ? "text-red-600" : "text-muted-foreground"}`}>
                            {over ? `${Math.abs(d.remaining).toLocaleString()} over` : `${d.remaining.toLocaleString()} left`}
                          </span>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          <ConfirmDialog
            open={!!confirmAction}
            onOpenChange={(open) => !open && setConfirmAction(null)}
            busy={loading}
            onConfirm={runLifecycleAction}
            {...(lifecycle[confirmAction] || {})}
          />

          {/* Change-plan flow with proration preview */}
          {changing && (
            <div className="space-y-4 rounded-lg border border-border bg-muted/30 p-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-foreground">
                  Change plan
                </h3>
                <button
                  type="button"
                  onClick={() => setChanging(false)}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  Close
                </button>
              </div>
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
                          ? ` — ${formatCurrency(
                              p.prices[0].amount,
                              p.prices[0].currency
                            )}/${p.interval_unit || "mo"}`
                          : ""}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>

              {previewLoading && (
                <p className="text-sm text-muted-foreground">
                  Calculating proration…
                </p>
              )}

              {preview && !previewLoading && (
                <div className="space-y-2 border-t border-border pt-3">
                  {prorationRow(
                    "Credit for unused time",
                    -preview.credit_amount,
                    preview.currency,
                    true
                  )}
                  {prorationRow(
                    "Prorated charge for new plan",
                    preview.charge_amount,
                    preview.currency,
                    true
                  )}
                  {preview.tax_amount > 0 &&
                    prorationRow(
                      "Tax",
                      preview.tax_amount,
                      preview.currency,
                      true
                    )}
                  <div className="border-t border-border pt-2">
                    {prorationRow(
                      preview.total_amount >= 0
                        ? "Due now"
                        : "Credited to account",
                      Math.abs(preview.total_amount),
                      preview.currency
                    )}
                  </div>
                  <p className="pt-1 text-xs text-muted-foreground">
                    Next invoice:{" "}
                    {formatCurrency(
                      preview.next_invoice_amount,
                      preview.currency
                    )}{" "}
                    on {formatDate(subscription.current_period_end)}
                  </p>
                  <Button
                    size="sm"
                    className="mt-2 w-full"
                    onClick={applyChange}
                    disabled={applying}
                  >
                    {applying ? "Applying…" : "Confirm plan change"}
                  </Button>
                </div>
              )}
            </div>
          )}

          {/* Details */}
          <div className="grid grid-cols-2 gap-x-4 gap-y-5">
            {detail("Customer", customer?.name || "Unknown")}
            {detail("Plan", `${planName} - ${interval}`)}
            {detail("Amount", formatCurrency(amountMinor, currency))}
            {detail("Created", formatDate(subscription.created_at))}
            {detail(
              "Current period",
              `${formatDate(subscription.current_period_start)} - ${formatDate(
                subscription.current_period_end
              )}`
            )}
            {detail(
              "Upcoming invoice",
              `${formatDate(subscription.current_period_end)} for ${formatCurrency(
                amountMinor,
                currency
              )}`
            )}
          </div>

          {/* Add-ons */}
          <div className="border-t border-border pt-6">
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-foreground">Add-ons</h3>
              <span className="text-xs text-muted-foreground">
                Billed from the next invoice
              </span>
            </div>

            {addons.length > 0 ? (
              <ul className="mb-4 divide-y divide-border rounded-lg border border-border">
                {addons.map((a) => (
                  <li
                    key={a.id}
                    className="flex items-center justify-between px-3 py-2 text-sm"
                  >
                    <span className="text-foreground">
                      {planName2(a.plan_id)}
                      {a.quantity > 1 && (
                        <span className="text-muted-foreground"> × {a.quantity}</span>
                      )}
                    </span>
                    <button
                      type="button"
                      onClick={() => removeAddon(a.id)}
                      disabled={addonBusy}
                      className="text-muted-foreground hover:text-red-600 disabled:opacity-50"
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
                          p.id !== subscription.plan_id &&
                          !addons.some((a) => a.plan_id === p.id)
                      )
                      .map((p) => (
                        <SelectItem key={p.id} value={p.id}>
                          {p.name}
                          {p.prices?.[0]
                            ? ` — ${formatCurrency(
                                p.prices[0].amount,
                                p.prices[0].currency
                              )}`
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
              <Button
                size="sm"
                variant="outline"
                onClick={addAddon}
                disabled={addonBusy || !addonPlanId}
              >
                <Plus className="h-3.5 w-3.5" />
                Add
              </Button>
            </div>
          </div>

          {/* Timeline — derived from the subscription's real dates
              (created_at / current_period_end); there is no per-subscription
              event history endpoint yet. */}
          <div className="border-t border-border pt-6">
            <h3 className="mb-4 text-sm font-semibold text-foreground">Timeline</h3>
            <ul className="-mb-6">
              <li>
                <div className="relative pb-6">
                  <span
                    className="absolute left-3 top-3 -ml-px h-full w-0.5 bg-border"
                    aria-hidden="true"
                  />
                  <div className="relative flex items-center gap-3">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-emerald-500 ring-4 ring-white">
                      <Check className="h-3.5 w-3.5 text-white" />
                    </span>
                    <div className="flex min-w-0 flex-1 justify-between gap-4">
                      <p className="text-sm text-foreground">Subscription created</p>
                      <time
                        dateTime={subscription.created_at}
                        className="whitespace-nowrap text-sm text-muted-foreground"
                      >
                        {formatDate(subscription.created_at)}
                      </time>
                    </div>
                  </div>
                </div>
              </li>
              <li>
                <div className="relative">
                  <div className="relative flex items-center gap-3">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-stone-400 ring-4 ring-white">
                      <RotateCw className="h-3.5 w-3.5 text-white" />
                    </span>
                    <div className="flex min-w-0 flex-1 justify-between gap-4">
                      <p className="text-sm text-foreground">Next renewal scheduled</p>
                      <time
                        dateTime={subscription.current_period_end}
                        className="whitespace-nowrap text-sm text-muted-foreground"
                      >
                        {formatDate(subscription.current_period_end)}
                      </time>
                    </div>
                  </div>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
