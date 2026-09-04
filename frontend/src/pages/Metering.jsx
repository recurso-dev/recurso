import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Gauge, Trash2, BellRing, Pencil } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { useCustomers, usePlans, useSubscriptions } from "@/lib/useCustomers";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormSheet } from "@/components/patterns/FormSheet";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useTableSort, sortRows } from "@/lib/tableSort";
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
  const queryClient = useQueryClient();
  const [pageError, setPageError] = useState(null); // delete-failure banner
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
  const [editAlert, setEditAlert] = useState(null); // the alert being edited, or null
  const [editAlertForm, setEditAlertForm] = useState({ threshold_type: "quantity", threshold: "" });
  const [alertForm, setAlertForm] = useState({
    subscription_id: "",
    metric_code: "",
    threshold_type: "quantity",
    threshold: "",
  });
  const [deleteTarget, setDeleteTarget] = useState(null);

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

  // Fully-loaded list (one fetch, no server paging), so sorting the complete
  // set client-side is honest; the sort persists in the URL (Batch F3).
  const { sort, onSortChange } = useTableSort();

  // Metrics + alerts load together; one cache entry for the page.
  const {
    data: metering,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["metering"],
    queryFn: async () => {
      const [m, a] = await Promise.all([api.getBillableMetrics(), api.getUsageAlerts()]);
      return { metrics: m.data.data || [], alerts: a.data.data || [] };
    },
  });
  const metrics = metering?.metrics ?? [];
  const alerts = metering?.alerts ?? [];
  // A failed delete surfaces in the same page-level banner as a failed load.
  const error =
    (queryError
      ? queryError?.response?.data?.error?.message ||
        queryError?.message ||
        "Failed to load metering"
      : null) || pageError;

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["metering"] });

  const metricMutation = useMutation({
    mutationFn: () => {
      const body = { ...metricForm };
      // field_name carries the counted property (unique) or the percentile
      // 1-99 (percentile); every other aggregation takes no field_name.
      if (body.aggregation_type !== "unique" && body.aggregation_type !== "percentile")
        delete body.field_name;
      // expression is only for the custom aggregation; the API rejects it
      // elsewhere.
      if (body.aggregation_type !== "custom") delete body.expression;
      return editingMetric
        ? api.updateBillableMetric(editingMetric.id, body)
        : api.createBillableMetric(body);
    },
    onSuccess: () => {
      setMetricOpen(false);
      setEditingMetric(null);
      setMetricForm({ name: "", code: "", aggregation_type: "sum", field_name: "", expression: "" });
      invalidate();
    },
    onError: (err) =>
      setActionError(
        err?.response?.data?.error?.message ||
          (editingMetric ? "Failed to update metric" : "Failed to create metric")
      ),
  });
  const submitMetric = () => {
    setActionError(null);
    metricMutation.mutate();
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

  const deleteMetricMutation = useMutation({
    mutationFn: () => api.deleteBillableMetric(deleteTarget.id),
    onSuccess: () => {
      setDeleteTarget(null);
      invalidate();
    },
    onError: (err) => {
      setDeleteTarget(null);
      setPageError(
        err?.response?.data?.error?.message ||
          "Delete failed — the metric may be referenced by a plan charge."
      );
    },
  });
  const deleting = deleteMetricMutation.isPending;
  const removeMetric = () => {
    if (!deleteTarget) return;
    setPageError(null);
    deleteMetricMutation.mutate();
  };

  const alertMutation = useMutation({
    mutationFn: () =>
      api.createUsageAlert({ ...alertForm, threshold: parseInt(alertForm.threshold, 10) }),
    onSuccess: () => {
      setAlertOpen(false);
      setAlertForm({ subscription_id: "", metric_code: "", threshold_type: "quantity", threshold: "" });
      invalidate();
    },
    onError: (err) =>
      setActionError(err?.response?.data?.error?.message || "Failed to create alert"),
  });
  const submitAlert = () => {
    setActionError(null);
    alertMutation.mutate();
  };

  const removeAlertMutation = useMutation({
    mutationFn: (alert) => api.deleteUsageAlert(alert.id),
    onSettled: invalidate, // refetch shows state either way
  });
  const removeAlert = (alert) => removeAlertMutation.mutate(alert);

  const openEditAlert = (alert) => {
    setActionError(null);
    setEditAlert(alert);
    setEditAlertForm({
      threshold_type: alert.threshold_type,
      threshold: String(alert.threshold),
    });
  };

  const editAlertMutation = useMutation({
    mutationFn: () =>
      api.updateUsageAlert(editAlert.id, {
        threshold_type: editAlertForm.threshold_type,
        threshold: parseInt(editAlertForm.threshold, 10),
      }),
    onSuccess: () => {
      setEditAlert(null);
      invalidate();
    },
    onError: (err) =>
      setActionError(err?.response?.data?.error?.message || "Failed to update alert"),
  });
  const submitEditAlert = () => {
    setActionError(null);
    editAlertMutation.mutate();
  };

  const saving =
    metricMutation.isPending || alertMutation.isPending || editAlertMutation.isPending;

  const metricColumns = [
    {
      key: "name",
      header: "Metric",
      sortable: true,
      sortValue: (m) => m.name,
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
            // Truncated in the cell; the full expression opens on hover AND on
            // keyboard focus (tabIndex) — never a hover-only native title.
            <TooltipProvider delayDuration={150}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <code
                    tabIndex={0}
                    className="max-w-[16rem] truncate rounded text-xs text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {m.expression}
                  </code>
                </TooltipTrigger>
                <TooltipContent className="max-w-sm break-all font-mono">{m.expression}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
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
        data={sortRows(metrics, sort, metricColumns)}
        sort={sort}
        onSortChange={onSortChange}
        rowHref={(m) => `/billable-metrics/${m.id}`}
        loading={loading}
        error={error}
        onRetry={refetch}
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
              <Button size="sm" variant="ghost" aria-label="Edit alert" onClick={() => openEditAlert(a)}>
                <Pencil className="h-4 w-4 text-muted-foreground" />
              </Button>
              <Button size="sm" variant="ghost" aria-label="Delete alert" onClick={() => removeAlert(a)}>
                <Trash2 className="h-4 w-4 text-muted-foreground" />
              </Button>
            </div>
          </div>
        ))}
      </div>

      {/* New / edit metric */}
      <FormSheet
        open={metricOpen}
        onOpenChange={(o) => {
          setMetricOpen(o);
          if (!o) {
            setEditingMetric(null);
            setMetricForm({ name: "", code: "", aggregation_type: "sum", field_name: "", expression: "" });
          }
        }}
        title={editingMetric ? "Edit billable metric" : "New billable metric"}
        description="A metric's code doubles as the usage-event dimension it aggregates."
        onSubmit={submitMetric}
        submitLabel={editingMetric ? "Save changes" : "Create metric"}
        busyLabel="Saving…"
        busy={saving}
        canSubmit={Boolean(metricForm.name && metricForm.code)}
        dirty={
          editingMetric
            ? metricForm.name !== (editingMetric.name || "") ||
              metricForm.aggregation_type !== (editingMetric.aggregation_type || "sum") ||
              metricForm.field_name !== (editingMetric.field_name || "") ||
              metricForm.expression !== (editingMetric.expression || "")
            : Boolean(metricForm.name || metricForm.code)
        }
        error={actionError}
      >
            <div>
              <Label htmlFor="name">Name</Label>
              <Input id="name"
                value={metricForm.name}
                onChange={(e) => setMetricForm({ ...metricForm, name: e.target.value })}
                placeholder="API calls"
              />
            </div>
            <div>
              <Label htmlFor="code-event-dimension-immutable">Code (= event dimension, immutable)</Label>
              <Input id="code-event-dimension-immutable"
                value={metricForm.code}
                onChange={(e) => setMetricForm({ ...metricForm, code: e.target.value })}
                placeholder="api_calls"
              />
            </div>
            <div>
              <Label htmlFor="aggregation">Aggregation</Label>
              <Select
                value={metricForm.aggregation_type}
                onValueChange={(v) => setMetricForm({ ...metricForm, aggregation_type: v })}
              >
                <SelectTrigger id="aggregation">
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
                <Label htmlFor="property-to-count-field-name">Property to count (field_name)</Label>
                <Input id="property-to-count-field-name"
                  value={metricForm.field_name}
                  onChange={(e) => setMetricForm({ ...metricForm, field_name: e.target.value })}
                  placeholder="user_id"
                />
              </div>
            )}
            {metricForm.aggregation_type === "percentile" && (
              <div>
                <Label htmlFor="percentile-1-99">Percentile (1–99)</Label>
                <Input id="percentile-1-99"
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
                <Label htmlFor="expression">Expression</Label>
                <Input id="expression"
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
      </FormSheet>

      {/* New alert */}
      <FormSheet
        open={alertOpen}
        onOpenChange={setAlertOpen}
        title="New usage alert"
        description="Fires once per billing period via webhook and email when the threshold is crossed."
        onSubmit={submitAlert}
        submitLabel="Create alert"
        busyLabel="Creating…"
        busy={saving}
        canSubmit={Boolean(alertForm.subscription_id && alertForm.metric_code && alertForm.threshold)}
        dirty={Boolean(alertForm.subscription_id || alertForm.metric_code || alertForm.threshold)}
        error={actionError}
      >
            <div>
              <Label htmlFor="subscription">Subscription</Label>
              <Select
                value={alertForm.subscription_id}
                onValueChange={(v) => setAlertForm({ ...alertForm, subscription_id: v })}
              >
                <SelectTrigger id="subscription">
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
              <Label htmlFor="metric">Metric</Label>
              <Select
                value={alertForm.metric_code}
                onValueChange={(v) => setAlertForm({ ...alertForm, metric_code: v })}
              >
                <SelectTrigger id="metric">
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
              <Label htmlFor="threshold-type">Threshold type</Label>
              <Select
                value={alertForm.threshold_type}
                onValueChange={(v) => setAlertForm({ ...alertForm, threshold_type: v })}
              >
                <SelectTrigger id="threshold-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="quantity">Absolute quantity</SelectItem>
                  <SelectItem value="percent_of_limit">Percent of entitlement limit</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label htmlFor="threshold">
                Threshold{alertForm.threshold_type === "percent_of_limit" ? " (%)" : ""}
              </Label>
              <Input id="threshold"
                type="number"
                min="1"
                value={alertForm.threshold}
                onChange={(e) => setAlertForm({ ...alertForm, threshold: e.target.value })}
              />
            </div>
      </FormSheet>

      {/* Edit alert — threshold only; subscription + metric are the alert's identity */}
      <FormSheet
        open={!!editAlert}
        onOpenChange={(o) => !o && setEditAlert(null)}
        title="Edit usage alert"
        description="Re-aim the threshold. Editing lets the alert fire again this period against the new line. To change the subscription or metric, delete and re-create."
        onSubmit={submitEditAlert}
        submitLabel="Save changes"
        busyLabel="Saving…"
        busy={saving}
        canSubmit={Boolean(editAlertForm.threshold)}
        dirty={
          Boolean(editAlert) &&
          (String(editAlertForm.threshold) !== String(editAlert?.threshold ?? "") ||
            editAlertForm.threshold_type !== (editAlert?.threshold_type || "quantity"))
        }
        error={actionError}
      >
            <div className="rounded-md bg-muted/40 p-3 text-sm">
              <span className="font-mono">{editAlert?.metric_code}</span>
              <span className="ml-2 text-xs text-muted-foreground">
                {(() => {
                  const s = subscriptions.find((x) => x.id === editAlert?.subscription_id);
                  return s ? subLabel(s) : editAlert ? `sub ${editAlert.subscription_id.slice(0, 8)}…` : "";
                })()}
              </span>
            </div>
            <div>
              <Label htmlFor="threshold-type-2">Threshold type</Label>
              <Select
                value={editAlertForm.threshold_type}
                onValueChange={(v) => setEditAlertForm({ ...editAlertForm, threshold_type: v })}
              >
                <SelectTrigger id="threshold-type-2">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="quantity">Absolute quantity</SelectItem>
                  <SelectItem value="percent_of_limit">Percent of entitlement limit</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label htmlFor="threshold-2">
                Threshold{editAlertForm.threshold_type === "percent_of_limit" ? " (%)" : ""}
              </Label>
              <Input id="threshold-2"
                type="number"
                min="1"
                value={editAlertForm.threshold}
                onChange={(e) => setEditAlertForm({ ...editAlertForm, threshold: e.target.value })}
              />
            </div>
      </FormSheet>

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
