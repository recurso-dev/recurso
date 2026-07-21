import { useEffect, useState } from "react";
import { Plus, Gauge, Trash2, BellRing, Pencil } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { useCustomers, usePlans, useSubscriptions } from "@/lib/useCustomers";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Usage-based billing configuration: billable metrics and usage alerts.
// Plan charges are edited per plan from this page's charge editor.
const Metering = () => {
  const [metrics, setMetrics] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [actionError, setActionError] = useState(null);

  const [metricOpen, setMetricOpen] = useState(false);
  const [editingMetric, setEditingMetric] = useState(null);
  const [metricForm, setMetricForm] = useState({
    name: "",
    code: "",
    aggregation_type: "sum",
    field_name: "",
    expression: "",
  });

  const [alertOpen, setAlertOpen] = useState(false);
  const [alertForm, setAlertForm] = useState({
    subscription_id: "",
    metric_code: "",
    threshold_type: "quantity",
    threshold: "",
  });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [deleting, setDeleting] = useState(false);

  // Subscriptions + names label the alert dialog's picker (replaces the old
  // paste-a-UUID input); all three lists come from the shared query cache.
  const { names: customerNames } = useCustomers();
  const subscriptions = useSubscriptions();
  const { names: planNames } = usePlans();

  const subLabel = (s) => {
    const cust = customerNames[s.customer_id] || `${String(s.customer_id).slice(0, 8)}…`;
    const plan = planNames[s.plan_id] || `${String(s.id).slice(0, 8)}…`;
    return `${cust} — ${plan}`;
  };

  const fetchAll = async () => {
    setLoading(true);
    setError(null);
    try {
      const [m, a] = await Promise.all([api.getBillableMetrics(), api.getUsageAlerts()]);
      setMetrics(m.data.data || []);
      setAlerts(a.data.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || err?.message || "Failed to load metering");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAll();
  }, []);

  const submitMetric = async () => {
    setActionError(null);
    setSaving(true);
    try {
      const body = { ...metricForm };
      // field_name carries the counted property (unique) or the percentile
      // 1-99 (percentile); every other aggregation takes no field_name.
      if (body.aggregation_type !== "unique" && body.aggregation_type !== "percentile")
        delete body.field_name;
      // expression is only for the custom aggregation; the API rejects it
      // elsewhere.
      if (body.aggregation_type !== "custom") delete body.expression;
      if (editingMetric) {
        await api.updateBillableMetric(editingMetric.id, body);
      } else {
        await api.createBillableMetric(body);
      }
      setMetricOpen(false);
      setEditingMetric(null);
      setMetricForm({ name: "", code: "", aggregation_type: "sum", field_name: "", expression: "" });
      fetchAll();
    } catch (err) {
      setActionError(
        err?.response?.data?.error?.message ||
          (editingMetric ? "Failed to update metric" : "Failed to create metric")
      );
    } finally {
      setSaving(false);
    }
  };

  const startEditMetric = (metric) => {
    setEditingMetric(metric);
    setMetricForm({
      name: metric.name || "",
      code: metric.code || "",
      aggregation_type: metric.aggregation_type || "sum",
      field_name: metric.field_name || "",
      expression: metric.expression || "",
    });
    setActionError(null);
    setMetricOpen(true);
  };

  const removeMetric = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await api.deleteBillableMetric(deleteTarget.id);
      setDeleteTarget(null);
      fetchAll();
    } catch (err) {
      setDeleteTarget(null);
      setError(
        err?.response?.data?.error?.message ||
          "Delete failed — the metric may be referenced by a plan charge."
      );
    } finally {
      setDeleting(false);
    }
  };

  const submitAlert = async () => {
    setActionError(null);
    setSaving(true);
    try {
      await api.createUsageAlert({
        ...alertForm,
        threshold: parseInt(alertForm.threshold, 10),
      });
      setAlertOpen(false);
      setAlertForm({ subscription_id: "", metric_code: "", threshold_type: "quantity", threshold: "" });
      fetchAll();
    } catch (err) {
      setActionError(err?.response?.data?.error?.message || "Failed to create alert");
    } finally {
      setSaving(false);
    }
  };

  const removeAlert = async (alert) => {
    try {
      await api.deleteUsageAlert(alert.id);
      fetchAll();
    } catch {
      /* refetch shows state */
    }
  };

  const metricColumns = [
    {
      key: "name",
      header: "Metric",
      cell: (m) => (
        <div>
          <div className="font-medium text-foreground">{m.name}</div>
          <div className="font-mono text-xs text-muted-foreground">{m.code}</div>
        </div>
      ),
    },
    {
      key: "aggregation",
      header: "Aggregation",
      cell: (m) => (
        <div className="flex flex-col gap-1">
          <Badge variant="neutral" className="w-fit font-mono">
            {m.aggregation_type}
            {m.field_name ? `(${m.field_name})` : ""}
          </Badge>
          {m.expression ? (
            <code className="max-w-[16rem] truncate text-xs text-muted-foreground" title={m.expression}>
              {m.expression}
            </code>
          ) : null}
        </div>
      ),
    },
    {
      key: "actions",
      header: "",
      cell: (m) => (
        <div className="flex justify-end gap-1">
          <Button
            size="sm"
            variant="ghost"
            aria-label={`Edit metric ${m.name}`}
            onClick={(e) => {
              e.stopPropagation();
              startEditMetric(m);
            }}
          >
            <Pencil className="h-4 w-4 text-muted-foreground" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            aria-label={`Delete metric ${m.name}`}
            onClick={(e) => {
              e.stopPropagation();
              setDeleteTarget(m);
            }}
          >
            <Trash2 className="h-4 w-4 text-muted-foreground" />
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Metering"
        description="Billable metrics aggregate usage events; charges on plans price them; alerts watch thresholds."
        actions={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setAlertOpen(true)}>
              <BellRing className="h-4 w-4" />
              New alert
            </Button>
            <Button onClick={() => setMetricOpen(true)}>
              <Plus className="h-4 w-4" />
              New metric
            </Button>
          </div>
        }
      />

      <DataTable
        columns={metricColumns}
        data={metrics}
        loading={loading}
        error={error}
        onRetry={fetchAll}
        empty={{
          icon: Gauge,
          title: "No billable metrics yet",
          description:
            "A metric's code doubles as the usage event dimension it aggregates (count, sum, max, unique, latest, percentile, weighted_sum, custom).",
          action: (
            <Button onClick={() => setMetricOpen(true)}>
              <Plus className="h-4 w-4" />
              New metric
            </Button>
          ),
        }}
      />

      <h2 className="mb-2 mt-8 text-sm font-semibold text-foreground">Usage alerts</h2>
      <div className="rounded-lg border border-border bg-white">
        {alerts.length === 0 && (
          <p className="p-4 text-sm text-muted-foreground">
            No alerts configured. Alerts fire once per billing period via the
            usage.alert.triggered webhook plus an email.
          </p>
        )}
        {alerts.map((a) => (
          <div
            key={a.id}
            className="flex items-center justify-between border-b border-border p-3 last:border-0"
          >
            <div className="text-sm">
              <span className="font-mono">{a.metric_code}</span>{" "}
              <span className="text-muted-foreground">
                {a.threshold_type === "quantity"
                  ? `≥ ${a.threshold.toLocaleString()}`
                  : `≥ ${a.threshold}% of limit`}
              </span>
              <span className="ml-2 text-xs text-muted-foreground">
                {(() => {
                  const s = subscriptions.find((x) => x.id === a.subscription_id);
                  return s ? subLabel(s) : `sub ${a.subscription_id.slice(0, 8)}…`;
                })()}
              </span>
            </div>
            <div className="flex items-center gap-2">
              {a.last_fired_period_start && <Badge variant="success">fired this period</Badge>}
              <Button size="sm" variant="ghost" aria-label="Delete alert" onClick={() => removeAlert(a)}>
                <Trash2 className="h-4 w-4 text-muted-foreground" />
              </Button>
            </div>
          </div>
        ))}
      </div>

      {/* New / edit metric */}
      <Sheet
        open={metricOpen}
        onOpenChange={(o) => {
          setMetricOpen(o);
          if (!o) {
            setEditingMetric(null);
            setMetricForm({ name: "", code: "", aggregation_type: "sum", field_name: "", expression: "" });
          }
        }}
      >
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>{editingMetric ? "Edit billable metric" : "New billable metric"}</SheetTitle>
            <SheetDescription>
              A metric&apos;s code doubles as the usage-event dimension it aggregates.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 overflow-y-auto px-6">
            <div>
              <Label>Name</Label>
              <Input
                value={metricForm.name}
                onChange={(e) => setMetricForm({ ...metricForm, name: e.target.value })}
                placeholder="API calls"
              />
            </div>
            <div>
              <Label>Code (= event dimension, immutable)</Label>
              <Input
                value={metricForm.code}
                onChange={(e) => setMetricForm({ ...metricForm, code: e.target.value })}
                placeholder="api_calls"
              />
            </div>
            <div>
              <Label>Aggregation</Label>
              <Select
                value={metricForm.aggregation_type}
                onValueChange={(v) => setMetricForm({ ...metricForm, aggregation_type: v })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="sum">sum — total quantity</SelectItem>
                  <SelectItem value="count">count — number of events</SelectItem>
                  <SelectItem value="max">max — largest event</SelectItem>
                  <SelectItem value="unique">unique — distinct property values</SelectItem>
                  <SelectItem value="latest">latest — most recent event</SelectItem>
                  <SelectItem value="percentile">percentile — p-th percentile (e.g. p95)</SelectItem>
                  <SelectItem value="weighted_sum">weighted_sum — time-weighted average (per-time resources)</SelectItem>
                  <SelectItem value="custom">custom — expression over each event</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {metricForm.aggregation_type === "unique" && (
              <div>
                <Label>Property to count (field_name)</Label>
                <Input
                  value={metricForm.field_name}
                  onChange={(e) => setMetricForm({ ...metricForm, field_name: e.target.value })}
                  placeholder="user_id"
                />
              </div>
            )}
            {metricForm.aggregation_type === "percentile" && (
              <div>
                <Label>Percentile (1–99)</Label>
                <Input
                  value={metricForm.field_name}
                  onChange={(e) => setMetricForm({ ...metricForm, field_name: e.target.value })}
                  inputMode="numeric"
                  placeholder="95"
                />
                <p className="mt-1 text-xs text-muted-foreground">
                  The percentile of event quantities to bill (e.g. 95 for p95).
                </p>
              </div>
            )}
            {metricForm.aggregation_type === "custom" && (
              <div>
                <Label>Expression</Label>
                <Input
                  value={metricForm.expression}
                  onChange={(e) => setMetricForm({ ...metricForm, expression: e.target.value })}
                  placeholder="quantity * properties.multiplier"
                  className="font-mono"
                />
                <p className="mt-1 text-xs text-muted-foreground">
                  Evaluated per event, then summed over the period. Reads{" "}
                  <code>quantity</code> and numeric <code>properties.*</code>{" "}
                  (e.g. <code>properties.bytes / 1000000</code>). Arithmetic only —
                  no functions or external access.
                </p>
              </div>
            )}
            {metricForm.aggregation_type === "weighted_sum" && (
              <p className="text-xs text-muted-foreground">
                Each event&apos;s quantity is a signed change to a running level
                (e.g. <code>+5</code> / <code>-2</code> seats); the metric bills the
                time-weighted average level over the period. The level carries
                forward from before the period, so a resource already active at
                period start is counted from the start.
              </p>
            )}
            {actionError && <p className="text-sm text-red-600">{actionError}</p>}
          </div>
          <SheetFooter>
            <Button
              onClick={submitMetric}
              disabled={saving || !metricForm.name || !metricForm.code}
            >
              {saving ? "Saving…" : editingMetric ? "Save changes" : "Create metric"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      {/* New alert */}
      <Sheet open={alertOpen} onOpenChange={setAlertOpen}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>New usage alert</SheetTitle>
            <SheetDescription>
              Fires once per billing period via webhook and email when the threshold is crossed.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 overflow-y-auto px-6">
            <div>
              <Label>Subscription</Label>
              <Select
                value={alertForm.subscription_id}
                onValueChange={(v) => setAlertForm({ ...alertForm, subscription_id: v })}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={
                      subscriptions.length === 0 ? "No subscriptions" : "Select a subscription"
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  {subscriptions
                    .filter((s) => s.status !== "canceled")
                    .map((s) => (
                      <SelectItem key={s.id} value={s.id}>
                        {subLabel(s)}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Metric</Label>
              <Select
                value={alertForm.metric_code}
                onValueChange={(v) => setAlertForm({ ...alertForm, metric_code: v })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a metric" />
                </SelectTrigger>
                <SelectContent>
                  {metrics.map((m) => (
                    <SelectItem key={m.id} value={m.code}>
                      {m.name} ({m.code})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Threshold type</Label>
              <Select
                value={alertForm.threshold_type}
                onValueChange={(v) => setAlertForm({ ...alertForm, threshold_type: v })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="quantity">Absolute quantity</SelectItem>
                  <SelectItem value="percent_of_limit">Percent of entitlement limit</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>
                Threshold{alertForm.threshold_type === "percent_of_limit" ? " (%)" : ""}
              </Label>
              <Input
                type="number"
                min="1"
                value={alertForm.threshold}
                onChange={(e) => setAlertForm({ ...alertForm, threshold: e.target.value })}
              />
            </div>
            {actionError && <p className="text-sm text-red-600">{actionError}</p>}
          </div>
          <SheetFooter>
            <Button
              onClick={submitAlert}
              disabled={
                saving || !alertForm.subscription_id || !alertForm.metric_code || !alertForm.threshold
              }
            >
              {saving ? "Creating…" : "Create alert"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title={`Delete metric ${deleteTarget?.name}?`}
        description="Usage already recorded is kept, but plans charging this metric will stop rating new events. This cannot be undone."
        confirmLabel="Delete metric"
        destructive
        busy={deleting}
        onConfirm={removeMetric}
      />
    </div>
  );
};

export default Metering;
