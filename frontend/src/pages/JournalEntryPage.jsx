import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";

import { endpoints } from "../lib/api";
import { codeLabel, refKind, refRoute } from "@/lib/ledgerCodes";
import { formatDateTime, shortId } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
} from "@/components/patterns/ObjectPage";
import { JournalEntries } from "@/components/patterns/JournalEntries";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { CopyableId } from "@/components/ui/copyable-id";

// A journal entry is one posted ledger transaction: a single balanced
// debit/credit. This page answers "why does this accounting entry exist?" — the
// posting in words (its code), what it references, and its two legs, each
// deep-linking to its ledger account.
export default function JournalEntryPage() {
  const { id } = useParams();

  const {
    data: entry,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["ledger-transaction", id],
    queryFn: async () => (await endpoints.getLedgerTransaction(id)).data.data,
    enabled: Boolean(id),
  });

  if (isLoading) {
    return (
      <div aria-busy="true">
        <Skeleton className="mb-6 h-16 w-full max-w-md" />
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2 space-y-6">
            <Skeleton className="h-40 w-full" />
            <Skeleton className="h-28 w-full" />
          </div>
          <Skeleton className="h-40 w-full" />
        </div>
      </div>
    );
  }

  if (error || !entry) {
    const is404 = error?.response?.status === 404;
    return (
      <ErrorState
        title={is404 ? "Journal entry not found" : "Couldn’t load this journal entry"}
        message={
          is404
            ? "This transaction doesn’t exist, or belongs to another workspace."
            : error?.response?.data?.error?.message || error?.message || "Please try again."
        }
        onRetry={is404 ? undefined : refetch}
      />
    );
  }

  const kind = refKind(entry.code);
  const sourceRoute = refRoute(entry.code, entry.reference_id);

  return (
    <div>
      <ObjectHeader
        backTo="/ledger"
        backLabel="Ledger"
        kicker="Journal entry"
        title={codeLabel(entry.code)}
        meta={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <CopyableId value={entry.transaction_id} />
            <span className="text-muted-foreground">{formatDateTime(entry.timestamp)}</span>
          </span>
        }
      />

      <ObjectPageLayout
        rail={
          <ObjectSection title="Details">
            <AttributeList
              columns={1}
              items={[
                { label: "Posting", value: `${codeLabel(entry.code)} (code ${entry.code})` },
                { label: "Date", value: formatDateTime(entry.timestamp) },
                { label: "Description", value: entry.description },
                { label: "Transaction ID", value: <CopyableId value={entry.transaction_id} /> },
                { label: "Reference", value: kind },
                {
                  label: "Accounting model",
                  value: entry.accounting_version ? `v${entry.accounting_version}` : null,
                },
                { label: "Entity", value: entry.entity_name || null },
              ]}
            />
          </ObjectSection>
        }
      >
        {/* The two legs — labeled DR/CR (never color-only), each account a link.
            Rendered with the SAME primitive the invoice uses, so a journal entry
            reads identically wherever it appears. */}
        <ObjectSection title="Journal entry" flush>
          <div className="px-6 py-4">
            <JournalEntries
              entries={[entry]}
              emptyMessage="This transaction has no legs."
            />
          </div>
        </ObjectSection>

        {/* Why it exists: the source object it posted against. Only invoice
            references are addressable objects today; others are labeled honestly
            rather than linked to a page that doesn't exist. */}
        <ObjectSection title="Source">
          {sourceRoute ? (
            <Link
              to={sourceRoute}
              className="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
            >
              <span className="capitalize">{kind}</span>
              <span className="font-mono text-xs text-muted-foreground">{shortId(entry.reference_id)}</span>
              <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          ) : entry.reference_id ? (
            <p className="text-sm text-muted-foreground">
              Posted against a <span className="capitalize">{kind}</span>{" "}
              <span className="font-mono text-xs">{shortId(entry.reference_id)}</span> — not a
              separately addressable object.
            </p>
          ) : (
            <p className="text-sm text-muted-foreground">No source reference on this posting.</p>
          )}
        </ObjectSection>
      </ObjectPageLayout>
    </div>
  );
}
