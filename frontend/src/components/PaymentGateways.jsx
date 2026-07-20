import { useEffect, useState } from "react";
import { CreditCard, Check, Copy, Link2 } from "lucide-react";

import { endpoints as api, API_ROOT } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";

const GATEWAYS = [
  {
    id: "stripe",
    name: "Stripe",
    publicLabel: "Publishable key",
    publicPlaceholder: "pk_live_…",
    secretPlaceholder: "sk_live_…",
  },
  {
    id: "razorpay",
    name: "Razorpay",
    publicLabel: "Key ID",
    publicPlaceholder: "rzp_live_…",
    secretPlaceholder: "Key secret",
  },
];

// Webhooks are served at the API origin (not under /v1); API_ROOT strips the
// /v1 suffix. Fall back to the app origin when the base is a relative dev path.
const webhookOrigin = API_ROOT && /^https?:\/\//.test(API_ROOT) ? API_ROOT : window.location.origin;

export default function PaymentGateways() {
  const [connections, setConnections] = useState([]);
  const [vaultReady, setVaultReady] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [connectTarget, setConnectTarget] = useState(null); // gateway being connected
  const [form, setForm] = useState({ mode: "test", public_key: "", secret_key: "", webhook_secret: "" });
  const [saving, setSaving] = useState(false);
  const [disconnectTarget, setDisconnectTarget] = useState(null);
  const [disconnecting, setDisconnecting] = useState(false);
  const [webhookInput, setWebhookInput] = useState({}); // provider -> in-progress secret

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getGatewayConnections();
      setConnections(res.data?.data?.connections || []);
      setVaultReady(res.data?.data?.vault_ready ?? true);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load payment gateways");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const connectionFor = (id) => connections.find((c) => c.provider === id);

  const openConnect = (gateway) => {
    setConnectTarget(gateway);
    setForm({ mode: "test", public_key: "", secret_key: "", webhook_secret: "" });
  };

  const submitConnect = async () => {
    setSaving(true);
    try {
      await api.connectGateway({
        provider: connectTarget.id,
        mode: form.mode,
        public_key: form.public_key,
        secret_key: form.secret_key,
        webhook_secret: form.webhook_secret,
      });
      toast.success(`${connectTarget.name} connected.`);
      setConnectTarget(null);
      load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to connect");
    } finally {
      setSaving(false);
    }
  };

  const disconnect = async () => {
    setDisconnecting(true);
    try {
      await api.disconnectGateway(disconnectTarget.provider);
      toast.success("Disconnected.");
      setDisconnectTarget(null);
      load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to disconnect");
    } finally {
      setDisconnecting(false);
    }
  };

  const saveWebhookSecret = async (provider) => {
    const secret = (webhookInput[provider] || "").trim();
    if (!secret) return;
    try {
      await api.setGatewayWebhookSecret(provider, secret);
      toast.success("Webhook secret saved.");
      setWebhookInput((p) => ({ ...p, [provider]: "" }));
      load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to save webhook secret");
    }
  };

  const copy = (text) => {
    navigator.clipboard?.writeText(text);
    toast.success("Copied to clipboard.");
  };

  return (
    <div>
      <h2 className="mb-3 text-sm font-semibold text-foreground">Payment gateways</h2>

      {!vaultReady && (
        <p className="mb-3 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-800">
          Connecting a gateway is disabled — the server has no credential encryption key
          (<code>GATEWAY_ENCRYPTION_KEY</code>) configured. Payments use the platform gateway
          until one is set.
        </p>
      )}
      {error && (
        <p className="mb-3 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800">
          {error}{" "}
          <button className="underline" onClick={load}>
            Retry
          </button>
        </p>
      )}

      {loading && <p className="text-sm text-muted-foreground">Loading gateways…</p>}

      <div className={"grid gap-3 sm:grid-cols-2" + (loading ? " hidden" : "")}>
        {GATEWAYS.map((gw) => {
          const conn = connectionFor(gw.id);
          const webhookUrl = conn ? webhookOrigin + conn.webhook_path : null;
          return (
            <Card key={gw.id}>
              <CardContent className="flex flex-col gap-3 p-5">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <CreditCard className="h-5 w-5 text-muted-foreground" />
                    <span className="font-medium text-foreground">{gw.name}</span>
                  </div>
                  {conn ? (
                    <Badge variant={conn.mode === "live" ? "success" : "neutral"}>
                      {conn.mode === "live" ? "Live" : "Test"}
                    </Badge>
                  ) : (
                    <Badge variant="neutral">Not connected</Badge>
                  )}
                </div>

                {conn ? (
                  <>
                    <p className="font-mono text-xs text-muted-foreground">
                      {conn.public_key || "—"}
                    </p>

                    {/* Webhook URL to paste into the gateway console */}
                    <div className="rounded-md border border-border bg-muted/40 p-2.5">
                      <div className="mb-1 flex items-center gap-1.5 text-xs font-medium text-foreground">
                        <Link2 className="h-3.5 w-3.5" />
                        Webhook URL
                      </div>
                      <div className="flex items-center gap-2">
                        <code className="flex-1 truncate text-xs text-muted-foreground">
                          {webhookUrl}
                        </code>
                        <button
                          type="button"
                          onClick={() => copy(webhookUrl)}
                          aria-label="Copy webhook URL"
                          className="text-stone-400 transition-colors hover:text-foreground"
                        >
                          <Copy className="h-3.5 w-3.5" />
                        </button>
                      </div>
                      <p className="mt-1.5 flex items-center gap-1 text-xs text-muted-foreground">
                        {conn.has_webhook_secret ? (
                          <>
                            <Check className="h-3.5 w-3.5 text-emerald-500" />
                            Signing secret set
                          </>
                        ) : (
                          "Create a webhook at this URL in your gateway, then paste its signing secret:"
                        )}
                      </p>
                      {!conn.has_webhook_secret && (
                        <div className="mt-1.5 flex items-center gap-2">
                          <Input
                            value={webhookInput[gw.id] || ""}
                            onChange={(e) =>
                              setWebhookInput((p) => ({ ...p, [gw.id]: e.target.value }))
                            }
                            placeholder="whsec_… / webhook secret"
                            aria-label={`${gw.name} webhook secret`}
                            className="h-8 font-mono"
                          />
                          <Button size="sm" onClick={() => saveWebhookSecret(gw.id)}>
                            Save
                          </Button>
                        </div>
                      )}
                    </div>

                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" onClick={() => openConnect(gw)}>
                        Update keys
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setDisconnectTarget(conn)}
                      >
                        Disconnect
                      </Button>
                    </div>
                  </>
                ) : (
                  <>
                    <p className="text-xs text-muted-foreground">
                      Use your own {gw.name} account to process this workspace's payments.
                    </p>
                    <Button
                      variant="outline"
                      size="sm"
                      className="self-start"
                      disabled={!vaultReady}
                      onClick={() => openConnect(gw)}
                    >
                      Connect
                    </Button>
                  </>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Connect / update sheet */}
      <Sheet open={!!connectTarget} onOpenChange={(open) => !open && setConnectTarget(null)}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>Connect {connectTarget?.name}</SheetTitle>
            <SheetDescription>
              Keys are encrypted at rest and never shown again. Find them in your{" "}
              {connectTarget?.name} dashboard.
            </SheetDescription>
          </SheetHeader>

          {connectTarget && (
            <div className="flex flex-1 flex-col gap-4 overflow-y-auto px-6 py-6">
              <div className="space-y-1.5">
                <Label htmlFor="gw-mode">Mode</Label>
                <select
                  id="gw-mode"
                  value={form.mode}
                  onChange={(e) => setForm({ ...form, mode: e.target.value })}
                  className="h-9 w-full rounded-md border border-input bg-transparent px-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <option value="test">Test</option>
                  <option value="live">Live</option>
                </select>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="gw-public">{connectTarget.publicLabel}</Label>
                <Input
                  id="gw-public"
                  value={form.public_key}
                  onChange={(e) => setForm({ ...form, public_key: e.target.value })}
                  placeholder={connectTarget.publicPlaceholder}
                  className="font-mono"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="gw-secret">Secret key</Label>
                <Input
                  id="gw-secret"
                  type="password"
                  value={form.secret_key}
                  onChange={(e) => setForm({ ...form, secret_key: e.target.value })}
                  placeholder={connectTarget.secretPlaceholder}
                  className="font-mono"
                />
              </div>
              <p className="text-xs text-muted-foreground">
                You can add the webhook signing secret after connecting, once you've created a
                webhook at the URL shown on the card.
              </p>
            </div>
          )}

          <SheetFooter>
            <Button variant="outline" onClick={() => setConnectTarget(null)} disabled={saving}>
              Cancel
            </Button>
            <Button
              onClick={submitConnect}
              disabled={saving || !form.secret_key.trim() || (connectTarget?.id === "razorpay" && !form.public_key.trim())}
            >
              {saving ? "Connecting…" : "Connect"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={!!disconnectTarget}
        onOpenChange={(open) => !open && setDisconnectTarget(null)}
        title="Disconnect this gateway?"
        description="New payments will fall back to the platform gateway. Existing transactions are unaffected. You can reconnect at any time."
        confirmLabel="Disconnect"
        destructive
        busy={disconnecting}
        onConfirm={disconnect}
      />
    </div>
  );
}
