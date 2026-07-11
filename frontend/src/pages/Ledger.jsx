import { useEffect, useMemo, useState } from "react";
import { BookOpen } from "lucide-react";

import { endpoints } from "../lib/api";
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
  const [accounts, setAccounts] = useState([]);
  const [entries, setEntries] = useState([]);
  const [loading, setLoading] = useState(true);
  const [entriesLoading, setEntriesLoading] = useState(false);
  const [error, setError] = useState(null);
  const [selectedAccountId, setSelectedAccountId] = useState("");

  // Fetch accounts on mount.
  useEffect(() => {
    fetchAccounts();
  }, []);

  // Fetch entries whenever the selected account changes.
  useEffect(() => {
    if (selectedAccountId) {
      fetchEntries(selectedAccountId);
    } else {
      setEntries([]);
    }
  }, [selectedAccountId]);

  const fetchAccounts = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await endpoints.getLedgerAccounts();
      const accs = response.data.data || [];
      setAccounts(accs);
      // Auto-select the first account if available.
      if (accs.length > 0) {
        setSelectedAccountId(accs[0].id);
      }
    } catch (err) {
      console.error("Failed to fetch ledger accounts:", err);
      setError("Failed to load accounts.");
    } finally {
      setLoading(false);
    }
  };

  const fetchEntries = async (accountId) => {
    setEntriesLoading(true);
    try {
      const response = await endpoints.getLedgerEntries({
        account_id: accountId,
        limit: 50,
      });
      setEntries(response.data.data || []);
    } catch (err) {
      // Entries failures are non-critical; log and leave the list empty.
      console.error("Failed to fetch ledger entries:", err);
    } finally {
      setEntriesLoading(false);
    }
  };

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
      cell: (e) => (
        <span className="font-mono text-xs text-muted-foreground">
          {e.debit_account_id}
        </span>
      ),
    },
    {
      key: "credit",
      header: "Credit",
      cell: (e) => (
        <span className="font-mono text-xs text-muted-foreground">
          {e.credit_account_id}
        </span>
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
        description="View double-entry ledger transactions and account balances."
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
                  {acc.name} ({acc.code})
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
            hint={`${selectedAccount.name} (${selectedAccount.code})`}
          />
        )}
      </div>

      <DataTable
        columns={columns}
        data={entries}
        loading={entriesLoading}
        error={error}
        onRetry={fetchAccounts}
        empty={{
          icon: BookOpen,
          title: "No entries found",
          description: "No ledger entries were found for this account.",
        }}
      />
    </div>
  );
}
