import { useEffect, useMemo, useState } from "react";
import { Plus, Wallet2 } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { CustomerName, CustomerSelect } from "@/components/patterns/CustomerSelect";
import { useCustomers } from "@/lib/useCustomers";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toMinorUnits, fromMinorUnits, currencyDecimals } from "@/lib/utils";

const fmtMoney = (minor, currency) => {
  const d = currencyDecimals(currency);
  return `${fromMinorUnits(minor, currency).toLocaleString(undefined, { minimumFractionDigits: d, maximumFractionDigits: d })} ${currency}`;
};

// Prepaid wallets (Lago-parity B1): balances, top-ups, and movement history.
const Wallets = () => {
  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [search, setSearch] = useState("");

  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ customer_id: "", currency: "INR" });
  const [topUpWallet, setTopUpWallet] = useState(null);
  const [topUpForm, setTopUpForm] = useState({ amount: "", source: "manual" });
  const [autoWallet, setAutoWallet] = useState(null);
  const [autoForm, setAutoForm] = useState({ threshold: "", amount: "" });
  const [txWallet, setTxWallet] = useState(null);
  const [txs, setTxs] = useState([]);
  const [actionError, setActionError] = useState(null);
  const [creating, setCreating] = useState(false);
  const { customers, names } = useCustomers();

  const fetchWallets = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getWallets();
      setWallets(res.data.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || err?.message || "Failed to load wallets");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWallets();
  }, []);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return wallets;
    return wallets.filter(
      (w) =>
        w.customer_id.toLowerCase().includes(q) ||
        (names[w.customer_id] || "").toLowerCase().includes(q) ||
        w.currency.toLowerCase().includes(q)
    );
  }, [wallets, search, names]);

  const submitCreate = async () => {
    setActionError(null);
    setCreating(true);
    try {
      await api.createWallet(createForm);
      setCreateOpen(false);
      setCreateForm({ customer_id: "", currency: "INR" });
      fetchWallets();
    } catch (err) {
      setActionError(err?.response?.data?.error?.message || "Failed to create wallet");
    } finally {
      setCreating(false);
    }
  };

  const submitTopUp = async () => {
    setActionError(null);
    try {
      await api.topUpWallet(topUpWallet.id, {
        amount: toMinorUnits(topUpForm.amount, topUpWallet.currency),
        source: topUpForm.source,
      });
      setTopUpWallet(null);
      setTopUpForm({ amount: "", source: "manual" });
      fetchWallets();
    } catch (err) {
      setActionError(err?.response?.data?.error?.message || "Top-up failed");
    }
  };

  const openAutoRecharge = (wallet) => {
    setActionError(null);
    setAutoWallet(wallet);
    setAutoForm({
      threshold: wallet.auto_recharge_threshold ? String(fromMinorUnits(wallet.auto_recharge_threshold, wallet.currency)) : "",
      amount: wallet.auto_recharge_amount ? String(fromMinorUnits(wallet.auto_recharge_amount, wallet.currency)) : "",
    });
  };

  // Backend requires threshold+amount together (both positive) or both null to clear.
  const submitAutoRecharge = async (disable = false) => {
    setActionError(null);
    try {
      const body = disable
        ? { auto_recharge_threshold: null, auto_recharge_amount: null }
        : {
            auto_recharge_threshold: toMinorUnits(autoForm.threshold, autoWallet.currency),
            auto_recharge_amount: toMinorUnits(autoForm.amount, autoWallet.currency),
          };
      await api.setWalletAutoRecharge(autoWallet.id, body);
      setAutoWallet(null);
      fetchWallets();
    } catch (err) {
      setActionError(err?.response?.data?.error?.message || "Failed to update auto-recharge");
    }
  };

  const openTransactions = async (wallet) => {
    setTxWallet(wallet);
    setTxs([]);
    try {
      const res = await api.getWalletTransactions(wallet.id, { limit: 50 });
      setTxs(res.data.data || []);
    } catch {
      setTxs([]);
    }
  };

  const columns = [
    {
      key: "customer",
      header: "Customer",
      cell: (w) => <CustomerName id={w.customer_id} names={names} />,
    },
    {
      key: "balance",
      header: "Balance",
      cell: (w) => (
        <span className="tabular-nums font-medium text-foreground">
          {fmtMoney(w.balance, w.currency)}
        </span>
      ),
    },
    {
      key: "auto",
      header: "Auto-recharge",
      cell: (w) =>
        w.auto_recharge_threshold ? (
          <Badge variant="success">
            below {fmtMoney(w.auto_recharge_threshold, w.currency)} → +
            {fmtMoney(w.auto_recharge_amount, w.currency)}
          </Badge>
        ) : (
          <span className="text-muted-foreground">off</span>
        ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (w) => (
        <div className="flex justify-end gap-2">
          <Button
            size="sm"
            variant="ghost"
            onClick={(e) => {
              e.stopPropagation();
              openAutoRecharge(w);
            }}
          >
            Auto-recharge
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={(e) => {
              e.stopPropagation();
              setTopUpWallet(w);
            }}
          >
            Top up
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Wallets"
        description="Prepaid balances drained before credit notes and the payment gateway."
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            Create wallet
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={filtered}
        loading={loading}
        error={error}
        onRetry={fetchWallets}
        onRowClick={openTransactions}
        search={{ value: search, onChange: setSearch, placeholder: "Search by customer or currency..." }}
        empty={{
          icon: Wallet2,
          title: "No wallets yet",
          description: "Create a wallet to hold prepaid balance for a customer.",
          action: (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              Create wallet
            </Button>
          ),
        }}
      />

      {/* Create wallet */}
      <Sheet open={createOpen} onOpenChange={setCreateOpen}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>Create wallet</SheetTitle>
            <SheetDescription>
              A prepaid balance drained before credit notes and the payment gateway.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 overflow-y-auto px-6">
            <div>
              <Label>Customer</Label>
              <CustomerSelect
                value={createForm.customer_id}
                onChange={(v) => setCreateForm({ ...createForm, customer_id: v })}
                customers={customers}
              />
            </div>
            <div>
              <Label>Currency</Label>
              <Input
                value={createForm.currency}
                onChange={(e) =>
                  setCreateForm({ ...createForm, currency: e.target.value.toUpperCase() })
                }
                maxLength={3}
              />
            </div>
            {actionError && <p className="text-sm text-red-600">{actionError}</p>}
          </div>
          <SheetFooter>
            <Button onClick={submitCreate} disabled={creating || !createForm.customer_id}>
              {creating ? "Creating…" : "Create wallet"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      {/* Top up */}
      <Dialog open={!!topUpWallet} onOpenChange={(open) => !open && setTopUpWallet(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Top up wallet</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label>Amount ({topUpWallet?.currency})</Label>
              <Input
                type="number"
                min="0.01"
                step="0.01"
                value={topUpForm.amount}
                onChange={(e) => setTopUpForm({ ...topUpForm, amount: e.target.value })}
                placeholder="5000.00"
              />
            </div>
            <div>
              <Label>Source</Label>
              <Select
                value={topUpForm.source}
                onValueChange={(v) => setTopUpForm({ ...topUpForm, source: v })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="manual">Manual (money received)</SelectItem>
                  <SelectItem value="promotional">Promotional credit</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {actionError && <p className="text-sm text-red-600">{actionError}</p>}
          </div>
          <DialogFooter>
            <Button onClick={submitTopUp} disabled={!topUpForm.amount}>
              Top up
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Auto-recharge */}
      <Dialog open={!!autoWallet} onOpenChange={(open) => !open && setAutoWallet(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Auto-recharge</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              When the balance falls below the threshold, the wallet is automatically
              topped up by the recharge amount using the customer&apos;s saved payment method.
            </p>
            <div>
              <Label>Threshold ({autoWallet?.currency})</Label>
              <Input
                type="number"
                min="0.01"
                step="0.01"
                value={autoForm.threshold}
                onChange={(e) => setAutoForm({ ...autoForm, threshold: e.target.value })}
                placeholder="1000.00"
              />
            </div>
            <div>
              <Label>Recharge amount ({autoWallet?.currency})</Label>
              <Input
                type="number"
                min="0.01"
                step="0.01"
                value={autoForm.amount}
                onChange={(e) => setAutoForm({ ...autoForm, amount: e.target.value })}
                placeholder="5000.00"
              />
            </div>
            {actionError && <p className="text-sm text-red-600">{actionError}</p>}
          </div>
          <DialogFooter className="sm:justify-between">
            {autoWallet?.auto_recharge_threshold ? (
              <Button
                variant="ghost"
                className="text-red-600 hover:text-red-600"
                onClick={() => submitAutoRecharge(true)}
              >
                Disable
              </Button>
            ) : (
              <span />
            )}
            <Button
              onClick={() => submitAutoRecharge(false)}
              disabled={!autoForm.threshold || !autoForm.amount}
            >
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Transactions */}
      <Dialog open={!!txWallet} onOpenChange={(open) => !open && setTxWallet(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Wallet transactions</DialogTitle>
          </DialogHeader>
          <div className="max-h-96 space-y-1 overflow-y-auto text-sm">
            {txs.length === 0 && <p className="text-muted-foreground">No movements yet.</p>}
            {txs.map((t) => (
              <div key={t.id} className="flex items-center justify-between border-b border-border py-2">
                <div className="flex items-center gap-2">
                  <Badge
                    variant={
                      t.type === "top_up" ? "success" : t.type === "drain" ? "neutral" : "destructive"
                    }
                  >
                    {t.type}
                  </Badge>
                  <span className="text-xs text-muted-foreground">
                    {new Date(t.created_at).toLocaleString()}
                    {t.source ? ` · ${t.source}` : ""}
                  </span>
                </div>
                <div className="tabular-nums">
                  {t.type === "top_up" ? "+" : "−"}
                  {fmtMoney(t.amount, txWallet?.currency || "")}
                  <span className="ml-2 text-xs text-muted-foreground">
                    → {fmtMoney(t.balance_after, txWallet?.currency || "")}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Wallets;
