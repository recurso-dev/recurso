import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";

import { endpoints } from "../lib/api";
import { formatDateTime } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
} from "@/components/patterns/ObjectPage";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Badge } from "@/components/ui/badge";
import { Money } from "@/components/ui/money";
import { CopyableId } from "@/components/ui/copyable-id";

const TYPE_LABEL = { 1: "asset", 2: "liability", 3: "equity", 4: "revenue", 5: "expense" };

/**
 * AccountPage — one ledger account as a first-class object at
 * /ledger/accounts/:id. Shows the account's identity, the debits/credits/balance
 * identity, and its journal activity as a per-account statement (which side each
 * posting hit, against which counterpart account). Makes accounts linkable — the
 * customer's ledger account, a journal leg — instead of a bare UUID.
 */
export default function AccountPage() {
  const { id } = useParams();

  // No single-account endpoint — the chart is small, so find it in the list.
  const {
    data: accounts = [],
    isLoading: accountsLoading,
    error: accountsError,
    refetch,
  } = useQuery({
    queryKey: ["ledger-accounts"],
    queryFn: async () => (await endpoints.getLedgerAccounts()).data.data || [],
  });

  const {
    data: entries = [],
    isLoading: entriesLoading,
  } = useQuery({
    queryKey: ["ledger-entries", id, "account-page"],
    queryFn: async () =>
      (await endpoints.getLedgerEntries({ account_id: id, limit: 100 })).data.data || [],
    enabled: Boolean(id),
  });

  const account = accounts.find((a) => a.id === id);

  // Per-customer AR sub-accounts aren't in the chart list, but every entry
  // carries its account's name + code (joined server-side) — so name the account
  // from its own postings when the list doesn't have it.
  const fromEntry = entries.find(
    (e) => e.debit_account_id === id || e.credit_account_id === id
  );
  const derivedName = fromEntry
    ? fromEntry.debit_account_id === id
      ? fromEntry.debit_account_name
      : fromEntry.credit_account_name
    : null;
  const derivedCode = fromEntry
    ? fromEntry.debit_account_id === id
      ? fromEntry.debit_account_code
      : fromEntry.credit_account_code
    : null;

  const name = account?.name || derivedName;
  const code = account?.code ?? derivedCode;
  const currency = account?.currency || "USD";

  if (accountsLoading || entriesLoading) {
    return (
      <div aria-busy="true">
        <Skeleton className="mb-2 h-4 w-24" />
        <Skeleton className="mb-6 h-8 w-64" />
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <Skeleton className="h-64 lg:col-span-2" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (accountsError || (!account && !fromEntry)) {
    return (
      <ErrorState
        title={accountsError ? "Couldn't load this account" : "Account not found"}
        message={
          accountsError
            ? accountsError?.response?.data?.error?.message || accountsError?.message
            : "This account has no postings and isn't in the chart of accounts."
        }
        onRetry={accountsError ? refetch : undefined}
      />
    );
  }

  return (
    <div>
      <ObjectHeader
        backTo="/ledger"
        backLabel="Ledger"
        kicker="Account"
        title={name || "Account"}
        badge={
          account ? (
            <Badge variant="neutral" className="capitalize">
              {TYPE_LABEL[account.type] || "account"}
            </Badge>
          ) : null
        }
        meta={
          <>
            {code != null && <span className="font-mono text-xs">{code}</span>}
            <span>{currency}</span>
            <CopyableId value={id} />
          </>
        }
      />

      <ObjectPageLayout
        rail={
          <ObjectSection title="Metadata">
            <AttributeList
              columns={1}
              items={[
                { label: "Account ID", value: <CopyableId value={id} /> },
                { label: "Code", value: code != null ? <span className="font-mono text-sm">{code}</span> : null },
                { label: "Type", value: account ? <span className="capitalize">{TYPE_LABEL[account.type]}</span> : null },
                { label: "Currency", value: currency },
              ]}
            />
          </ObjectSection>
        }
      >
        <ObjectSection title="Position">
          {account ? (
            <AttributeList
              columns={3}
              items={[
                {
                  label: "Debits posted",
                  value: <span className="font-mono tabular-nums"><Money amountMinor={account.debits_posted || 0} currency={currency} /></span>,
                },
                {
                  label: "Credits posted",
                  value: <span className="font-mono tabular-nums"><Money amountMinor={account.credits_posted || 0} currency={currency} /></span>,
                },
                {
                  label: "Net balance",
                  value: <span className="font-mono font-medium tabular-nums"><Money amountMinor={account.balance || 0} currency={currency} /></span>,
                },
              ]}
            />
          ) : (
            <p className="text-sm text-muted-foreground">
              This is a per-customer sub-account; its running balance isn’t in the chart of
              accounts. Its postings are below.
            </p>
          )}
        </ObjectSection>

        <ObjectSection title={`Journal activity${entries.length ? ` (${entries.length})` : ""}`} flush>
          {entries.length === 0 ? (
            <p className="px-6 py-4 text-sm text-muted-foreground">No postings on this account.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[640px] text-sm">
                <thead>
                  <tr className="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wide text-subtle">
                    <th className="px-6 py-2.5 font-medium">Date</th>
                    <th className="px-3 py-2.5 font-medium">Posting</th>
                    <th className="px-3 py-2.5 font-medium">Against</th>
                    <th className="px-3 py-2.5 text-right font-medium">Debit</th>
                    <th className="px-6 py-2.5 text-right font-medium">Credit</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {entries.map((e) => {
                    const isDebit = e.debit_account_id === id;
                    const counter = isDebit ? e.credit_account_name : e.debit_account_name;
                    const counterCode = isDebit ? e.credit_account_code : e.debit_account_code;
                    const counterId = isDebit ? e.credit_account_id : e.debit_account_id;
                    const amt = <Money amountMinor={e.amount} currency={currency} />;
                    return (
                      <tr key={e.transaction_id || e.id} className="hover:bg-muted/20">
                        <td className="whitespace-nowrap px-6 py-2.5 text-muted-foreground">
                          {formatDateTime(e.timestamp)}
                        </td>
                        <td className="px-3 py-2.5">
                          <Badge variant="neutral" className="font-mono text-xs">Code {e.code}</Badge>
                        </td>
                        <td className="px-3 py-2.5 text-foreground">
                          {counterId ? (
                            <Link
                              to={`/ledger/accounts/${counterId}`}
                              title="Open the counterpart account"
                              className="text-primary underline-offset-2 hover:underline"
                            >
                              <span className="text-muted-foreground">{counterCode}</span> {counter}
                            </Link>
                          ) : (
                            <>
                              <span className="text-muted-foreground">{counterCode}</span> {counter}
                            </>
                          )}
                        </td>
                        <td className="px-3 py-2.5 text-right font-mono tabular-nums text-foreground">
                          {isDebit ? amt : ""}
                        </td>
                        <td className="px-6 py-2.5 text-right font-mono tabular-nums text-foreground">
                          {isDebit ? "" : amt}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
          {entries.length >= 100 && (
            <p className="px-6 py-3 text-xs text-muted-foreground">
              Showing the 100 most recent postings.{" "}
              <Link to={`/ledger?account_id=${id}`} className="text-primary hover:underline">
                Open in the ledger explorer →
              </Link>
            </p>
          )}
        </ObjectSection>
      </ObjectPageLayout>
    </div>
  );
}
