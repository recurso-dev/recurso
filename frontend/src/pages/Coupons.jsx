import { useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Plus, BadgePercent } from "lucide-react";

import { endpoints as api } from "../lib/api";
import CouponDetail from "../components/slide-overs/CouponDetail";
import { cn, formatCurrency } from "@/lib/utils";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

const STATUS_FILTERS = ["all", "active", "inactive"];

const statusVariant = (status) =>
  ({ active: "success", inactive: "neutral" })[status] || "neutral";

const Coupons = () => {
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");

  const [selectedCoupon, setSelectedCoupon] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const [deactivateTarget, setDeactivateTarget] = useState(null);

  const queryClient = useQueryClient();
  const {
    data: coupons = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["coupons"],
    queryFn: async () => {
      const response = await api.getCoupons();
      // Map backend fields to frontend expectations (unchanged logic).
      return (response.data.data || []).map((c) => ({
        ...c,
        status: c.active ? "active" : "inactive",
        redemptions: 0,
        max_redemptions: null,
        // "percentage" is a legacy alias from pre-normalization seed data
        // (migration 000104 rewrites it; tolerated here for older rows).
        discount:
          c.discount_type === "percent" || c.discount_type === "percentage"
            ? `${c.discount_value}%`
            : formatCurrency(c.discount_value, c.currency),
        duration_in_months: c.duration_months,
      }));
    },
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load coupons"
    : null;

  // A toggle invalidates the whole "coupons" key so the list (and any other
  // coupons-keyed view) refetches — the standard prefix-invalidation contract
  // for mutations (ADR-005).
  const setActiveMutation = useMutation({
    mutationFn: ({ id, active }) => api.setCouponActive(id, active),
    onSuccess: (_data, { active }) => {
      toast.success(active ? "Coupon reactivated." : "Coupon deactivated.");
      setDeactivateTarget(null);
      queryClient.invalidateQueries({ queryKey: ["coupons"] });
    },
    onError: (err) => {
      toast.error(err?.response?.data?.error?.message || "Failed to update coupon");
    },
  });
  const toggling = setActiveMutation.isPending;
  // Reactivation is low-risk, so it skips the confirm; deactivation confirms.
  const setActive = (coupon, active) =>
    setActiveMutation.mutate({ id: coupon.id, active });

  const handleRowClick = (coupon) => {
    setSelectedCoupon(coupon);
    setIsDetailOpen(true);
  };

  const closeDetail = () => {
    setIsDetailOpen(false);
    setTimeout(() => setSelectedCoupon(null), 300);
  };

  const filteredCoupons = useMemo(() => {
    const q = search.trim().toLowerCase();
    return coupons.filter((c) => {
      if (statusFilter !== "all" && c.status !== statusFilter) return false;
      if (q && !(c.code || "").toLowerCase().includes(q)) return false;
      return true;
    });
  }, [coupons, statusFilter, search]);

  const columns = [
    {
      key: "code",
      header: "Coupon Code",
      cell: (c) => <span className="font-mono text-sm font-medium text-foreground">{c.code}</span>,
    },
    {
      key: "discount",
      header: "Discount",
      cell: (c) => <span className="text-muted-foreground">{c.discount}</span>,
    },
    {
      key: "duration",
      header: "Duration",
      cell: (c) => (
        <span className="capitalize text-muted-foreground">
          {c.duration === "repeating"
            ? c.duration_in_months
              ? `For ${c.duration_in_months} months`
              : "Repeating"
            : c.duration}
        </span>
      ),
    },
    {
      key: "redemptions",
      header: "Redemptions",
      cell: (c) => (
        <span className="tabular-nums text-muted-foreground">
          {c.redemptions} {c.max_redemptions ? `/ ${c.max_redemptions}` : ""}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (c) => (
        <Badge variant={statusVariant(c.status)} className="capitalize">
          {c.status}
        </Badge>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (c) => (
        <Button
          size="sm"
          variant={c.active ? "outline" : "ghost"}
          disabled={toggling}
          onClick={(e) => {
            e.stopPropagation();
            if (c.active) setDeactivateTarget(c);
            else setActive(c, true);
          }}
        >
          {c.active ? "Deactivate" : "Reactivate"}
        </Button>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Coupons"
        description="Create and manage discount codes for your customers."
        actions={
          <Button onClick={() => navigate("/coupons/new")}>
            <Plus className="h-4 w-4" />
            Create coupon
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={filteredCoupons}
        loading={loading}
        error={error}
        onRetry={refetch}
        onRowClick={handleRowClick}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search coupons...",
        }}
        toolbar={
          <div className="flex items-center gap-1 rounded-lg border border-border bg-white p-0.5">
            {STATUS_FILTERS.map((f) => (
              <button
                key={f}
                onClick={() => setStatusFilter(f)}
                className={cn(
                  "rounded-md px-3 py-1 text-sm font-medium capitalize transition-colors",
                  statusFilter === f
                    ? "bg-emerald-50 text-emerald-700"
                    : "text-stone-500 hover:text-stone-900"
                )}
              >
                {f}
              </button>
            ))}
          </div>
        }
        empty={{
          icon: BadgePercent,
          title:
            search || statusFilter !== "all" ? "No matching coupons" : "No coupons yet",
          description:
            search || statusFilter !== "all"
              ? "Try adjusting your search or filters."
              : "Create your first discount code to get started.",
          action:
            !search && statusFilter === "all" ? (
              <Button onClick={() => navigate("/coupons/new")}>
                <Plus className="h-4 w-4" />
                Create coupon
              </Button>
            ) : null,
        }}
      />

      <CouponDetail coupon={selectedCoupon} isOpen={isDetailOpen} onClose={closeDetail} />

      <ConfirmDialog
        open={!!deactivateTarget}
        onOpenChange={(o) => !o && setDeactivateTarget(null)}
        title={`Deactivate ${deactivateTarget?.code}?`}
        description="New subscriptions can no longer redeem this code. Customers already using it keep their discount. You can reactivate it later."
        confirmLabel="Deactivate coupon"
        destructive
        busy={toggling}
        onConfirm={() => setActive(deactivateTarget, false)}
      />
    </div>
  );
};

export default Coupons;
