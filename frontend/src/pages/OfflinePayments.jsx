import { useEffect, useState } from "react";
import { Plus, Banknote, Landmark } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
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

const PAYMENT_TYPES = [
  { value: "bank_transfer", label: "Bank transfer" },
  { value: "cash", label: "Cash" },
  { value: "cheque", label: "Cheque" },
];

const shortId = (id) => (id ? String(id).slice(0, 8) : "—");
const fmtDate = (v) => (v ? new Date(v).toLocaleString() : "—");

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
  const [payments, setPayments] = useState([]);
  const [paymentsLoading, setPaymentsLoading] = useState(true);
  const [paymentsError, setPaymentsError] = useState(null);
  const [vas, setVAs] = useState([]);
  const [vasLoading, setVAsLoading] = useState(true);
  const [vasError, setVAsError] = useState(null);
  const [recordOpen, setRecordOpen] = useState(false);
  const [payForm, setPayForm] = useState(emptyPayment);
  const [recording, setRecording] = useState(false);
  const [vaOpen, setVAOpen] = useState(false);
  const [vaForm, setVAForm] = useState(emptyVA);
  const [creatingVA, setCreatingVA] = useState(false);
  const [tab, setTab] = useState("payments");

  const fetchPayments = async () => {
    setPaymentsLoading(true);
    setPaymentsError(null);
    try {
      const res = await api.getOfflinePayments();
      setPayments(res.data.data || []);
    } catch (err) {
      setPaymentsError(err?.response?.data?.error?.message || "Failed to load payments");
    } finally {
      setPaymentsLoading(false);
    }
  };

  const fetchVAs = async () => {
    setVAsLoading(true);
    setVAsError(null);
    try {
      const res = await api.getVirtualAccounts();
      setVAs(res.data.data || []);
    } catch (err) {
      setVAsError(err?.response?.data?.error?.message || "Failed to load virtual accounts");
    } finally {
      setVAsLoading(false);
    }
  };

  useEffect(() => {
    fetchPayments();
    fetchVAs();
  }, []);

  const submitRecord = async () => {
    setRecording(true);
    try {
      const body = {
        customer_id: payForm.customer_id.trim(),
        payment_type: payForm.payment_type,
        amount: Math.round(parseFloat(payForm.amount) * 100),
        currency: payForm.currency,
        reference_number: payForm.reference_number.trim(),
        notes: payForm.notes.trim(),
      };
      if (payForm.invoice_id.trim()) body.invoice_id = payForm.invoice_id.trim();
      if (payForm.tds_amount) body.tds_amount = Math.round(parseFloat(payForm.tds_amount) * 100);
      await api.recordOfflinePayment(body);
      toast.success("Payment recorded.");
      setRecordOpen(false);
      setPayForm(emptyPayment);
      fetchPayments();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to record payment");
    } finally {
      setRecording(false);
    }
  };

  const submitVA = async () => {
    setCreatingVA(true);
    try {
      const body = {
        customer_id: vaForm.customer_id.trim(),
        amount: Math.round(parseFloat(vaForm.amount) * 100),
      };
      if (vaForm.invoice_id.trim()) body.invoice_id = vaForm.invoice_id.trim();
      await api.createVirtualAccount(body);
      toast.success("Virtual account created.");
      setVAOpen(false);
      setVAForm(emptyVA);
      fetchVAs();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to create virtual account");
    } finally {
      setCreatingVA(false);
    }
  };

  const paymentColumns = [
    {
      key: "customer",
      header: "Customer",
      cell: (p) => <span className="font-mono text-xs text-muted-foreground">{shortId(p.customer_id)}</span>,
    },
    {
      key: "type",
      header: "Type",
      cell: (p) => <span className="capitalize">{(p.payment_type || "").replace("_", " ")}</span>,
    },
    {
      key: "amount",
      header: "Amount",
      cell: (p) => (
        <div>
          <span className="tabular-nums font-medium">{formatCurrency(p.amount, p.currency)}</span>
          {p.tds_amount > 0 && (
            <p className="text-xs text-muted-foreground">
              + TDS {formatCurrency(p.tds_amount, p.currency)}
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
      cell: (p) => <span className="font-mono text-xs text-muted-foreground">{shortId(p.invoice_id)}</span>,
    },
    {
      key: "recorded",
      header: "Recorded",
      align: "right",
      cell: (p) => (
        <div className="text-right">
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
      cell: (v) => <span className="font-mono text-xs text-muted-foreground">{shortId(v.customer_id)}</span>,
    },
    {
      key: "expected",
      header: "Expected",
      cell: (v) => <span className="tabular-nums">{formatCurrency(v.amount_expected, "INR")}</span>,
    },
    {
      key: "received",
      header: "Received",
      cell: (v) => (
        <span className="tabular-nums font-medium">{formatCurrency(v.amount_received, "INR")}</span>
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
            onRetry={fetchPayments}
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
            onRetry={fetchVAs}
            empty={{
              icon: Landmark,
              title: "No virtual accounts",
              description: "Issue a dedicated account number a customer can transfer into.",
            }}
          />
        </TabsContent>
      </Tabs>

      {/* Record offline payment */}
      <Dialog open={recordOpen} onOpenChange={setRecordOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Record offline payment</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label>Customer ID</Label>
              <Input
                value={payForm.customer_id}
                onChange={(e) => setPayForm({ ...payForm, customer_id: e.target.value })}
                placeholder="uuid"
                className="font-mono"
              />
            </div>
            <div>
              <Label>Invoice ID (optional — settles the invoice)</Label>
              <Input
                value={payForm.invoice_id}
                onChange={(e) => setPayForm({ ...payForm, invoice_id: e.target.value })}
                placeholder="uuid"
                className="font-mono"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>Type</Label>
                <Select
                  value={payForm.payment_type}
                  onValueChange={(v) => setPayForm({ ...payForm, payment_type: v })}
                >
                  <SelectTrigger>
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
                <Label>Amount ({payForm.currency})</Label>
                <Input
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
                <Label>TDS withheld (optional)</Label>
                <Input
                  type="number"
                  min="0"
                  step="0.01"
                  value={payForm.tds_amount}
                  onChange={(e) => setPayForm({ ...payForm, tds_amount: e.target.value })}
                  placeholder="0.00"
                />
              </div>
              <div>
                <Label>Reference no.</Label>
                <Input
                  value={payForm.reference_number}
                  onChange={(e) => setPayForm({ ...payForm, reference_number: e.target.value })}
                  placeholder="UTR / cheque no."
                />
              </div>
            </div>
            <div>
              <Label>Notes (optional)</Label>
              <Input
                value={payForm.notes}
                onChange={(e) => setPayForm({ ...payForm, notes: e.target.value })}
                placeholder="Anything worth remembering"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={submitRecord}
              disabled={recording || !payForm.customer_id.trim() || !payForm.amount}
            >
              {recording ? "Recording…" : "Record payment"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* New virtual account */}
      <Dialog open={vaOpen} onOpenChange={setVAOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New virtual account</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label>Customer ID</Label>
              <Input
                value={vaForm.customer_id}
                onChange={(e) => setVAForm({ ...vaForm, customer_id: e.target.value })}
                placeholder="uuid"
                className="font-mono"
              />
            </div>
            <div>
              <Label>Invoice ID (optional)</Label>
              <Input
                value={vaForm.invoice_id}
                onChange={(e) => setVAForm({ ...vaForm, invoice_id: e.target.value })}
                placeholder="uuid"
                className="font-mono"
              />
            </div>
            <div>
              <Label>Expected amount (INR)</Label>
              <Input
                type="number"
                min="0.01"
                step="0.01"
                value={vaForm.amount}
                onChange={(e) => setVAForm({ ...vaForm, amount: e.target.value })}
                placeholder="25000.00"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={submitVA}
              disabled={creatingVA || !vaForm.customer_id.trim() || !vaForm.amount}
            >
              {creatingVA ? "Creating…" : "Create account"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default OfflinePayments;
