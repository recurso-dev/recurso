import { useEffect, useState } from "react";
import { Plus, Repeat2 } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { CustomerName, CustomerSelect } from "@/components/patterns/CustomerSelect";
import { useCustomers, usePlans, useSubscriptions } from "@/lib/useCustomers";
import { toast } from "@/components/ui/sonner";
import { formatCurrency, toMinorUnits } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const FREQUENCIES = ["weekly", "monthly", "quarterly", "yearly"];

const statusVariant = (status) =>
  ({ active: "success", authorized: "info", created: "neutral", paused: "warning", revoked: "destructive" })[
    status
  ] || "neutral";

const fmtDate = (v) => (v ? new Date(v).toLocaleDateString() : "—");

const emptyForm = { customer_id: "", vpa: "", max_amount: "", frequency: "monthly", subscription_id: "" };

// UPI Autopay mandates: standing authorizations to debit a customer up to a
// cap per cycle. Amounts are minor units; UPI mandates are INR.
const Mandates = () => {
  const [mandates, setMandates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [creating, setCreating] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState(null);
  const [revoking, setRevoking] = useState(false);
  const { customers, names } = useCustomers();
  // Subscriptions back the optional link picker in the create dialog; plans
  // give those options a human label. Both come from the shared query cache.
  const subscriptions = useSubscriptions();
  const { names: planNames } = usePlans();

  // Only the chosen customer's non-canceled subscriptions are linkable.
  const linkableSubs = subscriptions.filter(
    (s) => s.customer_id === form.customer_id && s.status !== "canceled"
  );

  const fetchMandates = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getMandates();
      setMandates(res.data.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load mandates");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMandates();
  }, []);

  const submitCreate = async () => {
    setCreating(true);
    try {
      const body = {
        customer_id: form.customer_id.trim(),
        vpa: form.vpa.trim(),
        max_amount: toMinorUnits(form.max_amount),
        frequency: form.frequency,
      };
      if (form.subscription_id.trim()) body.subscription_id = form.subscription_id.trim();
      await api.createMandate(body);
      toast.success("Mandate created.");
      setCreateOpen(false);
      setForm(emptyForm);
      fetchMandates();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to create mandate");
    } finally {
      setCreating(false);
    }
  };

  const confirmRevoke = async () => {
    if (!revokeTarget) return;
    setRevoking(true);
    try {
      await api.revokeMandate(revokeTarget.id);
      toast.success("Mandate revoked.");
      setRevokeTarget(null);
      fetchMandates();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to revoke mandate");
    } finally {
      setRevoking(false);
    }
  };

  const columns = [
    {
      key: "customer",
      header: "Customer",
      cell: (m) => <CustomerName id={m.customer_id} names={names} />,
    },
    {
      key: "vpa",
      header: "VPA",
      cell: (m) => <span className="font-mono text-xs">{m.vpa || "—"}</span>,
    },
    {
      key: "max",
      header: "Max / cycle",
      cell: (m) => (
        <span className="tabular-nums font-medium">{formatCurrency(m.max_amount, "INR")}</span>
      ),
    },
    { key: "frequency", header: "Frequency", cell: (m) => <span className="capitalize">{m.frequency}</span> },
    {
      key: "status",
      header: "Status",
      cell: (m) => <Badge variant={statusVariant(m.status)}>{m.status}</Badge>,
    },
    {
      key: "next",
      header: "Next debit",
      cell: (m) => <span className="text-sm text-muted-foreground">{fmtDate(m.next_debit_at)}</span>,
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (m) =>
        m.status !== "revoked" && (
          <Button
            size="sm"
            variant="outline"
            onClick={(e) => {
              e.stopPropagation();
              setRevokeTarget(m);
            }}
          >
            Revoke
          </Button>
        ),
    },
  ];

  const createButton = (
    <Button onClick={() => setCreateOpen(true)}>
      <Plus className="h-4 w-4" />
      New mandate
    </Button>
  );

  return (
    <div>
      <PageHeader
        title="Mandates"
        description="UPI Autopay authorizations — recurring debits up to a per-cycle cap."
        actions={createButton}
      />

      <DataTable
        columns={columns}
        data={mandates}
        loading={loading}
        error={error}
        onRetry={fetchMandates}
        empty={{
          icon: Repeat2,
          title: "No mandates yet",
          description: "Create a mandate to debit a customer's UPI account on a schedule.",
          action: createButton,
        }}
      />

      <Sheet open={createOpen} onOpenChange={setCreateOpen}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>New UPI mandate</SheetTitle>
            <SheetDescription>
              A standing authorization to debit the customer up to a cap per cycle.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 overflow-y-auto px-6">
            <div>
              <Label>Customer</Label>
              <CustomerSelect
                value={form.customer_id}
                onChange={(v) => setForm({ ...form, customer_id: v, subscription_id: "" })}
                customers={customers}
              />
            </div>
            <div>
              <Label>VPA (UPI ID)</Label>
              <Input
                value={form.vpa}
                onChange={(e) => setForm({ ...form, vpa: e.target.value })}
                placeholder="customer@upi"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>Max amount (INR)</Label>
                <Input
                  type="number"
                  min="0.01"
                  step="0.01"
                  value={form.max_amount}
                  onChange={(e) => setForm({ ...form, max_amount: e.target.value })}
                  placeholder="999.00"
                />
              </div>
              <div>
                <Label>Frequency</Label>
                <Select value={form.frequency} onValueChange={(v) => setForm({ ...form, frequency: v })}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {FREQUENCIES.map((f) => (
                      <SelectItem key={f} value={f} className="capitalize">
                        {f}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div>
              <Label>Subscription (optional)</Label>
              <Select
                value={form.subscription_id}
                onValueChange={(v) => setForm({ ...form, subscription_id: v === "none" ? "" : v })}
                disabled={!form.customer_id}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={
                      !form.customer_id
                        ? "Select a customer first"
                        : linkableSubs.length === 0
                          ? "No active subscriptions"
                          : "Link a subscription"
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Not linked</SelectItem>
                  {linkableSubs.map((s) => (
                    <SelectItem key={s.id} value={s.id}>
                      {planNames[s.plan_id] || `${String(s.id).slice(0, 8)}…`} · {s.status}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <SheetFooter>
            <Button
              onClick={submitCreate}
              disabled={creating || !form.customer_id.trim() || !form.vpa.trim() || !form.max_amount}
            >
              {creating ? "Creating…" : "Create mandate"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={!!revokeTarget}
        onOpenChange={(o) => !o && setRevokeTarget(null)}
        title="Revoke this mandate?"
        description="Future automatic debits stop immediately. This cannot be undone — the customer must authorize a new mandate to resume."
        confirmLabel="Revoke mandate"
        destructive
        busy={revoking}
        onConfirm={confirmRevoke}
      />
    </div>
  );
};

export default Mandates;
