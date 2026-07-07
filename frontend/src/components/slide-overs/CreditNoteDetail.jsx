import { formatCurrency } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

const CreditNoteDetail = ({ creditNote, isOpen, onClose }) => {
  if (!creditNote) return null;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Credit note details</SheetTitle>
          <p className="font-mono text-sm text-muted-foreground">ID: {creditNote.id}</p>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-6 py-6">
          <dl className="space-y-6">
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Total amount</dt>
              <dd className="mt-1 text-2xl font-bold tabular-nums text-foreground">
                {formatCurrency(creditNote.total, creditNote.currency)}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                Balance remaining
              </dt>
              <dd className="mt-1 text-lg font-semibold tabular-nums text-foreground">
                {formatCurrency(creditNote.balance, creditNote.currency)}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Customer ID</dt>
              <dd className="mt-1 font-mono text-sm text-foreground">
                {creditNote.customer_id}
              </dd>
            </div>
            {creditNote.reference && (
              <div>
                <dt className="text-sm font-medium text-muted-foreground">Reference</dt>
                <dd className="mt-1 text-sm text-foreground">{creditNote.reference}</dd>
              </div>
            )}
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Reason</dt>
              <dd className="mt-1 text-sm capitalize text-foreground">
                {creditNote.reason || "—"}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Status</dt>
              <dd className="mt-1">
                <Badge variant={creditNote.status === "active" ? "success" : "neutral"}>
                  {(creditNote.status || "").toUpperCase()}
                </Badge>
              </dd>
            </div>
          </dl>
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default CreditNoteDetail;
