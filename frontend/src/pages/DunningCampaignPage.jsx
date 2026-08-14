import { useState } from "react";
import { useParams } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Power, PowerOff, Settings2, Mail, MessageSquare, Bell, ShieldAlert } from "lucide-react";

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
import DunningCampaignDetail from "@/components/slide-overs/DunningCampaignDetail";

const CHANNEL_META = {
  email: { label: "Email", icon: Mail, variant: "info" },
  sms: { label: "SMS", icon: MessageSquare, variant: "secondary" },
  in_app: { label: "In-app", icon: Bell, variant: "neutral" },
};
const channelMeta = (ch) => CHANNEL_META[ch] || CHANNEL_META.email;

const TRIGGER_LABEL = {
  payment_failed: "a payment fails",
  invoice_overdue: "an invoice goes overdue",
};
const triggerPhrase = (t) => TRIGGER_LABEL[t] || t;

// A payment attempt's delay in plain language.
const delayLabel = (hours) => {
  if (!hours || hours <= 0) return "immediately";
  if (hours % 24 === 0) {
    const d = hours / 24;
    return `after ${d} day${d === 1 ? "" : "s"}`;
  }
  return `after ${hours} hour${hours === 1 ? "" : "s"}`;
};

/**
 * DunningCampaignPage — one dunning campaign as a first-class object at
 * /dunning-campaigns/:id. Replaces the card-grid + slide-over: the campaign's
 * trigger and its ordered step cadence rendered as a readable timeline (which
 * channel, after how long, payment-wall or not) — the "what actually happens
 * when we chase a failed payment" view. The existing step editor is reused as
 * the edit sheet, opened from here.
 */
export default function DunningCampaignPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);

  const {
    data: campaign,
    isLoading,
    error: campaignError,
    refetch,
  } = useQuery({
    queryKey: ["dunning-campaign", id],
    // NB: this endpoint returns the campaign directly, not wrapped in { data }.
    queryFn: async () => (await endpoints.getDunningCampaign(id)).data,
    enabled: Boolean(id),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["dunning-campaign", id] });
    queryClient.invalidateQueries({ queryKey: ["dunning-campaigns"] });
  };

  const toggleMutation = useMutation({
    mutationFn: (next) => endpoints.updateDunningCampaign(id, { is_active: next }),
    onSuccess: (_data, next) => {
      toast.success(next ? "Campaign activated." : "Campaign deactivated.");
      invalidate();
    },
    onError: (err) => toast.error(err?.response?.data?.error?.message || "Failed to update campaign"),
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

  if (campaignError || !campaign) {
    return (
      <ErrorState
        title={campaignError ? "Couldn't load this campaign" : "Campaign not found"}
        message={
          campaignError
            ? campaignError?.response?.data?.error?.message || campaignError?.message
            : "This campaign doesn't exist or isn't in your account."
        }
        onRetry={campaignError ? refetch : undefined}
      />
    );
  }

  const isActive = campaign.is_active;
  const steps = [...(campaign.steps || [])].sort((a, b) => a.step_order - b.step_order);

  return (
    <div>
      <ObjectHeader
        backTo="/dunning/campaigns"
        backLabel="Dunning Campaigns"
        kicker="Dunning campaign"
        title={campaign.name || "Campaign"}
        badge={<StatusBadge status={isActive ? "active" : "inactive"} />}
        meta={
          <>
            <span>Triggered when {triggerPhrase(campaign.trigger_event)}</span>
            <CopyableId value={campaign.id} />
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
                { label: "Campaign ID", value: <CopyableId value={campaign.id} /> },
                { label: "Trigger", value: <span className="capitalize">{(campaign.trigger_event || "").replace(/_/g, " ")}</span> },
                { label: "Status", value: <StatusBadge status={isActive ? "active" : "inactive"} /> },
                { label: "Steps", value: steps.length },
                { label: "Created", value: formatDateTime(campaign.created_at) },
              ]}
            />
          </ObjectSection>
        }
      >
        <ObjectSection title="How it runs">
          <p className="text-sm text-foreground">
            When {triggerPhrase(campaign.trigger_event)}, this sequence runs in order — each step
            waiting its delay — until the payment is recovered or the steps are exhausted.
          </p>
          {!isActive && (
            <p className="mt-2 text-sm text-muted-foreground">
              Deactivated — this campaign is not currently chasing any failed payments.
            </p>
          )}
        </ObjectSection>

        <ObjectSection title={`Step cadence${steps.length ? ` (${steps.length})` : ""}`}>
          {steps.length === 0 ? (
            <RelatedEmpty>
              No steps yet — use “Edit steps” to add the first outreach.
            </RelatedEmpty>
          ) : (
            <ol className="space-y-0">
              {steps.map((step, i) => {
                const meta = channelMeta(step.channel);
                const Icon = meta.icon;
                const last = i === steps.length - 1;
                return (
                  <li key={step.id} className="relative flex gap-4 pb-6 last:pb-0">
                    {/* Timeline rail */}
                    {!last && (
                      <span className="absolute left-[15px] top-8 h-full w-px bg-border" aria-hidden="true" />
                    )}
                    <span className="relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-border bg-muted text-xs font-semibold text-muted-foreground">
                      {step.step_order}
                    </span>
                    <div className="min-w-0 flex-1 pt-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={meta.variant}>
                          <Icon className="h-3 w-3" />
                          {meta.label}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {delayLabel(step.delay_hours)} {step.step_order === 1 ? "after the trigger" : "after the previous step"}
                        </span>
                        {step.is_payment_wall && (
                          <Badge variant="warning">
                            <ShieldAlert className="h-3 w-3" />
                            Payment wall
                          </Badge>
                        )}
                      </div>
                      <p className="mt-1.5 text-sm text-foreground">
                        {step.subject || step.template_name || <span className="text-muted-foreground">No subject</span>}
                      </p>
                      {step.body && (
                        <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">{step.body}</p>
                      )}
                    </div>
                  </li>
                );
              })}
            </ol>
          )}
        </ObjectSection>
      </ObjectPageLayout>

      {/* Reuse the existing step editor as the edit sheet. */}
      <DunningCampaignDetail
        campaignId={editOpen ? id : null}
        isOpen={editOpen}
        onClose={() => setEditOpen(false)}
        onChanged={invalidate}
      />
    </div>
  );
}
