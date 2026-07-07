import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
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

const QuoteDetail = ({ quote, isOpen, onClose }) => {
  if (!quote) return null;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Quote details</SheetTitle>
          <p className="font-mono text-sm text-muted-foreground">ID: {quote.id}</p>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-6 py-6">
          <dl className="space-y-6">
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Total amount</dt>
              <dd className="mt-1 text-2xl font-bold tabular-nums text-foreground">
                {formatCurrency(quote.total_amount, quote.currency)}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Customer ID</dt>
              <dd className="mt-1 font-mono text-sm text-foreground">
                {quote.customer_id}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Valid until</dt>
              <dd className="mt-1 text-sm text-foreground">
                {formatDate(quote.valid_until)}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Status</dt>
              <dd className="mt-1">
                <Badge variant={quoteStatusVariant(quote.status)}>
                  {(quote.status || "").toUpperCase()}
                </Badge>
              </dd>
            </div>
          </dl>
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default QuoteDetail;
