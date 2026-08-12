import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Banknote, Landmark } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { CustomerName, CustomerSelect } from "@/components/patterns/CustomerSelect";
import { useCustomers } from "@/lib/useCustomers";
import { toast } from "@/components/ui/sonner";
import { formatCurrency, toMinorUnits, fromMinorUnits, shortId, formatDateTime } from "@/lib/utils";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormSheet } from "@/components/patterns/FormSheet";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const PAYMENT_TYPES = [
  { value: "bank_transfer", label: "Bank transfer" },
  { value: "cash", label: "Cash" },
  { value: "cheque", label: "Cheque" },
];

const fmtDate = (v) => formatDateTime(v);

const emptyPayment = {
  customer_id: "",
  invoice_id: "",
  payment_type: "bank_transfer",
  amount: "",
  tds_amount: "",
  currency: "INR",
  reference_number: "",
  notes: "",
};
const emptyVA = { customer_id: "", invoice_id: "", amount: "" };

// Record money received outside the gateway (NEFT/cash/cheque) and issue
// virtual accounts customers can transfer into. Amounts are minor units.
const OfflinePayments = () => {
  const queryClient = useQueryClient();
  const [recordOpen, setRecordOpen] = useState(false);
  const [payForm, setPayForm] = useState(emptyPayment);
  const [vaOpen, setVAOpen] = useState(false);
  const [vaForm, setVAForm] = useState(emptyVA);
  const [tab, setTab] = useState("payments");
  const { customers, names } = useCustomers();

  // Invoices back the "settle this invoice" pickers; unpaid ones are offered.
  // Best-effort: on failure the pickers just offer nothing.
  const { data: invoices = [] } = useQuery({
    queryKey: ["invoices", "offline-pickers"],
    queryFn: async () => (await api.getInvoices())?.data?.data || [],
  });

  const openInvoicesFor = (customerId) =>
    invoices.filter(
      (i) => i.customer_id === customerId && ["open", "overdue", "past_due"].includes(i.status)
    );

  const invoiceLabel = (i) =>
    `${i.invoice_number || String(i.id).slice(0, 8)} · ${formatCurrency(i.total, i.currency)}`;

  const invoiceNumberById = (id) => {
    const inv = invoices.find((i) => i.id === id);
    return inv ? inv.invoice_number || String(id).slice(0, 8) : null;
  };

  const {
    data: payments = [],
    isLoading: paymentsLoading,
    error: paymentsQueryError,
    refetch: refetchPayments,
  } = useQuery({
    queryKey: ["offline-payments"],
    queryFn: async () => (await api.getOfflinePayments()).data.data || [],
  });
  const paymentsError = paymentsQueryError
    ? paymentsQueryError?.response?.data?.error?.message || "Failed to load payments"
    : null;

  const {
    data: vas = [],
    isLoading: vasLoading,
    error: vasQueryError,
    refetch: refetchVAs,
  } = useQuery({
    queryKey: ["virtual-accounts"],
    queryFn: async () => (await api.getVirtualAccounts()).data.data || [],
  });
  const vasError = vasQueryError
    ? vasQueryError?.response?.data?.error?.message || "Failed to load virtual accounts"
    : null;

  const recordMutation = useMutation({
    mutationFn: (body) => api.recordOfflinePayment(body),
    onSuccess: () => {
      toast.success("Payment recorded.");
      setRecordOpen(false);
      setPayForm(emptyPayment);
      // A recorded payment settles an invoice — refresh both feeds.
      queryClient.invalidateQueries({ queryKey: ["offline-payments"] });
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to record payment"),
  });
  const recording = recordMutation.isPending;

  const submitRecord = () => {
    const body = {
      customer_id: payForm.customer_id.trim(),
      payment_type: payForm.payment_type,
      amount: toMinorUnits(payForm.amount, payForm.currency),
      currency: payForm.currency,
      reference_number: payForm.reference_number.trim(),
      notes: payForm.notes.trim(),
    };
    if (payForm.invoice_id.trim()) body.invoice_id = payForm.invoice_id.trim();
    if (payForm.tds_amount) body.tds_amount = toMinorUnits(payForm.tds_amount, payForm.currency);
    recordMutation.mutate(body);
  };

  const vaMutation = useMutation({
    mutationFn: (body) => api.createVirtualAccount(body),
    onSuccess: () => {
      toast.success("Virtual account created.");
      setVAOpen(false);
      setVAForm(emptyVA);
      queryClient.invalidateQueries({ queryKey: ["virtual-accounts"] });
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to create virtual account"),
  });
  const creatingVA = vaMutation.isPending;

  const submitVA = () => {
    const body = {
      customer_id: vaForm.customer_id.trim(),
      amount: toMinorUnits(vaForm.amount, vaForm.currency),
    };
    if (vaForm.invoice_id.trim()) body.invoice_id = vaForm.invoice_id.trim();
    vaMutation.mutate(body);
  };

  const paymentColumns = [
    {
      key: "customer",
      header: "Customer",
      cell: (p) => <CustomerName id={p.customer_id} names={names} />,
    },
    {
      key: "type",
      header: "Type",
      cell: (p) => <span className="capitalize">{(p.payment_type || "").replace("_", " ")}</span>,
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (p) => (
        <div>
          <Money amountMinor={p.amount} currency={p.currency} className="font-medium" />
          {p.tds_amount > 0 && (
            <p className="text-xs text-muted-foreground">
              + TDS <Money amountMinor={p.tds_amount} currency={p.currency} className="text-xs" />
            </p>
          )}
        </div>
      ),
    },
    {
      key: "reference",
      header: "Reference",
      cell: (p) => <span className="font-mono text-xs">{p.reference_number || "—"}</span>,
    },
    {
      key: "invoice",
      header: "Invoice",
      cell: (p) =>
        invoiceNumberById(p.invoice_id) ? (
          <span className="text-sm text-foreground">{invoiceNumberById(p.invoice_id)}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">{shortId(p.invoice_id)}</span>
        ),
    },
    {
      key: "recorded",
      header: "Recorded",
      cell: (p) => (
        <div>
          <span className="text-sm text-muted-foreground">{fmtDate(p.recorded_at)}</span>
          {p.recorded_by && <p className="text-xs text-muted-foreground">by {p.recorded_by}</p>}
        </div>
      ),
    },
  ];

  const vaColumns = [
    {
      key: "account",
      header: "Account",
      cell: (v) => (
        <div>
          <span className="font-mono text-sm">{v.account_number}</span>
          <p className="text-xs text-muted-foreground">
            {v.ifsc_code} · {v.bank_name}
          </p>
        </div>
      ),
    },
    {
      key: "customer",
      header: "Customer",
      cell: (v) => <CustomerName id={v.customer_id} names={names} />,
    },
    {
      key: "expected",
      header: "Expected",
      align: "right",
      cell: (v) => <Money amountMinor={v.amount_expected} currency={v.currency || "INR"} />,
    },
    {
      key: "received",
      header: "Received",
      align: "right",
      cell: (v) => (
        <Money amountMinor={v.amount_received} currency={v.currency || "INR"} className="font-medium" />
      ),
    },
    {
      key: "status",
      header: "Status",
      align: "right",
      cell: (v) => (
        <Badge variant={v.status === "active" ? "success" : "neutral"}>{v.status}</Badge>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Offline payments"
        description="Record money received outside the gateway, and issue virtual accounts for bank transfers."
        actions={
          tab === "payments" ? (
            <Button onClick={() => setRecordOpen(true)}>
              <Plus className="h-4 w-4" />
              Record payment
            </Button>
          ) : (
            <Button onClick={() => setVAOpen(true)}>
              <Plus className="h-4 w-4" />
              New virtual account
            </Button>
          )
        }
      />

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="payments">Recorded payments</TabsTrigger>
          <TabsTrigger value="virtual-accounts">Virtual accounts</TabsTrigger>
        </TabsList>

        <TabsContent value="payments" className="mt-6">
          <DataTable
            columns={paymentColumns}
            data={payments}
            loading={paymentsLoading}
            error={paymentsError}
            onRetry={refetchPayments}
            empty={{
              icon: Banknote,
              title: "No offline payments recorded",
              description: "Record NEFT, cash, or cheque receipts to settle invoices.",
            }}
          />
        </TabsContent>

        <TabsContent value="virtual-accounts" className="mt-6">
          <DataTable
            columns={vaColumns}
            data={vas}
            loading={vasLoading}
            error={vasError}
            onRetry={refetchVAs}
            empty={{
              icon: Landmark,
              title: "No virtual accounts",
              description: "Issue a dedicated account number a customer can transfer into.",
            }}
          />
        </TabsContent>
      </Tabs>

      {/* Record offline payment */}
      <FormSheet
        open={recordOpen}
        onOpenChange={setRecordOpen}
        title="Record offline payment"
        description="Money received outside the gateway — NEFT, cash, or cheque."
        onSubmit={submitRecord}
        submitLabel="Record payment"
        busyLabel="Recording…"
        busy={recording}
        canSubmit={Boolean(payForm.customer_id.trim() && payForm.amount)}
        dirty={Boolean(payForm.customer_id || payForm.amount)}
      >
            <div>
              <Label htmlFor="record-customer">Customer</Label>
              <CustomerSelect
                id="record-customer"
                value={payForm.customer_id}
                onChange={(v) => setPayForm({ ...payForm, customer_id: v, invoice_id: "" })}
                customers={customers}
              />
            </div>
            <div>
              <Label htmlFor="record-invoice">Invoice (optional — settles the invoice)</Label>
              <Select
                value={payForm.invoice_id}
                onValueChange={(v) => {
                  const inv = invoices.find((i) => i.id === v);
                  setPayForm((f) => ({
                    ...f,
                    invoice_id: v === "none" ? "" : v,
                    // Prefill the open amount when none was typed yet.
                    amount:
                      v !== "none" && inv && !f.amount ? String(fromMinorUnits(inv.total, inv.currency)) : f.amount,
                    currency: v !== "none" && inv ? inv.currency : f.currency,
                  }));
                }}
                disabled={!payForm.customer_id}
              >
                <SelectTrigger id="record-invoice">
                  <SelectValue
                    placeholder={
                      !payForm.customer_id
                        ? "Select a customer first"
                        : openInvoicesFor(payForm.customer_id).length === 0
                          ? "No unpaid invoices"
                          : "Settle an invoice"
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Not linked</SelectItem>
                  {openInvoicesFor(payForm.customer_id).map((i) => (
                    <SelectItem key={i.id} value={i.id}>
                      {invoiceLabel(i)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="type">Type</Label>
                <Select
                  value={payForm.payment_type}
                  onValueChange={(v) => setPayForm({ ...payForm, payment_type: v })}
                >
                  <SelectTrigger id="type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PAYMENT_TYPES.map((t) => (
                      <SelectItem key={t.value} value={t.value}>
                        {t.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="amount">Amount ({payForm.currency})</Label>
                <Input id="amount"
                  type="number"
                  min="0.01"
                  step="0.01"
                  value={payForm.amount}
                  onChange={(e) => setPayForm({ ...payForm, amount: e.target.value })}
                  placeholder="10000.00"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="tds-withheld-optional">TDS withheld (optional)</Label>
                <Input id="tds-withheld-optional"
                  type="number"
                  min="0"
                  step="0.01"
                  value={payForm.tds_amount}
                  onChange={(e) => setPayForm({ ...payForm, tds_amount: e.target.value })}
                  placeholder="0.00"
                />
              </div>
              <div>
                <Label htmlFor="reference-no">Reference no.</Label>
                <Input id="reference-no"
                  value={payForm.reference_number}
                  onChange={(e) => setPayForm({ ...payForm, reference_number: e.target.value })}
                  placeholder="UTR / cheque no."
                />
              </div>
            </div>
            <div>
              <Label htmlFor="notes-optional">Notes (optional)</Label>
              <Input id="notes-optional"
                value={payForm.notes}
                onChange={(e) => setPayForm({ ...payForm, notes: e.target.value })}
                placeholder="Anything worth remembering"
              />
            </div>
      </FormSheet>

      {/* New virtual account */}
      <FormSheet
        open={vaOpen}
        onOpenChange={setVAOpen}
        title="New virtual account"
        description="A dedicated account number the customer can transfer into."
        onSubmit={submitVA}
        submitLabel="Create account"
        busyLabel="Creating…"
        busy={creatingVA}
        canSubmit={Boolean(vaForm.customer_id.trim() && vaForm.amount)}
        dirty={Boolean(vaForm.customer_id || vaForm.amount)}
      >
            <div>
              <Label htmlFor="va-customer">Customer</Label>
              <CustomerSelect
                id="va-customer"
                value={vaForm.customer_id}
                onChange={(v) => setVAForm({ ...vaForm, customer_id: v, invoice_id: "" })}
                customers={customers}
              />
            </div>
            <div>
              <Label htmlFor="va-invoice">Invoice (optional)</Label>
              <Select
                value={vaForm.invoice_id}
                onValueChange={(v) => {
                  const inv = invoices.find((i) => i.id === v);
                  setVAForm((f) => ({
                    ...f,
                    invoice_id: v === "none" ? "" : v,
                    amount: v !== "none" && inv && !f.amount ? String(fromMinorUnits(inv.total, inv.currency)) : f.amount,
                  }));
                }}
                disabled={!vaForm.customer_id}
              >
                <SelectTrigger id="va-invoice">
                  <SelectValue
                    placeholder={
                      !vaForm.customer_id
                        ? "Select a customer first"
                        : openInvoicesFor(vaForm.customer_id).length === 0
                          ? "No unpaid invoices"
                          : "Link an invoice"
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Not linked</SelectItem>
                  {openInvoicesFor(vaForm.customer_id).map((i) => (
                    <SelectItem key={i.id} value={i.id}>
                      {invoiceLabel(i)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label htmlFor="expected-amount-inr">Expected amount (INR)</Label>
              <Input id="expected-amount-inr"
                type="number"
                min="0.01"
                step="0.01"
                value={vaForm.amount}
                onChange={(e) => setVAForm({ ...vaForm, amount: e.target.value })}
                placeholder="25000.00"
              />
            </div>
      </FormSheet>
    </div>
  );
};

export default OfflinePayments;
