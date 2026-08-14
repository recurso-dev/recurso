import { useState } from "react";
import { useParams } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Power, PowerOff, Settings2, ClipboardList, Gift, ShieldQuestion } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatDateTime } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedEmpty,
} from "@/components/patterns/ObjectPage";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { Badge } from "@/components/ui/badge";
import { CopyableId } from "@/components/ui/copyable-id";
import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/sonner";
import CancelFlowDetail from "@/components/slide-overs/CancelFlowDetail";

const STEP_TYPES = {
  survey: { label: "Survey", icon: ClipboardList, variant: "info" },
  offer: { label: "Offer", icon: Gift, variant: "success" },
  confirmation: { label: "Confirmation", icon: ShieldQuestion, variant: "warning" },
};
const stepMeta = (type) => STEP_TYPES[type] || STEP_TYPES.survey;

// A one-line human summary of a step's config (mirrors the editor's).
const stepSummary = (step) => {
  const cfg = step.config || {};
  if (step.step_type === "survey") return `${(cfg.questions || []).length} cancellation reasons`;
  if (step.step_type === "offer") return cfg.headline || `${(cfg.offers || []).length} retention offers`;
  if (step.step_type === "confirmation") return cfg.message || "Confirm cancellation";
  return "";
};

const pct = (v) => (v == null ? "—" : `${Math.round(v * 100)}%`);

/**
 * CancelFlowPage — one cancellation flow as a first-class object at
 * /cancel-flows/:id. Replaces the card-grid + slide-over: the flow's retention
 * effectiveness (real FlowStats — save rate, saved count, offer-accept rate,
 * why customers cancel) alongside the ordered step sequence a canceling
 * customer walks through. The existing step editor is reused as the edit sheet.
 */
export default function CancelFlowPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);

  const {
    data: flow,
    isLoading,
    error: flowError,
    refetch,
  } = useQuery({
    queryKey: ["cancel-flow", id],
    // NB: this endpoint returns the flow directly, not wrapped in { data }.
    queryFn: async () => (await endpoints.getCancelFlow(id)).data,
    enabled: Boolean(id),
  });

  // Retention effectiveness — best-effort (empty until sessions exist).
  const { data: stats } = useQuery({
    queryKey: ["cancel-flow-stats", id],
    queryFn: async () => (await endpoints.getCancelFlowStats(id)).data,
    enabled: Boolean(id),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["cancel-flow", id] });
    queryClient.invalidateQueries({ queryKey: ["cancel-flow-stats", id] });
    queryClient.invalidateQueries({ queryKey: ["cancel-flows"] });
  };

  const toggleMutation = useMutation({
    mutationFn: (next) => endpoints.updateCancelFlow(id, { is_active: next }),
    onSuccess: (_data, next) => {
      toast.success(next ? "Flow activated." : "Flow deactivated.");
      invalidate();
    },
    onError: (err) => toast.error(err?.response?.data?.error?.message || "Failed to update flow"),
  });

  if (isLoading) {
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

  if (flowError || !flow) {
    return (
      <ErrorState
        title={flowError ? "Couldn't load this flow" : "Cancel flow not found"}
        message={
          flowError
            ? flowError?.response?.data?.error?.message || flowError?.message
            : "This cancellation flow doesn't exist or isn't in your account."
        }
        onRetry={flowError ? refetch : undefined}
      />
    );
  }

  const isActive = flow.is_active;
  const steps = [...(flow.steps || [])].sort((a, b) => a.step_order - b.step_order);
  const hasSessions = stats && stats.total_sessions > 0;
  const reasons = stats?.reason_breakdown
    ? Object.entries(stats.reason_breakdown).sort((a, b) => b[1] - a[1])
    : [];

  return (
    <div>
      <ObjectHeader
        backTo="/cancel-flows"
        backLabel="Cancel Flows"
        kicker="Cancel flow"
        title={flow.name || "Cancel flow"}
        badge={
          <span className="flex items-center gap-1.5">
            <StatusBadge status={isActive ? "active" : "inactive"} />
            {flow.is_default && <Badge variant="info">Default</Badge>}
          </span>
        }
        meta={
          <>
            <span>Cooldown {flow.cooldown_days} days</span>
            <CopyableId value={flow.id} />
          </>
        }
        actions={
          <>
            <Button variant="outline" onClick={() => setEditOpen(true)}>
              <Settings2 className="h-4 w-4" />
              Edit steps
            </Button>
            {isActive ? (
              <Button
                variant="ghost"
                className="text-destructive hover:text-destructive"
                disabled={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate(false)}
              >
                <PowerOff className="h-4 w-4" />
                Deactivate
              </Button>
            ) : (
              <Button
                variant="outline"
                disabled={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate(true)}
              >
                <Power className="h-4 w-4" />
                Activate
              </Button>
            )}
          </>
        }
      />

      <ObjectPageLayout
        rail={
          <ObjectSection title="Details">
            <AttributeList
              columns={1}
              items={[
                { label: "Flow ID", value: <CopyableId value={flow.id} /> },
                { label: "Default flow", value: flow.is_default ? "Yes" : "No" },
                { label: "Cooldown", value: `${flow.cooldown_days} days` },
                { label: "Steps", value: steps.length },
                { label: "Created", value: formatDateTime(flow.created_at) },
              ]}
            />
          </ObjectSection>
        }
      >
        <ObjectSection title="Effectiveness">
          {hasSessions ? (
            <>
              <AttributeList
                columns={4}
                items={[
                  { label: "Sessions", value: <span className="font-mono text-lg font-medium tabular-nums">{stats.total_sessions}</span> },
                  { label: "Saved", value: <span className="font-mono text-lg font-medium tabular-nums text-success">{stats.saved_count ?? 0}</span> },
                  { label: "Save rate", value: <span className="font-mono text-lg font-medium tabular-nums">{pct(stats.save_rate)}</span> },
                  { label: "Offer accept", value: <span className="font-mono text-lg font-medium tabular-nums">{pct(stats.offer_accept_rate)}</span> },
                ]}
              />
              <p className="mt-2 text-xs text-muted-foreground">
                {stats.saved_count ?? 0} of {stats.completed_count ?? 0} completed sessions kept the
                subscription — the retention this flow bought.
              </p>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              No cancellation sessions have run through this flow yet — effectiveness appears here
              once customers start hitting it.
            </p>
          )}
        </ObjectSection>

        {reasons.length > 0 && (
          <ObjectSection title="Why customers cancel" flush>
            <div className="divide-y divide-border">
              {reasons.map(([reason, count]) => (
                <div key={reason} className="flex items-center justify-between px-6 py-2.5 text-sm">
                  <span className="capitalize text-foreground">{reason.replace(/_/g, " ")}</span>
                  <span className="tabular-nums text-muted-foreground">{count}</span>
                </div>
              ))}
            </div>
          </ObjectSection>
        )}

        <ObjectSection title={`Steps${steps.length ? ` (${steps.length})` : ""}`}>
          {steps.length === 0 ? (
            <RelatedEmpty>
              No steps yet — use “Edit steps” to add a survey, a retention offer, or a confirmation.
            </RelatedEmpty>
          ) : (
            <ol className="space-y-0">
              {steps.map((step, i) => {
                const meta = stepMeta(step.step_type);
                const Icon = meta.icon;
                const last = i === steps.length - 1;
                return (
                  <li key={step.id} className="relative flex gap-4 pb-6 last:pb-0">
                    {!last && (
                      <span className="absolute left-[15px] top-8 h-full w-px bg-border" aria-hidden="true" />
                    )}
                    <span className="relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-border bg-muted text-xs font-semibold text-muted-foreground">
                      {step.step_order}
                    </span>
                    <div className="min-w-0 flex-1 pt-1">
                      <Badge variant={meta.variant}>
                        <Icon className="h-3 w-3" />
                        {meta.label}
                      </Badge>
                      <p className="mt-1.5 text-sm text-foreground">{stepSummary(step)}</p>
                    </div>
                  </li>
                );
              })}
            </ol>
          )}
        </ObjectSection>
      </ObjectPageLayout>

      {/* Reuse the existing step editor as the edit sheet. */}
      <CancelFlowDetail
        flowId={editOpen ? id : null}
        isOpen={editOpen}
        onClose={() => setEditOpen(false)}
        onChanged={invalidate}
      />
    </div>
  );
}
