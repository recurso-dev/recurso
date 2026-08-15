import { useState } from "react";
import { useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Pencil, X } from "lucide-react";

import { endpoints } from "../lib/api";
import { useObjectQuery } from "@/lib/useObjectQuery";
import PlanCharges from "../components/slide-overs/PlanCharges";
import PlanDetail from "../components/slide-overs/PlanDetail";
import { formatCurrency, formatDate } from "@/lib/utils";
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
import { AuditTrail } from "@/components/patterns/AuditTrail";
import { ObjectTimeline } from "@/components/patterns/ObjectTimeline";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { CopyableId } from "@/components/ui/copyable-id";

/**
 * PlanPage — the plan's full object page at /plans/:id, matching the
 * Customer/Subscription/Invoice pages. It's the read + navigation workspace
 * (pricing, entitlements, usage charges, the subscriptions using it, audit);
 * editing opens the existing PlanDetail sheet, so no edit logic is duplicated.
 */
export default function PlanPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);

  const {
    object: plan,
    loading,
    notFound,
    isError,
    error,
    refetch,
  } = useObjectQuery(
    ["plan", id],
    async () => (await endpoints.getPlan(id)).data.data,
    { enabled: Boolean(id) }
  );

  const { data: entitlements = [], error: entError } = useQuery({
    queryKey: ["plan-entitlements", id],
    queryFn: async () => (await endpoints.getPlanEntitlements(id)).data.data || [],
    enabled: Boolean(id),
  });

  // Reverse lookup: the subscriptions currently on this plan.
  const { data: subs = [] } = useQuery({
    queryKey: ["subscriptions", { plan_id: id }],
    queryFn: async () =>
      (await endpoints.getSubscriptions({ plan_id: id, limit: 50 })).data.data || [],
    enabled: Boolean(id),
  });

  if (loading) return <ObjectPageSkeleton />;
  if (notFound) {
    return (
      <ObjectNotFound
        objectLabel="plan"
        identifier={id ? String(id).slice(0, 8) : undefined}
        backTo="/plans"
        backLabel="Plans"
      />
    );
  }
  if (isError) {
    return <ObjectPageError objectLabel="plan" error={error} onRetry={refetch} backTo="/plans" backLabel="Plans" />;
  }

  const price = plan.prices && plan.prices[0];
  const currency = price ? price.currency.toUpperCase() : "USD";
  const interval = `${plan.interval_count > 1 ? `${plan.interval_count} ` : ""}${plan.interval_unit}`;

  const onChanged = (updated) => {
    queryClient.invalidateQueries({ queryKey: ["plan", id] });
    queryClient.invalidateQueries({ queryKey: ["plans"] });
    if (updated) queryClient.invalidateQueries({ queryKey: ["plan-entitlements", id] });
  };

  return (
    <div>
      <ObjectHeader
        backTo="/plans"
        backLabel="Plans"
        kicker="Plan"
        title={plan.name}
        badge={<StatusBadge status={plan.active ? "active" : "archived"} />}
        meta={
          <>
            <span className="font-mono text-xs">{plan.code}</span>
            <span>
              {price ? formatCurrency(price.amount, price.currency) : "—"} / {interval}
            </span>
            <CopyableId value={plan.id} />
          </>
        }
        actions={
          <Button onClick={() => setEditOpen(true)}>
            <Pencil className="h-4 w-4" />
            Edit
          </Button>
        }
      />

      <ObjectPageLayout
        rail={
          <>
            <ObjectSection title="Metadata">
              <AttributeList
                columns={1}
                items={[
                  { label: "Plan ID", value: <CopyableId value={plan.id} /> },
                  { label: "Code", value: <span className="font-mono text-sm">{plan.code}</span> },
                  { label: "Created", value: formatDate(plan.created_at) },
                  {
                    label: "HSN/SAC",
                    value: plan.hsn_code ? (
                      <span className="font-mono text-sm">{plan.hsn_code}</span>
                    ) : null,
                  },
                ]}
              />
            </ObjectSection>
            <ObjectSection title="Timeline">
              <ObjectTimeline objectId={plan.id} />
            </ObjectSection>
            <ObjectSection title="Audit trail">
              <AuditTrail entityType="plans" entityId={plan.id} />
            </ObjectSection>
          </>
        }
      >
        <ObjectSection title="Pricing">
          <AttributeList
            items={[
              {
                label: "Price",
                value: price ? (
                  <span>
                    {formatCurrency(price.amount, price.currency)} / {interval}
                  </span>
                ) : null,
              },
              { label: "Billing interval", value: <span className="capitalize">{interval}</span> },
              { label: "Currency", value: currency },
              { label: "Type", value: price?.type ? <span className="capitalize">{price.type}</span> : "recurring" },
            ]}
          />
          <p className="mt-4 text-xs text-muted-foreground">
            Price and code are fixed once a plan is created — create a new plan for a different
            price point.
          </p>
        </ObjectSection>

        <ObjectSection title={`Entitlements${entitlements.length ? ` (${entitlements.length})` : ""}`}>
          {entError ? (
            <p className="text-sm text-destructive">Failed to load entitlements.</p>
          ) : entitlements.length === 0 ? (
            <p className="text-sm text-muted-foreground">No entitlements configured for this plan.</p>
          ) : (
            <div className="flex flex-col gap-2">
              {entitlements.map((ent) => {
                const off = ent.kind === "boolean" && !ent.bool_value;
                return (
                  <div key={ent.feature_key} className="flex items-center gap-3">
                    {off ? (
                      <X className="h-4 w-4 text-subtle" aria-hidden="true" />
                    ) : (
                      <Check className="h-4 w-4 text-success" aria-hidden="true" />
                    )}
                    <span className="font-mono text-sm text-foreground">{ent.feature_key}</span>
                    {ent.kind === "limit" && (
                      <span className="text-xs text-muted-foreground">
                        limit: {ent.limit_value?.toLocaleString()}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </ObjectSection>

        <ObjectSection title="Usage charges">
          <PlanCharges planId={plan.id} currency={currency} />
        </ObjectSection>

        <ObjectSection title={`Subscriptions${subs.length ? ` (${subs.length})` : ""}`} flush>
          {subs.length === 0 ? (
            <RelatedEmpty>No subscriptions are on this plan yet.</RelatedEmpty>
          ) : (
            <div className="divide-y divide-border">
              {subs.map((s) => (
                <RelatedRow key={s.id} to={`/subscriptions/${s.id}`}>
                  <span className="min-w-0 truncate font-mono text-xs text-muted-foreground">
                    {s.id.slice(0, 8)}…
                  </span>
                  <StatusBadge status={s.status || "unknown"} />
                </RelatedRow>
              ))}
            </div>
          )}
        </ObjectSection>
      </ObjectPageLayout>

      <PlanDetail
        plan={plan}
        isOpen={editOpen}
        onClose={() => setEditOpen(false)}
        onChanged={onChanged}
      />
    </div>
  );
}
