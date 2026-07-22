import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { endpoints } from "../../lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { Button } from "@/components/ui/button";
import { Check, X } from "lucide-react";
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
  const { user } = useAuth();
  const queryClient = useQueryClient();

  const approveMutation = useMutation({
    mutationFn: () => endpoints.approveCreditNote(creditNote.id),
    onSuccess: () => {
      toast.success("Credit note approved successfully.");
      queryClient.invalidateQueries(["credit-notes"]);
      onClose();
    },
    onError: (err) => {
      toast.error(err?.response?.data?.error?.message || "Failed to approve credit note.");
    },
  });

  const rejectMutation = useMutation({
    mutationFn: () => endpoints.rejectCreditNote(creditNote.id),
    onSuccess: () => {
      toast.success("Credit note rejected.");
      queryClient.invalidateQueries(["credit-notes"]);
      onClose();
    },
    onError: (err) => {
      toast.error(err?.response?.data?.error?.message || "Failed to reject credit note.");
    },
  });

  if (!creditNote) return null;

  const currency = creditNote.currency;
  // API field is `amount` (not `total`).
  const amount = creditNote.amount ?? creditNote.total ?? 0;
  const isPending = creditNote.status === "pending_approval";
  const canApprove = user?.role === "admin" || user?.role === "owner";

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-3">
            Credit note
            <Badge variant={creditNote.status === "active" ? "success" : creditNote.status === "pending_approval" ? "warning" : "neutral"}>
              {(creditNote.status || "").replace("_", " ").toUpperCase() || "—"}
            </Badge>
          </SheetTitle>
        </SheetHeader>

        <div className="space-y-6 px-6 py-6">
          {isPending && canApprove && (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-4">
              <h3 className="text-sm font-medium text-amber-800">Approval Required</h3>
              <p className="mt-1 text-sm text-amber-700">
                This credit note is pending review. You can approve it to issue the credit or refund, or reject it.
              </p>
              <div className="mt-4 flex gap-3">
                <Button 
                  size="sm" 
                  onClick={() => approveMutation.mutate()}
                  disabled={approveMutation.isPending || rejectMutation.isPending}
                >
                  <Check className="mr-2 h-4 w-4" />
                  Approve
                </Button>
                <Button 
                  size="sm" 
                  variant="outline" 
                  onClick={() => rejectMutation.mutate()}
                  disabled={approveMutation.isPending || rejectMutation.isPending}
                >
                  <X className="mr-2 h-4 w-4" />
                  Reject
                </Button>
              </div>
            </div>
          )}

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
