import { shortId, formatDateTime } from "@/lib/utils";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Check, ArrowUpRight } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { EmptyState } from "@/components/patterns/EmptyState";
import { DataTable } from "@/components/patterns/DataTable";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/ui/status-badge";
import { Card, CardContent } from "@/components/ui/card";

const fmtDate = (v) => formatDateTime(v);

const errMsg = (err, fallback) =>
  err ? err?.response?.data?.error?.message || err?.message || fallback : null;

const Churn = () => {
  const queryClient = useQueryClient();
  // Shared cached customer directory (ADR-005) — ids stay the fallback label.
  const { names: customerNames } = useCustomers();

  const {
    data: alerts = [],
    isLoading: alertsLoading,
    error: alertsQueryError,
    refetch: refetchAlerts,
  } = useQuery({
    queryKey: ["churn-alerts"],
    queryFn: async () => (await api.getChurnAlerts()).data.data || [],
  });
  const alertsError = errMsg(alertsQueryError, "Failed to load churn alerts");

  const {
    data: highRisk = [],
    isLoading: hrLoading,
    error: hrQueryError,
    refetch: refetchHighRisk,
  } = useQuery({
    queryKey: ["high-risk-customers"],
    queryFn: async () => (await api.getHighRiskCustomers()).data.data || [],
  });
  const hrError = errMsg(hrQueryError, "Failed to load high-risk customers");

  const customerLabel = (id) =>
    customerNames[id] ? (
      <span className="text-sm text-foreground">{customerNames[id]}</span>
    ) : (
      <span className="font-mono text-xs text-muted-foreground">{shortId(id)}</span>
    );

  const ackMutation = useMutation({
    mutationFn: (id) => api.acknowledgeChurnAlert(id),
    onSuccess: (_res, id) => {
      // Drop the acknowledged alert from the cache immediately; the Dashboard's
      // needs-attention count reads the same feed, so refresh it too.
      queryClient.setQueryData(["churn-alerts"], (prev) =>
        (prev || []).filter((a) => a.id !== id)
      );
      queryClient.invalidateQueries({ queryKey: ["dashboard-overview"] });
      toast.success("Alert acknowledged.");
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to acknowledge"),
  });
  const acking = ackMutation.isPending ? ackMutation.variables : null;

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
      cell: (r) => <StatusBadge status={r.risk_level} />,
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
      ) : alertsError ? (
        <p className="rounded-md bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {alertsError}{" "}
          <button className="underline" onClick={() => refetchAlerts()}>
            Retry
          </button>
        </p>
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
                      <ArrowUpRight className="h-4 w-4 text-destructive" />
                      <span className="tabular-nums text-destructive">{a.new_score}</span>
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
                  onClick={() => ackMutation.mutate(a.id)}
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
        onRetry={refetchHighRisk}
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
