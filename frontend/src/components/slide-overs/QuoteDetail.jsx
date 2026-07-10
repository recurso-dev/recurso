import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
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

const QuoteDetail = ({ quote, isOpen, onClose }) => {
  if (!quote) return null;

  const currency = quote.currency;
  const items = Array.isArray(quote.line_items) ? quote.line_items : [];
  // API field is `total` (not `total_amount`).
  const total = quote.total ?? quote.total_amount ?? 0;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-3">
            <span className="font-mono">{quote.quote_number || "Quote"}</span>
            <Badge variant={quoteStatusVariant(quote.status)} className="capitalize">
              {quote.status || "draft"}
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
                        {it.quantity} × {formatCurrency(it.unit_price, currency)}
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
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default QuoteDetail;
