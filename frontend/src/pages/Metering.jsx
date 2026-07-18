import { useEffect, useState } from "react";
import { Plus, Gauge, Trash2, BellRing } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

// Usage-based billing configuration: billable metrics and usage alerts.
// Plan charges are edited per plan from this page's charge editor.
const Metering = () => {
  const [metrics, setMetrics] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [actionError, setActionError] = useState(null);

  const [metricOpen, setMetricOpen] = useState(false);
  const [metricForm, setMetricForm] = useState({
    name: "",
    code: "",
    aggregation_type: "sum",
    field_name: "",
  });

  const [alertOpen, setAlertOpen] = useState(false);
  const [alertForm, setAlertForm] = useState({
    subscription_id: "",
    metric_code: "",
    threshold_type: "quantity",
    threshold: "",
  });

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
    try {
      const body = { ...metricForm };
      if (body.aggregation_type !== "unique") delete body.field_name;
      await api.createBillableMetric(body);
      setMetricOpen(false);
      setMetricForm({ name: "", code: "", aggregation_type: "sum", field_name: "" });
      fetchAll();
    } catch (err) {
      setActionError(err?.response?.data?.error?.message || "Failed to create metric");
    }
  };

  const removeMetric = async (metric) => {
    try {
      await api.deleteBillableMetric(metric.id);
      fetchAll();
    } catch (err) {
      setError(
        err?.response?.data?.error?.message ||
          "Delete failed — the metric may be referenced by a plan charge."
      );
    }
  };

  const submitAlert = async () => {
    setActionError(null);
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
        <Badge variant="neutral" className="font-mono">
          {m.aggregation_type}
          {m.field_name ? `(${m.field_name})` : ""}
        </Badge>
      ),
    },
    {
      key: "actions",
      header: "",
      cell: (m) => (
        <Button
          size="sm"
          variant="ghost"
          onClick={(e) => {
            e.stopPropagation();
            removeMetric(m);
          }}
        >
          <Trash2 className="h-4 w-4 text-muted-foreground" />
        </Button>
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
            "A metric's code doubles as the usage event dimension it aggregates (count, sum, max, unique).",
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
              <span className="ml-2 font-mono text-xs text-muted-foreground">
                sub {a.subscription_id.slice(0, 8)}…
              </span>
            </div>
            <div className="flex items-center gap-2">
              {a.last_fired_period_start && <Badge variant="success">fired this period</Badge>}
              <Button size="sm" variant="ghost" onClick={() => removeAlert(a)}>
                <Trash2 className="h-4 w-4 text-muted-foreground" />
              </Button>
            </div>
          </div>
        ))}
      </div>

      {/* New metric */}
      <Dialog open={metricOpen} onOpenChange={setMetricOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New billable metric</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
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
              <select
                className="w-full rounded-md border border-border bg-white px-3 py-2 text-sm"
                value={metricForm.aggregation_type}
                onChange={(e) => setMetricForm({ ...metricForm, aggregation_type: e.target.value })}
              >
                <option value="sum">sum — total quantity</option>
                <option value="count">count — number of events</option>
                <option value="max">max — largest event</option>
                <option value="unique">unique — distinct property values</option>
              </select>
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
            {actionError && <p className="text-sm text-red-600">{actionError}</p>}
          </div>
          <DialogFooter>
            <Button onClick={submitMetric} disabled={!metricForm.name || !metricForm.code}>
              Create metric
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* New alert */}
      <Dialog open={alertOpen} onOpenChange={setAlertOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New usage alert</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label>Subscription ID</Label>
              <Input
                value={alertForm.subscription_id}
                onChange={(e) => setAlertForm({ ...alertForm, subscription_id: e.target.value })}
                placeholder="uuid"
              />
            </div>
            <div>
              <Label>Metric</Label>
              <select
                className="w-full rounded-md border border-border bg-white px-3 py-2 text-sm"
                value={alertForm.metric_code}
                onChange={(e) => setAlertForm({ ...alertForm, metric_code: e.target.value })}
              >
                <option value="">Select a metric…</option>
                {metrics.map((m) => (
                  <option key={m.id} value={m.code}>
                    {m.name} ({m.code})
                  </option>
                ))}
              </select>
            </div>
            <div>
              <Label>Threshold type</Label>
              <select
                className="w-full rounded-md border border-border bg-white px-3 py-2 text-sm"
                value={alertForm.threshold_type}
                onChange={(e) => setAlertForm({ ...alertForm, threshold_type: e.target.value })}
              >
                <option value="quantity">Absolute quantity</option>
                <option value="percent_of_limit">Percent of entitlement limit</option>
              </select>
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
          <DialogFooter>
            <Button
              onClick={submitAlert}
              disabled={!alertForm.subscription_id || !alertForm.metric_code || !alertForm.threshold}
            >
              Create alert
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Metering;
