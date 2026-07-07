import { useEffect, useMemo, useState } from "react";
import { BarChart } from "@tremor/react";
import { Activity, Download, Gauge, Layers, Users } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
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

const shortId = (id) => (id ? `${id.substring(0, 8)}...` : "Unknown");

export default function Usage() {
  const [usageStats, setUsageStats] = useState([]);
  const [loading, setLoading] = useState(true);
  const [customerFilter, setCustomerFilter] = useState("all");
  const [planFilter, setPlanFilter] = useState("all");

  useEffect(() => {
    const fetchUsage = async () => {
      try {
        const response = await api.getUsageStats();
        setUsageStats(response.data.data || []);
      } catch (error) {
        console.error("Failed to fetch usage stats:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchUsage();
  }, []);

  // Unique filter options derived from the stats.
  const uniqueCustomers = useMemo(
    () => [...new Set(usageStats.map((d) => d.customer_id))],
    [usageStats]
  );
  const uniquePlans = useMemo(
    () => [...new Set(usageStats.map((d) => d.plan_id))],
    [usageStats]
  );

  const filteredData = useMemo(
    () =>
      usageStats.filter((item) => {
        if (customerFilter !== "all" && item.customer_id !== customerFilter)
          return false;
        if (planFilter !== "all" && item.plan_id !== planFilter) return false;
        return true;
      }),
    [usageStats, customerFilter, planFilter]
  );

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
        title="Usage Metering"
        description="Metered usage aggregated by customer, plan, and dimension."
        actions={
          <Button variant="outline" onClick={exportCsv}>
            <Download className="h-4 w-4" />
            Export data
          </Button>
        }
      />

      {/* Filters */}
      <div className="mb-6 flex flex-wrap items-center gap-2">
        <Select value={customerFilter} onValueChange={setCustomerFilter}>
          <SelectTrigger className="w-auto min-w-[12rem]">
            <SelectValue placeholder="Customer: All" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All customers</SelectItem>
            {uniqueCustomers.map((c) => (
              <SelectItem key={c} value={c}>
                {shortId(c)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={planFilter} onValueChange={setPlanFilter}>
          <SelectTrigger className="w-auto min-w-[10rem]">
            <SelectValue placeholder="Plan: All" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All plans</SelectItem>
            {uniquePlans.map((p) => (
              <SelectItem key={p} value={p}>
                {shortId(p)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

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
          value={customersMetered.toLocaleString()}
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
            <BarChart
              className="h-72"
              data={byDimension}
              index="dimension"
              categories={["Units"]}
              colors={["emerald"]}
              valueFormatter={(v) => v.toLocaleString()}
              showLegend={false}
              showGridLines
              yAxisWidth={64}
            />
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
    </div>
  );
}
