import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, Repeat, MoreHorizontal } from "lucide-react";

import { endpoints } from "../lib/api";
import { useDebounce } from "../hooks/useDebounce";
import SubscriptionDetail from "../components/slide-overs/SubscriptionDetail";
import { formatCurrency, formatDate } from "@/lib/utils";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const PAGE_SIZE = 10;

// Map a subscription status to a Badge variant.
const statusVariant = (status) =>
  ({
    active: "success",
    paused: "warning",
    trialing: "info",
    past_due: "destructive",
    canceled: "neutral",
  })[status] || "neutral";

export default function Subscriptions() {
  const navigate = useNavigate();

  const [subs, setSubs] = useState([]);
  const [customers, setCustomers] = useState({});
  const [plans, setPlans] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("all");
  const [planFilter, setPlanFilter] = useState("all");
  const [dateFilter, setDateFilter] = useState("all");
  const debouncedSearch = useDebounce(search, 500);

  const [selectedSub, setSelectedSub] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = {
        q: debouncedSearch,
        page,
        limit: PAGE_SIZE,
        status: statusFilter === "all" ? "" : statusFilter,
      };

      const [subRes, custRes, planRes] = await Promise.all([
        endpoints.getSubscriptions(params),
        endpoints.getCustomers({ limit: 1000 }).catch(() => ({ data: { data: [] } })),
        endpoints.getPlans({ limit: 100 }).catch(() => ({ data: { data: [] } })),
      ]);

      const customerMap = {};
      (custRes.data.data || []).forEach((c) => {
        customerMap[c.id] = c;
      });
      setCustomers(customerMap);

      const planMap = {};
      (planRes.data.data || []).forEach((p) => {
        planMap[p.id] = p;
      });
      setPlans(planMap);

      setSubs(subRes.data.data || []);
    } catch (err) {
      setError(
        err?.response?.data?.error?.message ||
          err?.message ||
          "Failed to load subscriptions"
      );
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, page, statusFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Reset to page 1 when the query/status change.
  useEffect(() => {
    setPage(1);
  }, [debouncedSearch, statusFilter]);

  // Client-side plan + date filters over the fetched page (logic preserved).
  const filteredSubs = useMemo(() => {
    return subs.filter((s) => {
      if (planFilter !== "all" && s.plan_id !== planFilter) return false;
      if (dateFilter !== "all") {
        const start = new Date(s.current_period_start);
        const now = new Date();
        if (dateFilter === "30_days") {
          const thirtyDaysAgo = new Date(new Date().setDate(now.getDate() - 30));
          if (start < thirtyDaysAgo) return false;
        }
        if (dateFilter === "this_month") {
          const firstOfMonth = new Date(now.getFullYear(), now.getMonth(), 1);
          if (start < firstOfMonth) return false;
        }
      }
      return true;
    });
  }, [subs, planFilter, dateFilter]);

  const handleRowClick = (sub) => {
    setSelectedSub(sub);
    setIsDetailOpen(true);
  };

  const closeDetail = () => {
    setIsDetailOpen(false);
    setTimeout(() => setSelectedSub(null), 300);
  };

  const hasFilters =
    search || statusFilter !== "all" || planFilter !== "all" || dateFilter !== "all";

  const columns = [
    {
      key: "customer",
      header: "Customer",
      cell: (s) => {
        const customer = customers[s.customer_id];
        return (
          <div>
            <div className="text-sm font-medium text-foreground">
              {customer?.name || "Unknown"}
            </div>
            <div className="text-sm text-muted-foreground">
              {customer?.email || "No email"}
            </div>
          </div>
        );
      },
    },
    {
      key: "status",
      header: "Status",
      cell: (s) => (
        <Badge variant={statusVariant(s.status)} className="capitalize">
          {(s.status || "unknown").replace("_", " ")}
        </Badge>
      ),
    },
    {
      key: "plan",
      header: "Plan",
      cell: (s) => (
        <span className="text-muted-foreground">
          {plans[s.plan_id]?.name || s.plan_id?.slice(0, 8)}
        </span>
      ),
    },
    {
      key: "amount",
      header: "Amount",
      cell: (s) => {
        const plan = plans[s.plan_id];
        const price = plan?.prices?.[0];
        const amount = price ? price.amount : 0;
        const currency = price ? price.currency : "USD";
        const interval = plan?.interval_unit === "year" ? "yr" : "mo";
        return (
          <span className="tabular-nums text-foreground">
            <Money amountMinor={amount} currency={currency} />{" "}
            <span className="text-muted-foreground">/ {interval}</span>
          </span>
        );
      },
    },
    {
      key: "start",
      header: "Start date",
      cell: (s) => (
        <span className="text-muted-foreground">
          {formatDate(s.current_period_start)}
        </span>
      ),
    },
    {
      key: "next",
      header: "Next invoice",
      cell: (s) => (
        <span className="text-muted-foreground">
          {formatDate(s.current_period_end)}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: () => (
        <span className="inline-flex text-stone-400">
          <MoreHorizontal className="h-4 w-4" />
        </span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Subscriptions"
        description="Track and manage your recurring subscriptions."
        actions={
          <Button onClick={() => navigate("/subscriptions/new")}>
            <Plus className="h-4 w-4" />
            Add subscription
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={filteredSubs}
        loading={loading}
        error={error}
        onRetry={fetchData}
        onRowClick={handleRowClick}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search by customer name, email, or ID...",
        }}
        toolbar={
          <>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[150px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Status: All</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="paused">Paused</SelectItem>
                <SelectItem value="trialing">Trialing</SelectItem>
                <SelectItem value="past_due">Past Due</SelectItem>
                <SelectItem value="canceled">Canceled</SelectItem>
              </SelectContent>
            </Select>

            <Select value={planFilter} onValueChange={setPlanFilter}>
              <SelectTrigger className="w-[160px]">
                <SelectValue placeholder="Plan: All" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Plan: All</SelectItem>
                {Object.values(plans).map((plan) => (
                  <SelectItem key={plan.id} value={plan.id}>
                    {plan.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select value={dateFilter} onValueChange={setDateFilter}>
              <SelectTrigger className="w-[150px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Date: All time</SelectItem>
                <SelectItem value="30_days">Last 30 days</SelectItem>
                <SelectItem value="this_month">This month</SelectItem>
              </SelectContent>
            </Select>
          </>
        }
        empty={{
          icon: Repeat,
          title: hasFilters ? "No matching subscriptions" : "No subscriptions yet",
          description: hasFilters
            ? "Try adjusting your search or filters."
            : "Create a subscription to start recurring billing.",
          action: !hasFilters ? (
            <Button onClick={() => navigate("/subscriptions/new")}>
              <Plus className="h-4 w-4" />
              Add subscription
            </Button>
          ) : null,
        }}
        pagination={{
          page,
          onPrev: () => setPage((p) => Math.max(1, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext: subs.length >= PAGE_SIZE,
        }}
      />

      <SubscriptionDetail
        subscription={selectedSub}
        customer={selectedSub ? customers[selectedSub.customer_id] : null}
        plan={selectedSub ? plans[selectedSub.plan_id] : null}
        isOpen={isDetailOpen}
        onClose={closeDetail}
      />
    </div>
  );
}
