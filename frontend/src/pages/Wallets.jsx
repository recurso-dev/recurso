import { useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
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

const fmtWhen = (x) =>
  new Date(x).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });

// Human labels for movement types; unknown types fall back to a cleaned slug.
const TX_LABEL = { top_up: "Top-up", drain: "Drain", refund: "Refund", forfeit: "Forfeit" };
const txLabel = (t) =>
  TX_LABEL[t] || (t || "").replace(/_/g, " ").replace(/^\w/, (c) => c.toUpperCase());
const TX_SOURCE = { manual: "Manual", promotional: "Promotional credit", auto_recharge: "Auto-recharge" };

// Prepaid wallets (Lago-parity B1): balances, top-ups, and movement history.
const Wallets = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");

  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ customer_id: "", currency: "INR" });
  const [topUpWallet, setTopUpWallet] = useState(null);
  const [topUpForm, setTopUpForm] = useState({ amount: "", source: "manual" });
  const [autoWallet, setAutoWallet] = useState(null);
  const [autoForm, setAutoForm] = useState({ threshold: "", amount: "" });
  const [txWallet, setTxWallet] = useState(null);
  const [closingWallet, setClosingWallet] = useState(null);
  const [closeResult, setCloseResult] = useState(null);
  const [actionError, setActionError] = useState(null);
  const { customers, names } = useCustomers();

  const {
    data: wallets = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["wallets"],
    queryFn: async () => (await api.getWallets()).data.data || [],
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load wallets"
    : null;
  const invalidateWallets = () => queryClient.invalidateQueries({ queryKey: ["wallets"] });

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

  const createMutation = useMutation({
    mutationFn: () => api.createWallet(createForm),
    onSuccess: () => {
      setCreateOpen(false);
      setCreateForm({ customer_id: "", currency: "INR" });
      invalidateWallets();
    },
    onError: (err) =>
      setActionError(err?.response?.data?.error?.message || "Failed to create wallet"),
  });
  const creating = createMutation.isPending;
  const submitCreate = () => {
    setActionError(null);
    createMutation.mutate();
  };

  const topUpMutation = useMutation({
    mutationFn: () =>
      api.topUpWallet(topUpWallet.id, {
        amount: toMinorUnits(topUpForm.amount, topUpWallet.currency),
        source: topUpForm.source,
      }),
    onSuccess: () => {
      setTopUpWallet(null);
      setTopUpForm({ amount: "", source: "manual" });
      invalidateWallets();
    },
    onError: (err) => setActionError(err?.response?.data?.error?.message || "Top-up failed"),
  });
  const submitTopUp = () => {
    setActionError(null);
    topUpMutation.mutate();
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
  const autoMutation = useMutation({
    mutationFn: (disable) =>
      api.setWalletAutoRecharge(
        autoWallet.id,
        disable
          ? { auto_recharge_threshold: null, auto_recharge_amount: null }
          : {
              auto_recharge_threshold: toMinorUnits(autoForm.threshold, autoWallet.currency),
              auto_recharge_amount: toMinorUnits(autoForm.amount, autoWallet.currency),
            }
      ),
    onSuccess: () => {
      setAutoWallet(null);
      invalidateWallets();
    },
    onError: (err) =>
      setActionError(err?.response?.data?.error?.message || "Failed to update auto-recharge"),
  });
  const submitAutoRecharge = (disable = false) => {
    setActionError(null);
    autoMutation.mutate(disable);
  };

  // Closing settles the wallet: paid balance is refunded to the customer,
  // promotional balance is forfeited. Irreversible, so it goes through a
  // ConfirmDialog and the settlement result is surfaced afterward.
  const closeMutation = useMutation({
    mutationFn: () => api.closeWallet(closingWallet.id),
    onSuccess: (res) => {
      const { refunded = 0, forfeited = 0 } = res.data?.data || {};
      setCloseResult({ currency: closingWallet.currency, refunded, forfeited });
      setClosingWallet(null);
      invalidateWallets();
    },
    onError: (err) =>
      setActionError(err?.response?.data?.error?.message || "Failed to close wallet"),
  });
  const closing = closeMutation.isPending;
  const submitClose = () => {
    if (!closingWallet) return;
    setActionError(null);
    closeMutation.mutate();
  };

  // Transactions for the open wallet's sheet; cached per wallet.
  const { data: txs = [], isLoading: txsLoading } = useQuery({
    queryKey: ["wallet-transactions", txWallet?.id],
    queryFn: async () =>
      (await api.getWalletTransactions(txWallet.id, { limit: 50 })).data.data || [],
    enabled: !!txWallet,
  });

  const openTransactions = (wallet) => setTxWallet(wallet);

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
      cell: (w) =>
        w.closed_at ? (
          <div className="flex justify-end">
            <Badge variant="neutral">Closed</Badge>
          </div>
        ) : (
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
            <Button
              size="sm"
              variant="ghost"
              className="text-red-600 hover:text-red-600"
              onClick={(e) => {
                e.stopPropagation();
                setActionError(null);
                setClosingWallet(w);
              }}
            >
              Close
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
        onRetry={refetch}
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

      {/* Transactions — the wallet's movement history, house detail-Sheet style */}
      <Sheet open={!!txWallet} onOpenChange={(open) => !open && setTxWallet(null)}>
        <SheetContent side="right" className="w-full sm:max-w-md overflow-y-auto">
          {txWallet && (
            <>
              <SheetHeader>
                <SheetTitle>Wallet activity</SheetTitle>
                <SheetDescription>
                  <CustomerName id={txWallet.customer_id} names={names} /> · balance{" "}
                  <span className="tabular-nums font-medium text-foreground">
                    {fmtMoney(txWallet.balance, txWallet.currency)}
                  </span>
                </SheetDescription>
              </SheetHeader>

              <div className="mt-2 px-6 pb-6">
                {txsLoading ? (
                  <div className="space-y-3" aria-busy="true" aria-label="Loading transactions">
                    {[0, 1, 2].map((i) => (
                      <div key={i} className="h-12 animate-pulse rounded-md bg-muted" />
                    ))}
                  </div>
                ) : txs.length === 0 ? (
                  <p className="py-8 text-center text-sm text-muted-foreground">
                    No movements yet — top up this wallet to get started.
                  </p>
                ) : (
                  <ul className="divide-y divide-border">
                    {txs.map((t) => (
                      <li key={t.id} className="flex items-center justify-between gap-3 py-3">
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <Badge
                              variant={
                                t.type === "top_up"
                                  ? "success"
                                  : t.type === "drain"
                                    ? "neutral"
                                    : "destructive"
                              }
                            >
                              {txLabel(t.type)}
                            </Badge>
                            {t.source && (
                              <span className="truncate text-xs text-muted-foreground">
                                {TX_SOURCE[t.source] || t.source}
                              </span>
                            )}
                          </div>
                          <p className="mt-1 text-xs text-muted-foreground">{fmtWhen(t.created_at)}</p>
                        </div>
                        <div className="shrink-0 text-right">
                          <p
                            className={`tabular-nums text-sm font-medium ${
                              t.type === "top_up" ? "text-emerald-600" : "text-foreground"
                            }`}
                          >
                            {t.type === "top_up" ? "+" : "−"}
                            {fmtMoney(t.amount, txWallet.currency)}
                          </p>
                          <p className="tabular-nums text-xs text-muted-foreground">
                            balance {fmtMoney(t.balance_after, txWallet.currency)}
                          </p>
                        </div>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </>
          )}
        </SheetContent>
      </Sheet>

      {/* Close wallet — irreversible settlement */}
      <ConfirmDialog
        open={!!closingWallet}
        onOpenChange={(open) => !open && setClosingWallet(null)}
        title="Close this wallet?"
        description={
          closingWallet
            ? `The remaining ${fmtMoney(closingWallet.balance, closingWallet.currency)} will be settled: paid balance is refunded to the customer, promotional balance is forfeited. This can't be undone.`
            : ""
        }
        confirmLabel="Close wallet"
        destructive
        busy={closing}
        onConfirm={submitClose}
      />

      {/* Settlement result */}
      <Dialog open={!!closeResult} onOpenChange={(open) => !open && setCloseResult(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Wallet closed</DialogTitle>
          </DialogHeader>
          <div className="space-y-2 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Refunded to customer</span>
              <span className="tabular-nums font-medium">
                {fmtMoney(closeResult?.refunded || 0, closeResult?.currency || "")}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Promotional forfeited</span>
              <span className="tabular-nums font-medium">
                {fmtMoney(closeResult?.forfeited || 0, closeResult?.currency || "")}
              </span>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setCloseResult(null)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Wallets;
