import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Landmark, RefreshCw, Check } from "lucide-react";

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
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

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

const syncStatusVariant = (status) =>
  ({ success: "success", synced: "success", failed: "destructive", error: "destructive" })[
    status
  ] || "neutral";

const fmtDateTime = (v) => (v ? new Date(v).toLocaleString() : "—");

const Integrations = () => {
  const [connections, setConnections] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [logs, setLogs] = useState([]);
  const [logsLoading, setLogsLoading] = useState(true);
  const [logsError, setLogsError] = useState(null);
  const [connecting, setConnecting] = useState(null);
  const [syncing, setSyncing] = useState(false);
  const [disconnectTarget, setDisconnectTarget] = useState(null);
  const [tokenProvider, setTokenProvider] = useState(null); // provider being connected via sheet
  const [tokenForm, setTokenForm] = useState({ account_id: "", access_token: "" });
  const [disconnecting, setDisconnecting] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();

  const fetchConnections = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getAccountingConnections();
      setConnections(res.data.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load connections");
    } finally {
      setLoading(false);
    }
  };

  const fetchLogs = async () => {
    setLogsLoading(true);
    setLogsError(null);
    try {
      const res = await api.getAccountingSyncStatus();
      setLogs(res.data.data || []);
    } catch (err) {
      setLogsError(err?.response?.data?.error?.message || "Failed to load sync activity");
    } finally {
      setLogsLoading(false);
    }
  };

  useEffect(() => {
    fetchConnections();
    fetchLogs();
  }, []);

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

  const submitTokenConnect = async () => {
    if (!tokenProvider) return;
    setConnecting(tokenProvider.id);
    try {
      const body = tokenProvider.mode === "token" ? tokenForm : {};
      await api.connectAccountingToken(tokenProvider.id, body);
      toast.success(`${tokenProvider.name} connected.`);
      setTokenProvider(null);
      fetchConnections();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to connect");
    } finally {
      setConnecting(null);
    }
  };

  const handleDisconnect = async () => {
    if (!disconnectTarget) return;
    setDisconnecting(true);
    try {
      await api.disconnectAccounting(disconnectTarget.id);
      toast.success("Disconnected.");
      setDisconnectTarget(null);
      fetchConnections();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to disconnect");
    } finally {
      setDisconnecting(false);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      await api.triggerAccountingSync();
      toast.success("Sync triggered. Activity will update shortly.");
      fetchLogs();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Sync failed");
    } finally {
      setSyncing(false);
    }
  };

  const logColumns = [
    { key: "entity_type", header: "Entity" },
    { key: "action", header: "Action" },
    {
      key: "status",
      header: "Status",
      cell: (l) => (
        <div>
          <Badge variant={syncStatusVariant(l.status)}>{l.status}</Badge>
          {l.error_message && (
            <p className="mt-1 max-w-xs truncate text-xs text-red-600" title={l.error_message}>
              {l.error_message}
            </p>
          )}
        </div>
      ),
    },
    {
      key: "external_id",
      header: "External ID",
      cell: (l) => (
        <span className="font-mono text-xs text-muted-foreground">{l.external_id || "—"}</span>
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
            <Button onClick={handleSync} disabled={syncing}>
              <RefreshCw className={`h-4 w-4 ${syncing ? "animate-spin" : ""}`} />
              {syncing ? "Syncing..." : "Sync now"}
            </Button>
          )
        }
      />

      <div className="mb-8">
        <PaymentGateways />
      </div>

      <h2 className="mb-3 text-sm font-semibold text-foreground">Accounting</h2>
      {error && (
        <p className="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800">{error}</p>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        {PROVIDERS.map((p) => {
          const conn = connectionFor(p.id);
          return (
            <Card key={p.id}>
              <CardContent className="flex flex-col gap-4 p-6">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-md bg-emerald-50 text-emerald-600">
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
                        <dd className="max-w-[12rem] truncate text-red-600" title={conn.last_error}>
                          {conn.last_error}
                        </dd>
                      </div>
                    )}
                  </dl>
                )}

                <div className="mt-auto">
                  {conn ? (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setDisconnectTarget(conn)}
                    >
                      Disconnect
                    </Button>
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
        <h2 className="mb-3 text-sm font-semibold text-foreground">Sync activity</h2>
        <DataTable
          columns={logColumns}
          data={logs}
          loading={logsLoading}
          error={logsError}
          onRetry={fetchLogs}
          getRowId={(l) => l.id}
          empty={{
            icon: RefreshCw,
            title: "No sync activity yet",
            description: "Connect a provider and run a sync to see records here.",
          }}
        />
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
                <div>
                  <Label>Account ID</Label>
                  <Input
                    value={tokenForm.account_id}
                    onChange={(e) => setTokenForm({ ...tokenForm, account_id: e.target.value })}
                    placeholder="e.g. 1234567 or 1234567_SB1"
                    className="font-mono"
                  />
                </div>
                <div>
                  <Label>Access token</Label>
                  <Input
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
