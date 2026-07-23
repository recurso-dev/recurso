import { useState, useEffect } from "react";
import { RefreshCw, XCircle, FileDown, FileCode } from "lucide-react";

import { endpoints } from "../../lib/api";
import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const invoiceStatusVariant = (status) =>
  ({
    paid: "success",
    open: "info",
    overdue: "destructive",
    past_due: "destructive",
    void: "neutral",
    draft: "neutral",
  })[status] || "destructive";

const eInvoiceVariant = (status) =>
  ({
    GENERATED: "success",
    FAILED: "destructive",
    CANCELLED: "warning",
    NA: "neutral",
  })[status] || "neutral";

function EInvoiceStatusBadge({ status }) {
  if (!status || status === "PENDING")
    return <span className="text-sm text-muted-foreground">Pending</span>;
  return <Badge variant={eInvoiceVariant(status)}>{status}</Badge>;
}

function Field({ label, children }) {
  return (
    <div>
      <dt className="text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-sm text-foreground">{children}</dd>
    </div>
  );
}

// A single line in the amount breakdown box.
function Row({ label, value, strong, danger, border }) {
  return (
    <div
      className={
        "flex justify-between" +
        (border ? " border-t border-border pt-1.5" : "") +
        (strong
          ? " font-semibold text-foreground"
          : danger
            ? " font-medium text-red-600"
            : " text-muted-foreground")
      }
    >
      <span>{label}</span>
      <span className="tabular-nums">{value}</span>
    </div>
  );
}

// Map an EU e-invoice status (lowercase) to a Badge variant.
const euStatusVariant = (status) =>
  ({
    generated: "info",
    sent: "success",
    failed: "destructive",
  })[status] || "neutral";

const InvoiceDetail = ({ invoice, isOpen, onClose, onChanged }) => {
  const [retrying, setRetrying] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [cancelReason, setCancelReason] = useState("");
  const [cancelCode, setCancelCode] = useState(1);
  const [actionMessage, setActionMessage] = useState(null);
  const [euInvoice, setEuInvoice] = useState(null);
  const [euRetrying, setEuRetrying] = useState(false);

  // Fetch the invoice's EU e-invoice (EN 16931 / UBL) on open. It lives in its
  // own table, so it isn't on the invoice row — load it on demand. A tenant that
  // hasn't opted in (or a non-EU invoice) returns null and the section stays hidden.
  useEffect(() => {
    if (!isOpen || !invoice?.id) {
      setEuInvoice(null);
      return;
    }
    let cancelled = false;
    endpoints
      .getEUEInvoice(invoice.id)
      .then((res) => {
        if (!cancelled) setEuInvoice(res?.data?.data || null);
      })
      .catch(() => {
        if (!cancelled) setEuInvoice(null);
      });
    return () => {
      cancelled = true;
    };
  }, [isOpen, invoice?.id]);

  if (!invoice) return null;

  const handleEuRetry = async () => {
    setEuRetrying(true);
    setActionMessage(null);
    try {
      const res = await endpoints.retryEUEInvoice(invoice.id);
      setEuInvoice(res?.data?.data || null);
      setActionMessage({ type: "success", text: res?.data?.message || "EU e-invoice retried." });
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

  const handleRetry = async () => {
    setRetrying(true);
    setActionMessage(null);
    try {
      await endpoints.retryEInvoice(invoice.id);
      setActionMessage({ type: "success", text: "E-invoice retry initiated successfully." });
      if (onChanged) onChanged();
    } catch (err) {
      setActionMessage({
        type: "error",
        text: err?.response?.data?.error?.message || "Retry failed",
      });
    } finally {
      setRetrying(false);
    }
  };

  const handleCancel = async () => {
    setCancelling(true);
    setActionMessage(null);
    try {
      await endpoints.cancelEInvoice(invoice.id, {
        cancel_code: cancelCode,
        reason: cancelReason,
      });
      setActionMessage({ type: "success", text: "E-invoice cancelled successfully." });
      if (onChanged) onChanged();
      setShowCancelModal(false);
    } catch (err) {
      setActionMessage({
        type: "error",
        text: err?.response?.data?.error?.message || "Cancellation failed",
      });
    } finally {
      setCancelling(false);
    }
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

  const hasEInvoice =
    invoice.e_invoice_status &&
    invoice.e_invoice_status !== "NA" &&
    invoice.e_invoice_status !== "PENDING";

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Invoice details</SheetTitle>
          <div className="flex items-center gap-2">
            <p className="font-mono text-sm text-muted-foreground">
              {invoice.invoice_number}
            </p>
            <Badge variant={invoiceStatusVariant(invoice.status)}>
              {(invoice.status || "").toUpperCase()}
            </Badge>
          </div>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-6 py-6">
          <dl className="space-y-6">
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Amount</dt>
              <dd className="mt-1 text-2xl font-bold tabular-nums text-foreground">
                {formatCurrency(invoice.total, invoice.currency)}
              </dd>
            </div>

            {/* Line items — description, HSN, rate, amount (itemized tax) */}
            {Array.isArray(invoice.line_items) && invoice.line_items.length > 0 && (
              <div className="space-y-3">
                <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70">
                  Line items
                </p>
                <div className="space-y-2.5">
                  {invoice.line_items.map((li, i) => (
                    <div
                      key={li.id || i}
                      className="flex items-start justify-between gap-3"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm text-foreground">
                          {li.description || "Item"}
                        </p>
                        <p className="text-xs text-muted-foreground tabular-nums">
                          {li.quantity > 1 ? `${li.quantity} × ` : ""}
                          {li.hsn_code ? `HSN ${li.hsn_code}` : "—"}
                          {li.tax_rate ? ` · ${li.tax_rate}% GST` : ""}
                        </p>
                      </div>
                      <p className="shrink-0 tabular-nums text-sm text-foreground">
                        {formatCurrency(li.amount, invoice.currency)}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Amount breakdown: subtotal, GST split, total, paid, due */}
            <div className="space-y-1.5 rounded-md border border-border bg-stone-50 p-4 text-sm">
              <Row
                label="Subtotal"
                value={formatCurrency(invoice.subtotal, invoice.currency)}
              />
              {invoice.igst_amount > 0 && (
                <Row
                  label="IGST"
                  value={formatCurrency(invoice.igst_amount, invoice.currency)}
                />
              )}
              {invoice.cgst_amount > 0 && (
                <Row
                  label="CGST"
                  value={formatCurrency(invoice.cgst_amount, invoice.currency)}
                />
              )}
              {invoice.sgst_amount > 0 && (
                <Row
                  label="SGST"
                  value={formatCurrency(invoice.sgst_amount, invoice.currency)}
                />
              )}
              {!(
                invoice.igst_amount > 0 ||
                invoice.cgst_amount > 0 ||
                invoice.sgst_amount > 0
              ) &&
                invoice.tax_amount > 0 && (
                  <Row
                    label="Tax"
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

            <Field label="Customer ID">
              <span className="font-mono">{invoice.customer_id}</span>
            </Field>
            <Field label="Created at">
              {invoice.created_at ? new Date(invoice.created_at).toLocaleString() : "—"}
            </Field>
            <Field label="Due date">{formatDate(invoice.due_date)}</Field>
          </dl>

          {/* E-Invoice Section */}
          {hasEInvoice && (
            <>
              <Separator className="my-6" />
              <h3 className="mb-4 text-sm font-semibold text-foreground">E-Invoice</h3>
              <dl className="space-y-4">
                <div className="flex items-center justify-between">
                  <dt className="text-sm text-muted-foreground">Status</dt>
                  <dd>
                    <EInvoiceStatusBadge status={invoice.e_invoice_status} />
                  </dd>
                </div>
                {invoice.irn && (
                  <div>
                    <dt className="text-sm text-muted-foreground">IRN</dt>
                    <dd className="mt-1 break-all font-mono text-xs text-foreground">
                      {invoice.irn}
                    </dd>
                  </div>
                )}
                {invoice.ack_no && (
                  <div className="flex items-center justify-between">
                    <dt className="text-sm text-muted-foreground">Ack No</dt>
                    <dd className="font-mono text-sm text-foreground">{invoice.ack_no}</dd>
                  </div>
                )}
                {invoice.ack_date && (
                  <div className="flex items-center justify-between">
                    <dt className="text-sm text-muted-foreground">Ack Date</dt>
                    <dd className="text-sm text-foreground">{invoice.ack_date}</dd>
                  </div>
                )}
                {invoice.e_invoice_error_message && (
                  <div>
                    <dt className="text-sm text-muted-foreground">Error</dt>
                    <dd className="mt-1 text-sm text-red-600">
                      {invoice.e_invoice_error_message}
                    </dd>
                  </div>
                )}
              </dl>

              {actionMessage && (
                <div
                  className={
                    "mt-4 rounded-lg px-3 py-2 text-sm " +
                    (actionMessage.type === "success"
                      ? "bg-emerald-50 text-emerald-800"
                      : "bg-red-50 text-red-800")
                  }
                >
                  {actionMessage.text}
                </div>
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
                    onClick={() => setShowCancelModal(true)}
                    variant="destructive"
                    size="sm"
                  >
                    <XCircle className="h-4 w-4" />
                    Cancel IRN
                  </Button>
                )}
              </div>
            </>
          )}

          {/* EU E-Invoice Section (EN 16931 / UBL) — shown only when generated */}
          {euInvoice && (
            <>
              <Separator className="my-6" />
              <h3 className="mb-4 text-sm font-semibold text-foreground">EU e-invoice</h3>
              <dl className="space-y-4">
                <div className="flex items-center justify-between">
                  <dt className="text-sm text-muted-foreground">Status</dt>
                  <dd>
                    <Badge variant={euStatusVariant(euInvoice.status)}>
                      {(euInvoice.status || "").toUpperCase()}
                    </Badge>
                  </dd>
                </div>
                <div className="flex items-center justify-between">
                  <dt className="text-sm text-muted-foreground">Syntax</dt>
                  <dd className="text-sm text-foreground">
                    {(euInvoice.syntax || "ubl21").toUpperCase()}
                  </dd>
                </div>
                {euInvoice.message_id && (
                  <div>
                    <dt className="text-sm text-muted-foreground">Delivery ID</dt>
                    <dd className="mt-1 break-all font-mono text-xs text-foreground">
                      {euInvoice.message_id}
                    </dd>
                  </div>
                )}
                {euInvoice.status === "failed" && euInvoice.error_message && (
                  <div>
                    <dt className="text-sm text-muted-foreground">Error</dt>
                    <dd className="mt-1 text-sm text-red-600">{euInvoice.error_message}</dd>
                  </div>
                )}
              </dl>

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

              {actionMessage && (
                <div
                  className={
                    "mt-4 rounded-lg px-3 py-2 text-sm " +
                    (actionMessage.type === "success"
                      ? "bg-emerald-50 text-emerald-800"
                      : "bg-red-50 text-red-800")
                  }
                >
                  {actionMessage.text}
                </div>
              )}
            </>
          )}

          {/* Cancel IRN Modal */}
          {showCancelModal && (
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
                  onClick={handleCancel}
                  disabled={cancelling || !cancelReason}
                  variant="destructive"
                  size="sm"
                >
                  {cancelling ? "Cancelling..." : "Confirm cancel"}
                </Button>
                <Button
                  onClick={() => setShowCancelModal(false)}
                  variant="outline"
                  size="sm"
                >
                  Close
                </Button>
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="mt-8">
            <Button className="w-full" onClick={handleDownloadPdf}>
              <FileDown className="h-4 w-4" />
              Download PDF
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default InvoiceDetail;
