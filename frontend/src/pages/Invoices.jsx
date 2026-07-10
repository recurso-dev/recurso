import { useState, useCallback, useEffect } from "react";
import { FileText } from "lucide-react";

import { endpoints } from "../lib/api";
import { useToast } from "../components/Toast";
import InvoiceDetail from "../components/slide-overs/InvoiceDetail";
import { formatCurrency, formatDate } from "@/lib/utils";
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
  const [invoices, setInvoices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [search, setSearch] = useState("");
  const [selectedInvoice, setSelectedInvoice] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const toast = useToast();

  const fetchInvoices = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await endpoints.getInvoices();
      setInvoices(response.data.data || []);
    } catch (err) {
      const msg =
        err?.response?.data?.error?.message || err?.message || "Failed to load invoices";
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    fetchInvoices();
  }, [fetchInvoices]);

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
      cell: (inv) => (
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
        onRetry={fetchInvoices}
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
