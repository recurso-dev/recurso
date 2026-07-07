import { useEffect, useState } from "react";
import { Pause, Play, Check, RotateCw, ArrowLeftRight } from "lucide-react";
import { toast } from "sonner";

import { endpoints } from "../../lib/api";
import { cn, formatCurrency, formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
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

  useEffect(() => {
    if (!changing || plans.length) return;
    endpoints
      .getPlans({ limit: 100 })
      .then((res) => setPlans(res.data?.data || []))
      .catch(() => toast.error("Failed to load plans"));
  }, [changing, plans.length]);

  useEffect(() => {
    if (!newPlanId || !subscription) return;
    setPreviewLoading(true);
    setPreview(null);
    endpoints
      .previewPlanChange(subscription.id, newPlanId)
      .then((res) => setPreview(res.data))
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

  const handlePause = async () => {
    if (!confirm("Are you sure you want to pause this subscription?")) return;
    setLoading(true);
    try {
      await endpoints.pauseSubscription(subscription.id);
      if (onRefresh) onRefresh();
    } catch (err) {
      alert("Failed to pause subscription");
    } finally {
      setLoading(false);
    }
  };

  const handleResume = async () => {
    if (!confirm("Are you sure you want to resume this subscription?")) return;
    setLoading(true);
    try {
      await endpoints.resumeSubscription(subscription.id);
      if (onRefresh) onRefresh();
    } catch (err) {
      alert("Failed to resume subscription");
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
                  isActive ? "bg-emerald-400" : "bg-zinc-400"
                )}
              />
              <span
                className={cn(
                  "relative inline-flex h-2 w-2 rounded-full",
                  isActive ? "bg-emerald-500" : "bg-zinc-500"
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
                onClick={handlePause}
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
                onClick={handleResume}
                disabled={loading}
                className="text-emerald-700 hover:text-emerald-800"
              >
                <Play className="h-3.5 w-3.5" />
                Resume
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              className="text-red-600 hover:text-red-700"
            >
              Cancel
            </Button>
            <Button size="sm">Renew</Button>
          </div>

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
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-zinc-400 ring-4 ring-white">
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
