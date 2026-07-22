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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export default function Ledger() {
  const [selectedAccountId, setSelectedAccountId] = useState("");

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
      key: "id",
      header: "Transaction ID",
      cell: (e) => (
        <span className="font-mono text-xs text-foreground">{e.id}</span>
      ),
    },
    {
      key: "debit",
      header: "Debit",
      cell: (e) =>
        accountLabelById(e.debit_account_id) ? (
          <span className="text-sm text-foreground">{accountLabelById(e.debit_account_id)}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">{e.debit_account_id}</span>
        ),
    },
    {
      key: "credit",
      header: "Credit",
      cell: (e) =>
        accountLabelById(e.credit_account_id) ? (
          <span className="text-sm text-foreground">{accountLabelById(e.credit_account_id)}</span>
        ) : (
          <span className="font-mono text-xs text-muted-foreground">{e.credit_account_id}</span>
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
    {
      key: "code",
      header: "Code",
      cell: (e) => <Badge variant="neutral">Code {e.code}</Badge>,
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
        empty={{
          icon: BookOpen,
          title: "No entries found",
          description: "No ledger entries were found for this account.",
        }}
      />
    </div>
  );
}
