import { shortId } from "@/lib/utils";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { BarChart } from "@tremor/react";
import { Activity, Download, Gauge, Layers, RefreshCw, Users } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { makeChartTooltip, chartCategoryColors, chartDefaults } from "@/components/charts/ChartTooltip";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import { useCustomers } from "@/lib/useCustomers";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { ErrorState } from "@/components/patterns/ErrorState";
import { DataTable } from "@/components/patterns/DataTable";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Axis and tooltip share one formatter so they read identically.
const unitsFormatter = (v) => v.toLocaleString();
const unitsTooltip = makeChartTooltip(unitsFormatter);

export default function Usage() {
  const { names: customerNames } = useCustomers();
  const [eventDimension, setEventDimension] = useState("all");

  // Raw ingestion stream — answers "did my usage events actually land?".
  const {
    data: events = [],
    isLoading: eventsLoading,
    error: eventsQueryError,
    refetch: refetchEvents,
  } = useQuery({
    queryKey: ["usage-events", eventDimension],
    queryFn: async () => {
      const params = { limit: 50 };
      if (eventDimension !== "all") params.dimension = eventDimension;
      return (await api.getUsageEvents(params))?.data?.data || [];
    },
  });
  const eventsError = eventsQueryError
    ? eventsQueryError?.response?.data?.error?.message ||
      eventsQueryError?.message ||
      "Failed to load events"
    : null;

  const eventDimensions = useMemo(
    () => [...new Set(events.map((e) => e.dimension))].sort(),
    [events]
  );

  const eventColumns = [
    {
      key: "timestamp",
      header: "When",
      cell: (e) => (
        <span className="text-sm text-muted-foreground">
          {new Date(e.timestamp).toLocaleString()}
        </span>
      ),
    },
    {
      key: "customer",
      header: "Customer",
      cell: (e) => <CustomerName id={e.customer_id} names={customerNames} />,
    },
    {
      key: "dimension",
      header: "Dimension",
      cell: (e) => <Badge variant="neutral" className="font-mono">{e.dimension}</Badge>,
    },
    {
      key: "quantity",
      header: "Quantity",
      align: "right",
      cell: (e) => <span className="tabular-nums font-medium">{e.quantity.toLocaleString()}</span>,
    },
    {
      key: "txn",
      header: "Idempotency key",
      cell: (e) => (
        <span className="font-mono text-xs text-muted-foreground">{e.transaction_id || "—"}</span>
      ),
    },
  ];

  // Stats carry two values (per-dimension rows + the metered-customer count),
  // so the query returns both as one cached object.
  const {
    data: statsData,
    isLoading: loading,
    error: statsQueryError,
    refetch: refetchUsage,
  } = useQuery({
    queryKey: ["usage-stats"],
    queryFn: async () => {
      const response = await api.getUsageStats();
      return {
        stats: response.data.data || [],
        meteredCount: response.data.customers_metered ?? null,
      };
    },
  });
  const usageStats = statsData?.stats ?? [];
  const meteredCount = statsData?.meteredCount ?? null;
  const statsError = statsQueryError
    ? statsQueryError?.response?.data?.error?.message ||
      statsQueryError?.message ||
      "Failed to load usage metering"
    : null;


  const filteredData = usageStats;

  const totalUnits = useMemo(
    () => filteredData.reduce((acc, curr) => acc + curr.total_quantity, 0),
    [filteredData]
  );

  // Aggregate per dimension for the chart (stats are lifetime aggregates,
  // so a per-dimension breakdown is the honest visualization).
  const byDimension = useMemo(
    () =>
      Object.values(
        filteredData.reduce((acc, curr) => {
          const key = curr.dimension || "unknown";
          acc[key] = acc[key] || { dimension: key, Units: 0 };
          acc[key].Units += curr.total_quantity;
          return acc;
        }, {})
      ),
    [filteredData]
  );

  const customersMetered = useMemo(
    () => [...new Set(filteredData.map((d) => d.customer_id))].length,
    [filteredData]
  );
  const activeDimensions = useMemo(
    () => [...new Set(filteredData.map((d) => d.dimension))].length,
    [filteredData]
  );

  const exportCsv = () => {
    const rows = [
      ["customer_id", "plan_id", "dimension", "total_quantity"],
      ...filteredData.map((d) => [
        d.customer_id,
        d.plan_id,
        d.dimension,
        d.total_quantity,
      ]),
    ];
    const csv = rows.map((r) => r.join(",")).join("\n");
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = "usage-export.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  const columns = [
    {
      key: "customer",
      header: "Customer",
      cell: (item) => (
        <span className="font-medium text-foreground">
          {shortId(item.customer_id)}
        </span>
      ),
    },
    {
      key: "plan",
      header: "Plan",
      cell: (item) => (
        <span className="text-muted-foreground">
          {item.plan_id ? "Active Plan" : "-"}
        </span>
      ),
    },
    {
      key: "metric",
      header: "Metric",
      cell: (item) => (
        <span className="text-muted-foreground">{item.dimension}</span>
      ),
    },
    {
      key: "usage",
      header: "Usage",
      align: "right",
      cell: (item) => (
        <span className="tabular-nums text-foreground">
          {item.total_quantity}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: () => <Badge variant="success">Recorded</Badge>,
    },
    {
      key: "timestamp",
      header: "Timestamp",
      // Timestamp is not present in the lifetime aggregate.
      cell: () => <span className="text-muted-foreground">Recently</span>,
    },
  ];

  return (
    <div>
      <PageHeader
        title="Usage Explorer"
        description="Metered usage aggregated by customer, plan, and dimension."
        actions={
          <Button variant="outline" onClick={exportCsv}>
            <Download className="h-4 w-4" />
            Export data
          </Button>
        }
      />

      {statsError && !loading ? (
        <ErrorState
          title="Unable to load usage metering"
          message={statsError}
          onRetry={refetchUsage}
        />
      ) : (
      <>
      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <StatCard
          label="Total Units Consumed"
          value={totalUnits.toLocaleString()}
          icon={Gauge}
          hint="Lifetime"
        />
        <StatCard
          label="Customers Metered"
          value={(meteredCount ?? customersMetered).toLocaleString()}
          icon={Users}
          hint="With recorded usage"
        />
        <StatCard
          label="Active Dimensions"
          value={activeDimensions.toLocaleString()}
          icon={Layers}
          hint="Metric types"
        />
      </div>

      {/* Chart */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Usage by dimension</CardTitle>
          <p className="text-sm text-muted-foreground">
            {totalUnits.toLocaleString()} units · all recorded usage
          </p>
        </CardHeader>
        <CardContent>
          {loading ? (
            <Skeleton className="h-72 w-full" />
          ) : byDimension.length > 0 ? (
            <div role="img" aria-label="Usage quantity over time">
            <BarChart
              {...chartDefaults}
              className="h-72"
              data={byDimension}
              index="dimension"
              categories={["Units"]}
              colors={chartCategoryColors}
              valueFormatter={unitsFormatter}
              customTooltip={unitsTooltip}
              showLegend={false}
              showGridLines
              yAxisWidth={64}
            />
            </div>
          ) : (
            <EmptyState
              icon={Activity}
              title="No usage recorded yet"
              description="Metered usage will appear here once events are recorded."
            />
          )}
        </CardContent>
      </Card>

      {/* Table */}
      <div className="mt-6">
        <DataTable
          columns={columns}
          data={usageStats}
          loading={loading}
          empty={{
            icon: Activity,
            title: "No events found",
            description: "No metered usage events have been recorded yet.",
          }}
        />
      </div>

      {/* Raw event stream (ingestion debugging) */}
      <div className="mt-8">
        <div className="mb-3 flex items-center justify-between gap-3">
          <div>
            <h2 className="text-sm font-semibold text-foreground">Recent events</h2>
            <p className="text-xs text-muted-foreground">
              The raw ingestion stream, newest first — verify events are landing before
              they aggregate into metrics.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Select value={eventDimension} onValueChange={setEventDimension}>
              <SelectTrigger className="w-[180px]" aria-label="Filter by dimension">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All dimensions</SelectItem>
                {eventDimensions.map((d) => (
                  <SelectItem key={d} value={d}>
                    {d}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button variant="outline" size="sm" onClick={() => refetchEvents()} disabled={eventsLoading}>
              <RefreshCw className={`h-4 w-4 ${eventsLoading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
          </div>
        </div>
        <DataTable
          columns={eventColumns}
          data={events}
          loading={eventsLoading}
          error={eventsError}
          onRetry={refetchEvents}
          getRowId={(e) => e.id}
          empty={{
            icon: Activity,
            title: "No raw events",
            description:
              "POST /v1/usage/events (or the batch endpoint) to ingest usage; events appear here immediately.",
          }}
        />
      </div>
    </>
      )}
    </div>
  );
}
