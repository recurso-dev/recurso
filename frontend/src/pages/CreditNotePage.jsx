import { useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Check, X, Ban, Download } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatDateTime } from "@/lib/utils";
import { useAuth } from "@/auth/AuthProvider";
import { useCustomers } from "@/lib/useCustomers";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedRow,
} from "@/components/patterns/ObjectPage";
import { AttentionBanner } from "@/components/patterns/AttentionBanner";
import { JournalEntries } from "@/components/patterns/JournalEntries";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { Money } from "@/components/ui/money";
import { CopyableId } from "@/components/ui/copyable-id";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { toast } from "@/components/ui/sonner";

/**
 * CreditNotePage — one credit note as a first-class object at /credit-notes/:id.
 * Replaces the cramped detail slide-over: the note's identity + lifecycle, the
 * amount/balance/applied math, the statutory tax reversal, its relationships
 * (customer + the invoice it offsets), the approve/reject/void actions with
 * their money consequences spelled out, and — the new drill — the actual ledger
 * postings behind it (Customer-Credit liability, tax reversal, refund legs).
 */
export default function CreditNotePage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const { names } = useCustomers();

  const [confirm, setConfirm] = useState(null); // "approve" | "reject" | "void"

  const {
    data: cn,
    isLoading,
    error: cnError,
    refetch,
  } = useQuery({
    queryKey: ["creditNote", id],
    queryFn: async () => (await endpoints.getCreditNote(id)).data.data,
    enabled: Boolean(id),
  });

  const {
    data: journal = [],
    isLoading: journalLoading,
    error: journalError,
  } = useQuery({
    queryKey: ["creditNoteJournal", id],
    queryFn: async () =>
      (await endpoints.getCreditNoteJournalEntries(id)).data.data?.entries || [],
    enabled: Boolean(id),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["creditNote", id] });
    queryClient.invalidateQueries({ queryKey: ["creditNoteJournal", id] });
    queryClient.invalidateQueries({ queryKey: ["credit-notes"] });
  };

  const actionHandlers = (okMsg) => ({
    onSuccess: () => {
      toast.success(okMsg);
      setConfirm(null);
      invalidate();
    },
    onError: (err) => toast.error(err?.response?.data?.error?.message || "Action failed"),
  });
  const approveMutation = useMutation({
    mutationFn: () => endpoints.approveCreditNote(id),
    ...actionHandlers("Credit note approved."),
  });
  const rejectMutation = useMutation({
    mutationFn: () => endpoints.rejectCreditNote(id),
    ...actionHandlers("Credit note rejected."),
  });
  const voidMutation = useMutation({
    mutationFn: () => endpoints.voidCreditNote(id),
    ...actionHandlers("Credit note voided."),
  });

  const [downloading, setDownloading] = useState(false);
  const handleDownload = async () => {
    setDownloading(true);
    try {
      const res = await endpoints.getCreditNotePdf(id);
      const url = URL.createObjectURL(res.data);
      window.open(url, "_blank", "noreferrer");
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to open document.");
    } finally {
      setDownloading(false);
    }
  };

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

  if (cnError || !cn) {
    return (
      <ErrorState
        title={cnError ? "Couldn't load this credit note" : "Credit note not found"}
        message={
          cnError
            ? cnError?.response?.data?.error?.message || cnError?.message
            : "This credit note doesn't exist or isn't in your account."
        }
        onRetry={cnError ? refetch : undefined}
      />
    );
  }

  const currency = cn.currency;
  const applied = (cn.amount || 0) - (cn.balance || 0);
  const canApprove = user?.role === "admin" || user?.role === "owner";
  const isPending = cn.status === "pending_approval";
  // Void applies only to an issued adjustment credit with unspent balance — a
  // refund moved money at the gateway and can't be reversed with a ledger entry.
  const canVoid =
    canApprove && cn.type === "adjustment" && cn.status === "issued" && (cn.balance ?? 0) > 0;

  const attention = [];
  if (isPending) {
    attention.push({
      tone: "warning",
      text: canApprove
        ? "Pending approval — approving issues the credit (or refund) and posts it to the ledger; rejecting issues nothing."
        : "Pending approval — an admin or owner must approve this before any credit is issued.",
    });
  }
  if (cn.status === "rejected") {
    attention.push({ tone: "warning", text: "This credit note was rejected — no credit was issued." });
  }
  if (cn.refund_status === "failed") {
    attention.push({
      tone: "danger",
      text: `The refund failed${cn.refund_message ? `: ${cn.refund_message}` : ""}.`,
    });
  }

  const customerName = names[cn.customer_id] || cn.customer?.name;

  return (
    <div>
      <ObjectHeader
        backTo="/credit-notes"
        backLabel="Credit Notes"
        kicker="Credit note"
        title={cn.reference || `Credit note ${String(cn.id).slice(0, 8)}`}
        badge={<StatusBadge status={cn.status} />}
        meta={
          <>
            <span className="tabular-nums font-medium text-foreground">
              <Money amountMinor={cn.amount} currency={currency} />
            </span>
            {cn.type && <span className="capitalize">{cn.type}</span>}
            <CopyableId value={cn.id} />
          </>
        }
        actions={
          <>
            <Button variant="outline" onClick={handleDownload} disabled={downloading}>
              <Download className="h-4 w-4" />
              {downloading ? "Opening…" : "Document"}
            </Button>
            {isPending && canApprove && (
              <>
                <Button onClick={() => setConfirm("approve")}>
                  <Check className="h-4 w-4" />
                  Approve
                </Button>
                <Button variant="outline" onClick={() => setConfirm("reject")}>
                  <X className="h-4 w-4" />
                  Reject
                </Button>
              </>
            )}
            {canVoid && (
              <Button
                variant="ghost"
                className="text-destructive hover:text-destructive"
                onClick={() => setConfirm("void")}
              >
                <Ban className="h-4 w-4" />
                Void
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
                { label: "Credit note ID", value: <CopyableId value={cn.id} /> },
                ...(cn.reference ? [{ label: "Reference", value: cn.reference }] : []),
                {
                  label: "Customer",
                  value: (
                    <Link
                      to={`/customers/${cn.customer_id}`}
                      className="text-primary hover:underline"
                    >
                      {customerName || <CustomerName id={cn.customer_id} names={names} link={false} />}
                    </Link>
                  ),
                },
                { label: "Type", value: <span className="capitalize">{cn.type || "—"}</span> },
                { label: "Reason", value: <span className="capitalize">{cn.reason || "—"}</span> },
                ...(cn.type === "refund" && cn.refund_status
                  ? [{ label: "Refund status", value: <span className="capitalize">{cn.refund_status}</span> }]
                  : []),
                { label: "Created", value: formatDateTime(cn.created_at) },
                ...(cn.approved_at ? [{ label: "Approved", value: formatDateTime(cn.approved_at) }] : []),
                ...(cn.expires_at ? [{ label: "Expires", value: formatDateTime(cn.expires_at) }] : []),
              ]}
            />
          </ObjectSection>
        }
      >
        <ObjectSection title="Amounts">
          <AttributeList
            columns={3}
            items={[
              {
                label: "Credit amount",
                value: (
                  <span className="font-mono text-lg font-medium tabular-nums">
                    <Money amountMinor={cn.amount} currency={currency} />
                  </span>
                ),
              },
              {
                label: cn.type === "refund" ? "Refunded" : "Applied",
                value: (
                  <span className="font-mono tabular-nums">
                    <Money amountMinor={applied} currency={currency} />
                  </span>
                ),
              },
              {
                label: "Balance remaining",
                value: (
                  <span className="font-mono tabular-nums">
                    <Money amountMinor={cn.balance} currency={currency} />
                  </span>
                ),
              },
            ]}
          />
          {cn.type === "adjustment" && (cn.balance ?? 0) > 0 && (
            <p className="mt-3 text-xs text-muted-foreground">
              The remaining balance is spendable account credit — drained before the payment
              gateway on the customer&apos;s next invoice.
            </p>
          )}
        </ObjectSection>

        {cn.subtotal > 0 && (
          <ObjectSection title={`Tax reversed${cn.hsn_code ? ` · HSN ${cn.hsn_code}` : ""}`}>
            <AttributeList
              columns={2}
              items={[
                {
                  label: "Taxable value",
                  value: (
                    <span className="font-mono tabular-nums">
                      <Money amountMinor={cn.subtotal} currency={currency} />
                    </span>
                  ),
                },
                ...(cn.igst_amount > 0
                  ? [{ label: "IGST reversed", value: <span className="font-mono tabular-nums"><Money amountMinor={cn.igst_amount} currency={currency} /></span> }]
                  : []),
                ...(cn.cgst_amount > 0
                  ? [{ label: "CGST reversed", value: <span className="font-mono tabular-nums"><Money amountMinor={cn.cgst_amount} currency={currency} /></span> }]
                  : []),
                ...(cn.sgst_amount > 0
                  ? [{ label: "SGST reversed", value: <span className="font-mono tabular-nums"><Money amountMinor={cn.sgst_amount} currency={currency} /></span> }]
                  : []),
              ]}
            />
          </ObjectSection>
        )}

        <ObjectSection title="Related">
          <RelatedRow to={`/customers/${cn.customer_id}`}>
            <span className="text-foreground">
              {customerName || <CustomerName id={cn.customer_id} names={names} link={false} />}
            </span>
            <span className="text-xs text-muted-foreground">Customer →</span>
          </RelatedRow>
          {cn.invoice_id && (
            <RelatedRow to={`/invoices/${cn.invoice_id}`}>
              <span className="text-foreground">Offsets an invoice</span>
              <span className="text-xs text-muted-foreground">View invoice →</span>
            </RelatedRow>
          )}
        </ObjectSection>

        <ObjectSection title="Journal entries">
          <JournalEntries
            entries={journal}
            currency={currency}
            isLoading={journalLoading}
            error={journalError}
            emptyMessage="No ledger postings for this credit note yet — an approved credit posts its Customer-Credit and tax-reversal legs."
          />
        </ObjectSection>
      </ObjectPageLayout>

      <ConfirmDialog
        open={confirm === "approve"}
        onOpenChange={(o) => !o && setConfirm(null)}
        title="Approve this credit note?"
        description={`Approving issues ${cn.currency} credit of the full amount to the customer — it becomes spendable (or refundable) immediately and posts to the ledger.`}
        confirmLabel="Approve credit note"
        busy={approveMutation.isPending}
        onConfirm={() => approveMutation.mutate()}
      />
      <ConfirmDialog
        open={confirm === "reject"}
        onOpenChange={(o) => !o && setConfirm(null)}
        title="Reject this credit note?"
        description="The credit note is rejected and no credit is issued. This can't be undone."
        confirmLabel="Reject credit note"
        destructive
        busy={rejectMutation.isPending}
        onConfirm={() => rejectMutation.mutate()}
      />
      <ConfirmDialog
        open={confirm === "void"}
        onOpenChange={(o) => !o && setConfirm(null)}
        title="Void this credit note?"
        description="This cancels the credit and writes off the remaining balance. Already-applied credit is not affected. This can't be undone."
        confirmLabel="Void credit note"
        destructive
        busy={voidMutation.isPending}
        onConfirm={() => voidMutation.mutate()}
      />
    </div>
  );
}
