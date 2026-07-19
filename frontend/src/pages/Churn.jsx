import { useEffect, useState } from "react";
import { AlertTriangle, Check, ArrowUpRight } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { EmptyState } from "@/components/patterns/EmptyState";
import { DataTable } from "@/components/patterns/DataTable";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";

const riskVariant = (level) =>
  ({ critical: "destructive", high: "destructive", medium: "warning", low: "neutral" })[level] ||
  "neutral";

const shortId = (id) => (id ? String(id).slice(0, 8) : "—");
const fmtDate = (v) => (v ? new Date(v).toLocaleString() : "—");

const Churn = () => {
  const [alerts, setAlerts] = useState([]);
  const [alertsLoading, setAlertsLoading] = useState(true);
  const [highRisk, setHighRisk] = useState([]);
  const [hrLoading, setHrLoading] = useState(true);
  const [hrError, setHrError] = useState(null);
  const [acking, setAcking] = useState(null);
  const [customerNames, setCustomerNames] = useState({});

  const fetchAlerts = async () => {
    setAlertsLoading(true);
    try {
      const res = await api.getChurnAlerts();
      setAlerts(res.data.data || []);
    } catch {
      setAlerts([]);
    } finally {
      setAlertsLoading(false);
    }
  };

  const fetchHighRisk = async () => {
    setHrLoading(true);
    setHrError(null);
    try {
      const res = await api.getHighRiskCustomers();
      setHighRisk(res.data.data || []);
    } catch (err) {
      setHrError(err?.response?.data?.error?.message || "Failed to load high-risk customers");
    } finally {
      setHrLoading(false);
    }
  };

  useEffect(() => {
    fetchAlerts();
    fetchHighRisk();
    // Resolve customer ids to names (best-effort; ids remain the fallback).
    api
      .getCustomers({ limit: 1000 })
      .then((res) => {
        const names = {};
        (res?.data?.data || []).forEach((c) => {
          names[c.id] = c.name;
        });
        setCustomerNames(names);
      })
      .catch(() => {});
  }, []);

  const customerLabel = (id) =>
    customerNames[id] ? (
      <span className="text-sm text-foreground">{customerNames[id]}</span>
    ) : (
      <span className="font-mono text-xs text-muted-foreground">{shortId(id)}</span>
    );

  const acknowledge = async (id) => {
    setAcking(id);
    try {
      await api.acknowledgeChurnAlert(id);
      setAlerts((prev) => prev.filter((a) => a.id !== id));
      toast.success("Alert acknowledged.");
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to acknowledge");
    } finally {
      setAcking(null);
    }
  };

  const hrColumns = [
    {
      key: "customer_id",
      header: "Customer",
      cell: (r) => customerLabel(r.customer_id),
    },
    {
      key: "score",
      header: "Risk score",
      cell: (r) => <span className="tabular-nums font-medium text-foreground">{r.score}</span>,
    },
    {
      key: "risk_level",
      header: "Level",
      cell: (r) => <Badge variant={riskVariant(r.risk_level)}>{r.risk_level}</Badge>,
    },
    {
      key: "model_version",
      header: "Model",
      align: "right",
      cell: (r) => (
        <span className="text-xs text-muted-foreground">{r.model_version || "—"}</span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Churn risk"
        description="Customers whose churn score crossed the alert threshold, and everyone currently at high risk."
      />

      {/* Alerts */}
      <h2 className="mb-3 text-sm font-semibold text-foreground">Open alerts</h2>
      {alertsLoading ? (
        <CardGridSkeleton count={2} />
      ) : alerts.length === 0 ? (
        <EmptyState
          icon={AlertTriangle}
          title="No open churn alerts"
          description="You'll see an alert here when a customer's churn score spikes past the threshold."
        />
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {alerts.map((a) => (
            <Card key={a.id}>
              <CardContent className="flex flex-col gap-3 p-5">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <div className="text-xs">{customerLabel(a.customer_id)}</div>
                    <p className="mt-1 flex items-center gap-1.5 text-sm font-medium text-foreground">
                      <span className="tabular-nums">{a.previous_score}</span>
                      <ArrowUpRight className="h-4 w-4 text-red-600" />
                      <span className="tabular-nums text-red-600">{a.new_score}</span>
                    </p>
                  </div>
                  <Badge variant="destructive">{a.alert_type}</Badge>
                </div>
                <p className="text-xs text-muted-foreground">
                  Threshold {a.threshold} · {fmtDate(a.created_at)}
                </p>
                <Button
                  size="sm"
                  variant="outline"
                  className="self-start"
                  onClick={() => acknowledge(a.id)}
                  disabled={acking === a.id}
                >
                  <Check className="h-4 w-4" />
                  {acking === a.id ? "Acknowledging…" : "Acknowledge"}
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* High-risk customers */}
      <h2 className="mb-3 mt-8 text-sm font-semibold text-foreground">High-risk customers</h2>
      <DataTable
        columns={hrColumns}
        data={highRisk}
        loading={hrLoading}
        error={hrError}
        onRetry={fetchHighRisk}
        getRowId={(r) => r.customer_id}
        empty={{
          icon: AlertTriangle,
          title: "No high-risk customers",
          description: "Nobody is currently above the churn-risk threshold.",
        }}
      />
    </div>
  );
};

export default Churn;
