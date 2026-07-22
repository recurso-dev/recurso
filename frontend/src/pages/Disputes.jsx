import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { FileQuestion } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
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

const shortId = (id) => (id ? String(id).slice(0, 8) : "—");
const fmtDate = (v) => (v ? new Date(v).toLocaleString() : "—");

const textareaClass =
  "flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

// Customer-raised invoice disputes; admins close them with an optional note.
const Disputes = () => {
  const [statusFilter, setStatusFilter] = useState("open");
  const [resolveTarget, setResolveTarget] = useState(null);
  const [note, setNote] = useState("");
  const queryClient = useQueryClient();

  // Server-driven by status: each filter is its own cache entry.
  const {
    data: disputes = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["disputes", statusFilter],
    queryFn: async () =>
      (await api.getDisputes(statusFilter === "all" ? undefined : statusFilter)).data.data || [],
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load disputes"
    : null;

  const resolveMutation = useMutation({
    mutationFn: ({ id, note }) => api.resolveDispute(id, note),
    onSuccess: () => {
      toast.success("Dispute resolved.");
      setResolveTarget(null);
      setNote("");
      queryClient.invalidateQueries({ queryKey: ["disputes"] });
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to resolve dispute"),
  });
  const resolving = resolveMutation.isPending;

  const submitResolve = () => {
    if (!resolveTarget) return;
    resolveMutation.mutate({ id: resolveTarget.id, note: note.trim() });
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
      cell: (d) => <span className="font-mono text-xs text-muted-foreground">{shortId(d.customer_id)}</span>,
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
          <Badge variant={d.status === "open" ? "warning" : "success"}>{d.status}</Badge>
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
              setNote("");
              setResolveTarget(d);
            }}
          >
            Resolve
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
        data={disputes}
        loading={loading}
        error={error}
        onRetry={refetch}
        toolbar={
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="open">Open</SelectItem>
              <SelectItem value="resolved">Resolved</SelectItem>
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
            <DialogTitle>Resolve dispute</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Invoice <span className="font-mono">{shortId(resolveTarget?.invoice_id)}</span> —{" "}
              {resolveTarget?.reason}
            </p>
            <div>
              <Label>Resolution note (optional)</Label>
              <textarea
                className={textareaClass}
                rows={3}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="What was done about it — visible to the team."
              />
            </div>
          </div>
          <DialogFooter>
            <Button onClick={submitResolve} disabled={resolving}>
              {resolving ? "Resolving…" : "Mark resolved"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Disputes;
