import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router";
import { Plus, Gift, Package } from "lucide-react";

import { endpoints } from "../lib/api";
import { queryClient } from "@/lib/queryClient";
import { useDebounce } from "../hooks/useDebounce";
import BuyGiftModal from "../components/BuyGiftModal";
import PlanDetail from "../components/slide-overs/PlanDetail";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const CURRENCY_FILTERS = ["all", "USD", "INR"];
const INTERVAL_FILTERS = ["all", "month", "year"];
const PAGE_SIZE = 10;

export default function Plans() {
  const navigate = useNavigate();

  const [search, setSearch] = useState("");
  const [currencyFilter, setCurrencyFilter] = useState("all");
  const [intervalFilter, setIntervalFilter] = useState("all");
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebounce(search, 500);

  // URL-driven detail (/plans/:id) — shareable, refresh/back-safe.
  const { id: routeId } = useParams();
  const { data: routedObject } = useQuery({
    queryKey: ["plan", routeId],
    queryFn: async () => (await endpoints.getPlan(routeId)).data.data,
    enabled: Boolean(routeId),
  });
  const [isGiftModalOpen, setIsGiftModalOpen] = useState(false);

  // Fully server-driven list keyed by (page, q, currency, interval) —
  // filtering only the fetched page client-side silently hid matches on other
  // pages. placeholderData keeps the current page rendered while the next
  // loads. The backend defaults to limit 10, so paging must be explicit or
  // the list silently truncates past ten plans.
  const {
    data: plans = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: [
      "plans",
      { page, q: debouncedSearch, currency: currencyFilter, interval: intervalFilter },
    ],
    queryFn: async () => {
      const params = { page, limit: PAGE_SIZE };
      if (debouncedSearch) params.q = debouncedSearch;
      if (currencyFilter !== "all") params.currency = currencyFilter;
      if (intervalFilter !== "all") params.interval_unit = intervalFilter;
      return (await endpoints.getPlans(params)).data.data || [];
    },
    placeholderData: (prev) => prev,
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load plans"
    : null;

  // Reset to page 1 whenever any filter changes.
  useEffect(() => {
    setPage(1);
  }, [debouncedSearch, currencyFilter, intervalFilter]);


  const closeDetail = () => navigate("/plans");
  const isDetailOpen = Boolean(routeId);
  const selectedPlan = routedObject || null;

  // After an edit/archive in the detail sheet: show the server's version of the
  // plan immediately and refresh the list behind it.
  const handlePlanChanged = (updated) => {
    // The open detail is served by the ["plan", id] query — refetch it so the
    // sheet shows the server's version (GET /plans/:id includes prices).
    if (updated?.id) {
      queryClient.invalidateQueries({ queryKey: ["plan", updated.id] });
    }
    // Invalidating the "plans" prefix refreshes this server-driven list AND the
    // shared usePlans cache (Subscriptions/Metering/Mandates pickers) in one go.
    queryClient.invalidateQueries({ queryKey: ["plans"] });
  };

  const hasFilters = search || currencyFilter !== "all" || intervalFilter !== "all";

  const columns = [
    {
      key: "name",
      header: "Plan name",
      cell: (p) => <span className="font-medium text-foreground">{p.name}</span>,
    },
    {
      key: "code",
      header: "Plan ID",
      cell: (p) => (
        <span className="font-mono text-xs text-muted-foreground">{p.code}</span>
      ),
    },
    {
      key: "price",
      header: "Price",
      align: "right",
      cell: (p) =>
        p.prices && p.prices.length > 0 ? (
          <Money amountMinor={p.prices[0].amount} currency={p.prices[0].currency} />
        ) : (
          <span className="text-muted-foreground">Free</span>
        ),
    },
    {
      key: "interval",
      header: "Billing interval",
      cell: (p) => (
        <span className="capitalize text-muted-foreground">{p.interval_unit}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (p) => (
        <StatusBadge status={p.active ? "active" : "archived"} />
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Plans"
        description="Define the pricing plans customers can subscribe to."
        actions={
          <>
            <Button variant="outline" onClick={() => setIsGiftModalOpen(true)}>
              <Gift className="h-4 w-4" />
              Gift plan
            </Button>
            <Button onClick={() => navigate("/plans/new")}>
              <Plus className="h-4 w-4" />
              New plan
            </Button>
          </>
        }
      />

      <DataTable
        columns={columns}
        data={plans}
        loading={loading}
        error={error}
        onRetry={refetch}
        rowHref={(row) => `/plans/${row.id}`}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search by plan name or ID...",
        }}
        toolbar={
          <>
            <Select value={currencyFilter} onValueChange={setCurrencyFilter}>
              <SelectTrigger className="w-[150px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CURRENCY_FILTERS.map((c) => (
                  <SelectItem key={c} value={c}>
                    {c === "all" ? "Currency: All" : `Currency: ${c}`}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={intervalFilter} onValueChange={setIntervalFilter}>
              <SelectTrigger className="w-[150px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {INTERVAL_FILTERS.map((i) => (
                  <SelectItem key={i} value={i} className="capitalize">
                    {i === "all" ? "Interval: All" : `Interval: ${i}`}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </>
        }
        empty={{
          icon: Package,
          title: hasFilters ? "No matching plans" : "No plans yet",
          description: hasFilters
            ? "Try adjusting your search or filters."
            : "Create your first plan to start billing customers.",
          action: !hasFilters ? (
            <Button onClick={() => navigate("/plans/new")}>
              <Plus className="h-4 w-4" />
              New plan
            </Button>
          ) : null,
        }}
        pagination={{
          page,
          onPrev: () => setPage((p) => Math.max(1, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext: plans.length >= PAGE_SIZE,
        }}
      />

      <PlanDetail
        plan={selectedPlan}
        isOpen={isDetailOpen}
        onClose={closeDetail}
        onChanged={handlePlanChanged}
      />

      <BuyGiftModal
        isOpen={isGiftModalOpen}
        onClose={() => setIsGiftModalOpen(false)}
        plans={plans}
      />
    </div>
  );
}
