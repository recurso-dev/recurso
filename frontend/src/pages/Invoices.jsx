import { useState } from "react";
import { useSearchParams } from "react-router";
import { FileText, Download } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import { formatCurrency, formatDate } from "@/lib/utils";
import { moneyColumn } from "@/components/patterns/columns";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { StatusBadge } from "@/components/ui/status-badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

// Status filter chips. "past_due" folds overdue + past_due into one bucket.
const STATUS_FILTERS = [
  { key: "all", label: "All" },
  { key: "paid", label: "Paid" },
  { key: "open", label: "Open" },
  { key: "past_due", label: "Past due" },
  { key: "void", label: "Void" },
  { key: "draft", label: "Draft" },
];

const matchesStatus = (inv, key) => {
  if (key === "all") return true;
  if (key === "past_due") return inv.status === "past_due" || inv.status === "overdue";
  return inv.status === key;
};

// AR aging bucket for one invoice — mirrors the backend's aging SQL exactly
// (GetInvoiceAgingRows): outstanding open/past_due rows, bucketed by how far
// past due_date they are. Lets the aging report's buckets deep-link here.
const AGING_LABELS = {
  current: "Current (not yet due)",
  "1-30": "Overdue 1–30 days",
  "31-60": "Overdue 31–60 days",
  "61-90": "Overdue 61–90 days",
  "90+": "Overdue 90+ days",
};
const agingBucketOf = (inv) => {
  const due = inv.due_date ? new Date(inv.due_date) : null;
  if (!due || due >= new Date()) return "current";
  const days = (Date.now() - due.getTime()) / 86400000;
  if (days <= 30) return "1-30";
  if (days <= 60) return "31-60";
  if (days <= 90) return "61-90";
  return "90+";
};
const matchesAging = (inv, bucket) =>
  ["open", "past_due", "overdue"].includes(inv.status) &&
  (inv.total || 0) - (inv.amount_paid || 0) > 0 &&
  agingBucketOf(inv) === bucket;

function toCSV(rows) {
  if (rows.length === 0) return "";
  const cols = Object.keys(rows[0]);
  const esc = (v) =>
    v == null
      ? ""
      : /[",\n]/.test(String(v))
        ? `"${String(v).replace(/"/g, '""')}"`
        : String(v);
  return [cols.join(","), ...rows.map((r) => cols.map((c) => esc(r[c])).join(","))].join("\n");
}

function downloadCSV(text, name) {
  const blob = new Blob([text], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 5_000);
}

function EInvoiceBadge({ status }) {
  if (!status || status === "PENDING")
    return <span className="text-sm text-muted-foreground">—</span>;
  return <StatusBadge status={status} />;
}

const Invoices = () => {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  // ?aging=<bucket> deep-links from the invoice-aging report's buckets.
  const [searchParams, setSearchParams] = useSearchParams();
  const agingFilter = AGING_LABELS[searchParams.get("aging")] ? searchParams.get("aging") : null;

  // Invoices come from the shared query cache (60s fresh — revisiting the
  // page reuses the cached list); customer names from the shared hook.
  const {
    data,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["invoices", "all"],
    queryFn: async () => {
      // Page through the server-paginated endpoint (≤250/page) so the status
      // chips, the aging math, and above all the CSV export see EVERY invoice
      // — one max-size page silently truncated all three past 250 invoices.
      // Beyond the safety cap we keep the newest pages and say so on the page
      // instead of pretending the set is complete.
      const PER_PAGE = 250;
      const MAX_PAGES = 40; // 10k invoices
      const first = await endpoints.getInvoices({ per_page: PER_PAGE, page: 1 });
      const rows = [...(first?.data?.data || [])];
      const totalPages = first?.data?.pagination?.total_pages || 1;
      const pages = Math.min(totalPages, MAX_PAGES);
      if (pages > 1) {
        const rest = await Promise.all(
          Array.from({ length: pages - 1 }, (_, i) =>
            endpoints.getInvoices({ per_page: PER_PAGE, page: i + 2 })
          )
        );
        for (const res of rest) rows.push(...(res?.data?.data || []));
      }
      return {
        rows,
        truncated: totalPages > MAX_PAGES,
        total: first?.data?.pagination?.total ?? rows.length,
      };
    },
  });
  const invoices = data?.rows || [];
  const truncated = data?.truncated || false;
  const totalCount = data?.total ?? invoices.length;
  const error = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load invoices"
    : null;
  const { names: customerNames } = useCustomers();

  const filteredInvoices = invoices.filter((inv) => {
    if (agingFilter && !matchesAging(inv, agingFilter)) return false;
    if (!matchesStatus(inv, statusFilter)) return false;
    if (!search) return true;
    const s = search.toLowerCase();
    return (
      inv.invoice_number?.toLowerCase().includes(s) ||
      inv.customer_id?.toLowerCase().includes(s) ||
      inv.status?.toLowerCase().includes(s)
    );
  });

  const exportCsv = () => {
    const rows = filteredInvoices.map((inv) => ({
      number: inv.invoice_number || "",
      customer: customerNames[inv.customer_id] || inv.customer_id || "",
      amount: formatCurrency(inv.total, inv.currency),
      currency: inv.currency || "",
      status: inv.status || "",
      e_invoice: inv.e_invoice_status || "",
      date: inv.created_at ? formatDate(inv.created_at) : "",
    }));
    downloadCSV(toCSV(rows), `invoices-${new Date().toISOString().slice(0, 10)}.csv`);
  };

  // Client sorting is honest here: this page loads the COMPLETE invoice set
  // (page-through fetch), so sorting spans everything, not one server page.
  const columns = [
    {
      key: "invoice_number",
      header: "Number",
      sortable: true,
      cell: (inv) => (
        <span className="font-medium text-foreground">{inv.invoice_number}</span>
      ),
    },
    {
      key: "customer",
      header: "Customer",
      sortable: true,
      sortValue: (inv) => customerNames[inv.customer_id] || "",
      cell: (inv) =>
        customerNames[inv.customer_id] ? (
          <span className="text-sm text-foreground">{customerNames[inv.customer_id]}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">
            {inv.customer_id ? `${inv.customer_id.slice(0, 8)}…` : "—"}
          </span>
        ),
    },
    moneyColumn({
      key: "amount",
      header: "Amount",
      amount: (inv) => inv.total,
      currency: (inv) => inv.currency,
      sortable: true,
    }),
    {
      key: "status",
      header: "Status",
      sortable: true,
      cell: (inv) => (
        <StatusBadge status={inv.status} />
      ),
    },
    {
      key: "e_invoice",
      header: "E-Invoice",
      hideBelow: "md",
      cell: (inv) => <EInvoiceBadge status={inv.e_invoice_status} />,
    },
    {
      key: "date",
      header: "Date",
      sortable: true,
      sortValue: (inv) => inv.created_at || "",
      cell: (inv) => (
        <span className="text-muted-foreground">{formatDate(inv.created_at)}</span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Invoices"
        description="View and manage customer invoices."
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={exportCsv}
            disabled={filteredInvoices.length === 0}
          >
            <Download className="h-4 w-4" />
            Export CSV
          </Button>
        }
      />

      {truncated && (
        <p
          role="status"
          className="mb-4 rounded-md border border-warning/20 bg-warning/5 px-3 py-2 text-sm text-warning"
        >
          Showing the newest {invoices.length.toLocaleString()} of{" "}
          {totalCount.toLocaleString()} invoices — filters and the CSV export cover only
          these. Use the API for a complete export.
        </p>
      )}

      <DataTable
        columns={columns}
        data={filteredInvoices}
        loading={loading}
        error={error}
        onRetry={refetch}
        rowHref={(inv) => `/invoices/${inv.id}`}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search invoices...",
        }}
        toolbar={
          <div className="flex flex-wrap gap-1.5">
            {agingFilter && (
              <button
                type="button"
                onClick={() => setSearchParams({}, { replace: true })}
                title="Clear the aging filter"
                className="rounded-full border border-warning/20 bg-warning/5 px-3 py-1 text-xs font-medium text-warning transition-colors hover:bg-warning/15"
              >
                {AGING_LABELS[agingFilter]} ×
              </button>
            )}
            {STATUS_FILTERS.map((f) => (
              <button
                key={f.key}
                type="button"
                onClick={() => setStatusFilter(f.key)}
                className={cn(
                  "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
                  statusFilter === f.key
                    ? "border-success/20 bg-success/5 text-success"
                    : "border-border text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
              >
                {f.label}
              </button>
            ))}
          </div>
        }
        empty={{
          icon: FileText,
          title: statusFilter === "all" ? "No invoices yet" : "No invoices match this filter",
          description:
            statusFilter === "all"
              ? "Invoices will appear here once subscriptions are billed."
              : "Try a different status filter.",
        }}
      />

    </div>
  );
};

export default Invoices;
