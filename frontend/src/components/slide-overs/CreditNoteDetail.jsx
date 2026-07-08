import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

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

const CreditNoteDetail = ({ creditNote, isOpen, onClose }) => {
  if (!creditNote) return null;

  const currency = creditNote.currency;
  // API field is `amount` (not `total`).
  const amount = creditNote.amount ?? creditNote.total ?? 0;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-3">
            Credit note
            <Badge variant={creditNote.status === "active" ? "success" : "neutral"}>
              {(creditNote.status || "").toUpperCase() || "—"}
            </Badge>
          </SheetTitle>
        </SheetHeader>

        <div className="space-y-6 px-6 py-6">
          <div className="flex items-end justify-between gap-4">
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Total amount
              </p>
              <p className="mt-1 text-2xl font-bold tabular-nums text-foreground">
                {formatCurrency(amount, currency)}
              </p>
            </div>
            <div className="text-right">
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Balance remaining
              </p>
              <p className="mt-1 text-lg font-semibold tabular-nums text-foreground">
                {formatCurrency(creditNote.balance, currency)}
              </p>
            </div>
          </div>

          <Separator />

          <dl className="space-y-5">
            <Field label="Customer ID" mono>
              {creditNote.customer_id}
            </Field>
            {creditNote.type && (
              <Field label="Type">
                <span className="capitalize">{creditNote.type}</span>
              </Field>
            )}
            <Field label="Reason">
              <span className="capitalize">{creditNote.reason || "—"}</span>
            </Field>
            {creditNote.reference && (
              <Field label="Reference">{creditNote.reference}</Field>
            )}
            {creditNote.refund_status && (
              <Field label="Refund status">
                <span className="capitalize">{creditNote.refund_status}</span>
              </Field>
            )}
            <Field label="Created">
              {creditNote.created_at ? formatDate(creditNote.created_at) : "—"}
            </Field>
          </dl>
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default CreditNoteDetail;
