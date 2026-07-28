import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Send, Check, X, ArrowRight, Trash2 } from "lucide-react";

import { endpoints } from "../../lib/api";
import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

const quoteStatusVariant = (status) =>
  ({
    draft: "neutral",
    sent: "info",
    accepted: "success",
    declined: "destructive",
    expired: "warning",
  })[status] || "info";

const Field = ({ label, children, mono }) => (
  <div>
    <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
      {label}
    </dt>
    <dd className={`mt-1 text-sm text-foreground ${mono ? "font-mono" : ""}`}>
      {children}
    </dd>
  </div>
);

const QuoteDetail = ({ quote, isOpen, onClose, onChanged }) => {
  const queryClient = useQueryClient();
  // Optimistic local status so acting on a quote updates the badge and reveals
  // the next action (e.g. Accept → Convert) without reopening the sheet. Reset
  // whenever a different quote is opened.
  const [statusOverride, setStatusOverride] = useState(null);
  const [actionError, setActionError] = useState(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  useEffect(() => {
    setStatusOverride(null);
    setActionError(null);
    setConfirmDelete(false);
  }, [quote?.id]);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["quotes"] });
  const onActionError = (verb) => (err) => {
    setActionError(
      err?.response?.data?.error?.message || `Failed to ${verb} quote`,
    );
  };

  const sendMutation = useMutation({
    mutationFn: (id) => endpoints.sendQuote(id),
    onSuccess: () => {
      setStatusOverride("sent");
      invalidate();
      onChanged?.();
    },
    onError: onActionError("send"),
  });
  const acceptMutation = useMutation({
    mutationFn: (id) => endpoints.acceptQuote(id),
    onSuccess: () => {
      setStatusOverride("accepted");
      invalidate();
      onChanged?.();
    },
    onError: onActionError("accept"),
  });
  const declineMutation = useMutation({
    mutationFn: (id) => endpoints.declineQuote(id),
    onSuccess: () => {
      setStatusOverride("declined");
      invalidate();
      onChanged?.();
    },
    onError: onActionError("decline"),
  });
  const convertMutation = useMutation({
    mutationFn: (id) => endpoints.convertQuoteToInvoice(id),
    onSuccess: () => {
      invalidate();
      // A converted quote becomes an invoice — refresh that list and close.
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
      onChanged?.();
      onClose();
    },
    onError: onActionError("convert"),
  });
  const deleteMutation = useMutation({
    mutationFn: (id) => endpoints.deleteQuote(id),
    onSuccess: () => {
      invalidate();
      onChanged?.();
      setConfirmDelete(false);
      onClose();
    },
    onError: (err) => {
      setConfirmDelete(false);
      onActionError("delete")(err);
    },
  });

  if (!quote) return null;

  const currency = quote.currency;
  const items = Array.isArray(quote.line_items) ? quote.line_items : [];
  // API field is `total` (not `total_amount`).
  const total = quote.total ?? quote.total_amount ?? 0;

  const status = statusOverride || quote.status;
  const busy =
    sendMutation.isPending ||
    acceptMutation.isPending ||
    declineMutation.isPending ||
    convertMutation.isPending ||
    deleteMutation.isPending;
  const canSend = status === "draft" || status === "sent";
  const canDecide = status === "sent";
  const canConvert = status === "accepted" && !quote.invoice_id;
  // A quote that never became an invoice can be deleted; accepted/converted
  // ones are kept for the audit trail.
  const canDelete = status === "draft" || status === "declined";
  const hasActions = canSend || canDecide || canConvert || canDelete;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-3">
            <span className="font-mono">{quote.quote_number || "Quote"}</span>
            <Badge variant={quoteStatusVariant(status)} className="capitalize">
              {status || "draft"}
            </Badge>
          </SheetTitle>
        </SheetHeader>

        <div className="space-y-6 px-6 py-6">
          {/* Headline total */}
          <div>
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Total amount
            </p>
            <p className="mt-1 text-2xl font-bold tabular-nums text-foreground">
              {formatCurrency(total, currency)}
            </p>
          </div>

          <Separator />

          {/* Line items */}
          <div className="space-y-3">
            <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70">
              Line items
            </p>
            {items.length === 0 ? (
              <p className="text-sm text-muted-foreground">No line items.</p>
            ) : (
              <div className="space-y-2">
                {items.map((it, i) => (
                  <div
                    key={i}
                    className="flex items-start justify-between gap-3 text-sm"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-foreground">
                        {it.description || "Item"}
                      </p>
                      <p className="text-xs text-muted-foreground tabular-nums">
                        {it.quantity} ×{" "}
                        {formatCurrency(
                          // Fall back to amount ÷ quantity when unit_price is
                          // 0/missing (e.g. legacy or imported line items), so
                          // the subtitle never reads "1 × $0.00" against a
                          // non-zero amount.
                          it.unit_price ||
                            (it.quantity
                              ? Math.round(it.amount / it.quantity)
                              : it.amount),
                          currency,
                        )}
                      </p>
                    </div>
                    <p className="shrink-0 tabular-nums text-foreground">
                      {formatCurrency(it.amount, currency)}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Totals breakdown */}
          <div className="space-y-1.5 rounded-md border border-border bg-stone-50 p-4 text-sm">
            <div className="flex justify-between text-muted-foreground">
              <span>Subtotal</span>
              <span className="tabular-nums">
                {formatCurrency(quote.subtotal, currency)}
              </span>
            </div>
            {quote.discount_amount > 0 && (
              <div className="flex justify-between text-muted-foreground">
                <span>Discount</span>
                <span className="tabular-nums">
                  −{formatCurrency(quote.discount_amount, currency)}
                </span>
              </div>
            )}
            {quote.tax_amount > 0 && (
              <div className="flex justify-between text-muted-foreground">
                <span>Tax</span>
                <span className="tabular-nums">
                  {formatCurrency(quote.tax_amount, currency)}
                </span>
              </div>
            )}
            <div className="flex justify-between border-t border-border pt-1.5 font-semibold text-foreground">
              <span>Total</span>
              <span className="tabular-nums">{formatCurrency(total, currency)}</span>
            </div>
          </div>

          <Separator />

          {/* Meta */}
          <dl className="space-y-5">
            <Field label="Customer ID" mono>
              {quote.customer_id}
            </Field>
            <Field label="Valid until">
              {quote.valid_until ? formatDate(quote.valid_until) : "—"}
            </Field>
            <Field label="Created">
              {quote.created_at ? formatDate(quote.created_at) : "—"}
            </Field>
            {quote.notes && <Field label="Notes">{quote.notes}</Field>}
            {quote.terms && <Field label="Terms">{quote.terms}</Field>}
          </dl>

          {/* Actions — state-appropriate, mirroring the quote lifecycle. */}
          {hasActions && (
            <div className="space-y-3 border-t border-border pt-5">
              {actionError && (
                <p className="text-sm text-red-600" role="alert">
                  {actionError}
                </p>
              )}
              <div className="flex flex-wrap gap-2">
                {canSend && (
                  <Button
                    variant="outline"
                    disabled={busy}
                    onClick={() => sendMutation.mutate(quote.id)}
                  >
                    <Send className="h-4 w-4" />
                    {status === "sent" ? "Resend" : "Send"}
                  </Button>
                )}
                {canDecide && (
                  <Button
                    disabled={busy}
                    onClick={() => acceptMutation.mutate(quote.id)}
                  >
                    <Check className="h-4 w-4" />
                    Accept
                  </Button>
                )}
                {canDecide && (
                  <Button
                    variant="outline"
                    className="text-red-600 hover:text-red-700"
                    disabled={busy}
                    onClick={() => declineMutation.mutate(quote.id)}
                  >
                    <X className="h-4 w-4" />
                    Decline
                  </Button>
                )}
                {canConvert && (
                  <Button
                    disabled={busy}
                    onClick={() => convertMutation.mutate(quote.id)}
                  >
                    Convert to invoice
                    <ArrowRight className="h-4 w-4" />
                  </Button>
                )}
                {canDelete && (
                  <Button
                    variant="ghost"
                    className="text-red-600 hover:bg-red-50 hover:text-red-700"
                    disabled={busy}
                    onClick={() => setConfirmDelete(true)}
                  >
                    <Trash2 className="h-4 w-4" />
                    Delete
                  </Button>
                )}
              </div>
            </div>
          )}
        </div>

        <ConfirmDialog
          open={confirmDelete}
          onOpenChange={setConfirmDelete}
          title="Delete this quote?"
          description={`Quote ${quote.quote_number || ""} will be permanently removed. This can't be undone.`}
          confirmLabel="Delete quote"
          destructive
          busy={deleteMutation.isPending}
          onConfirm={() => deleteMutation.mutate(quote.id)}
        />
      </SheetContent>
    </Sheet>
  );
};

export default QuoteDetail;
