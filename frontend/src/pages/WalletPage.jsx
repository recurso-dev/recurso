import { useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { endpoints } from "../lib/api";
import { useObjectQuery } from "@/lib/useObjectQuery";
import { formatDateTime, toMinorUnits, fromMinorUnits } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedRow,
  RelatedEmpty,
  ObjectPageSkeleton,
  ObjectNotFound,
  ObjectPageError,
} from "@/components/patterns/ObjectPage";
import { AttentionBanner } from "@/components/patterns/AttentionBanner";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import { useCustomers } from "@/lib/useCustomers";
import { Badge } from "@/components/ui/badge";
import { Money } from "@/components/ui/money";
import { CopyableId } from "@/components/ui/copyable-id";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
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

const TX_LABEL = {
  top_up: "Top-up",
  drain: "Drain",
  expiry: "Expiry",
  refund: "Refund",
  forfeit: "Forfeit",
};
const txLabel = (t) => TX_LABEL[t] || (t || "").replace(/_/g, " ");
const TX_SOURCE = { manual: "Manual", promotional: "Promotional", auto_recharge: "Auto-recharge" };
// Top-up adds; everything else removes drainable balance.
const isCredit = (t) => t === "top_up";
const TX_TONE = { top_up: "success", drain: "neutral", expiry: "destructive", refund: "neutral", forfeit: "destructive" };

const DAY = 24 * 60 * 60 * 1000;

/**
 * WalletPage — one prepaid wallet as a first-class object at /wallets/:id.
 * A wallet is money, so it gets the full treatment: current balance and how it
 * splits into refundable-paid vs forfeitable-promotional residue, the
 * auto-recharge rule, and the append-only movement ledger where each drain
 * links to the invoice it settled. The wallet's transactions ARE its audit
 * trail (wallets emit nothing to the /events feed).
 */
export default function WalletPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const { names } = useCustomers();

  const [topUpOpen, setTopUpOpen] = useState(false);
  const [topUpForm, setTopUpForm] = useState({ amount: "", source: "manual" });
  const [autoOpen, setAutoOpen] = useState(false);
  const [autoForm, setAutoForm] = useState({ threshold: "", amount: "" });
  const [closeOpen, setCloseOpen] = useState(false);
  const [closeResult, setCloseResult] = useState(null);
  const [actionError, setActionError] = useState(null);

  const {
    object: wallet,
    loading: walletLoading,
    notFound,
    isError,
    error: walletError,
    refetch,
  } = useObjectQuery(
    ["wallet", id],
    async () => (await endpoints.getWallet(id)).data.data,
    { enabled: Boolean(id) }
  );

  const { data: txs = [], isLoading: txsLoading } = useQuery({
    queryKey: ["wallet-transactions", id, "object-page"],
    queryFn: async () =>
      (await endpoints.getWalletTransactions(id, { limit: 100 })).data.data || [],
    enabled: Boolean(id),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["wallet", id] });
    queryClient.invalidateQueries({ queryKey: ["wallet-transactions", id] });
    queryClient.invalidateQueries({ queryKey: ["wallets"] });
  };

  const currency = wallet?.currency || "USD";
  const closed = Boolean(wallet?.closed_at);

  const topUpMutation = useMutation({
    mutationFn: () =>
      endpoints.topUpWallet(id, {
        amount: toMinorUnits(topUpForm.amount, currency),
        source: topUpForm.source,
      }),
    onSuccess: () => {
      setTopUpOpen(false);
      setTopUpForm({ amount: "", source: "manual" });
      invalidate();
    },
    onError: (err) => setActionError(err?.response?.data?.error?.message || "Top-up failed"),
  });

  const autoMutation = useMutation({
    mutationFn: (disable) =>
      endpoints.setWalletAutoRecharge(
        id,
        disable
          ? { auto_recharge_threshold: null, auto_recharge_amount: null }
          : {
              auto_recharge_threshold: toMinorUnits(autoForm.threshold, currency),
              auto_recharge_amount: toMinorUnits(autoForm.amount, currency),
            }
      ),
    onSuccess: () => {
      setAutoOpen(false);
      invalidate();
    },
    onError: (err) =>
      setActionError(err?.response?.data?.error?.message || "Failed to update auto-recharge"),
  });

  const closeMutation = useMutation({
    mutationFn: () => endpoints.closeWallet(id),
    onSuccess: (res) => {
      const { refunded = 0, forfeited = 0 } = res.data?.data || {};
      setCloseResult({ refunded, forfeited });
      setCloseOpen(false);
      invalidate();
    },
    onError: (err) => setActionError(err?.response?.data?.error?.message || "Failed to close wallet"),
  });

  const openAuto = () => {
    setActionError(null);
    setAutoForm({
      threshold: wallet?.auto_recharge_threshold
        ? String(fromMinorUnits(wallet.auto_recharge_threshold, currency))
        : "",
      amount: wallet?.auto_recharge_amount
        ? String(fromMinorUnits(wallet.auto_recharge_amount, currency))
        : "",
    });
    setAutoOpen(true);
  };

  if (walletLoading || txsLoading) return <ObjectPageSkeleton />;
  if (notFound) {
    return (
      <ObjectNotFound
        objectLabel="wallet"
        identifier={id ? String(id).slice(0, 8) : undefined}
        backTo="/wallets"
        backLabel="Wallets"
      />
    );
  }
  if (isError) {
    return <ObjectPageError objectLabel="wallet" error={walletError} onRetry={refetch} backTo="/wallets" backLabel="Wallets" />;
  }

  // Best-effort residue split: undrained top-up residue by source. Only shown
  // when the fetched movements fully account for the balance — otherwise we'd
  // be guessing, so we hide it rather than mislead.
  const openTopUps = txs.filter((t) => t.type === "top_up" && (t.remaining || 0) > 0);
  const paidResidue = openTopUps
    .filter((t) => t.source !== "promotional")
    .reduce((s, t) => s + (t.remaining || 0), 0);
  const promoResidue = openTopUps
    .filter((t) => t.source === "promotional")
    .reduce((s, t) => s + (t.remaining || 0), 0);
  const residueReconciles = paidResidue + promoResidue === wallet.balance;

  // Promotional residue expiring within 30 days — a real, dated warning.
  const now = Date.now();
  const expiringSoon = openTopUps.filter(
    (t) => t.expires_at && new Date(t.expires_at).getTime() - now < 30 * DAY
  );

  const attention = [];
  if (closed) {
    attention.push({
      tone: "warning",
      text: `This wallet was closed on ${formatDateTime(wallet.closed_at)}. It holds no balance and accepts no top-ups or drains.`,
    });
  }
  if (!closed && expiringSoon.length > 0) {
    const soonest = expiringSoon
      .map((t) => new Date(t.expires_at).getTime())
      .sort((a, b) => a - b)[0];
    attention.push({
      tone: "warning",
      text: `Promotional credit expires ${formatDateTime(new Date(soonest).toISOString())} — it will be written off if not spent.`,
    });
  }

  return (
    <div>
      <ObjectHeader
        backTo="/wallets"
        backLabel="Wallets"
        kicker="Wallet"
        title={
          <>
            <CustomerName id={wallet.customer_id} names={names} link={false} /> · {currency}
          </>
        }
        badge={
          closed ? (
            <Badge variant="neutral">Closed</Badge>
          ) : (
            <Badge variant="success">Open</Badge>
          )
        }
        meta={
          <>
            <span className="tabular-nums font-medium text-foreground">
              <Money amountMinor={wallet.balance} currency={currency} />
            </span>
            <span>balance</span>
            <CopyableId value={id} />
          </>
        }
        actions={
          closed ? null : (
            <>
              <Button variant="outline" onClick={openAuto}>
                Auto-recharge
              </Button>
              <Button
                variant="outline"
                onClick={() => {
                  setActionError(null);
                  setTopUpForm({ amount: "", source: "manual" });
                  setTopUpOpen(true);
                }}
              >
                Top up
              </Button>
              <Button
                variant="ghost"
                className="text-destructive hover:text-destructive"
                onClick={() => {
                  setActionError(null);
                  setCloseOpen(true);
                }}
              >
                Close
              </Button>
            </>
          )
        }
      />

      <AttentionBanner items={attention} className="mb-6" />

      <ObjectPageLayout
        rail={
          <>
            <ObjectSection title="Details">
              <AttributeList
                columns={1}
                items={[
                  { label: "Wallet ID", value: <CopyableId value={id} /> },
                  {
                    label: "Customer",
                    value: (
                      <Link
                        to={`/customers/${wallet.customer_id}`}
                        className="text-primary hover:underline"
                      >
                        <CustomerName id={wallet.customer_id} names={names} link={false} />
                      </Link>
                    ),
                  },
                  { label: "Currency", value: currency },
                  {
                    label: "Auto-recharge",
                    value: wallet.auto_recharge_threshold ? (
                      <span className="text-sm">
                        below{" "}
                        <Money amountMinor={wallet.auto_recharge_threshold} currency={currency} /> → +
                        <Money amountMinor={wallet.auto_recharge_amount} currency={currency} />
                      </span>
                    ) : (
                      <span className="text-muted-foreground">off</span>
                    ),
                  },
                  { label: "Created", value: formatDateTime(wallet.created_at) },
                  ...(closed
                    ? [{ label: "Closed", value: formatDateTime(wallet.closed_at) }]
                    : []),
                ]}
              />
            </ObjectSection>

            <ObjectSection title="Customer">
              <RelatedRow to={`/customers/${wallet.customer_id}`}>
                <span className="text-foreground">
                  <CustomerName id={wallet.customer_id} names={names} link={false} />
                </span>
                <span className="text-xs text-muted-foreground">View customer →</span>
              </RelatedRow>
            </ObjectSection>
          </>
        }
      >
        <ObjectSection title="Balance">
          <AttributeList
            columns={residueReconciles ? 3 : 1}
            items={[
              {
                label: "Drainable balance",
                value: (
                  <span className="font-mono text-lg font-medium tabular-nums">
                    <Money amountMinor={wallet.balance} currency={currency} />
                  </span>
                ),
              },
              ...(residueReconciles
                ? [
                    {
                      label: "Refundable (paid)",
                      value: (
                        <span className="font-mono tabular-nums">
                          <Money amountMinor={paidResidue} currency={currency} />
                        </span>
                      ),
                    },
                    {
                      label: "Forfeitable (promo)",
                      value: (
                        <span className="font-mono tabular-nums">
                          <Money amountMinor={promoResidue} currency={currency} />
                        </span>
                      ),
                    },
                  ]
                : []),
            ]}
          />
          <p className="mt-3 text-xs text-muted-foreground">
            Drained before adjustment credit notes and the payment gateway, oldest-expiring
            residue first.
            {residueReconciles
              ? " On closure the paid portion is refunded to the customer and the promotional portion is forfeited."
              : ""}
          </p>
        </ObjectSection>

        <ObjectSection title={`Movements${txs.length ? ` (${txs.length})` : ""}`} flush>
          {txs.length === 0 ? (
            <RelatedEmpty>
              No movements yet — top up this wallet to give the customer prepaid balance.
            </RelatedEmpty>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[640px] text-sm">
                <thead>
                  <tr className="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
                    <th className="px-6 py-2.5 font-medium">Date</th>
                    <th className="px-3 py-2.5 font-medium">Movement</th>
                    <th className="px-3 py-2.5 font-medium">Detail</th>
                    <th className="px-3 py-2.5 text-right font-medium">Amount</th>
                    <th className="px-6 py-2.5 text-right font-medium">Balance</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {txs.map((t) => (
                    <tr key={t.id} className="hover:bg-muted/20">
                      <td className="whitespace-nowrap px-6 py-2.5 text-muted-foreground">
                        {formatDateTime(t.created_at)}
                      </td>
                      <td className="px-3 py-2.5">
                        <Badge variant={TX_TONE[t.type] || "neutral"}>{txLabel(t.type)}</Badge>
                      </td>
                      <td className="px-3 py-2.5 text-muted-foreground">
                        {t.type === "top_up" && t.source ? TX_SOURCE[t.source] || t.source : null}
                        {t.type === "drain" && t.invoice_id ? (
                          <Link
                            to={`/invoices/${t.invoice_id}`}
                            className="text-primary hover:underline"
                          >
                            settled an invoice →
                          </Link>
                        ) : null}
                        {t.expires_at ? (
                          <span className="ml-1 text-xs">
                            expires {formatDateTime(t.expires_at)}
                          </span>
                        ) : null}
                      </td>
                      <td
                        className={`px-3 py-2.5 text-right font-mono tabular-nums ${
                          isCredit(t.type) ? "text-success" : "text-foreground"
                        }`}
                      >
                        {isCredit(t.type) ? "+" : "−"}
                        <Money amountMinor={t.amount} currency={currency} />
                      </td>
                      <td className="px-6 py-2.5 text-right font-mono tabular-nums text-muted-foreground">
                        <Money amountMinor={t.balance_after} currency={currency} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {txs.length >= 100 && (
            <p className="px-6 py-3 text-xs text-muted-foreground">
              Showing the 100 most recent movements.
            </p>
          )}
        </ObjectSection>
      </ObjectPageLayout>

      {/* Top up */}
      <Dialog open={topUpOpen} onOpenChange={(o) => !o && setTopUpOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Top up wallet</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label htmlFor="topup-amount">Amount ({currency})</Label>
              <Input
                id="topup-amount"
                type="number"
                min="0.01"
                step="0.01"
                value={topUpForm.amount}
                onChange={(e) => setTopUpForm({ ...topUpForm, amount: e.target.value })}
                placeholder="5000.00"
              />
            </div>
            <div>
              <Label htmlFor="topup-source">Source</Label>
              <Select
                value={topUpForm.source}
                onValueChange={(v) => setTopUpForm({ ...topUpForm, source: v })}
              >
                <SelectTrigger id="topup-source">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="manual">Manual (money received)</SelectItem>
                  <SelectItem value="promotional">Promotional credit (non-refundable)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {topUpForm.source === "promotional" && (
              <p className="text-xs text-muted-foreground">
                Promotional credit posts no cash — it's granted, and is forfeited (not refunded)
                if the wallet is closed.
              </p>
            )}
            {actionError && <p className="text-sm text-destructive">{actionError}</p>}
          </div>
          <DialogFooter>
            <Button
              onClick={() => {
                setActionError(null);
                topUpMutation.mutate();
              }}
              disabled={!topUpForm.amount || topUpMutation.isPending}
            >
              {topUpMutation.isPending ? "Topping up…" : "Top up"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Auto-recharge */}
      <Dialog open={autoOpen} onOpenChange={(o) => !o && setAutoOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Auto-recharge</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              When the balance falls below the threshold, the wallet is topped up by the recharge
              amount using the customer&apos;s saved payment method.
            </p>
            <div>
              <Label htmlFor="auto-threshold">Threshold ({currency})</Label>
              <Input
                id="auto-threshold"
                type="number"
                min="0.01"
                step="0.01"
                value={autoForm.threshold}
                onChange={(e) => setAutoForm({ ...autoForm, threshold: e.target.value })}
                placeholder="1000.00"
              />
            </div>
            <div>
              <Label htmlFor="auto-amount">Recharge amount ({currency})</Label>
              <Input
                id="auto-amount"
                type="number"
                min="0.01"
                step="0.01"
                value={autoForm.amount}
                onChange={(e) => setAutoForm({ ...autoForm, amount: e.target.value })}
                placeholder="5000.00"
              />
            </div>
            {actionError && <p className="text-sm text-destructive">{actionError}</p>}
          </div>
          <DialogFooter className="sm:justify-between">
            {wallet.auto_recharge_threshold ? (
              <Button
                variant="ghost"
                className="text-destructive hover:text-destructive"
                onClick={() => {
                  setActionError(null);
                  autoMutation.mutate(true);
                }}
              >
                Disable
              </Button>
            ) : (
              <span />
            )}
            <Button
              onClick={() => {
                setActionError(null);
                autoMutation.mutate(false);
              }}
              disabled={!autoForm.threshold || !autoForm.amount || autoMutation.isPending}
            >
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Close — irreversible settlement */}
      <ConfirmDialog
        open={closeOpen}
        onOpenChange={(o) => !o && setCloseOpen(false)}
        title="Close this wallet?"
        description={
          residueReconciles
            ? `The remaining balance settles now: ${fromMinorUnits(paidResidue, currency)} ${currency} is refunded to the customer and ${fromMinorUnits(promoResidue, currency)} ${currency} of promotional credit is forfeited. This can't be undone.`
            : `The remaining balance of ${fromMinorUnits(wallet.balance, currency)} ${currency} settles now: the paid portion is refunded to the customer, the promotional portion is forfeited. This can't be undone.`
        }
        confirmLabel="Close wallet"
        destructive
        busy={closeMutation.isPending}
        onConfirm={() => {
          setActionError(null);
          closeMutation.mutate();
        }}
      />

      {/* Settlement result */}
      <Dialog open={!!closeResult} onOpenChange={(o) => !o && setCloseResult(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Wallet closed</DialogTitle>
          </DialogHeader>
          <div className="space-y-2 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Refunded to customer</span>
              <span className="tabular-nums font-medium">
                <Money amountMinor={closeResult?.refunded || 0} currency={currency} />
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Promotional forfeited</span>
              <span className="tabular-nums font-medium">
                <Money amountMinor={closeResult?.forfeited || 0} currency={currency} />
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
}
