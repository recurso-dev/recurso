import { useState, useEffect } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { FileText } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import InvoiceDetail from "../components/slide-overs/InvoiceDetail";
import { formatDate } from "@/lib/utils";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";

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
  const [selectedInvoice, setSelectedInvoice] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();

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
    if (!search) return true;
    const s = search.toLowerCase();
    return (
      inv.invoice_number?.toLowerCase().includes(s) ||
      inv.customer_id?.toLowerCase().includes(s) ||
      inv.status?.toLowerCase().includes(s)
    );
  });

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
      <PageHeader title="Invoices" description="View and manage customer invoices." />

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
        empty={{
          icon: FileText,
          title: "No invoices yet",
          description: "Invoices will appear here once subscriptions are billed.",
        }}
      />

      <InvoiceDetail
        invoice={selectedInvoice}
        isOpen={isDetailOpen}
        onClose={closeDetail}
      />
    </div>
  );
};

export default Invoices;
