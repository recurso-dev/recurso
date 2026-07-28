import { useState, useEffect } from "react";
import { useLocation, useNavigate } from "react-router";
import { FileText, Download } from "lucide-react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import InvoiceDetail from "../components/slide-overs/InvoiceDetail";
import { formatCurrency, formatDate } from "@/lib/utils";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";
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

// Map an invoice status to a Badge variant.
const invoiceStatusVariant = (status) =>
  ({
    paid: "success",
    open: "info",
    overdue: "destructive",
    past_due: "destructive",
    void: "neutral",
    draft: "neutral",
  })[status] || "neutral";

// Map an e-invoice status to a Badge variant.
const eInvoiceVariant = (status) =>
  ({
    GENERATED: "success",
    FAILED: "destructive",
    CANCELLED: "warning",
    NA: "neutral",
  })[status] || "neutral";

function EInvoiceBadge({ status }) {
  if (!status || status === "PENDING")
    return <span className="text-sm text-muted-foreground">—</span>;
  return <Badge variant={eInvoiceVariant(status)}>{status}</Badge>;
}

const Invoices = () => {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [selectedInvoice, setSelectedInvoice] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

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
      const res = await endpoints.getInvoices();
      return res?.data?.data || [];
    },
  });
  const invoices = data || [];
  const error = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load invoices"
    : null;
  const { names: customerNames } = useCustomers();

  // Deep-link from Home's recent-invoices rows: /invoices with
  // { state: { openInvoiceId } } auto-opens that invoice's detail sheet once.
  useEffect(() => {
    const id = location.state?.openInvoiceId;
    if (!id || loading) return;
    const inv = invoices.find((i) => i.id === id);
    if (inv) {
      setSelectedInvoice(inv);
      setIsDetailOpen(true);
    }
    // Consume the state so back/refresh doesn't reopen it.
    navigate(location.pathname, { replace: true, state: null });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading, invoices]);

  const filteredInvoices = invoices.filter((inv) => {
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

  const handleRowClick = (invoice) => {
    setSelectedInvoice(invoice);
    setIsDetailOpen(true);
  };

  const closeDetail = () => {
    setIsDetailOpen(false);
    setTimeout(() => setSelectedInvoice(null), 300);
  };

  const columns = [
    {
      key: "invoice_number",
      header: "Number",
      cell: (inv) => (
        <span className="font-medium text-foreground">{inv.invoice_number}</span>
      ),
    },
    {
      key: "customer",
      header: "Customer",
      cell: (inv) =>
        customerNames[inv.customer_id] ? (
          <span className="text-sm text-foreground">{customerNames[inv.customer_id]}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">
            {inv.customer_id ? `${inv.customer_id.slice(0, 8)}…` : "—"}
          </span>
        ),
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (inv) => (
        <Money amountMinor={inv.total} currency={inv.currency} />
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (inv) => (
        <Badge variant={invoiceStatusVariant(inv.status)}>{inv.status}</Badge>
      ),
    },
    {
      key: "e_invoice",
      header: "E-Invoice",
      cell: (inv) => <EInvoiceBadge status={inv.e_invoice_status} />,
    },
    {
      key: "date",
      header: "Date",
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

      <DataTable
        columns={columns}
        data={filteredInvoices}
        loading={loading}
        error={error}
        onRetry={refetch}
        onRowClick={handleRowClick}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search invoices...",
        }}
        toolbar={
          <div className="flex flex-wrap gap-1.5">
            {STATUS_FILTERS.map((f) => (
              <button
                key={f.key}
                type="button"
                onClick={() => setStatusFilter(f.key)}
                className={cn(
                  "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
                  statusFilter === f.key
                    ? "border-emerald-200 bg-emerald-50 text-emerald-700"
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

      <InvoiceDetail
        invoice={selectedInvoice}
        isOpen={isDetailOpen}
        onClose={closeDetail}
        onChanged={() => queryClient.invalidateQueries({ queryKey: ["invoices"] })}
      />
    </div>
  );
};

export default Invoices;
