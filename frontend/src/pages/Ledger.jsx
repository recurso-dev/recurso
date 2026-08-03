import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { BookOpen } from "lucide-react";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Posting codes (ADR-002): what each movement IS, in words. "Code 3" means
// nothing to an operator; "Payment" does.
const CODE_LABEL = {
  1: "Invoice raised",
  2: "Revenue recognized",
  3: "Payment",
  4: "Refund",
  5: "Refund — deferred reversal",
  6: "Output tax",
  7: "Credit applied",
  8: "Credit note",
  9: "Refund — tax reversal",
  10: "TDS receivable",
  11: "Wallet top-up",
  12: "Wallet drain",
  13: "Wallet refund",
  14: "Wallet forfeit",
  15: "Wallet expiry",
  16: "Downgrade credit",
  17: "Downgrade — tax reversal",
  18: "Credit expiry",
  19: "Payment reversal",
  20: "Credit void",
  21: "Downgrade — revenue reversal",
};
const codeLabel = (c) => CODE_LABEL[c] || `Code ${c}`;

const fmtWhen = (x) =>
  x ? new Date(x).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }) : "—";

export default function Ledger() {
  const [selectedAccountId, setSelectedAccountId] = useState("");
  const [selectedEntry, setSelectedEntry] = useState(null);

  const accountsQuery = useQuery({
    queryKey: ["ledger-accounts"],
    queryFn: async () => (await endpoints.getLedgerAccounts()).data.data || [],
  });
  // Stable ref (only changes with the query result) so the effect/memo below
  // that depend on `accounts` don't re-run every render.
  const accounts = useMemo(() => accountsQuery.data ?? [], [accountsQuery.data]);
  const loading = accountsQuery.isLoading;
  const error = accountsQuery.error ? "Failed to load accounts." : null;

  // Entries for the selected account; disabled until one is chosen.
  const entriesQuery = useQuery({
    queryKey: ["ledger-entries", selectedAccountId],
    queryFn: async () =>
      (await endpoints.getLedgerEntries({ account_id: selectedAccountId, limit: 50 })).data.data || [],
    enabled: !!selectedAccountId,
  });
  const entries = entriesQuery.data ?? [];
  const entriesLoading = entriesQuery.isFetching;
  // Every customer has their own AR sub-account (same name + code 1100, id ==
  // customer id) — label them with the customer so the picker isn't a wall of
  // identical "Accounts Receivable (1100)" rows.
  const { names: customerNames } = useCustomers();

  const accountLabel = (acc) =>
    customerNames[acc.id]
      ? `${acc.name} — ${customerNames[acc.id]} (${acc.code})`
      : `${acc.name} (${acc.code})`;

  const accountLabelById = (id) => {
    const acc = accounts.find((a) => a.id === id);
    if (acc) return accountLabel(acc);
    return null;
  };

  // Auto-select the first account once accounts load (matches prior behavior).
  useEffect(() => {
    if (!selectedAccountId && accounts.length > 0) {
      setSelectedAccountId(accounts[0].id);
    }
  }, [accounts, selectedAccountId]);

  const selectedAccount = useMemo(
    () => accounts.find((a) => a.id === selectedAccountId),
    [accounts, selectedAccountId]
  );

  const columns = [
    {
      key: "when",
      header: "Date",
      cell: (e) => (
        <span className="whitespace-nowrap text-sm text-muted-foreground">{fmtWhen(e.timestamp)}</span>
      ),
    },
    {
      key: "type",
      header: "Posting",
      cell: (e) => <Badge variant="neutral">{codeLabel(e.code)}</Badge>,
    },
    {
      key: "description",
      header: "Description",
      cell: (e) => (
        <span className="block max-w-xs truncate text-sm text-foreground" title={e.description || undefined}>
          {e.description || <span className="text-muted-foreground">—</span>}
        </span>
      ),
    },
    {
      key: "debit",
      header: "Debit",
      cell: (e) =>
        accountLabelById(e.debit_account_id) ? (
          <span className="text-sm text-foreground">{accountLabelById(e.debit_account_id)}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">{String(e.debit_account_id).slice(0, 8)}…</span>
        ),
    },
    {
      key: "credit",
      header: "Credit",
      cell: (e) =>
        accountLabelById(e.credit_account_id) ? (
          <span className="text-sm text-foreground">{accountLabelById(e.credit_account_id)}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">{String(e.credit_account_id).slice(0, 8)}…</span>
        ),
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (e) => (
        <span className="font-medium tabular-nums text-foreground">
          {formatCurrency(e.amount, selectedAccount?.currency)}
        </span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Financial Ledger"
        description="Double-entry ledger transactions and account balances. PostgreSQL is the authoritative ledger; TigerBeetle, when enabled, is an optional mirror."
      />

      {/* Account selector + current balance */}
      <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-3">
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">
            Select account
          </label>
          <Select
            value={selectedAccountId}
            onValueChange={setSelectedAccountId}
            disabled={loading || accounts.length === 0}
          >
            <SelectTrigger>
              <SelectValue
                placeholder={
                  loading
                    ? "Loading accounts..."
                    : accounts.length === 0
                      ? "No accounts found"
                      : "Select account"
                }
              />
            </SelectTrigger>
            <SelectContent>
              {accounts.map((acc) => (
                <SelectItem key={acc.id} value={acc.id}>
                  {accountLabel(acc)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {selectedAccount && (
          <StatCard
            className="md:col-span-1"
            label="Current Balance"
            value={formatCurrency(selectedAccount.balance || 0, selectedAccount.currency)}
            icon={BookOpen}
            hint={accountLabel(selectedAccount)}
          />
        )}
      </div>

      <DataTable
        columns={columns}
        data={entries}
        loading={entriesLoading}
        error={error}
        onRetry={accountsQuery.refetch}
        onRowClick={(e) => setSelectedEntry(e)}
        getRowId={(e) => e.id}
        empty={{
          icon: BookOpen,
          title: "No entries found",
          description: "No ledger entries were found for this account.",
        }}
      />

      {/* Entry detail: the full posting — both legs, reference, ids. This is
          the "explain any number" surface for a single ledger movement. */}
      <Sheet open={!!selectedEntry} onOpenChange={(o) => !o && setSelectedEntry(null)}>
        <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
          {selectedEntry && (
            <>
              <SheetHeader>
                <SheetTitle className="flex items-center gap-2">
                  Ledger posting
                  <Badge variant="neutral">{codeLabel(selectedEntry.code)}</Badge>
                </SheetTitle>
                <SheetDescription>{fmtWhen(selectedEntry.timestamp)}</SheetDescription>
              </SheetHeader>
              <div className="mt-2 space-y-5 px-6 pb-6">
                <div className="rounded-lg border border-border bg-muted/20 p-4 text-center">
                  <p className="text-2xl font-semibold tabular-nums text-foreground">
                    {formatCurrency(selectedEntry.amount, selectedAccount?.currency)}
                  </p>
                  {selectedEntry.description && (
                    <p className="mt-1 text-sm text-muted-foreground">{selectedEntry.description}</p>
                  )}
                </div>
                <dl className="space-y-3">
                  <div>
                    <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Debit account</dt>
                    <dd className="mt-0.5 text-sm text-foreground">
                      {accountLabelById(selectedEntry.debit_account_id) || (
                        <span className="font-mono text-xs">{selectedEntry.debit_account_id}</span>
                      )}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Credit account</dt>
                    <dd className="mt-0.5 text-sm text-foreground">
                      {accountLabelById(selectedEntry.credit_account_id) || (
                        <span className="font-mono text-xs">{selectedEntry.credit_account_id}</span>
                      )}
                    </dd>
                  </div>
                  {selectedEntry.reference_id && (
                    <div>
                      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Reference (invoice / payment)</dt>
                      <dd className="mt-0.5 font-mono text-xs text-foreground">{selectedEntry.reference_id}</dd>
                    </div>
                  )}
                  <div>
                    <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Transaction ID</dt>
                    <dd className="mt-0.5 font-mono text-xs text-muted-foreground">{selectedEntry.id}</dd>
                  </div>
                </dl>
              </div>
            </>
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}
