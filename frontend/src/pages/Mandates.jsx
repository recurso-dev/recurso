import { useState } from "react";
import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { Plus, Repeat2 } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { CustomerName, CustomerSelect } from "@/components/patterns/CustomerSelect";
import { useCustomers, usePlans, useSubscriptions } from "@/lib/useCustomers";
import { toast } from "@/components/ui/sonner";
import { toMinorUnits, formatDate } from "@/lib/utils";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormSheet } from "@/components/patterns/FormSheet";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const FREQUENCIES = ["weekly", "monthly", "quarterly", "yearly"];

const fmtDate = (v) => formatDate(v);

const emptyForm = { customer_id: "", currency: "INR", vpa: "", max_amount: "", frequency: "monthly", subscription_id: "" };

// Mandate rails by currency: INR rides UPI (VPA required); EUR/GBP ride
// SEPA/Bacs bank debit — the customer authorizes on the gateway's hosted
// page, so no VPA exists.
const MANDATE_CURRENCIES = [
  { code: "INR", label: "INR — UPI (India)" },
  { code: "EUR", label: "EUR — SEPA bank debit" },
  { code: "GBP", label: "GBP — Bacs bank debit" },
];

// UPI Autopay mandates: standing authorizations to debit a customer up to a
// cap per cycle. Amounts are minor units; UPI mandates are INR.
const PER_PAGE = 25;

const Mandates = () => {
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [revokeTarget, setRevokeTarget] = useState(null);
  const [page, setPage] = useState(1);
  const queryClient = useQueryClient();
  const { customers, names } = useCustomers();
  // Subscriptions back the optional link picker in the create dialog; plans
  // give those options a human label. Both come from the shared query cache.
  const subscriptions = useSubscriptions();
  const { names: planNames } = usePlans();

  // Only the chosen customer's non-canceled subscriptions are linkable.
  const linkableSubs = subscriptions.filter(
    (s) => s.customer_id === form.customer_id && s.status !== "canceled"
  );

  const {
    data: mandates = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["mandates", page],
    // Fetch PER_PAGE+1 to detect a next page without a server-side total count.
    queryFn: async () =>
      (await api.getMandates({ limit: PER_PAGE + 1, offset: (page - 1) * PER_PAGE }))
        .data.data || [],
    placeholderData: keepPreviousData,
  });
  const hasNext = mandates.length > PER_PAGE;
  const pageRows = hasNext ? mandates.slice(0, PER_PAGE) : mandates;
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load mandates"
    : null;

  const createMutation = useMutation({
    mutationFn: (body) => api.createMandate(body),
    onSuccess: (res) => {
      const authUrl = res?.data?.data?.auth_url;
      if (authUrl) {
        // Bank-debit mandates need the customer to authorize on the
        // gateway's hosted page — surface the link, don't bury it.
        toast.success("Mandate created — send the customer their authorization link.", {
          duration: 15000,
          action: {
            label: "Copy link",
            onClick: () => navigator.clipboard?.writeText(authUrl),
          },
        });
      } else {
        toast.success("Mandate created.");
      }
      setCreateOpen(false);
      setForm(emptyForm);
      queryClient.invalidateQueries({ queryKey: ["mandates"] });
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to create mandate"),
  });
  const creating = createMutation.isPending;

  const revokeMutation = useMutation({
    mutationFn: (id) => api.revokeMandate(id),
    onSuccess: () => {
      toast.success("Mandate revoked.");
      setRevokeTarget(null);
      queryClient.invalidateQueries({ queryKey: ["mandates"] });
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to revoke mandate"),
  });
  const revoking = revokeMutation.isPending;

  const submitCreate = () => {
    const isUPI = form.currency === "INR";
    const body = {
      customer_id: form.customer_id.trim(),
      currency: form.currency,
      max_amount: toMinorUnits(form.max_amount, form.currency),
      frequency: form.frequency,
    };
    if (isUPI) body.vpa = form.vpa.trim();
    if (form.subscription_id.trim()) body.subscription_id = form.subscription_id.trim();
    createMutation.mutate(body);
  };

  const confirmRevoke = () => {
    if (!revokeTarget) return;
    revokeMutation.mutate(revokeTarget.id);
  };

  const columns = [
    {
      key: "customer",
      header: "Customer",
      cell: (m) => <CustomerName id={m.customer_id} names={names} />,
    },
    {
      key: "vpa",
      header: "Method",
      cell: (m) =>
        m.vpa ? (
          <span className="font-mono text-xs">{m.vpa}</span>
        ) : (
          <span className="text-xs text-muted-foreground">
            {m.payment_method === "bank_debit" ? "Bank debit" : "—"}
          </span>
        ),
    },
    {
      key: "max",
      header: "Max / cycle",
      align: "right",
      cell: (m) => <Money amountMinor={m.max_amount} currency={m.currency || "INR"} />,
    },
    { key: "frequency", header: "Frequency", cell: (m) => <span className="capitalize">{m.frequency}</span> },
    {
      key: "status",
      header: "Status",
      cell: (m) => <StatusBadge status={m.status} />,
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
        empty={{
          icon: Repeat2,
          title: "No mandates yet",
          description: "Create a mandate to debit a customer's UPI account on a schedule.",
          action: createButton,
        }}
      />

      <FormSheet
        open={createOpen}
        onOpenChange={setCreateOpen}
        title="New mandate"
        description="A standing authorization to debit the customer up to a cap per cycle — UPI for INR, SEPA/Bacs bank debit for EUR/GBP."
        onSubmit={submitCreate}
        submitLabel="Create mandate"
        busyLabel="Creating…"
        busy={creating}
        canSubmit={Boolean(
          form.customer_id.trim() &&
            (form.currency !== "INR" || form.vpa.trim()) &&
            form.max_amount
        )}
        dirty={Boolean(form.customer_id || form.vpa || form.max_amount)}
      >
            <div>
              <Label htmlFor="mandate-customer">Customer</Label>
              <CustomerSelect
                id="mandate-customer"
                value={form.customer_id}
                onChange={(v) => setForm({ ...form, customer_id: v, subscription_id: "" })}
                customers={customers}
              />
            </div>
            <div>
              <Label htmlFor="currency">Currency</Label>
              <Select value={form.currency} onValueChange={(v) => setForm({ ...form, currency: v })}>
                <SelectTrigger id="currency">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MANDATE_CURRENCIES.map((c) => (
                    <SelectItem key={c.code} value={c.code}>
                      {c.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {form.currency === "INR" ? (
              <div>
                <Label htmlFor="vpa-upi-id">VPA (UPI ID)</Label>
                <Input id="vpa-upi-id"
                  value={form.vpa}
                  onChange={(e) => setForm({ ...form, vpa: e.target.value })}
                  placeholder="customer@upi"
                />
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">
                The customer authorizes this mandate on the payment provider's hosted page —
                you'll get the link after creating it.
              </p>
            )}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="max-amount">Max amount ({form.currency})</Label>
                <Input id="max-amount"
                  type="number"
                  min="0.01"
                  step="0.01"
                  value={form.max_amount}
                  onChange={(e) => setForm({ ...form, max_amount: e.target.value })}
                  placeholder="999.00"
                />
              </div>
              <div>
                <Label htmlFor="frequency">Frequency</Label>
                <Select value={form.frequency} onValueChange={(v) => setForm({ ...form, frequency: v })}>
                  <SelectTrigger id="frequency">
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
              <Label htmlFor="subscription-optional">Subscription (optional)</Label>
              <Select
                value={form.subscription_id}
                onValueChange={(v) => setForm({ ...form, subscription_id: v === "none" ? "" : v })}
                disabled={!form.customer_id}
              >
                <SelectTrigger id="subscription-optional">
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
      </FormSheet>

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
