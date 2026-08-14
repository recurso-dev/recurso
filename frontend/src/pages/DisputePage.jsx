import { useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Check, X } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatDateTime, shortId } from "@/lib/utils";
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
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { Money } from "@/components/ui/money";
import { CopyableId } from "@/components/ui/copyable-id";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

const textareaClass =
  "flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

/**
 * DisputePage — one customer-raised invoice dispute as a first-class object at
 * /disputes/:id. Gives the query its own addressable page: the reason, the
 * invoice it contests (with its real amount/status, linked), the customer, the
 * resolution note + outcome, and the Review action (accept — optionally issuing
 * a credit — or reject) with its consequence spelled out.
 */
export default function DisputePage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const { names } = useCustomers();

  const [reviewOpen, setReviewOpen] = useState(false);
  const [note, setNote] = useState("");
  const [issueCredit, setIssueCredit] = useState(false);

  const {
    data: dispute,
    isLoading,
    error: disputeError,
    refetch,
  } = useQuery({
    queryKey: ["dispute", id],
    queryFn: async () => (await endpoints.getDispute(id)).data.data,
    enabled: Boolean(id),
  });

  // The contested invoice — its real number/amount/status enrich the relation.
  // Best-effort: on failure the relation still links by id.
  const { data: invoice } = useQuery({
    queryKey: ["invoice", dispute?.invoice_id, "dispute-page"],
    queryFn: async () => (await endpoints.getInvoice(dispute.invoice_id)).data.data,
    enabled: Boolean(dispute?.invoice_id),
  });

  const resolveMutation = useMutation({
    mutationFn: (body) => endpoints.resolveDispute(id, body),
    onSuccess: (res, body) => {
      toast.success(
        body.outcome === "reject"
          ? "Dispute rejected."
          : res?.data?.credit_note
            ? "Dispute accepted — credit note issued."
            : "Dispute accepted."
      );
      setReviewOpen(false);
      queryClient.invalidateQueries({ queryKey: ["dispute", id] });
      queryClient.invalidateQueries({ queryKey: ["disputes"] });
      if (res?.data?.credit_note) {
        queryClient.invalidateQueries({ queryKey: ["credit-notes"] });
      }
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to resolve dispute"),
  });

  const submit = (outcome) => {
    const body = { outcome, note: note.trim() };
    if (outcome === "accept" && issueCredit) body.issue_credit = true;
    resolveMutation.mutate(body);
  };

  const openReview = () => {
    setNote("");
    setIssueCredit(false);
    setReviewOpen(true);
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

  if (disputeError || !dispute) {
    return (
      <ErrorState
        title={disputeError ? "Couldn't load this dispute" : "Dispute not found"}
        message={
          disputeError
            ? disputeError?.response?.data?.error?.message || disputeError?.message
            : "This dispute doesn't exist or isn't in your account."
        }
        onRetry={disputeError ? refetch : undefined}
      />
    );
  }

  const isOpen = dispute.status === "open";
  const invoiceLabel = invoice?.invoice_number || shortId(dispute.invoice_id);

  const attention = [];
  if (isOpen) {
    attention.push({
      tone: "warning",
      text: "Open — awaiting review. Accepting closes it in the customer's favor (and can issue a credit); rejecting declines it.",
    });
  } else if (dispute.status === "rejected") {
    attention.push({ tone: "warning", text: "This dispute was reviewed and rejected." });
  }

  return (
    <div>
      <ObjectHeader
        backTo="/disputes"
        backLabel="Disputes"
        kicker="Dispute"
        title={`Dispute on ${invoiceLabel}`}
        badge={<StatusBadge status={dispute.status} kind="dispute" flashOnChange />}
        meta={
          <>
            <CustomerName id={dispute.customer_id} names={names} link={false} />
            <span>{formatDateTime(dispute.created_at)}</span>
            <CopyableId value={dispute.id} />
          </>
        }
        actions={
          isOpen ? (
            <Button onClick={openReview}>Review</Button>
          ) : null
        }
      />

      <AttentionBanner items={attention} className="mb-6" />

      <ObjectPageLayout
        rail={
          <ObjectSection title="Details">
            <AttributeList
              columns={1}
              items={[
                { label: "Dispute ID", value: <CopyableId value={dispute.id} /> },
                {
                  label: "Customer",
                  value: (
                    <Link
                      to={`/customers/${dispute.customer_id}`}
                      className="text-primary hover:underline"
                    >
                      <CustomerName id={dispute.customer_id} names={names} link={false} />
                    </Link>
                  ),
                },
                { label: "Status", value: <StatusBadge status={dispute.status} kind="dispute" /> },
                { label: "Raised", value: formatDateTime(dispute.created_at) },
                ...(dispute.resolved_at
                  ? [{ label: "Resolved", value: formatDateTime(dispute.resolved_at) }]
                  : []),
              ]}
            />
          </ObjectSection>
        }
      >
        <ObjectSection title="What the customer disputed">
          <p className="text-sm text-foreground">{dispute.reason || "No reason given."}</p>
        </ObjectSection>

        <ObjectSection title="Resolution">
          {isOpen ? (
            <p className="text-sm text-muted-foreground">
              Not yet resolved — use Review to accept (optionally issuing a credit) or reject.
            </p>
          ) : (
            <AttributeList
              columns={1}
              items={[
                {
                  label: "Outcome",
                  value: (
                    <span className="capitalize">
                      {dispute.status === "resolved" ? "Accepted" : dispute.status}
                    </span>
                  ),
                },
                ...(dispute.resolved_at
                  ? [{ label: "Resolved", value: formatDateTime(dispute.resolved_at) }]
                  : []),
                { label: "Note", value: dispute.note || <span className="text-muted-foreground">—</span> },
              ]}
            />
          )}
        </ObjectSection>

        <ObjectSection title="Related">
          <RelatedRow to={`/invoices/${dispute.invoice_id}`}>
            <span className="min-w-0 truncate text-foreground">Invoice {invoiceLabel}</span>
            <span className="flex shrink-0 items-center gap-3">
              {invoice ? <Money amountMinor={invoice.total} currency={invoice.currency} /> : null}
              {invoice ? <StatusBadge status={invoice.status} /> : null}
            </span>
          </RelatedRow>
          <RelatedRow to={`/customers/${dispute.customer_id}`}>
            <span className="text-foreground">
              <CustomerName id={dispute.customer_id} names={names} link={false} />
            </span>
            <span className="text-xs text-muted-foreground">Customer →</span>
          </RelatedRow>
        </ObjectSection>
      </ObjectPageLayout>

      {/* Review — accept (optionally with a credit) or reject */}
      <Dialog open={reviewOpen} onOpenChange={(o) => !o && setReviewOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Review dispute</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Invoice <span className="font-mono">{invoiceLabel}</span> — {dispute.reason}
            </p>

            <label className="flex items-start gap-2.5 rounded-md border border-border p-3 text-sm">
              <input
                type="checkbox"
                className="mt-0.5 h-4 w-4 accent-primary"
                checked={issueCredit}
                onChange={(e) => setIssueCredit(e.target.checked)}
              />
              <span>
                <span className="font-medium text-foreground">Issue a credit note on accept</span>
                <span className="mt-0.5 block text-xs text-muted-foreground">
                  Adds an account credit for the invoice&apos;s outstanding amount
                  {invoice ? (
                    <>
                      {" "}
                      (up to <Money amountMinor={invoice.total} currency={invoice.currency} />)
                    </>
                  ) : null}
                  . Leave unchecked to accept without a credit.
                </span>
              </span>
            </label>

            <div>
              <Label htmlFor="dispute-note">Note (optional)</Label>
              <Textarea
                id="dispute-note"
                className={textareaClass}
                rows={3}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="What was decided — visible to the team."
              />
            </div>
          </div>
          <DialogFooter className="gap-2 sm:justify-between">
            <Button
              variant="outline"
              className="text-destructive hover:bg-destructive/10 hover:text-destructive"
              onClick={() => submit("reject")}
              disabled={resolveMutation.isPending}
            >
              <X className="h-4 w-4" />
              Reject
            </Button>
            <Button onClick={() => submit("accept")} disabled={resolveMutation.isPending}>
              <Check className="h-4 w-4" />
              {resolveMutation.isPending ? "Working…" : issueCredit ? "Accept & credit" : "Accept"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
