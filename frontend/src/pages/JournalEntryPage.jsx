import { Link, useParams } from "react-router";
import { ArrowRight } from "lucide-react";

import { endpoints } from "../lib/api";
import { useObjectQuery } from "@/lib/useObjectQuery";
import { codeLabel, refKind, refRoute } from "@/lib/ledgerCodes";
import { formatDateTime, shortId } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  ObjectPageSkeleton,
  ObjectNotFound,
  ObjectPageError,
} from "@/components/patterns/ObjectPage";
import { JournalEntries } from "@/components/patterns/JournalEntries";
import { CopyableId } from "@/components/ui/copyable-id";
import { Money } from "@/components/ui/money";

// A journal entry is one posted ledger transaction: a single balanced
// debit/credit. This page answers "why does this accounting entry exist?" — the
// posting in words (its code), what it references, and its two legs, each
// deep-linking to its ledger account.
export default function JournalEntryPage() {
  const { id } = useParams();

  const {
    object: entry,
    loading,
    notFound,
    isError,
    error,
    refetch,
  } = useObjectQuery(
    ["ledger-transaction", id],
    async () => (await endpoints.getLedgerTransaction(id)).data.data,
    { enabled: Boolean(id) }
  );

  if (loading) return <ObjectPageSkeleton />;
  if (notFound) {
    return (
      <ObjectNotFound
        objectLabel="journal entry"
        identifier={id ? String(id).slice(0, 8) : undefined}
        backTo="/ledger"
        backLabel="Ledger"
      />
    );
  }
  if (isError) {
    return (
      <ObjectPageError objectLabel="journal entry" error={error} onRetry={refetch} backTo="/ledger" backLabel="Ledger" />
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
        amount={<Money amountMinor={entry.amount} size="hero" />}
        amountLabel="posted to the ledger"
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
