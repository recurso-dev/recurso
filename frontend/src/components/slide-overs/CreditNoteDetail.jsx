import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { endpoints } from "../../lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Check, X, Download, Copy, Ban } from "lucide-react";
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
  const [downloading, setDownloading] = useState(false);
  const [confirmVoid, setConfirmVoid] = useState(false);
  // "approve" | "reject" | null — approving MOVES MONEY (issues the credit or
  // refund); it must be at least as guarded as its sibling void (audit §7).
  const [confirmDecision, setConfirmDecision] = useState(null);

  const voidMutation = useMutation({
    mutationFn: () => endpoints.voidCreditNote(creditNote.id),
    onSuccess: () => {
      toast.success("Credit note voided.");
      queryClient.invalidateQueries({ queryKey: ["credit-notes"] });
      queryClient.invalidateQueries({ queryKey: ["creditNote", creditNote.id] });
      setConfirmVoid(false);
      onClose();
    },
    onError: (err) => {
      toast.error(err?.response?.data?.error?.message || "Failed to void credit note.");
    },
  });

  const handleDownload = async () => {
    setDownloading(true);
    try {
      const res = await endpoints.getCreditNotePdf(creditNote.id);
      const url = URL.createObjectURL(res.data);
      window.open(url, "_blank", "noreferrer");
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to open credit note document.");
    } finally {
      setDownloading(false);
    }
  };

  const handleCopyId = async () => {
    try {
      await navigator.clipboard.writeText(creditNote.id);
      toast.success("Credit note ID copied.");
    } catch {
      toast.error("Couldn't copy to clipboard.");
    }
  };

  const approveMutation = useMutation({
    mutationFn: () => endpoints.approveCreditNote(creditNote.id),
    onSuccess: () => {
      toast.success("Credit note approved successfully.");
      queryClient.invalidateQueries({ queryKey: ["credit-notes"] });
      queryClient.invalidateQueries({ queryKey: ["creditNote", creditNote.id] });
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
      queryClient.invalidateQueries({ queryKey: ["credit-notes"] });
      queryClient.invalidateQueries({ queryKey: ["creditNote", creditNote.id] });
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
  // Void applies only to an issued account-credit with an unspent balance; a
  // refund moved money at the gateway and can't be undone with a ledger entry.
  const canVoid =
    canApprove &&
    creditNote.type === "adjustment" &&
    creditNote.status === "issued" &&
    (creditNote.balance ?? 0) > 0;

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
            <div className="rounded-lg border border-warning/20 bg-warning/5 p-4">
              <h3 className="text-sm font-medium text-warning">Approval Required</h3>
              <p className="mt-1 text-sm text-warning">
                This credit note is pending review. You can approve it to issue the credit or refund, or reject it.
              </p>
              <div className="mt-4 flex gap-3">
                <Button
                  size="sm"
                  onClick={() => setConfirmDecision("approve")}
                  disabled={approveMutation.isPending || rejectMutation.isPending}
                >
                  <Check className="mr-2 h-4 w-4" />
                  Approve
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setConfirmDecision("reject")}
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

          {/* Tax breakdown (present when the note recorded one at creation —
              the same figures the downloadable CDN document shows). */}
          {creditNote.subtotal > 0 && (
            <div className="rounded-lg border border-border bg-muted/30 p-4">
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Tax breakdown
                {creditNote.hsn_code ? ` · HSN ${creditNote.hsn_code}` : ""}
              </p>
              <dl className="mt-3 space-y-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Taxable value</dt>
                  <dd className="tabular-nums font-medium">
                    {formatCurrency(creditNote.subtotal, currency)}
                  </dd>
                </div>
                {creditNote.igst_amount > 0 && (
                  <div className="flex justify-between">
                    <dt className="text-muted-foreground">IGST reversed</dt>
                    <dd className="tabular-nums">
                      {formatCurrency(creditNote.igst_amount, currency)}
                    </dd>
                  </div>
                )}
                {creditNote.cgst_amount > 0 && (
                  <div className="flex justify-between">
                    <dt className="text-muted-foreground">CGST reversed</dt>
                    <dd className="tabular-nums">
                      {formatCurrency(creditNote.cgst_amount, currency)}
                    </dd>
                  </div>
                )}
                {creditNote.sgst_amount > 0 && (
                  <div className="flex justify-between">
                    <dt className="text-muted-foreground">SGST reversed</dt>
                    <dd className="tabular-nums">
                      {formatCurrency(creditNote.sgst_amount, currency)}
                    </dd>
                  </div>
                )}
              </dl>
            </div>
          )}

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

          <Separator />

          <div className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleDownload}
              disabled={downloading}
            >
              <Download className="mr-2 h-4 w-4" />
              {downloading ? "Opening…" : "Download document"}
            </Button>
            <Button variant="ghost" size="sm" onClick={handleCopyId}>
              <Copy className="mr-2 h-4 w-4" />
              Copy ID
            </Button>
            {canVoid && (
              <Button
                variant="ghost"
                size="sm"
                className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                onClick={() => setConfirmVoid(true)}
              >
                <Ban className="mr-2 h-4 w-4" />
                Void
              </Button>
            )}
          </div>
        </div>

        <ConfirmDialog
          open={confirmDecision === "approve"}
          onOpenChange={(o) => !o && setConfirmDecision(null)}
          title="Approve this credit note?"
          description={`Approving issues ${formatCurrency(
            creditNote.amount,
            currency
          )} of credit to the customer — it becomes spendable (or refundable) immediately.`}
          confirmLabel="Approve credit note"
          busy={approveMutation.isPending}
          onConfirm={() =>
            approveMutation.mutate(undefined, { onSettled: () => setConfirmDecision(null) })
          }
        />
        <ConfirmDialog
          open={confirmDecision === "reject"}
          onOpenChange={(o) => !o && setConfirmDecision(null)}
          title="Reject this credit note?"
          description="The credit note is rejected and no credit is issued. This can't be undone."
          confirmLabel="Reject credit note"
          destructive
          busy={rejectMutation.isPending}
          onConfirm={() =>
            rejectMutation.mutate(undefined, { onSettled: () => setConfirmDecision(null) })
          }
        />
        <ConfirmDialog
          open={confirmVoid}
          onOpenChange={setConfirmVoid}
          title="Void this credit note?"
          description={`This cancels the credit and writes off the remaining ${formatCurrency(
            creditNote.balance,
            currency
          )} balance. Already-applied credit is not affected. This can't be undone.`}
          confirmLabel="Void credit note"
          destructive
          busy={voidMutation.isPending}
          onConfirm={() => voidMutation.mutate()}
        />
      </SheetContent>
    </Sheet>
  );
};

export default CreditNoteDetail;
