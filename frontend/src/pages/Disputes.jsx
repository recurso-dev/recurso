import { shortId, formatDateTime } from "@/lib/utils";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { FileQuestion, Check, X } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { Textarea } from "@/components/ui/textarea";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { Label } from "@/components/ui/label";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import { useCustomers } from "@/lib/useCustomers";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const fmtDate = (v) => formatDateTime(v);

const textareaClass =
  "flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

// Customer-raised invoice disputes; admins accept (optionally issuing a credit)
// or reject them.
const PER_PAGE = 25;

const Disputes = () => {
  const [statusFilter, setStatusFilter] = useState("open");
  const [page, setPage] = useState(1);
  const [resolveTarget, setResolveTarget] = useState(null);
  const [note, setNote] = useState("");
  const [issueCredit, setIssueCredit] = useState(false);
  const { names } = useCustomers();
  const queryClient = useQueryClient();

  const openReview = (d) => {
    setNote("");
    setIssueCredit(false);
    setResolveTarget(d);
  };

  // Server-driven by status: each filter is its own cache entry.
  const {
    data: disputes = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["disputes", statusFilter, page],
    // Fetch PER_PAGE+1 so a full page tells us there's a next one, without the
    // backend needing to return a total count.
    queryFn: async () =>
      (
        await api.getDisputes(statusFilter === "all" ? undefined : statusFilter, {
          limit: PER_PAGE + 1,
          offset: (page - 1) * PER_PAGE,
        })
      ).data.data || [],
    placeholderData: keepPreviousData,
  });
  const hasNext = disputes.length > PER_PAGE;
  const pageRows = hasNext ? disputes.slice(0, PER_PAGE) : disputes;
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load disputes"
    : null;

  const resolveMutation = useMutation({
    mutationFn: ({ id, body }) => api.resolveDispute(id, body),
    onSuccess: (res, { body }) => {
      toast.success(
        body.outcome === "reject"
          ? "Dispute rejected."
          : res?.data?.credit_note
            ? "Dispute accepted — credit note issued."
            : "Dispute accepted."
      );
      setResolveTarget(null);
      queryClient.invalidateQueries({ queryKey: ["disputes"] });
      if (res?.data?.credit_note) {
        queryClient.invalidateQueries({ queryKey: ["credit-notes"] });
      }
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to resolve dispute"),
  });
  const resolving = resolveMutation.isPending;

  const submit = (outcome) => {
    if (!resolveTarget) return;
    const body = { outcome, note: note.trim() };
    // Credit defaults server-side to the invoice's amount due (full disputed
    // amount) — a partial credit can be issued from the Credit Notes page.
    if (outcome === "accept" && issueCredit) body.issue_credit = true;
    resolveMutation.mutate({ id: resolveTarget.id, body });
  };

  const columns = [
    {
      key: "invoice",
      header: "Invoice",
      cell: (d) => <span className="font-mono text-xs text-muted-foreground">{shortId(d.invoice_id)}</span>,
    },
    {
      key: "customer",
      header: "Customer",
      cell: (d) => <CustomerName id={d.customer_id} names={names} />,
    },
    {
      key: "reason",
      header: "Reason",
      cell: (d) => (
        <span className="block max-w-sm truncate text-sm" title={d.reason}>
          {d.reason || "—"}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (d) => (
        <div>
          <StatusBadge status={d.status} kind="dispute" />
          {d.note && (
            <p className="mt-1 max-w-xs truncate text-xs text-muted-foreground" title={d.note}>
              {d.note}
            </p>
          )}
        </div>
      ),
    },
    {
      key: "created_at",
      header: "Raised",
      cell: (d) => <span className="text-sm text-muted-foreground">{fmtDate(d.created_at)}</span>,
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (d) =>
        d.status === "open" && (
          <Button
            size="sm"
            variant="outline"
            onClick={(e) => {
              e.stopPropagation();
              openReview(d);
            }}
          >
            Review
          </Button>
        ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Disputes"
        description="Invoice queries raised by customers from the portal."
      />

      <DataTable
        columns={columns}
        data={pageRows}
        loading={loading}
        error={error}
        onRetry={refetch}
        pagination={{
          page,
          onPrev: () => setPage((p) => Math.max(1, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext,
        }}
        toolbar={
          <Select
            value={statusFilter}
            onValueChange={(v) => {
              setStatusFilter(v);
              setPage(1);
            }}
          >
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="open">Open</SelectItem>
              <SelectItem value="resolved">Resolved</SelectItem>
              <SelectItem value="rejected">Rejected</SelectItem>
              <SelectItem value="all">All</SelectItem>
            </SelectContent>
          </Select>
        }
        empty={{
          icon: FileQuestion,
          title: statusFilter === "open" ? "No open disputes" : "No disputes",
          description: "Customer-raised invoice disputes appear here for resolution.",
        }}
      />

      <Dialog open={!!resolveTarget} onOpenChange={(o) => !o && setResolveTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Review dispute</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Invoice <span className="font-mono">{shortId(resolveTarget?.invoice_id)}</span> —{" "}
              {resolveTarget?.reason}
            </p>

            <label className="flex items-start gap-2.5 rounded-md border border-border p-3 text-sm">
              <input
                type="checkbox"
                className="mt-0.5 h-4 w-4 accent-primary"
                checked={issueCredit}
                onChange={(e) => setIssueCredit(e.target.checked)}
              />
              <span>
                <span className="font-medium text-foreground">
                  Issue a credit note on accept
                </span>
                <span className="mt-0.5 block text-xs text-muted-foreground">
                  Adds an account credit for the invoice&apos;s outstanding amount. Leave
                  unchecked to accept without a credit.
                </span>
              </span>
            </label>

            <div>
              <Label>Note (optional)</Label>
              <Textarea
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
              disabled={resolving}
            >
              <X className="h-4 w-4" />
              Reject
            </Button>
            <Button onClick={() => submit("accept")} disabled={resolving}>
              <Check className="h-4 w-4" />
              {resolving ? "Working…" : issueCredit ? "Accept & credit" : "Accept"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Disputes;
