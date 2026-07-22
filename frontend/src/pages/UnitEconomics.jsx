import { useQuery } from "@tanstack/react-query";
import { Gauge } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Card } from "@/components/ui/card";

export default function UnitEconomics() {
  const {
    data,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["unit-economics"],
    queryFn: async () => (await endpoints.getUnitEconomics()).data?.data || null,
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load unit economics"
    : null;

  const cur = data?.reporting_currency || "USD";
  const money = (n) => formatCurrency(n || 0, cur);
  const hasData = data && (data.mrr > 0 || data.active_customers > 0);

  return (
    <div>
      <PageHeader
        title="Unit Economics"
        description="Revenue per account and per subscription, plus lifetime value."
      />

      {loading ? (
        <CardGridSkeleton count={3} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={refetch} />
        </Card>
      ) : (
        data && (
          <div className="flex flex-col gap-6">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <StatCard
                label="ARPA"
                value={money(data.arpa)}
                hint="Avg. revenue per account (customer) / month"
              />
              <StatCard
                label="ARPU"
                value={money(data.arpu)}
                hint="Avg. revenue per subscription / month"
              />
              <StatCard
                label="LTV"
                value={data.has_ltv ? money(data.ltv) : "—"}
                hint={
                  data.has_ltv
                    ? `at ${(data.monthly_churn_rate || 0).toFixed(1)}% monthly churn`
                    : "Needs a few weeks of MRR history"
                }
              />
            </div>

            {!hasData ? (
              <Card className="overflow-hidden">
                <EmptyState
                  icon={Gauge}
                  title="No revenue yet"
                  description="Once you have active subscriptions, ARPA, ARPU and LTV appear here."
                />
              </Card>
            ) : (
              <Card className="p-6">
                <h2 className="mb-4 text-base font-semibold text-foreground">Basis</h2>
                <dl className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                  <Metric label="MRR" value={money(data.mrr)} />
                  <Metric label="Active customers" value={(data.active_customers || 0).toLocaleString()} />
                  <Metric label="Active subscriptions" value={(data.active_subscriptions || 0).toLocaleString()} />
                  <Metric
                    label="Monthly churn"
                    value={data.has_ltv ? `${(data.monthly_churn_rate || 0).toFixed(1)}%` : "—"}
                  />
                </dl>
                {!data.has_ltv && (
                  <p className="mt-4 text-xs text-muted-foreground">
                    LTV = ARPA ÷ monthly revenue churn. Churn is measured from captured MRR history,
                    which is still accruing — LTV will populate once there’s a trailing month to compare.
                  </p>
                )}
              </Card>
            )}
          </div>
        )
      )}
    </div>
  );
}

function Metric({ label, value }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="mt-1 font-mono text-lg font-semibold tabular-nums text-foreground">{value}</dd>
    </div>
  );
}
