import { useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw, XCircle, FileDown, FileCode, Eye, Send } from "lucide-react";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import { cn, formatCurrency, formatDate, formatDateTime } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
} from "@/components/patterns/ObjectPage";
import { AuditTrail } from "@/components/patterns/AuditTrail";
import { ObjectTimeline } from "@/components/patterns/ObjectTimeline";
import { AttentionBanner } from "@/components/patterns/AttentionBanner";
import { JournalEntries } from "@/components/patterns/JournalEntries";
import { PaymentAttempts } from "@/components/patterns/PaymentAttempts";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Alert } from "@/components/ui/alert";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StatusBadge } from "@/components/ui/status-badge";
import { CopyableId } from "@/components/ui/copyable-id";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

// A single line in the amount breakdown box.
function Row({ label, value, strong, danger, border }) {
  return (
    <div
      className={cn(
        "flex justify-between",
        border && "border-t border-border pt-1.5",
        strong
          ? "font-semibold text-foreground"
          : danger
            ? "font-medium text-destructive"
            : "text-muted-foreground"
      )}
    >
      <span>{label}</span>
      <span className="tabular-nums">{value}</span>
    </div>
  );
}

/**
 * InvoicePage — the invoice's full object page at /invoices/:id
 * (DASHBOARD_REDESIGN.md Phase 5): identity header with send/preview/PDF
 * actions, amount breakdown + line items, statutory e-invoice sections
 * (India IRP, EU EN 16931/UBL), details + metadata + audit rail.
 */
export default function InvoicePage() {
  const { id } = useParams();
  const { names: customerNames } = useCustomers();

  const {
    data: invoice,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["invoice", id],
    queryFn: async () => (await endpoints.getInvoice(id)).data.data,
    enabled: Boolean(id),
  });

  // EU e-invoice (EN 16931 / UBL) lives in its own table; a tenant that
  // hasn't opted in (or a non-EU invoice) returns null → section hides.
  const { data: euInvoice, refetch: refetchEu } = useQuery({
    queryKey: ["euEInvoice", id],
    queryFn: async () => {
      try {
        return (await endpoints.getEUEInvoice(id)).data.data || null;
      } catch {
        return null;
      }
    },
    enabled: Boolean(id),
  });

  // The finance-accounting side: this invoice's ledger postings.
  const {
    data: journal = [],
    isLoading: journalLoading,
    error: journalError,
  } = useQuery({
    queryKey: ["invoiceJournal", id],
    queryFn: async () => (await endpoints.getInvoiceJournalEntries(id)).data.data?.entries || [],
    enabled: Boolean(id),
  });

  // The collection side: how we tried to get paid (attempt lifecycle + failures).
  const {
    data: attempts = [],
    isLoading: attemptsLoading,
    error: attemptsError,
  } = useQuery({
    queryKey: ["invoiceAttempts", id],
    queryFn: async () => (await endpoints.getInvoicePaymentAttempts(id)).data.data?.attempts || [],
    enabled: Boolean(id),
  });

  const [retrying, setRetrying] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [showCancelPanel, setShowCancelPanel] = useState(false);
  const [cancelReason, setCancelReason] = useState("");
  const [cancelCode, setCancelCode] = useState(1);
  const [actionMessage, setActionMessage] = useState(null);
  const [euRetrying, setEuRetrying] = useState(false);
  const [previewHtml, setPreviewHtml] = useState(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const [confirmSend, setConfirmSend] = useState(false);

  if (isLoading) {
    return (
      <div aria-busy="true">
        <Skeleton className="mb-2 h-4 w-24" />
        <Skeleton className="mb-6 h-8 w-64" />
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <Skeleton className="h-64 lg:col-span-2" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (error || !invoice) {
    const status = error?.response?.status;
    return (
      <ErrorState
        title={status === 404 ? "Invoice not found" : "Couldn't load this invoice"}
        message={
          status === 404
            ? "This invoice doesn't exist or was removed."
            : error?.response?.data?.error?.message || error?.message
        }
        onRetry={status === 404 ? undefined : refetch}
      />
    );
  }

  // Presentation regime — the backend stamps invoice.tax_regime from the
  // seller's jurisdiction; fall back to the invoice's own data. Non-GST
  // invoices hide every GST artifact (HSN, CGST/SGST/IGST, TDS, IRN).
  const taxRegime =
    invoice.tax_regime ||
    (invoice.igst_amount > 0 ||
    invoice.cgst_amount > 0 ||
    invoice.sgst_amount > 0 ||
    String(invoice.currency || "").toUpperCase() === "INR"
      ? "gst"
      : "plain");
  const isGST = taxRegime === "gst";
  const taxLineLabel =
    taxRegime === "sales_tax" ? "Sales Tax" : taxRegime === "vat" ? "VAT" : "Tax";

  const hasEInvoice =
    invoice.e_invoice_status &&
    invoice.e_invoice_status !== "NA" &&
    invoice.e_invoice_status !== "PENDING";

  const handleRetry = async () => {
    setRetrying(true);
    setActionMessage(null);
    try {
      await endpoints.retryEInvoice(invoice.id);
      setActionMessage({ type: "success", text: "E-invoice retry initiated successfully." });
      refetch();
    } catch (err) {
      setActionMessage({
        type: "error",
        text: err?.response?.data?.error?.message || "Retry failed",
      });
    } finally {
      setRetrying(false);
    }
  };

  const handleCancelIrn = async () => {
    setCancelling(true);
    setActionMessage(null);
    try {
      await endpoints.cancelEInvoice(invoice.id, {
        cancel_code: cancelCode,
        reason: cancelReason,
      });
      setActionMessage({ type: "success", text: "E-invoice cancelled successfully." });
      setShowCancelPanel(false);
      refetch();
    } catch (err) {
      setActionMessage({
        type: "error",
        text: err?.response?.data?.error?.message || "Cancellation failed",
      });
    } finally {
      setCancelling(false);
    }
  };

  const handleEuRetry = async () => {
    setEuRetrying(true);
    setActionMessage(null);
    try {
      const res = await endpoints.retryEUEInvoice(invoice.id);
      setActionMessage({
        type: "success",
        text: res?.data?.message || "EU e-invoice retried.",
      });
      refetchEu();
    } catch (err) {
      setActionMessage({
        type: "error",
        text: err?.response?.data?.error?.message || "EU e-invoice retry failed",
      });
    } finally {
      setEuRetrying(false);
    }
  };

  const handleDownloadUbl = () => {
    if (!euInvoice?.document) return;
    const blob = new Blob([euInvoice.document], { type: "application/xml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${invoice.invoice_number || invoice.id}-ubl.xml`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
  };

  const handleDownloadPdf = async () => {
    setActionMessage(null);
    try {
      const res = await endpoints.getInvoicePdf(invoice.id);
      const url = URL.createObjectURL(res.data);
      window.open(url, "_blank", "noreferrer");
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (err) {
      setActionMessage({
        type: "error",
        text: err?.response?.data?.error?.message || "PDF download failed",
      });
    }
  };

  const handleSend = async () => {
    setConfirmSend(false);
    setSending(true);
    setActionMessage(null);
    try {
      const res = await endpoints.sendInvoice(invoice.id);
      setActionMessage({ type: "success", text: res?.data?.message || "Invoice email sent." });
    } catch (err) {
      setActionMessage({
        type: "error",
        text: err?.response?.data?.error?.message || "Failed to send invoice email",
      });
    } finally {
      setSending(false);
    }
  };

  const handlePreview = async () => {
    setActionMessage(null);
    setPreviewLoading(true);
    setPreviewHtml(""); // opens the dialog in a loading state
    try {
      const res = await endpoints.getInvoicePreview(invoice.id);
      setPreviewHtml(typeof res.data === "string" ? res.data : "");
    } catch (err) {
      setPreviewHtml(null);
      setActionMessage({
        type: "error",
        text: err?.response?.data?.error?.message || "Preview failed",
      });
    } finally {
      setPreviewLoading(false);
    }
  };

  // Layer 3 — why is it in this state? Grounded in the invoice's own dunning
  // fields; silent when the invoice is healthy.
  const attention = [];
  if (invoice.status === "past_due") {
    attention.push({
      tone: "danger",
      text: (
        <>
          Payment past due
          {invoice.last_payment_error ? `: ${invoice.last_payment_error}` : ""}
          {invoice.next_retry_at ? ` — next retry ${formatDate(invoice.next_retry_at)}` : ""}.
        </>
      ),
    });
  } else if (invoice.status === "uncollectible") {
    attention.push({
      tone: "danger",
      text: "Written off as uncollectible — the receivable was reversed in the ledger.",
    });
  } else if (invoice.status === "void") {
    attention.push({ tone: "warning", text: "This invoice is void — it can no longer be paid." });
  }

  return (
    <div>
      <ObjectHeader
        backTo="/invoices"
        backLabel="Invoices"
        kicker="Invoice"
        title={invoice.invoice_number || invoice.id.slice(0, 8)}
        badge={<StatusBadge status={invoice.status || "unknown"} />}
        meta={
          <>
            <CopyableId value={invoice.id} />
            {invoice.due_date && <span>Due {formatDate(invoice.due_date)}</span>}
          </>
        }
        actions={
          <>
            <Button variant="outline" onClick={() => setConfirmSend(true)} disabled={sending}>
              <Send className="h-4 w-4" />
              {sending ? "Sending…" : "Send"}
            </Button>
            <Button variant="outline" onClick={handlePreview} disabled={previewLoading}>
              <Eye className="h-4 w-4" />
              {previewLoading ? "Loading…" : "Preview"}
            </Button>
            <Button onClick={handleDownloadPdf}>
              <FileDown className="h-4 w-4" />
              Download PDF
            </Button>
          </>
        }
      />

      <AttentionBanner items={attention} />

      {actionMessage && (
        <Alert
          variant={actionMessage.type === "success" ? "success" : "danger"}
          className="mb-6"
        >
          {actionMessage.text}
        </Alert>
      )}

      <ObjectPageLayout
        rail={
          <>
            <ObjectSection title="Details">
              <AttributeList
                columns={1}
                items={[
                  {
                    label: "Customer",
                    value: invoice.customer_id ? (
                      <Link
                        to={`/customers/${invoice.customer_id}`}
                        className="text-primary underline-offset-2 hover:underline"
                      >
                        {customerNames[invoice.customer_id] ||
                          `${String(invoice.customer_id).slice(0, 8)}…`}
                      </Link>
                    ) : null,
                  },
                  {
                    label: "Subscription",
                    value: invoice.subscription_id ? (
                      <Link
                        to={`/subscriptions/${invoice.subscription_id}`}
                        className="font-mono text-xs text-primary underline-offset-2 hover:underline"
                      >
                        {String(invoice.subscription_id).slice(0, 8)}…
                      </Link>
                    ) : null,
                  },
                  { label: "Created", value: formatDateTime(invoice.created_at) },
                  { label: "Due date", value: formatDate(invoice.due_date) },
                  {
                    label: "Billing reason",
                    value: invoice.billing_reason ? (
                      <span className="capitalize">
                        {invoice.billing_reason.replace(/_/g, " ")}
                      </span>
                    ) : null,
                  },
                  { label: "Invoice ID", value: <CopyableId value={invoice.id} /> },
                ]}
              />
            </ObjectSection>
            <ObjectSection title="Timeline">
              <ObjectTimeline objectId={invoice.id} />
            </ObjectSection>
            <ObjectSection title="Audit trail">
              <AuditTrail entityType="invoices" entityId={invoice.id} />
            </ObjectSection>
          </>
        }
      >
        <ObjectSection title="Amount">
          <p className="mb-4 text-3xl font-bold tabular-nums text-foreground">
            {formatCurrency(invoice.total, invoice.currency)}
          </p>

          {Array.isArray(invoice.line_items) && invoice.line_items.length > 0 && (
            <div className="mb-4 space-y-3">
              <p className="text-xs font-semibold uppercase tracking-wider text-subtle">
                Line items
              </p>
              <div className="space-y-2.5">
                {invoice.line_items.map((li, i) => (
                  <div key={li.id || i} className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm text-foreground">
                        {li.description || "Item"}
                      </p>
                      <p className="text-xs tabular-nums text-muted-foreground">
                        {li.quantity > 1 ? `${li.quantity} × ` : ""}
                        {isGST
                          ? `${li.hsn_code ? `HSN ${li.hsn_code}` : "—"}${
                              li.tax_rate ? ` · ${li.tax_rate}% GST` : ""
                            }`
                          : li.tax_rate
                            ? `${li.tax_rate}% ${taxLineLabel}`
                            : ""}
                      </p>
                    </div>
                    <p className="shrink-0 text-sm tabular-nums text-foreground">
                      {formatCurrency(li.amount, invoice.currency)}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="space-y-1.5 rounded-md border border-border bg-muted p-4 text-sm">
            <Row label="Subtotal" value={formatCurrency(invoice.subtotal, invoice.currency)} />
            {invoice.igst_amount > 0 && (
              <Row label="IGST" value={formatCurrency(invoice.igst_amount, invoice.currency)} />
            )}
            {invoice.cgst_amount > 0 && (
              <Row label="CGST" value={formatCurrency(invoice.cgst_amount, invoice.currency)} />
            )}
            {invoice.sgst_amount > 0 && (
              <Row label="SGST" value={formatCurrency(invoice.sgst_amount, invoice.currency)} />
            )}
            {!(
              invoice.igst_amount > 0 ||
              invoice.cgst_amount > 0 ||
              invoice.sgst_amount > 0
            ) &&
              invoice.tax_amount > 0 && (
                <Row
                  label={taxLineLabel}
                  value={formatCurrency(invoice.tax_amount, invoice.currency)}
                />
              )}
            {invoice.tds_amount > 0 && (
              <Row
                label="TDS withheld"
                value={`−${formatCurrency(invoice.tds_amount, invoice.currency)}`}
              />
            )}
            <Row
              label="Total"
              value={formatCurrency(invoice.total, invoice.currency)}
              strong
              border
            />
            {invoice.credit_applied > 0 && (
              <Row
                label="Credit applied"
                value={formatCurrency(invoice.credit_applied, invoice.currency)}
              />
            )}
            <Row
              label="Amount paid"
              value={formatCurrency(invoice.amount_paid, invoice.currency)}
            />
            <Row
              label="Amount due"
              value={formatCurrency(invoice.amount_due, invoice.currency)}
              strong={!(invoice.amount_due > 0)}
              danger={invoice.amount_due > 0}
            />
          </div>
        </ObjectSection>

        {/* The collection side: how we tried to get paid. Shown only when there
            were gateway attempts (credit/offline invoices have none). */}
        {(attemptsLoading || attempts.length > 0) && (
          <ObjectSection
            title="Payments"
            action={
              <span className="text-xs text-muted-foreground">Attempt lifecycle &amp; failures</span>
            }
          >
            <PaymentAttempts
              attempts={attempts}
              currency={invoice.currency}
              isLoading={attemptsLoading}
              error={attemptsError}
            />
          </ObjectSection>
        )}

        {/* The finance-accounting side of the same invoice: its ledger postings. */}
        <ObjectSection
          title="Journal entries"
          action={
            <span className="text-xs text-muted-foreground">
              What this invoice posted to the ledger
            </span>
          }
        >
          <JournalEntries
            entries={journal}
            currency={invoice.currency}
            isLoading={journalLoading}
            error={journalError}
          />
        </ObjectSection>

        {/* India IRP e-invoice — never on a non-GST invoice */}
        {hasEInvoice && isGST && (
          <ObjectSection
            title="E-Invoice (IRP)"
            action={<StatusBadge status={invoice.e_invoice_status} />}
          >
            <AttributeList
              columns={1}
              items={[
                {
                  label: "IRN",
                  value: invoice.irn ? (
                    <span className="break-all font-mono text-xs">{invoice.irn}</span>
                  ) : null,
                },
                {
                  label: "Ack No",
                  value: invoice.ack_no ? (
                    <span className="font-mono">{invoice.ack_no}</span>
                  ) : null,
                },
                { label: "Ack Date", value: invoice.ack_date },
              ]}
            />
            {invoice.e_invoice_error_message && (
              <Alert variant="danger" className="mt-4">
                {invoice.e_invoice_error_message}
              </Alert>
            )}
            <div className="mt-4 flex gap-3">
              {invoice.e_invoice_status === "FAILED" && (
                <Button onClick={handleRetry} disabled={retrying} size="sm">
                  <RefreshCw className="h-4 w-4" />
                  {retrying ? "Retrying..." : "Retry"}
                </Button>
              )}
              {invoice.e_invoice_status === "GENERATED" && (
                <Button
                  onClick={() => setShowCancelPanel((s) => !s)}
                  variant="destructive"
                  size="sm"
                >
                  <XCircle className="h-4 w-4" />
                  Cancel IRN
                </Button>
              )}
            </div>

            {showCancelPanel && (
              <div className="mt-4 space-y-3 rounded-lg border border-border bg-muted/40 p-4">
                <h4 className="text-sm font-medium text-foreground">Cancel IRN</h4>
                <div className="space-y-1.5">
                  <Label className="text-xs text-muted-foreground">Cancel reason</Label>
                  <Select
                    value={String(cancelCode)}
                    onValueChange={(v) => setCancelCode(Number(v))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1">Duplicate</SelectItem>
                      <SelectItem value="2">Data Entry Mistake</SelectItem>
                      <SelectItem value="3">Order Cancelled</SelectItem>
                      <SelectItem value="4">Others</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs text-muted-foreground">Remarks</Label>
                  <Input
                    type="text"
                    value={cancelReason}
                    onChange={(e) => setCancelReason(e.target.value)}
                    placeholder="Enter reason for cancellation"
                  />
                </div>
                <div className="flex gap-2">
                  <Button
                    onClick={handleCancelIrn}
                    disabled={cancelling || !cancelReason}
                    variant="destructive"
                    size="sm"
                  >
                    {cancelling ? "Cancelling..." : "Confirm cancel"}
                  </Button>
                  <Button onClick={() => setShowCancelPanel(false)} variant="outline" size="sm">
                    Close
                  </Button>
                </div>
              </div>
            )}
          </ObjectSection>
        )}

        {/* EU e-invoice (EN 16931 / UBL) — shown only when generated */}
        {euInvoice && (
          <ObjectSection
            title="EU e-invoice"
            action={<StatusBadge status={euInvoice.status} />}
          >
            <AttributeList
              columns={1}
              items={[
                { label: "Syntax", value: (euInvoice.syntax || "ubl21").toUpperCase() },
                {
                  label: "Delivery ID",
                  value: euInvoice.message_id ? (
                    <span className="break-all font-mono text-xs">{euInvoice.message_id}</span>
                  ) : null,
                },
              ]}
            />
            {euInvoice.status === "failed" && euInvoice.error_message && (
              <Alert variant="danger" className="mt-4">
                {euInvoice.error_message}
              </Alert>
            )}
            <div className="mt-4 flex flex-wrap gap-3">
              {euInvoice.document && (
                <Button onClick={handleDownloadUbl} variant="outline" size="sm">
                  <FileCode className="h-4 w-4" />
                  Download UBL
                </Button>
              )}
              {euInvoice.status === "failed" && (
                <Button onClick={handleEuRetry} disabled={euRetrying} size="sm">
                  <RefreshCw className="h-4 w-4" />
                  {euRetrying ? "Retrying…" : "Retry"}
                </Button>
              )}
            </div>
          </ObjectSection>
        )}
      </ObjectPageLayout>

      <ConfirmDialog
        open={confirmSend}
        onOpenChange={setConfirmSend}
        title="Send this invoice to the customer?"
        description="The invoice is emailed to the customer's billing address."
        confirmLabel="Send invoice"
        busy={sending}
        onConfirm={handleSend}
      />

      {/* HTML invoice preview */}
      <Dialog open={previewHtml !== null} onOpenChange={(o) => !o && setPreviewHtml(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Invoice {invoice.invoice_number}</DialogTitle>
          </DialogHeader>
          <div className="h-[70vh] overflow-hidden rounded-md border border-border bg-white">
            {previewHtml ? (
              <iframe
                title="Invoice preview"
                srcDoc={previewHtml}
                className="h-full w-full"
                sandbox=""
              />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
                Loading preview…
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
