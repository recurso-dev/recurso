import { useCallback, useEffect, useState } from "react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Card } from "@/components/ui/card";

const iso = (d) => d.toISOString().slice(0, 10);

// Board-grade overview: composes the shipped analytics endpoints into one view.
// Each endpoint is fetched independently (Promise.allSettled) so a single
// failure degrades one tile rather than the whole page.
export default function ExecutiveSummary() {
  const [m, setM] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const now = new Date();
    const start = new Date(now);
    start.setDate(start.getDate() - 30);

    const [mrr, wf, ue, aging, rev] = await Promise.allSettled([
      endpoints.getMRR(),
      endpoints.getMRRWaterfall(iso(start), iso(now)),
      endpoints.getUnitEconomics(),
      endpoints.getInvoiceAging(),
      endpoints.getRevenueRecognition(now.getMonth() + 1, now.getFullYear()),
    ]);

    // getMRR returns the object directly; the rest wrap it in { data }.
    const ok = (r) => (r.status === "fulfilled" ? r.value?.data : null);
    const inner = (r) => (r.status === "fulfilled" ? r.value?.data?.data : null);

    const next = {
      mrr: ok(mrr),
      wf: inner(wf),
      ue: inner(ue),
      aging: inner(aging),
      rev: inner(rev),
    };
    if (!next.mrr && !next.wf && !next.ue && !next.aging && !next.rev) {
      setError("Could not load metrics. Is the server running?");
      setM(null);
    } else {
      setM(next);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const cur = m?.mrr?.reporting_currency || m?.ue?.reporting_currency || "USD";
  const money = (n) => (n == null ? "—" : formatCurrency(n, cur));
  const pct = (n) => (n == null ? "—" : `${Number(n).toFixed(1)}%`);
  const signed = (n) => (n == null ? "—" : `${n >= 0 ? "+" : "−"}${formatCurrency(Math.abs(n), cur)}`);

  const mrrVal = m?.mrr?.normalized_mrr ?? m?.mrr?.mrr ?? null;
  const netChange = m?.wf ? (m.wf.ending_mrr || 0) - (m.wf.starting_mrr || 0) : null;
  const ndr = m?.wf?.has_start_history ? m.wf.net_dollar_retention : null;
  const overdue =
    m?.aging?.buckets != null
      ? (m.aging.total_outstanding || 0) - (m.aging.buckets.find((b) => b.label === "current")?.amount || 0)
      : null;

  return (
    <div>
      <PageHeader
        title="Executive Summary"
        description="Your revenue at a glance — recurring revenue, movement, unit economics, and receivables."
      />

      {loading ? (
        <CardGridSkeleton count={4} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={load} />
        </Card>
      ) : (
        m && (
          <div className="flex flex-col gap-8">
            <Section title="Revenue">
              <StatCard label="MRR" value={money(mrrVal)} hint="Monthly recurring revenue" />
              <StatCard label="ARR" value={money(m?.mrr?.arr)} hint="Annual run-rate" />
              <StatCard label="Net change (30d)" value={signed(netChange)} hint="Ending vs. starting MRR" />
              <StatCard label="Net dollar retention" value={pct(ndr)} hint={ndr == null ? "Needs MRR history" : "Trailing 30 days"} />
            </Section>

            <Section title="MRR movement (30 days)">
              <StatCard label="New" value={money(m?.wf?.new)} hint="From new subscriptions" />
              <StatCard label="Expansion" value={money(m?.wf?.expansion)} hint="Upgrades" />
              <StatCard label="Contraction" value={m?.wf ? `−${formatCurrency(m.wf.contraction || 0, cur)}` : "—"} hint="Downgrades" />
              <StatCard label="Churned" value={m?.wf ? `−${formatCurrency(m.wf.churned || 0, cur)}` : "—"} hint="Cancellations" />
            </Section>

            <Section title="Unit economics">
              <StatCard label="ARPA" value={money(m?.ue?.arpa)} hint="Per account / month" />
              <StatCard label="ARPU" value={money(m?.ue?.arpu)} hint="Per subscription / month" />
              <StatCard label="LTV" value={m?.ue?.has_ltv ? money(m.ue.ltv) : "—"} hint={m?.ue?.has_ltv ? "At current churn" : "Needs history"} />
              <StatCard label="Active customers" value={m?.ue ? (m.ue.active_customers || 0).toLocaleString() : "—"} hint={m?.ue ? `${(m.ue.active_subscriptions || 0).toLocaleString()} subscriptions` : ""} />
            </Section>

            <Section title="Cash & recognition">
              <StatCard label="Outstanding" value={money(m?.aging?.total_outstanding)} hint={m?.aging ? `${m.aging.total_count || 0} open invoices` : ""} />
              <StatCard label="Overdue" value={money(overdue)} hint="Past due date" />
              <StatCard label="Deferred revenue" value={money(m?.rev?.deferred_balance)} hint="Unearned, still to recognize" />
              <StatCard label="Recognized (MTD)" value={money(m?.rev?.recognized_amount)} hint="This month" />
            </Section>
          </div>
        )
      )}
    </div>
  );
}

function Section({ title, children }) {
  return (
    <div>
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">{title}</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">{children}</div>
    </div>
  );
}
