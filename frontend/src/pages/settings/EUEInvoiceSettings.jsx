import { useCallback, useEffect, useState } from "react";
import { Save } from "lucide-react";

import { endpoints } from "@/lib/api";
import { cn } from "@/lib/utils";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormField } from "@/components/patterns/FormField";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";

const EMPTY = {
  enabled: false,
  legal_name: "",
  vat_number: "",
  country_code: "",
  street: "",
  city: "",
  postal_zone: "",
};

// EU e-invoicing settings (Track C): the opt-in flag + the EN 16931 seller
// identity used to generate UBL 2.1 invoices. Enabling requires a complete
// seller identity (the backend rejects an incomplete opt-in).
export default function EUEInvoiceSettings() {
  const [config, setConfig] = useState(EMPTY);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const fetchConfig = useCallback(async () => {
    try {
      const res = await endpoints.getEUEInvoiceConfig();
      setConfig({ ...EMPTY, ...(res.data?.data || {}) });
    } catch {
      /* leave defaults */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const set = (patch) => setConfig((prev) => ({ ...prev, ...patch }));

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      await endpoints.updateEUEInvoiceConfig(config);
      toast.success("EU e-invoicing settings saved");
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to save settings");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="EU e-invoicing"
        description="Generate EN 16931 (UBL 2.1) structured invoices. Off by default — enable it only for tenants under an EU e-invoicing mandate."
      />

      {loading ? (
        <Skeleton className="h-96 w-full rounded-xl" />
      ) : (
        <form onSubmit={handleSave}>
          <Card>
            <CardContent className="space-y-6 pt-6">
              <div className="flex items-center justify-between rounded-lg border border-border p-4">
                <div>
                  <h3 className="text-sm font-medium text-foreground">Enable EU e-invoicing</h3>
                  <p className="text-sm text-muted-foreground">
                    Generate an EN 16931 UBL document on each invoice, delivered via the
                    configured transport.
                  </p>
                </div>
                <button
                  type="button"
                  role="switch"
                  aria-checked={config.enabled}
                  onClick={() => set({ enabled: !config.enabled })}
                  className={cn(
                    "relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                    config.enabled ? "bg-primary" : "bg-stone-200",
                  )}
                >
                  <span
                    className={cn(
                      "pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out",
                      config.enabled ? "translate-x-5" : "translate-x-0",
                    )}
                  />
                </button>
              </div>

              <p className="text-xs text-muted-foreground">
                Seller party — the details every generated invoice carries. Required to enable.
              </p>

              <FormField label="Legal name" htmlFor="legal_name">
                <Input
                  id="legal_name"
                  value={config.legal_name}
                  onChange={(e) => set({ legal_name: e.target.value })}
                  placeholder="Acme GmbH"
                />
              </FormField>

              <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                <FormField label="VAT number" htmlFor="vat_number">
                  <Input
                    id="vat_number"
                    value={config.vat_number}
                    onChange={(e) => set({ vat_number: e.target.value.toUpperCase() })}
                    placeholder="DE123456789"
                    className="font-mono"
                  />
                </FormField>
                <FormField label="Country (ISO code)" htmlFor="country_code">
                  <Input
                    id="country_code"
                    value={config.country_code}
                    onChange={(e) => set({ country_code: e.target.value.toUpperCase().slice(0, 2) })}
                    placeholder="DE"
                    maxLength={2}
                    className="font-mono uppercase"
                  />
                </FormField>
              </div>

              <FormField label="Street" htmlFor="street">
                <Input
                  id="street"
                  value={config.street}
                  onChange={(e) => set({ street: e.target.value })}
                  placeholder="Hauptstr. 1"
                />
              </FormField>

              <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                <FormField label="City" htmlFor="city">
                  <Input
                    id="city"
                    value={config.city}
                    onChange={(e) => set({ city: e.target.value })}
                    placeholder="Berlin"
                  />
                </FormField>
                <FormField label="Postal code" htmlFor="postal_zone">
                  <Input
                    id="postal_zone"
                    value={config.postal_zone}
                    onChange={(e) => set({ postal_zone: e.target.value })}
                    placeholder="10115"
                  />
                </FormField>
              </div>

              <div className="flex justify-end">
                <Button type="submit" disabled={saving}>
                  <Save className="h-4 w-4" />
                  {saving ? "Saving…" : "Save settings"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </form>
      )}
    </div>
  );
}
