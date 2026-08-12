import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router";
import { Landmark, RefreshCw, Check, Copy, ExternalLink } from "lucide-react";
import { formatDateTime } from "@/lib/utils";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import { DataTable } from "@/components/patterns/DataTable";
import PaymentGateways from "@/components/PaymentGateways";
import IntegrationConnections from "@/components/IntegrationConnections";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/ui/status-badge";
import { Card, CardContent } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { ProviderGuide } from "@/components/patterns/ProviderGuide";

// QuickBooks/Xero use the browser OAuth flow (main.go InitiateOAuth);
// NetSuite takes a pasted SuiteTalk token, Tally is a local JSONL export.
const PROVIDERS = [
  {
    id: "quickbooks",
    name: "QuickBooks Online",
    description: "Push customers, invoices, and payments to QuickBooks.",
    mode: "oauth",
  },
  {
    id: "xero",
    name: "Xero",
    description: "Sync your billing data to Xero's accounting ledger.",
    mode: "oauth",
  },
  {
    id: "netsuite",
    name: "NetSuite",
    description: "Sync to NetSuite via the SuiteTalk REST API (experimental).",
    mode: "token",
    guide: {
      steps: [
        "In NetSuite: Setup → Integration → Manage Integrations → New; enable OAuth 2.0 (client credentials).",
        "Obtain an access token for that integration (Setup → Users/Roles → Access Tokens, or your OAuth 2.0 flow).",
        "Account ID: Setup → Company → Company Information — e.g. 1234567, or 1234567_SB1 for a sandbox.",
      ],
      url: "https://system.netsuite.com/",
      urlLabel: "Open NetSuite",
    },
  },
  {
    id: "tally",
    name: "Tally",
    description: "Export data as import files for Tally ERP — nothing leaves your server.",
    mode: "local",
  },
];

// Error codes the OAuth callback redirect can carry (see the backend's
// redirectToIntegrations) mapped to human-readable messages.
const OAUTH_ERRORS = {
  missing_code: "The provider did not return an authorization code. Please try again.",
  invalid_state: "The connection link was invalid or expired. Please try again.",
  unsupported_provider: "That provider is not supported.",
  exchange_failed: "The provider rejected the token exchange. Please try again.",
  org_lookup_failed: "Could not resolve your Xero organisation.",
  save_failed: "Connected, but saving the connection failed. Please try again.",
};

const isFailedSync = (status) => ["failed", "error"].includes(status);

const fmtDateTime = (v) => formatDateTime(v);

// Turn a raw provider error into an actionable next step. Matches on common
// substrings so a failed sync tells the operator what to actually do, not just
// that it failed.
const errorHint = (msg) => {
  const m = String(msg || "").toLowerCase();
  if (m.includes("email"))
    return "The record is missing a valid email address, which this provider requires before it will accept it. Fix the customer's email, then re-sync.";
  if (m.includes("token") || m.includes("unauthor") || m.includes("401") || m.includes("expired") || m.includes("revoked"))
    return "The connection's authorization has expired or been revoked. Reconnect the provider above, then re-sync.";
  if (m.includes("rate") || m.includes("429") || m.includes("throttl"))
    return "The provider throttled the request. Recurso retries automatically on the next sync — or re-sync manually in a moment.";
  if (m.includes("duplicate") || m.includes("already exists"))
    return "The provider already has a record with this identifier. This usually clears once IDs reconcile on the next sync.";
  if (m.includes("not found") || m.includes("404"))
    return "A referenced record doesn't exist on the provider yet. Sync its parent record first (e.g. the customer before the invoice), then re-sync.";
  return null;
};

// Where in the app a synced record actually lives — the "track it down" link.
const ENTITY_PAGES = {
  invoice: { to: "/invoices", label: "Open Invoices" },
  customer: { to: "/customers", label: "Open Customers" },
  product: { to: "/plans", label: "Open Plans" },
  plan: { to: "/plans", label: "Open Plans" },
};

const copyId = async (v, label) => {
  try {
    await navigator.clipboard.writeText(v);
    toast.success(`${label} copied.`);
  } catch {
    toast.error("Couldn't copy to clipboard.");
  }
};

const Integrations = () => {
  const queryClient = useQueryClient();
  const [logOffset, setLogOffset] = useState(0);
  const [logProvider, setLogProvider] = useState("all");
  const [logStatus, setLogStatus] = useState("all");
  const [logSearch, setLogSearch] = useState("");
  const [logSearchInput, setLogSearchInput] = useState("");
  const [connecting, setConnecting] = useState(null);
  const [selectedLog, setSelectedLog] = useState(null);
  const [disconnectTarget, setDisconnectTarget] = useState(null);
  const [tokenProvider, setTokenProvider] = useState(null); // provider being connected via sheet
  const [tokenForm, setTokenForm] = useState({ account_id: "", access_token: "" });
  const [searchParams, setSearchParams] = useSearchParams();

  const {
    data: connections = [],
    isLoading: loading,
    error: connQueryError,
  } = useQuery({
    queryKey: ["accounting-connections"],
    queryFn: async () => (await api.getAccountingConnections()).data.data || [],
  });
  const error = connQueryError
    ? connQueryError?.response?.data?.error?.message || "Failed to load connections"
    : null;

  const LOG_PAGE_SIZE = 25;

  // One cache entry per filter+search+page combination; previous rows stay
  // visible while the next page loads.
  const {
    data: logData,
    isLoading: logsLoading,
    error: logsQueryError,
    refetch: refetchLogs,
  } = useQuery({
    queryKey: ["accounting-sync-logs", { logProvider, logStatus, logSearch, logOffset }],
    queryFn: async () => {
      const res = await api.getAccountingSyncStatus({
        limit: LOG_PAGE_SIZE,
        offset: logOffset,
        ...(logProvider !== "all" && { provider: logProvider }),
        ...(logStatus !== "all" && { status: logStatus }),
        ...(logSearch && { search: logSearch }),
      });
      return {
        logs: res.data.data || [],
        total: res.data.total ?? (res.data.data || []).length,
      };
    },
    placeholderData: keepPreviousData,
  });
  const logs = logData?.logs ?? [];
  const logTotal = logData?.total ?? 0;
  const logsError = logsQueryError
    ? logsQueryError?.response?.data?.error?.message || "Failed to load sync activity"
    : null;

  useEffect(() => {
    const t = setTimeout(() => {
      setLogOffset(0);
      setLogSearch(logSearchInput.trim());
    }, 400);
    return () => clearTimeout(t);
  }, [logSearchInput]);

  // The OAuth callback lands back here with ?connected=<provider> or
  // ?error=<code> — surface it as a toast once, then clean the URL.
  useEffect(() => {
    const connected = searchParams.get("connected");
    const oauthError = searchParams.get("error");
    if (!connected && !oauthError) return;
    if (connected) {
      const name = PROVIDERS.find((p) => p.id === connected)?.name || connected;
      toast.success(`${name} connected.`);
    } else {
      toast.error(OAUTH_ERRORS[oauthError] || "Connection failed. Please try again.");
    }
    setSearchParams({}, { replace: true });
    // Runs once for the params present at mount; the redirect is a full page load.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const connectionFor = (providerId) =>
    connections.find((c) => c.provider === providerId && c.is_active);

  const hasActiveConnection = connections.some((c) => c.is_active);

  const handleConnect = async (providerId) => {
    const provider = PROVIDERS.find((p) => p.id === providerId);
    // Token/local providers configure in a slide-over instead of OAuth handoff.
    if (provider?.mode === "token" || provider?.mode === "local") {
      setTokenForm({ account_id: "", access_token: "" });
      setTokenProvider(provider);
      return;
    }
    setConnecting(providerId);
    try {
      const res = await api.connectAccounting(providerId);
      const authUrl = res.data?.auth_url;
      if (!authUrl) {
        toast.error("OAuth is not configured for this provider on the server.");
        return;
      }
      // Hand off to the provider's consent screen; we return via the backend callback.
      window.location.href = authUrl;
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to start connection");
      setConnecting(null);
    }
  };

  const tokenConnectMutation = useMutation({
    mutationFn: () =>
      api.connectAccountingToken(
        tokenProvider.id,
        tokenProvider.mode === "token" ? tokenForm : {}
      ),
    onSuccess: () => {
      toast.success(`${tokenProvider.name} connected.`);
      setTokenProvider(null);
      queryClient.invalidateQueries({ queryKey: ["accounting-connections"] });
    },
    onError: (err) => toast.error(err?.response?.data?.error?.message || "Failed to connect"),
    onSettled: () => setConnecting(null),
  });
  const submitTokenConnect = () => {
    if (!tokenProvider) return;
    setConnecting(tokenProvider.id);
    tokenConnectMutation.mutate();
  };

  const disconnectMutation = useMutation({
    mutationFn: () => api.disconnectAccounting(disconnectTarget.id),
    onSuccess: () => {
      toast.success("Disconnected.");
      setDisconnectTarget(null);
      queryClient.invalidateQueries({ queryKey: ["accounting-connections"] });
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to disconnect"),
  });
  const disconnecting = disconnectMutation.isPending;
  const handleDisconnect = () => {
    if (!disconnectTarget) return;
    disconnectMutation.mutate();
  };

  const syncMutation = useMutation({
    mutationFn: (provider) => api.triggerAccountingSync(provider),
    onSuccess: (res, provider) => {
      if (res.data?.status === "sync_already_running") {
        toast.message("A sync is already running — watch Sync activity for progress.");
      } else if (provider) {
        toast.success(
          `${provider.charAt(0).toUpperCase() + provider.slice(1)} sync started — changed records only.`,
        );
      } else {
        toast.success("Sync started in the background. Activity will update as records push.");
      }
      // Jump back to the newest activity.
      setLogOffset(0);
      queryClient.invalidateQueries({ queryKey: ["accounting-sync-logs"] });
    },
    onError: (err) => toast.error(err?.response?.data?.error?.message || "Sync failed"),
  });
  const syncing = syncMutation.isPending;
  const handleSync = (provider) => syncMutation.mutate(provider);

  const logColumns = [
    {
      key: "provider",
      header: "Integration",
      cell: (l) => (
        <span className="text-sm font-medium capitalize">{l.provider || "—"}</span>
      ),
    },
    { key: "entity_type", header: "Entity" },
    { key: "action", header: "Action" },
    {
      key: "status",
      header: "Status",
      cell: (l) => (
        <div>
          <StatusBadge status={l.status} />
          {l.error_message && (
            <p className="mt-1 max-w-md whitespace-normal break-words text-xs text-destructive">
              {l.error_message}
            </p>
          )}
        </div>
      ),
    },
    {
      key: "external_id",
      header: "Record",
      // Success rows show the provider's id; error rows fall back to the
      // INTERNAL entity id, so the operator can find which invoice/customer
      // failed (an error row with "—" was un-actionable).
      cell: (l) => (
        <div className="min-w-0">
          {l.entity_name && (
            <p className="truncate text-sm font-medium" title={l.entity_name}>
              {l.entity_name}
            </p>
          )}
          <span className="font-mono text-xs text-muted-foreground">
            {l.external_id || (l.entity_id ? `${String(l.entity_id).slice(0, 8)}… (internal)` : "—")}
          </span>
        </div>
      ),
    },
    {
      key: "synced_at",
      header: "Synced",
      align: "right",
      cell: (l) => (
        <span className="text-sm text-muted-foreground">{fmtDateTime(l.synced_at)}</span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Integrations"
        description="Connect your payment gateways and accounting systems."
        actions={
          hasActiveConnection && (
            <Button onClick={() => handleSync()} disabled={syncing}>
              <RefreshCw className={`h-4 w-4 ${syncing ? "animate-spin" : ""}`} />
              {syncing ? "Syncing..." : "Sync now"}
            </Button>
          )
        }
      />

      <div className="mb-8">
        <PaymentGateways />
      </div>

      <div className="mb-8">
        <h2 className="mb-3 text-sm font-semibold text-foreground">Tax, CRM &amp; storage</h2>
        <IntegrationConnections />
      </div>

      <h2 className="mb-3 text-sm font-semibold text-foreground">Accounting</h2>
      {error && (
        <p className="mb-4 rounded-md bg-destructive/5 px-3 py-2 text-sm text-destructive">{error}</p>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        {PROVIDERS.map((p) => {
          const conn = connectionFor(p.id);
          return (
            <Card key={p.id}>
              <CardContent className="flex flex-col gap-4 p-6">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-md bg-success/5 text-success">
                      <Landmark className="h-5 w-5" />
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-foreground">{p.name}</p>
                      <p className="text-xs text-muted-foreground">{p.description}</p>
                    </div>
                  </div>
                  {conn ? (
                    <Badge variant="success">
                      <Check className="h-3 w-3" />
                      Connected
                    </Badge>
                  ) : (
                    <Badge variant="neutral">Not connected</Badge>
                  )}
                </div>

                {conn && (
                  <dl className="space-y-1 border-t border-border pt-3 text-xs">
                    {conn.realm_id && (
                      <div className="flex justify-between gap-2">
                        <dt className="text-muted-foreground">Organisation</dt>
                        <dd className="font-mono text-foreground">{conn.realm_id}</dd>
                      </div>
                    )}
                    <div className="flex justify-between gap-2">
                      <dt className="text-muted-foreground">Last sync</dt>
                      <dd className="text-foreground">{fmtDateTime(conn.last_sync_at)}</dd>
                    </div>
                    <div className="flex justify-between gap-2">
                      <dt className="text-muted-foreground">Status</dt>
                      <dd className="text-foreground">{conn.sync_status || "idle"}</dd>
                    </div>
                    {conn.last_error && (
                      <div className="flex justify-between gap-2">
                        <dt className="text-muted-foreground">Last error</dt>
                        <dd className="max-w-[12rem] truncate text-destructive" title={conn.last_error}>
                          {conn.last_error}
                        </dd>
                      </div>
                    )}
                  </dl>
                )}

                <div className="mt-auto">
                  {conn ? (
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        onClick={() => handleSync(p.id)}
                        disabled={syncing}
                      >
                        {conn.sync_status === "syncing" ? "Syncing…" : "Sync"}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setDisconnectTarget(conn)}
                      >
                        Disconnect
                      </Button>
                    </div>
                  ) : (
                    <Button
                      size="sm"
                      onClick={() => handleConnect(p.id)}
                      disabled={connecting === p.id || loading}
                    >
                      {connecting === p.id ? "Connecting..." : "Connect"}
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <div className="mt-8">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-sm font-semibold text-foreground">Sync activity</h2>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              value={logSearchInput}
              onChange={(e) => setLogSearchInput(e.target.value)}
              placeholder="Search record id…"
              className="h-8 w-44 text-xs"
              aria-label="Search sync records"
            />
            <select
              value={logProvider}
              onChange={(e) => {
                setLogOffset(0);
                setLogProvider(e.target.value);
              }}
              className="h-8 rounded-md border border-input bg-transparent px-2 text-xs text-foreground capitalize focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              aria-label="Filter by integration"
            >
              <option value="all">All integrations</option>
              {PROVIDERS.map((pv) => (
                <option key={pv.id} value={pv.id}>
                  {pv.name}
                </option>
              ))}
            </select>
            <select
              value={logStatus}
              onChange={(e) => {
                setLogOffset(0);
                setLogStatus(e.target.value);
              }}
              className="h-8 rounded-md border border-input bg-transparent px-2 text-xs text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              aria-label="Filter by status"
            >
              <option value="all">All statuses</option>
              <option value="success">Success only</option>
              <option value="error">Errors only</option>
            </select>
          </div>
        </div>
        <DataTable
          columns={logColumns}
          data={logs}
          loading={logsLoading}
          error={logsError}
          onRetry={refetchLogs}
          onRowClick={(l) => setSelectedLog(l)}
          getRowId={(l) => l.id}
          empty={{
            icon: RefreshCw,
            title: "No sync activity yet",
            description: "Connect a provider and run a sync to see records here.",
          }}
        />

        {/* Sync-record detail: everything about one sync attempt, with the IDs
            copyable and a jump to where the record lives in the app. */}
        <Sheet open={!!selectedLog} onOpenChange={(o) => !o && setSelectedLog(null)}>
          <SheetContent className="overflow-y-auto sm:max-w-md">
            {selectedLog && (
              <>
                <SheetHeader>
                  <SheetTitle className="flex items-center gap-2">
                    <span className="capitalize">{selectedLog.provider}</span>
                    <span className="text-muted-foreground">·</span>
                    <span className="capitalize">{selectedLog.entity_type}</span>
                    <StatusBadge status={selectedLog.status} />
                  </SheetTitle>
                </SheetHeader>

                <div className="mt-4 space-y-5 px-1">
                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3 rounded-lg border border-border bg-muted/20 p-3">
                    <div>
                      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Record</dt>
                      <dd className="mt-0.5 truncate text-sm font-medium" title={selectedLog.entity_name || undefined}>
                        {selectedLog.entity_name || "—"}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Action</dt>
                      <dd className="mt-0.5 text-sm capitalize">{selectedLog.action}</dd>
                    </div>
                    <div>
                      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Internal ID</dt>
                      <dd className="mt-0.5">
                        {selectedLog.entity_id ? (
                          <button
                            type="button"
                            onClick={() => copyId(selectedLog.entity_id, "Internal ID")}
                            className="group inline-flex max-w-full items-center gap-1.5 text-left"
                            title="Copy internal ID"
                          >
                            <span className="truncate font-mono text-xs">{selectedLog.entity_id}</span>
                            <Copy className="h-3 w-3 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                          </button>
                        ) : (
                          <span className="text-sm text-muted-foreground">—</span>
                        )}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Provider ID</dt>
                      <dd className="mt-0.5">
                        {selectedLog.external_id ? (
                          <button
                            type="button"
                            onClick={() => copyId(selectedLog.external_id, "Provider ID")}
                            className="group inline-flex max-w-full items-center gap-1.5 text-left"
                            title="Copy provider ID"
                          >
                            <span className="truncate font-mono text-xs">{selectedLog.external_id}</span>
                            <Copy className="h-3 w-3 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                          </button>
                        ) : (
                          <span className="text-sm text-muted-foreground">not assigned yet</span>
                        )}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Direction</dt>
                      <dd className="mt-0.5 text-sm">
                        Recurso <span className="text-muted-foreground">→</span>{" "}
                        <span className="capitalize">{selectedLog.provider}</span>
                      </dd>
                    </div>
                    <div>
                      <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Synced</dt>
                      <dd className="mt-0.5 text-sm">{fmtDateTime(selectedLog.synced_at)}</dd>
                    </div>
                  </dl>

                  {selectedLog.error_message && (
                    <div className="space-y-2">
                      <p className="text-xs font-medium text-foreground">What went wrong</p>
                      <p className="rounded-md border border-destructive/20 bg-destructive/5 p-3 font-mono text-xs leading-relaxed text-destructive">
                        {selectedLog.error_message}
                      </p>
                      {errorHint(selectedLog.error_message) && (
                        <p className="rounded-md border border-warning/20 bg-warning/5 p-3 text-xs leading-relaxed text-warning">
                          <span className="font-semibold">How to fix: </span>
                          {errorHint(selectedLog.error_message)}
                        </p>
                      )}
                    </div>
                  )}

                  <div className="flex flex-wrap items-center gap-2">
                    {isFailedSync(selectedLog.status) && (
                      <Button
                        size="sm"
                        onClick={() => {
                          handleSync(selectedLog.provider);
                          setSelectedLog(null);
                        }}
                        disabled={syncing}
                      >
                        <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${syncing ? "animate-spin" : ""}`} />
                        Re-sync {selectedLog.provider}
                      </Button>
                    )}
                    {ENTITY_PAGES[selectedLog.entity_type] && (
                      <Button variant="outline" size="sm" asChild>
                        <Link to={ENTITY_PAGES[selectedLog.entity_type].to}>
                          <ExternalLink className="mr-1.5 h-3.5 w-3.5" />
                          {ENTITY_PAGES[selectedLog.entity_type].label}
                        </Link>
                      </Button>
                    )}
                  </div>
                </div>
              </>
            )}
          </SheetContent>
        </Sheet>
        {logTotal > LOG_PAGE_SIZE && (
          <div className="mt-2 flex items-center justify-between text-xs text-muted-foreground">
            <span>
              {logOffset + 1}–{Math.min(logOffset + LOG_PAGE_SIZE, logTotal)} of {logTotal}
            </span>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={logOffset === 0 || logsLoading}
                onClick={() => setLogOffset(Math.max(0, logOffset - LOG_PAGE_SIZE))}
              >
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={logOffset + LOG_PAGE_SIZE >= logTotal || logsLoading}
                onClick={() => setLogOffset(logOffset + LOG_PAGE_SIZE)}
              >
                Next
              </Button>
            </div>
          </div>
        )}
      </div>

      <ConfirmDialog
        open={!!disconnectTarget}
        onOpenChange={(open) => !open && setDisconnectTarget(null)}
        title={`Disconnect ${
          PROVIDERS.find((p) => p.id === disconnectTarget?.provider)?.name || "provider"
        }?`}
        description="Syncing stops immediately. Existing synced records are kept in the accounting system."
        confirmLabel="Disconnect"
        destructive
        busy={disconnecting}
        onConfirm={handleDisconnect}
      />

      {/* Token/local provider connect (NetSuite, Tally) */}
      <Sheet open={!!tokenProvider} onOpenChange={(o) => !o && setTokenProvider(null)}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>Connect {tokenProvider?.name}</SheetTitle>
            <SheetDescription>
              {tokenProvider?.mode === "token"
                ? "Paste credentials from your NetSuite account — Setup → Integration → OAuth 2.0. Experimental: verify in a sandbox account first."
                : "Tally sync writes JSONL export files on the server for Tally's import tooling. No credentials needed — no data leaves your infrastructure."}
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 overflow-y-auto px-6">
            {tokenProvider?.mode === "token" && (
              <>
                <ProviderGuide guide={tokenProvider.guide} />
                <div>
                  <Label htmlFor="account-id">Account ID</Label>
                  <Input id="account-id"
                    value={tokenForm.account_id}
                    onChange={(e) => setTokenForm({ ...tokenForm, account_id: e.target.value })}
                    placeholder="e.g. 1234567 or 1234567_SB1"
                    className="font-mono"
                  />
                </div>
                <div>
                  <Label htmlFor="access-token">Access token</Label>
                  <Input id="access-token"
                    type="password"
                    value={tokenForm.access_token}
                    onChange={(e) => setTokenForm({ ...tokenForm, access_token: e.target.value })}
                    placeholder="SuiteTalk OAuth 2.0 access token"
                    className="font-mono"
                  />
                  <p className="mt-1 text-xs text-muted-foreground">
                    Stored server-side and never shown again.
                  </p>
                </div>
              </>
            )}
          </div>
          <SheetFooter>
            <Button
              onClick={submitTokenConnect}
              disabled={
                connecting === tokenProvider?.id ||
                (tokenProvider?.mode === "token" &&
                  (!tokenForm.account_id.trim() || !tokenForm.access_token.trim()))
              }
            >
              {connecting === tokenProvider?.id
                ? "Connecting…"
                : `Connect ${tokenProvider?.name || ""}`}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
};

export default Integrations;
