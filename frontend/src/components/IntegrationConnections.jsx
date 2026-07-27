import { useEffect, useState } from "react";
import { Check, Plug } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { ProviderGuide } from "@/components/patterns/ProviderGuide";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";

// The providers Recurso can bring-your-own, grouped by category, with the
// config fields each one needs. `secret` fields are write-only (never returned
// by the API); non-secret fields (region/bucket/endpoints) show back on the card.
const SECTIONS = [
  {
    category: "tax",
    title: "Tax",
    blurb: "Use your own sales-tax account for US tax calculation.",
    providers: [
      {
        id: "taxjar",
        name: "TaxJar",
        fields: [{ key: "api_key", label: "API key", secret: true, placeholder: "Live API token" }],
        guide: {
          steps: [
            "Sign in to TaxJar and open Account → TaxJar API.",
            "Copy your Live API token (API access requires TaxJar's Professional plan).",
            "Paste it here. Sandbox tokens won't work — this connection calls TaxJar's production API.",
          ],
          url: "https://app.taxjar.com/account#api-access",
          urlLabel: "Open TaxJar API settings",
        },
      },
      {
        id: "avalara",
        name: "Avalara AvaTax",
        fields: [
          { key: "account_id", label: "Account ID", placeholder: "e.g. 2001234567" },
          { key: "license_key", label: "License key", secret: true, placeholder: "Shown once at generation" },
          { key: "company_code", label: "Company code", placeholder: "e.g. DEFAULT" },
        ],
        guide: {
          steps: [
            "Sign in to the AvaTax admin console.",
            "Settings → License and API keys — generate or copy your License key (it is shown only once).",
            "Your Account ID appears in the console header and on the same page.",
            "Company code: Settings → Manage companies (a fresh account's is usually DEFAULT).",
          ],
          url: "https://admin.avalara.com/",
          urlLabel: "Open AvaTax console",
        },
      },
    ],
  },
  {
    category: "crm",
    title: "CRM",
    blurb: "Sync customers to your own CRM daily.",
    providers: [
      {
        id: "hubspot",
        name: "HubSpot",
        fields: [
          {
            key: "access_token",
            label: "Access token (Service Key or private-app token)",
            secret: true,
            placeholder: "Service Key or pat-na1-…",
          },
        ],
        guide: {
          steps: [
            "Recommended — Service Key (HubSpot's new path for single-account API access): Development → Keys → Service Keys → Create service key, with the crm.objects.contacts Read and Write scopes.",
            "Or a private app: Settings → Integrations → Private Apps (new platform: Development → Legacy Apps), same scopes; copy the pat- token from its Auth tab.",
            "Either credential pastes here — both authenticate the same way. A developer API key will NOT work.",
          ],
          url: "https://app.hubspot.com/l/private-apps",
          urlLabel: "Open HubSpot developer tools",
        },
      },
    ],
  },
  {
    category: "storage",
    title: "Storage",
    blurb: "Export your general ledger to your own object storage daily.",
    providers: [
      {
        id: "s3",
        name: "Amazon S3 / MinIO / R2",
        fields: [
          { key: "bucket", label: "Bucket", placeholder: "e.g. acme-ledger-exports" },
          { key: "region", label: "Region", placeholder: "e.g. us-east-1 (R2: auto)" },
          { key: "access_key_id", label: "Access key ID" },
          { key: "secret_access_key", label: "Secret access key", secret: true },
          {
            key: "endpoint",
            label: "Endpoint (MinIO/R2)",
            optional: true,
            placeholder: "https://<account-id>.r2.cloudflarestorage.com",
          },
          { key: "prefix", label: "Key prefix", optional: true, placeholder: "e.g. recurso/ledger/" },
        ],
        guide: {
          steps: [
            "Amazon S3: IAM → Users → Security credentials → Create access key, for a user allowed s3:PutObject on your bucket. Region is the bucket's region; leave Endpoint empty.",
            "Cloudflare R2: R2 → Manage API Tokens → Create API Token (Object Read & Write). Endpoint is https://<account-id>.r2.cloudflarestorage.com; use auto as the region.",
            "MinIO: create an access key in the MinIO console; Endpoint is your MinIO server URL.",
          ],
        },
      },
    ],
  },
];

export default function IntegrationConnections() {
  const [connections, setConnections] = useState([]);
  const [vaultReady, setVaultReady] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [connectTarget, setConnectTarget] = useState(null); // { category, provider }
  const [form, setForm] = useState({});
  const [saving, setSaving] = useState(false);
  const [disconnectTarget, setDisconnectTarget] = useState(null);
  const [disconnecting, setDisconnecting] = useState(false);
  const [crmSyncing, setCrmSyncing] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getIntegrationConnections();
      setConnections(res.data?.data?.connections || []);
      setVaultReady(res.data?.data?.vault_ready ?? true);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load integrations");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const connectionFor = (category, provider) =>
    connections.find((c) => c.category === category && c.provider === provider);

  const openConnect = (category, provider) => {
    setConnectTarget({ category, provider });
    setForm({});
  };

  const canSave = () => {
    if (!connectTarget) return false;
    return connectTarget.provider.fields.every((f) => f.optional || (form[f.key] || "").trim());
  };

  const submitConnect = async () => {
    setSaving(true);
    try {
      const config = {};
      connectTarget.provider.fields.forEach((f) => {
        const v = (form[f.key] || "").trim();
        if (v) config[f.key] = v;
      });
      await api.connectIntegration({
        category: connectTarget.category,
        provider: connectTarget.provider.id,
        config,
      });
      toast.success(`${connectTarget.provider.name} connected.`);
      setConnectTarget(null);
      load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to connect");
    } finally {
      setSaving(false);
    }
  };

  // Test a fresh CRM connection immediately instead of waiting for the
  // daily sweep. Surfaces the provider's own rejection (bad token / scopes).
  const syncCRM = async () => {
    setCrmSyncing(true);
    try {
      const res = await api.syncCRMNow();
      const n = res.data?.data?.contacts_synced ?? 0;
      toast.success(
        n > 0
          ? `Synced ${n} contact${n === 1 ? "" : "s"} to your CRM.`
          : "Connection works — no contacts with an email to sync yet.",
      );
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "CRM sync failed");
    } finally {
      setCrmSyncing(false);
    }
  };

  const disconnect = async () => {
    setDisconnecting(true);
    try {
      await api.disconnectIntegration(disconnectTarget.category, disconnectTarget.provider);
      toast.success("Disconnected.");
      setDisconnectTarget(null);
      load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to disconnect");
    } finally {
      setDisconnecting(false);
    }
  };

  return (
    <div>
      {!vaultReady && (
        <p className="mb-3 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-800">
          Connecting an integration is disabled — the server has no credential encryption key
          (<code>GATEWAY_ENCRYPTION_KEY</code>) configured. These integrations use the
          deployment's environment config until one is set.
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
      {loading && <p className="text-sm text-muted-foreground">Loading integrations…</p>}

      {!loading &&
        SECTIONS.map((section) => (
          <div key={section.category} className="mb-6">
            <h3 className="mb-1 text-sm font-semibold text-foreground">{section.title}</h3>
            <p className="mb-3 text-xs text-muted-foreground">{section.blurb}</p>
            <div className="grid gap-3 sm:grid-cols-2">
              {section.providers.map((provider) => {
                const conn = connectionFor(section.category, provider.id);
                return (
                  <Card key={provider.id}>
                    <CardContent className="flex flex-col gap-3 p-5">
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-2">
                          <Plug className="h-5 w-5 text-muted-foreground" />
                          <span className="font-medium text-foreground">{provider.name}</span>
                        </div>
                        {conn ? (
                          <Badge variant="success">Connected</Badge>
                        ) : (
                          <Badge variant="neutral">Not connected</Badge>
                        )}
                      </div>

                      {conn ? (
                        <>
                          {Object.keys(conn.config || {}).length > 0 && (
                            <dl className="text-xs text-muted-foreground">
                              {Object.entries(conn.config).map(([k, v]) => (
                                <div key={k} className="flex gap-1.5">
                                  <dt className="font-medium">{k}:</dt>
                                  <dd className="truncate font-mono">{v}</dd>
                                </div>
                              ))}
                            </dl>
                          )}
                          {conn.has_secrets && (
                            <p className="flex items-center gap-1 text-xs text-muted-foreground">
                              <Check className="h-3.5 w-3.5 text-emerald-500" />
                              Credentials stored
                            </p>
                          )}
                          <div className="flex gap-2">
                            {section.category === "crm" && (
                              <Button size="sm" onClick={syncCRM} disabled={crmSyncing}>
                                {crmSyncing ? "Syncing…" : "Sync now"}
                              </Button>
                            )}
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => openConnect(section.category, provider)}
                            >
                              Update
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() =>
                                setDisconnectTarget({
                                  category: section.category,
                                  provider: provider.id,
                                  name: provider.name,
                                })
                              }
                            >
                              Disconnect
                            </Button>
                          </div>
                        </>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          className="self-start"
                          disabled={!vaultReady}
                          onClick={() => openConnect(section.category, provider)}
                        >
                          Connect
                        </Button>
                      )}
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          </div>
        ))}

      {/* Connect / update sheet */}
      <Sheet open={!!connectTarget} onOpenChange={(open) => !open && setConnectTarget(null)}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>Connect {connectTarget?.provider.name}</SheetTitle>
            <SheetDescription>
              Credentials are encrypted at rest and never shown again.
            </SheetDescription>
          </SheetHeader>

          {connectTarget && (
            <div className="flex flex-1 flex-col gap-4 overflow-y-auto px-6 py-6">
              <ProviderGuide guide={connectTarget.provider.guide} />
              {connectTarget.provider.fields.map((f) => (
                <div key={f.key} className="space-y-1.5">
                  <Label htmlFor={`f-${f.key}`}>
                    {f.label}
                    {f.optional && (
                      <span className="ml-1 text-xs text-muted-foreground">(optional)</span>
                    )}
                  </Label>
                  <Input
                    id={`f-${f.key}`}
                    type={f.secret ? "password" : "text"}
                    value={form[f.key] || ""}
                    onChange={(e) => setForm({ ...form, [f.key]: e.target.value })}
                    placeholder={f.placeholder}
                    className="font-mono"
                  />
                </div>
              ))}
            </div>
          )}

          <SheetFooter>
            <Button variant="outline" onClick={() => setConnectTarget(null)} disabled={saving}>
              Cancel
            </Button>
            <Button onClick={submitConnect} disabled={saving || !canSave()}>
              {saving ? "Connecting…" : "Connect"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={!!disconnectTarget}
        onOpenChange={(open) => !open && setDisconnectTarget(null)}
        title="Disconnect this integration?"
        description="This workspace will fall back to the deployment's environment config (if any). You can reconnect at any time."
        confirmLabel="Disconnect"
        destructive
        busy={disconnecting}
        onConfirm={disconnect}
      />
    </div>
  );
}
