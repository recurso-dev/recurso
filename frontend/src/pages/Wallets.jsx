import { useEffect, useMemo, useState } from "react";
import { Plus, Wallet2 } from "lucide-react";

import { endpoints as api } from "../lib/api";
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
import { Label } from "@/components/ui/label";

const fmtMoney = (minor, currency) =>
  `${(minor / 100).toLocaleString(undefined, { minimumFractionDigits: 2 })} ${currency}`;

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
  const [txWallet, setTxWallet] = useState(null);
  const [txs, setTxs] = useState([]);
  const [actionError, setActionError] = useState(null);

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
      (w) => w.customer_id.toLowerCase().includes(q) || w.currency.toLowerCase().includes(q)
    );
  }, [wallets, search]);

  const submitCreate = async () => {
    setActionError(null);
    try {
      await api.createWallet(createForm);
      setCreateOpen(false);
      setCreateForm({ customer_id: "", currency: "INR" });
      fetchWallets();
    } catch (err) {
      setActionError(err?.response?.data?.error?.message || "Failed to create wallet");
    }
  };

  const submitTopUp = async () => {
    setActionError(null);
    try {
      await api.topUpWallet(topUpWallet.id, {
        amount: Math.round(parseFloat(topUpForm.amount) * 100),
        source: topUpForm.source,
      });
      setTopUpWallet(null);
      setTopUpForm({ amount: "", source: "manual" });
      fetchWallets();
    } catch (err) {
      setActionError(err?.response?.data?.error?.message || "Top-up failed");
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
      cell: (w) => <span className="font-mono text-xs text-muted-foreground">{w.customer_id}</span>,
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
      cell: (w) => (
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
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create wallet</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label>Customer ID</Label>
              <Input
                value={createForm.customer_id}
                onChange={(e) => setCreateForm({ ...createForm, customer_id: e.target.value })}
                placeholder="uuid"
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
          <DialogFooter>
            <Button onClick={submitCreate}>Create</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
              <select
                className="w-full rounded-md border border-border bg-white px-3 py-2 text-sm"
                value={topUpForm.source}
                onChange={(e) => setTopUpForm({ ...topUpForm, source: e.target.value })}
              >
                <option value="manual">Manual (money received)</option>
                <option value="promotional">Promotional credit</option>
              </select>
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
