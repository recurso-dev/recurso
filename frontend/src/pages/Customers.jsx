import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, Users, Link2 } from "lucide-react";

import { useQuery, useQueryClient } from "@tanstack/react-query";

import { endpoints } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { useDebounce } from "../hooks/useDebounce";
import CustomerDetail from "../components/slide-overs/CustomerDetail";
import { cn, formatDate } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

const STATUS_FILTERS = ["all", "active", "inactive"];
const PAGE_SIZE = 10;

// Risk score → labelled badge.
function RiskBadge({ score }) {
  if (score == null) return <span className="text-xs text-muted-foreground">—</span>;
  let variant = "success";
  let label = "Low";
  if (score >= 50) {
    variant = "destructive";
    label = "High";
  } else if (score >= 20) {
    variant = "warning";
    label = "Medium";
  }
  return (
    <Badge variant={variant}>
      {score} • {label}
    </Badge>
  );
}

export default function Customers() {
  const navigate = useNavigate();

  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebounce(search, 500);

  const [selected, setSelected] = useState(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const queryClient = useQueryClient();
  // Server-driven list: every (page, q, status) combination is its own cache
  // entry; placeholderData keeps the previous page rendered while the next
  // one loads (no skeleton flash when paging).
  const {
    data: customersData,
    isLoading: loading,
    error: queryError,
  } = useQuery({
    queryKey: ["customers", { page, q: debouncedSearch, status }],
    queryFn: async () => {
      const params = { page, limit: PAGE_SIZE };
      if (debouncedSearch) params.q = debouncedSearch;
      if (status !== "all") params.status = status;
      const res = await endpoints.getCustomers(params);
      return res.data.data || [];
    },
    placeholderData: (prev) => prev,
  });
  const customers = useMemo(() => customersData || [], [customersData]);
  const error = queryError
    ? queryError?.response?.data?.error?.message ||
      queryError?.message ||
      "Failed to load customers"
    : null;

  // Mutations elsewhere (create page, detail sheet, pickers' shared cache)
  // see fresh data by invalidating every customers-keyed query.
  const fetchCustomers = useCallback(
    () => queryClient.invalidateQueries({ queryKey: ["customers"] }),
    [queryClient]
  );

  // Reset to page 1 whenever the query changes.
  useEffect(() => {
    setPage(1);
  }, [debouncedSearch, status]);

  const openDetail = (customer) => {
    setSelected(customer);
    setDetailOpen(true);
  };
  const closeDetail = () => {
    setDetailOpen(false);
    setTimeout(() => setSelected(null), 300);
  };

  // After an edit/archive in the detail sheet: show the server's version
  // immediately and refresh the list behind it.
  const handleCustomerChanged = (updated) => {
    if (updated?.id) {
      setSelected((prev) => (prev && prev.id === updated.id ? { ...prev, ...updated } : prev));
    }
    fetchCustomers();
  };

  const copyPortalLink = useCallback(
    (e, customer) => {
      e.stopPropagation();
      // The portal is entered via magic-link login; prefill the customer's
      // email. (The old /portal/{tenant}/{customer} path never existed.)
      const url = `${window.location.origin}/portal/login?email=${encodeURIComponent(customer.email || "")}`;
      navigator.clipboard.writeText(url);
      toast.success("Portal link copied");
    },
    []
  );

  const columns = useMemo(
    () => [
      {
        key: "customer",
        header: "Customer",
        cell: (c) => (
          <div>
            <div className="text-sm font-medium text-foreground">{c.name}</div>
            <div className="text-sm text-muted-foreground">{c.email}</div>
          </div>
        ),
      },
      {
        key: "status",
        header: "Status",
        cell: (c) =>
          c.active === false ? (
            <Badge variant="warning">Archived</Badge>
          ) : c.active_subs > 0 ? (
            <Badge variant="success">Active</Badge>
          ) : (
            <Badge variant="neutral">Inactive</Badge>
          ),
      },
      { key: "risk", header: "Risk", cell: (c) => <RiskBadge score={c.risk_score} /> },
      {
        key: "subs",
        header: "Subscriptions",
        cell: (c) => (
          <span className="tabular-nums text-muted-foreground">{c.active_subs ?? 0}</span>
        ),
      },
      {
        key: "joined",
        header: "Joined",
        cell: (c) => <span className="text-muted-foreground">{formatDate(c.created_at)}</span>,
      },
      {
        key: "actions",
        header: "",
        align: "right",
        cell: (c) => (
          <button
            onClick={(e) => copyPortalLink(e, c)}
            className="text-stone-400 transition-colors hover:text-emerald-600"
            title="Copy portal link"
            aria-label={`Copy portal link for ${c.name}`}
          >
            <Link2 className="h-4 w-4" />
          </button>
        ),
      },
    ],
    [copyPortalLink]
  );

  return (
    <div>
      <PageHeader
        title="Customers"
        description="Manage your customer base and their subscriptions."
        actions={
          <Button onClick={() => navigate("/customers/new")}>
            <Plus className="h-4 w-4" />
            Add customer
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={customers}
        loading={loading}
        error={error}
        onRetry={fetchCustomers}
        onRowClick={openDetail}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search by name or email...",
        }}
        toolbar={
          <div className="flex items-center gap-1 rounded-lg border border-border bg-white p-0.5">
            {STATUS_FILTERS.map((f) => (
              <button
                key={f}
                onClick={() => setStatus(f)}
                className={cn(
                  "rounded-md px-3 py-1 text-sm font-medium capitalize transition-colors",
                  status === f
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
          icon: Users,
          title:
            search || status !== "all" ? "No matching customers" : "No customers yet",
          description:
            search || status !== "all"
              ? "Try adjusting your search or filters."
              : "Add your first customer to get started.",
          action:
            !search && status === "all" ? (
              <Button onClick={() => navigate("/customers/new")}>
                <Plus className="h-4 w-4" />
                Add customer
              </Button>
            ) : null,
        }}
        pagination={{
          page,
          onPrev: () => setPage((p) => Math.max(1, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext: customers.length >= PAGE_SIZE,
        }}
      />

      <CustomerDetail
        customer={selected}
        isOpen={detailOpen}
        onClose={closeDetail}
        onChanged={handleCustomerChanged}
      />
    </div>
  );
}
