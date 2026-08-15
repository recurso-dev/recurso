import { useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "@/components/ui/sonner";
import { usePlans, useCustomers } from "@/lib/useCustomers";
import { Plus, Gift, CheckCircle2, Clock, Ban } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatDate } from "@/lib/utils";
import { useUrlState, useResetPageOnChange } from "@/lib/useUrlState";
import { useTableSort, sortRows } from "@/lib/tableSort";
import { LIST_PAGE_SIZE, fetchAllPages, pageSlice } from "@/lib/pagination";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { DataTable } from "@/components/patterns/DataTable";
import { ListNotice } from "@/components/patterns/ListNotice";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

function Gifts() {
  const [showCreate, setShowCreate] = useState(false);
  const [cancelTarget, setCancelTarget] = useState(null);
  const [cancelNotice, setCancelNotice] = useState(null);
  const [form, setForm] = useState({
    buyer_customer_id: "",
    plan_id: "",
    duration_months: 12,
  });

  const queryClient = useQueryClient();
  // Reference data from the shared cache (ADR-005).
  const { plans } = usePlans();
  const { customers } = useCustomers();

  const [page, setPage] = useUrlState("page", 1, { parse: Number });
  // URL-persisted sort over the complete (page-through) set — Batch F3.
  const { sort, sortKey, onSortChange } = useTableSort();

  // The StatCards below (total / redeemed / pending) count every gift, so the
  // page holds the complete set — page through it to a bounded cap rather than
  // let the backend default (per_page 50) silently drop rows. (BACKEND GAP: add
  // a total / stats to /gifts for true server-side pagination.)
  const {
    data,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["gifts", "all"],
    queryFn: () =>
      fetchAllPages((offset, limit) =>
        endpoints
          .getGifts({ page: offset / limit + 1, per_page: limit })
          .then((r) => (Array.isArray(r.data?.data) ? r.data.data : [])),
      ),
    placeholderData: (prev) => prev,
  });
  const gifts = useMemo(() => data?.rows ?? [], [data]);
  const truncated = data?.truncated ?? false;
  const error = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load gifts"
    : null;
  // A sort change starts back at page 1 (URL-safe: separate tick, skips mount).
  useResetPageOnChange(setPage, [sortKey]);

  const createMutation = useMutation({
    mutationFn: (payload) => endpoints.purchaseGift(payload),
    onSuccess: () => {
      setShowCreate(false);
      setForm({ buyer_customer_id: "", plan_id: "", duration_months: 12 });
      queryClient.invalidateQueries({ queryKey: ["gifts"] });
    },
    onError: (err) => {
      // Money moves here — a failed purchase must be told to the operator,
      // not just the console (audit bug 6).
      toast.error(err?.response?.data?.error?.message || "Failed to purchase gift");
    },
  });
  const creating = createMutation.isPending;

  // Canceling a purchased gift compensates the buyer: a paid purchase becomes
  // an account credit, an unpaid invoice is voided.
  const cancelMutation = useMutation({
    mutationFn: (id) => endpoints.cancelGift(id),
    onSuccess: (response) => {
      const res = response.data?.data;
      setCancelTarget(null);
      setCancelNotice(
        res?.credit_note
          ? "Gift canceled — the purchase amount was issued to the buyer as an account credit."
          : res?.invoice_voided
            ? "Gift canceled — the unpaid purchase invoice was voided."
            : "Gift canceled.",
      );
      queryClient.invalidateQueries({ queryKey: ["gifts"] });
    },
    onError: (err) => {
      setCancelTarget(null);
      setCancelNotice(
        err?.response?.data?.error?.message || err?.message || "Failed to cancel gift",
      );
    },
  });

  const handleCreate = (e) => {
    e.preventDefault();
    if (!form.buyer_customer_id || !form.plan_id) return;
    createMutation.mutate({
      buyer_customer_id: form.buyer_customer_id,
      plan_id: form.plan_id,
      duration_months: parseInt(form.duration_months),
    });
  };

  const redeemedCount = gifts.filter((g) => g.status === "redeemed").length;
  const pendingCount = gifts.filter((g) => g.status === "purchased").length;

  const columns = [
    {
      key: "code",
      header: "Gift Code",
      cell: (g) => (
        <span className="rounded-md bg-muted px-2 py-1 font-mono text-sm text-foreground">
          {g.code}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      sortable: true,
      sortValue: (g) => g.status,
      cell: (g) => (
        <StatusBadge status={g.status} />
      ),
    },
    {
      key: "duration",
      header: "Duration",
      sortable: true,
      sortValue: (g) => g.duration_months,
      cell: (g) => <span className="text-muted-foreground">{g.duration_months} Months</span>,
    },
    {
      key: "recipient",
      header: "Recipient",
      cell: (g) => <span className="text-muted-foreground">{g.recipient_email || "—"}</span>,
    },
    {
      key: "purchased",
      header: "Purchased",
      sortable: true,
      sortValue: (g) => g.created_at || "",
      cell: (g) => <span className="text-muted-foreground">{formatDate(g.created_at)}</span>,
    },
    {
      key: "actions",
      header: "",
      cell: (g) =>
        g.status === "purchased" ? (
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive"
            onClick={() => setCancelTarget(g)}
          >
            <Ban className="h-4 w-4" />
            Cancel
          </Button>
        ) : null,
    },
  ];

  // Sort the full set, THEN paginate (ordering spans everything).
  const pagedGifts = pageSlice(sortRows(gifts, sort, columns), page);

  return (
    <div>
      <PageHeader
        title="Gift Subscriptions"
        description="Manage purchased gift subscriptions and track redemptions."
        actions={
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4" />
            Create gift
          </Button>
        }
      />

      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard label="Total Gifts Sold" value={gifts.length.toLocaleString()} icon={Gift} />
        <StatCard label="Redeemed" value={redeemedCount.toLocaleString()} icon={CheckCircle2} />
        <StatCard label="Pending" value={pendingCount.toLocaleString()} icon={Clock} />
      </div>

      {cancelNotice && (
        <div
          role="status"
          className="mb-4 flex items-center justify-between rounded-md border bg-muted/50 px-4 py-3 text-sm text-foreground"
        >
          <span>{cancelNotice}</span>
          <Button variant="ghost" size="sm" onClick={() => setCancelNotice(null)}>
            Dismiss
          </Button>
        </div>
      )}

      {truncated && (
        <ListNotice>
          Showing the first {gifts.length.toLocaleString()} gifts. Use the API for
          the complete set.
        </ListNotice>
      )}

      <DataTable
        columns={columns}
        data={pagedGifts}
        sort={sort}
        onSortChange={onSortChange}
        loading={loading}
        error={error}
        onRetry={refetch}
        empty={{
          icon: Gift,
          title: "No gifts yet",
          description: "Create your first gift subscription for a customer.",
          action: (
            <Button onClick={() => setShowCreate(true)}>
              <Plus className="h-4 w-4" />
              Create gift
            </Button>
          ),
        }}
        pagination={{
          page,
          pageSize: LIST_PAGE_SIZE,
          total: gifts.length,
          onPageChange: setPage,
        }}
      />

      <Sheet open={showCreate} onOpenChange={setShowCreate}>
        <SheetContent side="right" className="w-full sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>Create gift subscription</SheetTitle>
            <SheetDescription>Purchase a gift subscription on behalf of a customer.</SheetDescription>
          </SheetHeader>

          <form
            id="create-gift-form"
            onSubmit={handleCreate}
            className="flex-1 space-y-6 overflow-y-auto px-6 py-6"
          >
            <FormField label="Buyer customer" htmlFor="buyer_customer_id" required>
              <Select
                value={form.buyer_customer_id}
                onValueChange={(v) => setForm({ ...form, buyer_customer_id: v })}
              >
                <SelectTrigger id="buyer_customer_id">
                  <SelectValue placeholder="Select customer..." />
                </SelectTrigger>
                <SelectContent>
                  {customers.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name} ({c.email})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField label="Plan" htmlFor="plan_id" required>
              <Select value={form.plan_id} onValueChange={(v) => setForm({ ...form, plan_id: v })}>
                <SelectTrigger id="plan_id">
                  <SelectValue placeholder="Select plan..." />
                </SelectTrigger>
                <SelectContent>
                  {plans.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField label="Duration (months)" htmlFor="duration_months" required>
              <Input
                id="duration_months"
                type="number"
                min="1"
                max="36"
                required
                value={form.duration_months}
                onChange={(e) => setForm({ ...form, duration_months: e.target.value })}
              />
            </FormField>
          </form>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button type="submit" form="create-gift-form" disabled={creating}>
              {creating ? "Creating..." : "Create gift"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      {/* Cancel gift — buyer is made whole (credit or void), redemption is blocked */}
      <ConfirmDialog
        open={!!cancelTarget}
        onOpenChange={(open) => !open && setCancelTarget(null)}
        title="Cancel this gift?"
        description={
          cancelTarget
            ? `Gift ${cancelTarget.code} can no longer be redeemed. If the purchase was paid, the buyer receives the amount back as an account credit; an unpaid purchase invoice is voided.`
            : ""
        }
        confirmLabel="Cancel gift"
        destructive
        busy={cancelMutation.isPending}
        onConfirm={() => cancelMutation.mutate(cancelTarget.id)}
      />
    </div>
  );
}

export default Gifts;
