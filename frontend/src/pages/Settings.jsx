import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Save } from "lucide-react";

import { endpoints } from "@/lib/api";
import { COUNTRIES, COUNTRY_NAME } from "@/lib/countries";
import { toast } from "@/components/ui/sonner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";


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

  // Business country switches the tax regime (GST / sales tax / VAT) the
  // moment it saves — picking a value asks for confirmation first instead of
  // auto-saving on select (audit §7).
  const [pendingCountry, setPendingCountry] = useState(null);
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

  return (
    <div>
      <PageHeader
        title="General"
        description="Your company identity and business country. Pick a section on the left for taxes, branding, and platform settings."
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
                  onValueChange={(code) => setPendingCountry(code)}
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
            <div className="flex justify-end">
              <Button onClick={handleSave} disabled={saving || loading}>
                <Save className="h-4 w-4" />
                {saving ? "Saving..." : "Save changes"}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <ConfirmDialog
        open={Boolean(pendingCountry)}
        onOpenChange={(o) => !o && setPendingCountry(null)}
        title={`Set business country to ${pendingCountry ? COUNTRY_NAME[pendingCountry] || pendingCountry : ""}?`}
        description="This switches your tax regime and invoice format immediately. New invoices are taxed under the new regime."
        confirmLabel="Change country"
        busy={countryMutation.isPending}
        onConfirm={() =>
          countryMutation.mutate(pendingCountry, { onSettled: () => setPendingCountry(null) })
        }
      />
    </div>
  );
}
