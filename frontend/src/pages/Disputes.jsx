import { useEffect, useState } from "react";
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
  const [disputes, setDisputes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [statusFilter, setStatusFilter] = useState("open");
  const [resolveTarget, setResolveTarget] = useState(null);
  const [note, setNote] = useState("");
  const [resolving, setResolving] = useState(false);

  const fetchDisputes = async (status = statusFilter) => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getDisputes(status === "all" ? undefined : status);
      setDisputes(res.data.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load disputes");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDisputes(statusFilter);
    // statusFilter is the only input; fetchDisputes identity is per-render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter]);

  const submitResolve = async () => {
    if (!resolveTarget) return;
    setResolving(true);
    try {
      await api.resolveDispute(resolveTarget.id, note.trim());
      toast.success("Dispute resolved.");
      setResolveTarget(null);
      setNote("");
      fetchDisputes();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to resolve dispute");
    } finally {
      setResolving(false);
    }
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
        onRetry={() => fetchDisputes()}
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
