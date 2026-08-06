import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { BookOpen } from "lucide-react";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import { formatCurrency } from "@/lib/utils";
import { Money } from "@/components/ui/money";
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
  22: "Write-off",
  23: "Write-off — tax reversal",
  24: "Write-off recovery",
  25: "Write-off recovery — tax",
  26: "Bad debt (write-off)",
  27: "Bad debt recovery",
};
const codeLabel = (c) => CODE_LABEL[c] || `Code ${c}`;

// What a transaction's reference_id points at, derived from its code (each
// posting site in service/ledger.go stamps one reference kind per code).
// Invoice references drill through to the invoice; the rest are labeled
// honestly rather than the old ambiguous "invoice / payment".
const REF_KIND = {
  1: "invoice", 3: "invoice", 6: "invoice", 10: "invoice", 12: "invoice",
  19: "invoice", 22: "invoice", 23: "invoice", 24: "invoice", 25: "invoice",
  26: "invoice", 27: "invoice",
  4: "credit note", 5: "credit note", 9: "credit note", 16: "credit note",
  17: "credit note", 18: "credit note", 20: "credit note", 21: "credit note",
  2: "recognition entry",
  11: "wallet transaction", 13: "wallet transaction", 14: "wallet transaction",
  15: "wallet transaction",
};
const refKind = (c) => REF_KIND[c] || "source record";

const fmtWhen = (x) =>
  x ? new Date(x).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }) : "—";

const PAGE_SIZE = 50;

export default function Ledger() {
  // URL-addressable: /ledger?account_id=…&code=3 (or ?account_code=2100 to
  // resolve a tenant-level account by chart code). Reports deep-link here so
  // any figure is two clicks from the journal legs behind it.
  const [searchParams, setSearchParams] = useSearchParams();
  const [selectedAccountId, setSelectedAccountId] = useState(searchParams.get("account_id") || "");
  const [selectedEntry, setSelectedEntry] = useState(null);
  const [page, setPage] = useState(0);
  const [codeFilter, setCodeFilter] = useState(searchParams.get("code") || "all");

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
    queryKey: ["ledger-entries", selectedAccountId, page, codeFilter],
    queryFn: async () =>
      (
        await endpoints.getLedgerEntries({
          account_id: selectedAccountId,
          limit: PAGE_SIZE,
          offset: page * PAGE_SIZE,
          ...(codeFilter !== "all" ? { code: Number(codeFilter) } : {}),
        })
      ).data.data || [],
    enabled: !!selectedAccountId,
    keepPreviousData: true,
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

  // Prefer a human account label over the raw UUID. Per-customer AR sub-accounts
  // aren't in the tenant `accounts` list, so accountLabelById() misses them —
  // but every ledger entry carries its own account name + code (joined
  // server-side), so we can still name them (and, for AR, tag the customer).
  const accountLabelFromEntry = (id, name, code) => {
    const viaAccounts = accountLabelById(id);
    if (viaAccounts) return viaAccounts;
    if (name) {
      return customerNames[id] ? `${name} — ${customerNames[id]} (${code})` : `${name} (${code})`;
    }
    return null;
  };

  // Auto-select once accounts load: an explicit ?account_code picks the
  // matching tenant-level account when it's unambiguous (per-customer AR
  // sub-accounts all share 1100, so that code stays ambiguous and falls
  // through); otherwise the first account, matching prior behavior.
  useEffect(() => {
    if (selectedAccountId || accounts.length === 0) return;
    const codeParam = searchParams.get("account_code");
    if (codeParam) {
      const matches = accounts.filter((a) => String(a.code) === codeParam);
      if (matches.length === 1) {
        setSelectedAccountId(matches[0].id);
        return;
      }
    }
    setSelectedAccountId(accounts[0].id);
  }, [accounts, selectedAccountId, searchParams]);

  // Reflect the current view in the URL (replace, not push) so it is
  // shareable and the back button isn't spammed.
  useEffect(() => {
    if (!selectedAccountId) return;
    const next = new URLSearchParams();
    next.set("account_id", selectedAccountId);
    if (codeFilter !== "all") next.set("code", codeFilter);
    setSearchParams(next, { replace: true });
  }, [selectedAccountId, codeFilter, setSearchParams]);

  // A new account or posting filter starts back at the first page.
  useEffect(() => {
    setPage(0);
  }, [selectedAccountId, codeFilter]);

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
      cell: (e) => {
        const label = accountLabelFromEntry(
          e.debit_account_id,
          e.debit_account_name,
          e.debit_account_code
        );
        return label ? (
          <span className="text-sm text-foreground">{label}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">{String(e.debit_account_id).slice(0, 8)}…</span>
        );
      },
    },
    {
      key: "credit",
      header: "Credit",
      cell: (e) => {
        const label = accountLabelFromEntry(
          e.credit_account_id,
          e.credit_account_name,
          e.credit_account_code
        );
        return label ? (
          <span className="text-sm text-foreground">{label}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">{String(e.credit_account_id).slice(0, 8)}…</span>
        );
      },
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (e) => (
        <Money amountMinor={e.amount} currency={selectedAccount?.currency} className="font-medium" />
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

        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">Posting type</label>
          <Select value={codeFilter} onValueChange={setCodeFilter}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All postings</SelectItem>
              {Object.entries(CODE_LABEL).map(([c, label]) => (
                <SelectItem key={c} value={c}>{label}</SelectItem>
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
        pagination={{
          page: page + 1,
          onPrev: () => setPage((p) => Math.max(0, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext: entries.length === PAGE_SIZE,
        }}
        empty={{
          icon: BookOpen,
          title: page > 0 ? "No more entries" : "No entries found",
          description:
            codeFilter !== "all"
              ? "No postings of this type for this account."
              : "No ledger entries were found for this account.",
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
                      {accountLabelFromEntry(
                        selectedEntry.debit_account_id,
                        selectedEntry.debit_account_name,
                        selectedEntry.debit_account_code
                      ) || (
                        <span className="font-mono text-xs">{selectedEntry.debit_account_id}</span>
                      )}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Credit account</dt>
                    <dd className="mt-0.5 text-sm text-foreground">
                      {accountLabelFromEntry(
                        selectedEntry.credit_account_id,
                        selectedEntry.credit_account_name,
                        selectedEntry.credit_account_code
                      ) || (
                        <span className="font-mono text-xs">{selectedEntry.credit_account_id}</span>
                      )}
                    </dd>
                  </div>
                  {selectedEntry.reference_id && (
                    <div>
                      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                        Reference ({refKind(selectedEntry.code)})
                      </dt>
                      <dd className="mt-0.5 font-mono text-xs text-foreground">
                        {refKind(selectedEntry.code) === "invoice" ? (
                          <Link
                            to="/invoices"
                            state={{ openInvoiceId: selectedEntry.reference_id }}
                            className="text-primary underline-offset-2 hover:underline"
                          >
                            {selectedEntry.reference_id}
                          </Link>
                        ) : (
                          selectedEntry.reference_id
                        )}
                      </dd>
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
