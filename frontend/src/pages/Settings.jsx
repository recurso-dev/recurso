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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

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
    },
    {
      to: "/settings/irp",
      icon: FileCheck2,
      title: "E-invoicing (IRP)",
      description: "Connect the Invoice Registration Portal for e-invoices.",
    },
    {
      to: "/settings/eu-einvoice",
      icon: Globe,
      title: "EU e-invoicing",
      description: "EN 16931 (UBL) structured invoices and your seller identity.",
    },
    {
      to: "/settings/tax-nexus",
      icon: MapPinned,
      title: "US sales-tax nexus",
      description: "Declare collection states and monitor economic thresholds.",
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
          </CardContent>
        </Card>

        <Card className="mt-6">
          <CardContent className="divide-y divide-border p-0">
            {settingsLinks.map(({ to, icon: Icon, title, description }) => (
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
                    <p className="text-sm font-medium text-foreground">{title}</p>
                    <p className="text-xs text-muted-foreground">{description}</p>
                  </div>
                </div>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </Link>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
