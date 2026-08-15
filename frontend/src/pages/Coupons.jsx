import { useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { Plus, BadgePercent } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { cn, formatCurrency } from "@/lib/utils";
import { useUrlState, useResetPageOnChange } from "@/lib/useUrlState";
import { useTableSort, sortRows } from "@/lib/tableSort";
import { LIST_PAGE_SIZE, fetchAllPages, pageSlice } from "@/lib/pagination";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { ListNotice } from "@/components/patterns/ListNotice";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

const STATUS_FILTERS = ["all", "active", "inactive"];

const Coupons = () => {
  const navigate = useNavigate();
  // List state in the URL so returning from a coupon restores search / filter /
  // page (useUrlState).
  const [search, setSearch] = useUrlState("q", "");
  const [statusFilter, setStatusFilter] = useUrlState("status", "all");
  const [page, setPage] = useUrlState("page", 1, { parse: Number });
  // URL-persisted sort over the complete (page-through) set — Batch F3.
  const { sort, sortKey, onSortChange } = useTableSort();

  const [deactivateTarget, setDeactivateTarget] = useState(null);

  const queryClient = useQueryClient();
  // /coupons has no server-side search or status filter (both are applied
  // client-side below), so the page must hold the complete set — page through it
  // to a bounded cap rather than let the backend's default limit silently drop
  // rows. (BACKEND GAP: add search / status / total to /coupons for true
  // server-side pagination.)
  const {
    data,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["coupons", "all"],
    queryFn: async () => {
      const { rows, truncated } = await fetchAllPages((offset, limit) =>
        api.getCoupons({ limit, offset }).then((r) => r.data.data || []),
      );
      // Map backend fields to frontend expectations (unchanged logic).
      const coupons = rows.map((c) => ({
        ...c,
        status: c.active ? "active" : "inactive",
        // "percentage" is a legacy alias from pre-normalization seed data
        // (migration 000104 rewrites it; tolerated here for older rows).
        discount:
          c.discount_type === "percent" || c.discount_type === "percentage"
            ? `${c.discount_value}%`
            : formatCurrency(c.discount_value, c.currency),
        duration_in_months: c.duration_months,
      }));
      return { coupons, truncated };
    },
    placeholderData: (prev) => prev,
  });
  const coupons = useMemo(() => data?.coupons ?? [], [data]);
  const truncated = data?.truncated ?? false;
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

  const filteredCoupons = useMemo(() => {
    const q = search.trim().toLowerCase();
    return coupons.filter((c) => {
      if (statusFilter !== "all" && c.status !== statusFilter) return false;
      if (q && !(c.code || "").toLowerCase().includes(q)) return false;
      return true;
    });
  }, [coupons, statusFilter, search]);

  // A new search/filter OR a sort change starts back at page 1. Reset in an
  // effect (a separate tick from the filter's own URL write, so the two don't
  // clobber each other), skip the first run so a page restored from the URL on
  // mount survives, and hold setPage in a ref so a plain page change never
  // re-triggers this. pagedCoupons is computed after `columns`, over the sorted
  // full set.
  useResetPageOnChange(setPage, [search, statusFilter, sortKey]);

  const columns = [
    {
      key: "code",
      header: "Coupon Code",
      sortable: true,
      sortValue: (c) => c.code || "",
      cell: (c) => <span className="font-mono text-sm font-medium text-foreground">{c.code}</span>,
    },
    {
      // Not sortable: `discount` mixes units (a percent for percent coupons, a
      // money amount for fixed ones), so any single ordering would be a lie.
      key: "discount",
      header: "Discount",
      align: "right",
      cell: (c) => <span className="tabular-nums text-foreground">{c.discount}</span>,
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
      key: "status",
      header: "Status",
      sortable: true,
      sortValue: (c) => c.status,
      cell: (c) => (
        <StatusBadge status={c.status} />
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

  // Sort the full filtered set, THEN paginate (ordering spans everything).
  const pagedCoupons = pageSlice(sortRows(filteredCoupons, sort, columns), page);

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

      {truncated && (
        <ListNotice>
          Showing the first {coupons.length.toLocaleString()} coupons. Refine your
          search or use the API for the complete set.
        </ListNotice>
      )}

      <DataTable
        columns={columns}
        data={pagedCoupons}
        sort={sort}
        onSortChange={onSortChange}
        loading={loading}
        error={error}
        onRetry={refetch}
        rowHref={(c) => `/coupons/${c.id}`}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search coupons...",
        }}
        toolbar={
          <div className="flex items-center gap-1 rounded-lg border border-border bg-white p-0.5">
            {STATUS_FILTERS.map((f) => (
              <button type="button"
                key={f}
                onClick={() => setStatusFilter(f)}
                className={cn(
                  "rounded-md px-3 py-1 text-sm font-medium capitalize transition-colors",
                  statusFilter === f
                    ? "bg-success/5 text-success"
                    : "text-muted-foreground hover:text-foreground"
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
        pagination={{
          page,
          pageSize: LIST_PAGE_SIZE,
          total: filteredCoupons.length,
          onPageChange: setPage,
        }}
      />

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
