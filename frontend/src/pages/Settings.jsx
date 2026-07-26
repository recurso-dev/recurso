import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Save, ShieldCheck, ChevronRight, Receipt, FileCheck2, MapPinned, Globe, Bot, Building2 } from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Curated business-country list. The code is the ISO 3166-1 alpha-2 stored on
// the primary entity; it drives the tax regime and invoice format.
const COUNTRIES = [
  { code: "US", name: "United States" },
  { code: "IN", name: "India" },
  { code: "GB", name: "United Kingdom" },
  { code: "DE", name: "Germany" },
  { code: "FR", name: "France" },
  { code: "ES", name: "Spain" },
  { code: "IT", name: "Italy" },
  { code: "NL", name: "Netherlands" },
  { code: "IE", name: "Ireland" },
  { code: "CA", name: "Canada" },
  { code: "AU", name: "Australia" },
  { code: "SG", name: "Singapore" },
  { code: "AE", name: "United Arab Emirates" },
];
const COUNTRY_NAME = Object.fromEntries(COUNTRIES.map((c) => [c.code, c.name]));

const EU = new Set([
  "AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR", "DE", "GR",
  "HU", "IE", "IT", "LV", "LT", "LU", "MT", "NL", "PL", "PT", "RO", "SK",
  "SI", "ES", "SE",
]);
// The tax regime an ISO-2 country maps to — mirrors the backend's
// RegimeForCountry so the hub highlights the setup that matches.
const regionOf = (cc) => {
  const c = (cc || "").toUpperCase();
  if (c === "IN") return "IN";
  if (c === "US") return "US";
  if (c === "GB" || EU.has(c)) return "EU";
  return "other";
};

export default function Settings() {
  const queryClient = useQueryClient();
  const [account, setAccount] = useState({ name: "", email: "" });

  // Shared account resource — keyed ["account"] so Profile (read-only view)
  // and this editor read the same cache and a save here refreshes both.
  const { data, isLoading: loading } = useQuery({
    queryKey: ["account"],
    queryFn: async () => (await endpoints.getAccount()).data.data || null,
  });
  useEffect(() => {
    if (data) setAccount({ name: data.name, email: data.email });
  }, [data]);

  // The tenant's legal entities — the primary entity's country_code is the
  // business country that drives the tax regime + invoice format (see #185).
  const { data: entities = [] } = useQuery({
    queryKey: ["entities"],
    queryFn: async () => (await endpoints.getEntities()).data.data || [],
  });
  const primary = entities.find((e) => e.is_primary) || entities[0] || null;
  const businessCountry = primary?.country_code || "";
  const region = regionOf(businessCountry);

  const saveMutation = useMutation({
    mutationFn: (payload) => endpoints.updateAccount(payload),
    onSuccess: () => {
      toast.success("Settings saved successfully.");
      queryClient.invalidateQueries({ queryKey: ["account"] });
    },
    onError: (error) => {
      console.error("Failed to update account:", error);
      toast.error("Failed to save settings.");
    },
  });
  const saving = saveMutation.isPending;
  const handleSave = () => saveMutation.mutate(account);

  // Business country saves immediately against the primary entity (its own
  // control), independent of the account Save button.
  const countryMutation = useMutation({
    mutationFn: (code) =>
      endpoints.updateEntity(primary.id, {
        name: primary.name,
        legal_name: primary.legal_name,
        invoice_prefix: primary.invoice_prefix,
        country_code: code,
      }),
    onSuccess: (_res, code) => {
      toast.success(`Business country set to ${COUNTRY_NAME[code] || code}.`);
      queryClient.invalidateQueries({ queryKey: ["entities"] });
    },
    onError: (error) => {
      console.error("Failed to update business country:", error);
      toast.error("Failed to update business country.");
    },
  });

  // Each tax setup declares the regions it's relevant to; the hub badges and
  // floats the ones matching the business region without hiding the rest.
  const settingsLinks = [
    {
      to: "/security",
      icon: ShieldCheck,
      title: "Security",
      description: "Two-factor authentication and active sessions.",
    },
    {
      to: "/settings/gst",
      icon: Receipt,
      title: "GST configuration",
      description: "GSTIN, business details, and tax rates for invoices.",
      regions: ["IN"],
    },
    {
      to: "/settings/irp",
      icon: FileCheck2,
      title: "E-invoicing (IRP)",
      description: "Connect the Invoice Registration Portal for e-invoices.",
      regions: ["IN"],
    },
    {
      to: "/settings/eu-einvoice",
      icon: Globe,
      title: "EU e-invoicing",
      description: "EN 16931 (UBL) structured invoices and your seller identity.",
      regions: ["EU"],
    },
    {
      to: "/settings/tax-nexus",
      icon: MapPinned,
      title: "US sales-tax nexus",
      description: "Declare collection states and monitor economic thresholds.",
      regions: ["US"],
    },
    {
      to: "/settings/tax-us",
      icon: Receipt,
      title: "US tax identity (W-9)",
      description: "Legal name and EIN shown as the seller on US invoices.",
      regions: ["US"],
    },
    {
      to: "/settings/mcp",
      icon: Bot,
      title: "MCP server",
      description: "Let AI agents operate your billing, and gate money-path actions.",
    },
    {
      to: "/settings/entities",
      icon: Building2,
      title: "Legal entities",
      description: "Bill under multiple legal entities with per-entity books and invoice series.",
    },
  ];
  // Float the region-relevant tax setups to the top, keeping everything else in
  // place. Stable within each group.
  const relevant = (l) => (region !== "other" && l.regions?.includes(region) ? 0 : 1);
  const orderedLinks = settingsLinks
    .map((l, i) => ({ l, i }))
    .sort((a, b) => relevant(a.l) - relevant(b.l) || a.i - b.i)
    .map(({ l }) => l);

  return (
    <div>
      <PageHeader
        title="Settings"
        description="Manage your account information."
        actions={
          <Button onClick={handleSave} disabled={saving || loading}>
            <Save className="h-4 w-4" />
            {saving ? "Saving..." : "Save changes"}
          </Button>
        }
      />

      <div className="max-w-2xl">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">General information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-5">
            <FormField label="Company name" htmlFor="company-name">
              <Input
                id="company-name"
                value={account.name}
                onChange={(e) => setAccount({ ...account, name: e.target.value })}
                placeholder="e.g. Acme Corp"
                disabled={loading}
              />
            </FormField>
            <FormField label="Support email" htmlFor="support-email">
              <Input
                id="support-email"
                type="email"
                value={account.email}
                onChange={(e) => setAccount({ ...account, email: e.target.value })}
                placeholder="support@example.com"
                disabled={loading}
              />
            </FormField>
            {primary && (
              <FormField
                label="Business country"
                htmlFor="business-country"
                description="Sets your tax regime (India GST · US sales tax · EU VAT) and invoice format. Stored on your primary legal entity."
              >
                <Select
                  value={businessCountry || undefined}
                  onValueChange={(code) => countryMutation.mutate(code)}
                  disabled={countryMutation.isPending}
                >
                  <SelectTrigger id="business-country" className="w-full">
                    <SelectValue placeholder="Select your business country" />
                  </SelectTrigger>
                  <SelectContent>
                    {COUNTRIES.map((c) => (
                      <SelectItem key={c.code} value={c.code}>
                        {c.name} ({c.code})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            )}
          </CardContent>
        </Card>

        <Card className="mt-6">
          <CardContent className="divide-y divide-border p-0">
            {orderedLinks.map(({ to, icon: Icon, title, description, regions }) => {
              const isRelevant = region !== "other" && regions?.includes(region);
              return (
                <Link
                  key={to}
                  to={to}
                  className="flex items-center justify-between gap-4 px-6 py-4 transition-colors hover:bg-muted/50"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-md bg-emerald-50 text-emerald-600">
                      <Icon className="h-4 w-4" />
                    </div>
                    <div>
                      <p className="flex items-center gap-2 text-sm font-medium text-foreground">
                        {title}
                        {isRelevant && (
                          <Badge variant="success" className="text-[10px]">
                            For your region
                          </Badge>
                        )}
                      </p>
                      <p className="text-xs text-muted-foreground">{description}</p>
                    </div>
                  </div>
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                </Link>
              );
            })}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
