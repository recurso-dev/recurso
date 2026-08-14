import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";

import { endpoints } from "../lib/api";
import { formatDateTime } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedRow,
  RelatedEmpty,
} from "@/components/patterns/ObjectPage";
import { AuditTrail } from "@/components/patterns/AuditTrail";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/ui/status-badge";
import { CopyableId } from "@/components/ui/copyable-id";

/**
 * MeterPage — a billable metric as a first-class object at
 * /billable-metrics/:id. Answers the meter questions: what it measures, how it
 * aggregates, which plans price on it (the reverse lookup), and the recent
 * events feeding it. Event → Meter → Aggregation → Pricing, made navigable.
 */
export default function MeterPage() {
  const { id } = useParams();

  const {
    data: metric,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["billable-metric", id],
    queryFn: async () => (await endpoints.getBillableMetric(id)).data.data,
    enabled: Boolean(id),
  });

  // Reverse lookup: which plans/charges price on this meter.
  const { data: charges = [] } = useQuery({
    queryKey: ["metric-charges", id],
    queryFn: async () => (await endpoints.getMetricCharges(id)).data.data || [],
    enabled: Boolean(id),
  });

  // Recent events feeding the meter (dimension == the metric's code).
  const { data: events = [] } = useQuery({
    queryKey: ["metric-events", metric?.code],
    queryFn: async () =>
      (await endpoints.getUsageEvents({ dimension: metric.code, limit: 20 })).data.data || [],
    enabled: Boolean(metric?.code),
  });

  if (isLoading) {
    return (
      <div aria-busy="true">
        <Skeleton className="mb-2 h-4 w-24" />
        <Skeleton className="mb-6 h-8 w-64" />
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <Skeleton className="h-64 lg:col-span-2" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (error || !metric) {
    const status = error?.response?.status;
    return (
      <ErrorState
        title={status === 404 ? "Meter not found" : "Couldn't load this meter"}
        message={
          status === 404
            ? "This billable metric doesn't exist or was removed."
            : error?.response?.data?.error?.message || error?.message
        }
        onRetry={status === 404 ? undefined : refetch}
      />
    );
  }

  return (
    <div>
      <ObjectHeader
        backTo="/metering"
        backLabel="Metering"
        kicker="Meter"
        title={metric.name}
        badge={<Badge variant="neutral" className="font-mono">{metric.aggregation_type}</Badge>}
        meta={
          <>
            <span className="font-mono text-xs">{metric.code}</span>
            <CopyableId value={metric.id} />
          </>
        }
      />

      <ObjectPageLayout
        rail={
          <>
            <ObjectSection title="Metadata">
              <AttributeList
                columns={1}
                items={[
                  { label: "Metric ID", value: <CopyableId value={metric.id} /> },
                  { label: "Code", value: <span className="font-mono text-sm">{metric.code}</span> },
                ]}
              />
            </ObjectSection>
            <ObjectSection title="Audit trail">
              <AuditTrail entityType="billable-metrics" entityId={metric.id} />
            </ObjectSection>
          </>
        }
      >
        <ObjectSection title="Definition">
          <AttributeList
            items={[
              { label: "Name", value: metric.name },
              { label: "Event dimension (code)", value: <span className="font-mono text-sm">{metric.code}</span> },
              {
                label: "Aggregation",
                value: (
                  <span>
                    <span className="font-medium">{metric.aggregation_type}</span>
                    {metric.field_name ? (
                      <span className="text-muted-foreground"> of {metric.field_name}</span>
                    ) : null}
                  </span>
                ),
              },
              {
                label: "Field",
                value: metric.field_name ? (
                  <span className="font-mono text-sm">{metric.field_name}</span>
                ) : null,
              },
            ]}
          />
          <p className="mt-4 text-xs text-muted-foreground">
            Each usage event with dimension{" "}
            <span className="font-mono">{metric.code}</span> feeds this meter; the meter reduces
            them to one quantity per period by{" "}
            <span className="font-medium">{metric.aggregation_type}</span>.
          </p>
        </ObjectSection>

        <ObjectSection title={`Plans pricing on this meter${charges.length ? ` (${charges.length})` : ""}`} flush>
          {charges.length === 0 ? (
            <RelatedEmpty>No plan prices on this meter yet.</RelatedEmpty>
          ) : (
            <div className="divide-y divide-border">
              {charges.map((c) => (
                <RelatedRow key={c.charge_id} to={`/plans/${c.plan_id}`}>
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="truncate font-medium text-foreground">{c.plan_name}</span>
                    {!c.plan_active && (
                      <Badge variant="neutral" className="text-[10px]">archived</Badge>
                    )}
                  </span>
                  <span className="flex shrink-0 items-center gap-3">
                    <span className="font-mono text-xs text-muted-foreground">{c.charge_model}</span>
                    {c.pay_in_advance && (
                      <Badge variant="neutral" className="text-[10px]">in advance</Badge>
                    )}
                    <StatusBadge status={c.plan_active ? "active" : "archived"} />
                  </span>
                </RelatedRow>
              ))}
            </div>
          )}
        </ObjectSection>

        <ObjectSection
          title={`Recent events${events.length ? ` (${events.length})` : ""}`}
          flush
          action={
            <Link
              to={`/usage?dimension=${metric.code}`}
              className="text-xs font-medium text-primary underline-offset-2 hover:underline"
            >
              Open in Usage Explorer
            </Link>
          }
        >
          {events.length === 0 ? (
            <RelatedEmpty>No usage events have fed this meter yet.</RelatedEmpty>
          ) : (
            <div className="divide-y divide-border">
              {events.map((e) => (
                <div key={e.id || e.transaction_id} className="flex items-center justify-between gap-3 px-6 py-2.5 text-sm">
                  <span className="text-muted-foreground">{formatDateTime(e.timestamp)}</span>
                  <span className="font-mono tabular-nums text-foreground">
                    {e.quantity?.toLocaleString?.() ?? e.quantity}
                  </span>
                </div>
              ))}
            </div>
          )}
        </ObjectSection>
      </ObjectPageLayout>
    </div>
  );
}
