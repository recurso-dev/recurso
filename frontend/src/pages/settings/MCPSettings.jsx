import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Save, Bot, AlertTriangle } from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// The money-path tools an agent gains when Tier-3 is enabled — shown so the
// operator knows exactly what they're opting into.
const TIER3_TOOLS = [
  "Convert a quote to an invoice",
  "Cancel a subscription",
  "Issue a credit note / refund",
  "Top up a customer wallet",
  "Add a one-off charge",
  "Bill accrued usage now",
];

export default function MCPSettings() {
  const queryClient = useQueryClient();
  const [tier3Enabled, setTier3Enabled] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const {
    data,
    isLoading: loading,
    isError: loadError,
    refetch,
  } = useQuery({
    queryKey: ["mcp-settings"],
    queryFn: async () => (await endpoints.getMCPSettings()).data?.data || null,
  });

  useEffect(() => {
    if (data) setTier3Enabled(!!data.tier3_enabled);
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: (payload) => endpoints.updateMCPSettings(payload),
    onSuccess: () => {
      toast.success("MCP settings saved.");
      queryClient.invalidateQueries({ queryKey: ["mcp-settings"] });
    },
    onError: () => toast.error("Failed to save MCP settings."),
  });
  const saving = saveMutation.isPending;
  const persist = () =>
    saveMutation.mutate(
      { tier3_enabled: tier3Enabled },
      { onSettled: () => setConfirmOpen(false) }
    );
  // Newly granting money-path tools is a high-consequence change — confirm it.
  // Turning them off, or saving with no change to the grant, saves directly.
  const enablingMoneyTools = tier3Enabled && !data?.tier3_enabled;
  const save = () => (enablingMoneyTools ? setConfirmOpen(true) : persist());

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="MCP server"
        description="Let AI agents operate your billing over the Model Context Protocol."
        actions={
          <Button onClick={save} disabled={saving || loading}>
            <Save className="h-4 w-4" />
            {saving ? "Saving..." : "Save settings"}
          </Button>
        }
      />

      {loading ? (
        <Skeleton className="h-48 w-full rounded-xl" />
      ) : loadError ? (
        <ErrorState
          title="Couldn't load MCP settings"
          message="We couldn't reach the settings service. Please try again."
          onRetry={refetch}
        />
      ) : (
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <Bot className="h-4 w-4 text-success" />
                Agent access
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <p className="text-sm text-muted-foreground">
                Reads and simulations are always available to a connected agent.
                Curated writes (create customer, record usage, draft quote) are
                on by default. The switch below controls the sensitive
                money-path tools, which are <span className="font-medium text-foreground">off</span> until
                you turn them on.
              </p>

              <div className="flex items-start gap-3 rounded-lg border border-warning/20 bg-warning/5 p-3 text-sm text-warning">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                <p>
                  Enabling money-path tools lets an agent move money on your
                  account. Only enable this for trusted, supervised agents.
                </p>
              </div>

              <div className="flex items-start justify-between gap-3">
                <span className="text-sm">
                  <span id="mcp-tier3-label" className="font-medium text-foreground">
                    Allow money-path tools
                  </span>
                  <span className="mt-0.5 block text-xs text-muted-foreground">
                    Grants an authenticated agent the actions listed below.
                  </span>
                </span>
                <Switch
                  aria-labelledby="mcp-tier3-label"
                  checked={tier3Enabled}
                  onCheckedChange={setTier3Enabled}
                  className="mt-0.5"
                />
              </div>

              <div className="rounded-lg border border-border bg-muted/40 p-4">
                <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                  Money-path tools
                </p>
                <ul className="grid gap-1.5 sm:grid-cols-2">
                  {TIER3_TOOLS.map((t) => (
                    <li key={t} className="text-sm text-foreground">
                      • {t}
                    </li>
                  ))}
                </ul>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Allow agents to move money?"
        description="Connected agents will be able to issue refunds and credit notes, top up wallets, add charges, cancel subscriptions, and bill usage on your account. Only enable this for trusted, supervised agents."
        confirmLabel="Enable money-path tools"
        destructive
        busy={saving}
        onConfirm={persist}
      />
    </div>
  );
}
