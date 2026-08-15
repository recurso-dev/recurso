import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { Plus, Receipt } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatDate } from "@/lib/utils";
import { useUrlState, useResetPageOnChange } from "@/lib/useUrlState";
import { LIST_PAGE_SIZE, fetchAllPages, pageSlice } from "@/lib/pagination";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { ListNotice } from "@/components/patterns/ListNotice";
import { StatusBadge } from "@/components/ui/status-badge";
import { Button } from "@/components/ui/button";

const CreditNotes = () => {
  const navigate = useNavigate();
  // List state in the URL so returning from a credit note restores search / page.
  const [search, setSearch] = useUrlState("q", "");
  const [page, setPage] = useUrlState("page", 1, { parse: Number });

  // /credit-notes has no free-text search server-side (search is over id +
  // customer name below), so the page holds the complete set — page through it
  // to a bounded cap instead of letting the backend default truncate it.
  // (BACKEND GAP: add search + total to /credit-notes for true server-side
  // pagination.)
  const {
    data,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["credit-notes", "all"],
    queryFn: () =>
      fetchAllPages((offset, limit) =>
        endpoints.getCreditNotes({ limit, offset }).then((r) => r.data.data || []),
      ),
    placeholderData: (prev) => prev,
  });
  const creditNotes = useMemo(() => data?.rows ?? [], [data]);
  const truncated = data?.truncated ?? false;
  const error = queryError ? "Failed to load credit notes." : null;

  const filteredNotes = useMemo(() => {
    const q = search.toLowerCase();
    return creditNotes.filter(
      (cn) =>
        cn.id.toLowerCase().includes(q) ||
        (cn.customer?.name || "").toLowerCase().includes(q),
    );
  }, [creditNotes, search]);
  const pagedNotes = pageSlice(filteredNotes, page);
  // A new search starts back at page 1 (URL-safe: separate tick, skips mount).
  useResetPageOnChange(setPage, [search]);

  const columns = [
    {
      key: "id",
      header: "ID",
      cell: (cn) => (
        <span className="font-mono text-xs text-foreground">
          {cn.reference || cn.id.slice(0, 8)}
        </span>
      ),
    },
    {
      key: "customer",
      header: "Customer",
      cell: (cn) => (
        <span className="text-muted-foreground">
          {cn.customer ? cn.customer.name : "Unknown Customer"}
        </span>
      ),
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (cn) => <Money amountMinor={cn.amount} currency={cn.currency} />,
    },
    {
      key: "balance",
      header: "Balance",
      align: "right",
      cell: (cn) => <Money amountMinor={cn.balance} currency={cn.currency} className="text-muted-foreground" />,
    },
    {
      key: "status",
      header: "Status",
      cell: (cn) => (
        <StatusBadge status={cn.status} />
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (cn) => (
        <span className="text-muted-foreground">{formatDate(cn.created_at)}</span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Credit Notes"
        description="Manage customer credits and refunds."
        actions={
          <Button onClick={() => navigate("/credit-notes/new")}>
            <Plus className="h-4 w-4" />
            Create credit note
          </Button>
        }
      />

      {truncated && (
        <ListNotice>
          Showing the first {creditNotes.length.toLocaleString()} credit notes.
          Refine your search or use the API for the complete set.
        </ListNotice>
      )}

      <DataTable
        columns={columns}
        data={pagedNotes}
        loading={loading}
        error={error}
        onRetry={refetch}
        rowHref={(row) => `/credit-notes/${row.id}`}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search by ID or customer...",
        }}
        empty={{
          icon: Receipt,
          title: search ? "No matching credit notes" : "No credit notes yet",
          description: search
            ? "Try adjusting your search."
            : "Issue a credit note to get started.",
          action: !search ? (
            <Button onClick={() => navigate("/credit-notes/new")}>
              <Plus className="h-4 w-4" />
              Create credit note
            </Button>
          ) : null,
        }}
        pagination={{
          page,
          pageSize: LIST_PAGE_SIZE,
          total: filteredNotes.length,
          onPageChange: setPage,
        }}
      />
    </div>
  );
};

export default CreditNotes;
