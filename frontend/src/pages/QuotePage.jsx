import { useState } from "react";
import { Link, useParams, useNavigate } from "react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Send, Check, X, ArrowRight, Trash2, Pencil } from "lucide-react";

import { endpoints } from "../lib/api";
import { useObjectQuery } from "@/lib/useObjectQuery";
import { formatDateTime } from "@/lib/utils";
import { useCustomers } from "@/lib/useCustomers";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedRow,
  ObjectPageSkeleton,
  ObjectNotFound,
  ObjectPageError,
} from "@/components/patterns/ObjectPage";
import { AttentionBanner } from "@/components/patterns/AttentionBanner";
import { StatusBadge } from "@/components/ui/status-badge";
import { Money } from "@/components/ui/money";
import { CopyableId } from "@/components/ui/copyable-id";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { toast } from "@/components/ui/sonner";

/**
 * QuotePage — one price quote as a first-class object at /quotes/:id. Replaces
 * the detail slide-over: the quote's line items and totals math, its lifecycle
 * (draft → sent → accepted/declined → converted), the customer and the invoice
 * it converted into (linked), and the full action set (edit/send/accept/decline/
 * convert/delete) with the money-moving ones (convert) gated by a confirm.
 */
export default function QuotePage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { names } = useCustomers();

  const [confirm, setConfirm] = useState(null); // "convert" | "delete"

  const {
    object: quote,
    loading,
    notFound,
    isError,
    error: quoteError,
    refetch,
  } = useObjectQuery(
    ["quote", id],
    async () => (await endpoints.getQuote(id)).data.data,
    { enabled: Boolean(id) }
  );

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["quote", id] });
    queryClient.invalidateQueries({ queryKey: ["quotes"] });
  };
  const onErr = (verb) => (err) =>
    toast.error(err?.response?.data?.error?.message || `Failed to ${verb} quote`);

  const sendMutation = useMutation({
    mutationFn: () => endpoints.sendQuote(id),
    onSuccess: () => {
      toast.success("Quote sent.");
      invalidate();
    },
    onError: onErr("send"),
  });
  const acceptMutation = useMutation({
    mutationFn: () => endpoints.acceptQuote(id),
    onSuccess: () => {
      toast.success("Quote accepted.");
      invalidate();
    },
    onError: onErr("accept"),
  });
  const declineMutation = useMutation({
    mutationFn: () => endpoints.declineQuote(id),
    onSuccess: () => {
      toast.success("Quote declined.");
      invalidate();
    },
    onError: onErr("decline"),
  });
  const convertMutation = useMutation({
    mutationFn: () => endpoints.convertQuoteToInvoice(id),
    onSuccess: (res) => {
      toast.success("Quote converted to an invoice.");
      setConfirm(null);
      invalidate();
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
      const invId = res?.data?.data?.invoice_id || res?.data?.data?.id;
      if (invId) navigate(`/invoices/${invId}`);
    },
    onError: (err) => {
      setConfirm(null);
      onErr("convert")(err);
    },
  });
  const deleteMutation = useMutation({
    mutationFn: () => endpoints.deleteQuote(id),
    onSuccess: () => {
      toast.success("Quote deleted.");
      queryClient.invalidateQueries({ queryKey: ["quotes"] });
      navigate("/quotes");
    },
    onError: (err) => {
      setConfirm(null);
      onErr("delete")(err);
    },
  });

  if (loading) return <ObjectPageSkeleton />;
  if (notFound) {
    return (
      <ObjectNotFound
        objectLabel="quote"
        identifier={id ? String(id).slice(0, 8) : undefined}
        backTo="/quotes"
        backLabel="Quotes"
      />
    );
  }
  if (isError) {
    return <ObjectPageError objectLabel="quote" error={quoteError} onRetry={refetch} backTo="/quotes" backLabel="Quotes" />;
  }

  const currency = quote.currency;
  const items = Array.isArray(quote.line_items) ? quote.line_items : [];
  const status = quote.status;
  const busy =
    sendMutation.isPending ||
    acceptMutation.isPending ||
    declineMutation.isPending ||
    convertMutation.isPending ||
    deleteMutation.isPending;

  const canSend = status === "draft" || status === "sent";
  const canDecide = status === "sent";
  const canConvert = status === "accepted" && !quote.invoice_id;
  const canEdit = status === "draft";
  const canDelete = status === "draft" || status === "declined";

  const expired =
    quote.valid_until &&
    new Date(quote.valid_until).getTime() < Date.now() &&
    !["accepted", "declined"].includes(status) &&
    !quote.invoice_id;

  const attention = [];
  if (quote.invoice_id) {
    attention.push({
      tone: "warning",
      text: "This quote was converted to an invoice — it's now locked.",
      to: `/invoices/${quote.invoice_id}`,
    });
  } else if (canConvert) {
    attention.push({
      tone: "warning",
      text: "Accepted and awaiting conversion — convert it to raise the invoice.",
    });
  } else if (expired) {
    attention.push({
      tone: "warning",
      text: `This quote expired on ${formatDateTime(quote.valid_until)}.`,
    });
  }

  const unitPriceOf = (it) =>
    it.unit_price || (it.quantity ? Math.round(it.amount / it.quantity) : it.amount);

  return (
    <div>
      <ObjectHeader
        backTo="/quotes"
        backLabel="Quotes"
        kicker="Quote"
        title={quote.quote_number || "Quote"}
        badge={<StatusBadge status={status} />}
        meta={
          <>
            <span className="tabular-nums font-medium text-foreground">
              <Money amountMinor={quote.total} currency={currency} />
            </span>
            <CustomerName id={quote.customer_id} names={names} link={false} />
            <CopyableId value={quote.id} />
          </>
        }
        actions={
          <>
            {canEdit && (
              <Button variant="outline" disabled={busy} onClick={() => navigate(`/quotes/${id}/edit`)}>
                <Pencil className="h-4 w-4" />
                Edit
              </Button>
            )}
            {canSend && (
              <Button variant="outline" disabled={busy} onClick={() => sendMutation.mutate()}>
                <Send className="h-4 w-4" />
                {status === "sent" ? "Resend" : "Send"}
              </Button>
            )}
            {canDecide && (
              <Button disabled={busy} onClick={() => acceptMutation.mutate()}>
                <Check className="h-4 w-4" />
                Accept
              </Button>
            )}
            {canDecide && (
              <Button
                variant="outline"
                className="text-destructive hover:text-destructive"
                disabled={busy}
                onClick={() => declineMutation.mutate()}
              >
                <X className="h-4 w-4" />
                Decline
              </Button>
            )}
            {canConvert && (
              <Button disabled={busy} onClick={() => setConfirm("convert")}>
                Convert to invoice
                <ArrowRight className="h-4 w-4" />
              </Button>
            )}
            {canDelete && (
              <Button
                variant="ghost"
                className="text-destructive hover:text-destructive"
                disabled={busy}
                onClick={() => setConfirm("delete")}
              >
                <Trash2 className="h-4 w-4" />
                Delete
              </Button>
            )}
          </>
        }
      />

      <AttentionBanner items={attention} className="mb-6" />

      <ObjectPageLayout
        rail={
          <ObjectSection title="Details">
            <AttributeList
              columns={1}
              items={[
                { label: "Quote ID", value: <CopyableId value={quote.id} /> },
                {
                  label: "Customer",
                  value: (
                    <Link
                      to={`/customers/${quote.customer_id}`}
                      className="text-primary hover:underline"
                    >
                      <CustomerName id={quote.customer_id} names={names} link={false} />
                    </Link>
                  ),
                },
                { label: "Currency", value: currency },
                {
                  label: "Valid until",
                  value: quote.valid_until ? formatDateTime(quote.valid_until) : "—",
                },
                { label: "Created", value: formatDateTime(quote.created_at) },
                ...(quote.accepted_at
                  ? [{ label: "Accepted", value: formatDateTime(quote.accepted_at) }]
                  : []),
                ...(quote.declined_at
                  ? [{ label: "Declined", value: formatDateTime(quote.declined_at) }]
                  : []),
              ]}
            />
          </ObjectSection>
        }
      >
        <ObjectSection title={`Line items${items.length ? ` (${items.length})` : ""}`} flush>
          {items.length === 0 ? (
            <p className="px-6 py-4 text-sm text-muted-foreground">No line items.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[560px] text-sm" aria-label="Quote line items">
                <thead>
                  <tr className="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wide text-subtle">
                    <th scope="col" className="px-6 py-2.5 font-medium">Description</th>
                    <th scope="col" className="px-3 py-2.5 text-right font-medium">Qty</th>
                    <th scope="col" className="px-3 py-2.5 text-right font-medium">Unit price</th>
                    <th scope="col" className="px-6 py-2.5 text-right font-medium">Amount</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {items.map((it, i) => (
                    <tr key={i} className="hover:bg-muted/20">
                      <td className="px-6 py-2.5 text-foreground">{it.description || "Item"}</td>
                      <td className="px-3 py-2.5 text-right tabular-nums text-muted-foreground">
                        {it.quantity}
                      </td>
                      <td className="px-3 py-2.5 text-right font-mono tabular-nums text-muted-foreground">
                        <Money amountMinor={unitPriceOf(it)} currency={currency} />
                      </td>
                      <td className="px-6 py-2.5 text-right font-mono tabular-nums text-foreground">
                        <Money amountMinor={it.amount} currency={currency} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </ObjectSection>

        <ObjectSection title="Totals">
          <dl className="space-y-1.5 text-sm">
            <div className="flex justify-between text-muted-foreground">
              <dt>Subtotal</dt>
              <dd className="tabular-nums"><Money amountMinor={quote.subtotal} currency={currency} /></dd>
            </div>
            {quote.discount_amount > 0 && (
              <div className="flex justify-between text-muted-foreground">
                <dt>Discount</dt>
                <dd className="tabular-nums">−<Money amountMinor={quote.discount_amount} currency={currency} /></dd>
              </div>
            )}
            {quote.tax_amount > 0 && (
              <div className="flex justify-between text-muted-foreground">
                <dt>Tax</dt>
                <dd className="tabular-nums"><Money amountMinor={quote.tax_amount} currency={currency} /></dd>
              </div>
            )}
            <div className="flex justify-between border-t border-border pt-1.5 font-semibold text-foreground">
              <dt>Total</dt>
              <dd className="tabular-nums"><Money amountMinor={quote.total} currency={currency} /></dd>
            </div>
          </dl>
        </ObjectSection>

        {(quote.notes || quote.terms) && (
          <ObjectSection title="Notes & terms">
            <AttributeList
              columns={1}
              items={[
                ...(quote.notes ? [{ label: "Notes", value: quote.notes }] : []),
                ...(quote.terms ? [{ label: "Terms", value: quote.terms }] : []),
              ]}
            />
          </ObjectSection>
        )}

        <ObjectSection title="Related">
          <RelatedRow to={`/customers/${quote.customer_id}`}>
            <span className="text-foreground">
              <CustomerName id={quote.customer_id} names={names} link={false} />
            </span>
            <span className="text-xs text-muted-foreground">Customer →</span>
          </RelatedRow>
          {quote.invoice_id && (
            <RelatedRow to={`/invoices/${quote.invoice_id}`}>
              <span className="text-foreground">Converted invoice</span>
              <span className="text-xs text-muted-foreground">View invoice →</span>
            </RelatedRow>
          )}
        </ObjectSection>
      </ObjectPageLayout>

      <ConfirmDialog
        open={confirm === "convert"}
        onOpenChange={(o) => !o && setConfirm(null)}
        title="Convert this quote to an invoice?"
        description="An invoice is created for the quoted amount and the quote is locked. This can't be undone."
        confirmLabel="Convert to invoice"
        busy={convertMutation.isPending}
        onConfirm={() => convertMutation.mutate()}
      />
      <ConfirmDialog
        open={confirm === "delete"}
        onOpenChange={(o) => !o && setConfirm(null)}
        title="Delete this quote?"
        description={`Quote ${quote.quote_number || ""} will be permanently removed. This can't be undone.`}
        confirmLabel="Delete quote"
        destructive
        busy={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}
