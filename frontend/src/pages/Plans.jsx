import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, Gift, Package, MoreHorizontal } from "lucide-react";

import { endpoints } from "../lib/api";
import BuyGiftModal from "../components/BuyGiftModal";
import PlanDetail from "../components/slide-overs/PlanDetail";
import { formatCurrency } from "@/lib/utils";
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

const CURRENCY_FILTERS = ["all", "USD", "INR"];
const INTERVAL_FILTERS = ["all", "month", "year"];

export default function Plans() {
  const navigate = useNavigate();

  const [plans, setPlans] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [search, setSearch] = useState("");
  const [currencyFilter, setCurrencyFilter] = useState("all");
  const [intervalFilter, setIntervalFilter] = useState("all");

  const [selectedPlan, setSelectedPlan] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const [isGiftModalOpen, setIsGiftModalOpen] = useState(false);

  const fetchPlans = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await endpoints.getPlans();
      setPlans(response.data.data || []);
    } catch (err) {
      setError(
        err?.response?.data?.error?.message || err?.message || "Failed to load plans"
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPlans();
  }, [fetchPlans]);

  // Filter logic — preserved from the original (currency over prices, interval unit),
  // plus a client-side name/code search over the already-fetched list.
  const filteredPlans = useMemo(() => {
    const q = search.trim().toLowerCase();
    return plans.filter((p) => {
      if (currencyFilter !== "all") {
        const hasCurrency = (p.prices || []).some(
          (price) => price.currency === currencyFilter
        );
        if (!hasCurrency) return false;
      }
      if (intervalFilter !== "all" && p.interval_unit !== intervalFilter) {
        return false;
      }
      if (q) {
        const haystack = `${p.name || ""} ${p.code || ""}`.toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  }, [plans, search, currencyFilter, intervalFilter]);

  const handleRowClick = (plan) => {
    setSelectedPlan(plan);
    setIsDetailOpen(true);
  };

  const closeDetail = () => {
    setIsDetailOpen(false);
    setTimeout(() => setSelectedPlan(null), 300);
  };

  // After an edit/archive in the detail sheet: show the server's version of the
  // plan immediately and refresh the list behind it.
  const handlePlanChanged = (updated) => {
    if (updated?.id) {
      // The PUT response has no prices array — keep the ones we already had.
      setSelectedPlan((prev) =>
        prev && prev.id === updated.id ? { ...prev, ...updated, prices: updated.prices || prev.prices } : prev
      );
    }
    fetchPlans();
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
      cell: (p) =>
        p.prices && p.prices.length > 0 ? (
          <span className="tabular-nums text-foreground">
            {formatCurrency(p.prices[0].amount, p.prices[0].currency)}
          </span>
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
        <Badge variant={p.active ? "success" : "neutral"}>
          {p.active ? "Active" : "Archived"}
        </Badge>
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
        data={filteredPlans}
        loading={loading}
        error={error}
        onRetry={fetchPlans}
        onRowClick={handleRowClick}
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
