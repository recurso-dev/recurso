import { useEffect, useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Save } from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormField } from "@/components/patterns/FormField";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { ErrorState } from "@/components/patterns/ErrorState";

const EMPTY = { legal_name: "", ein: "", address: "" };

// US tax identity (W-9): the seller party shown on US sales-tax invoices.
// Presentation only — it does not change how tax is computed.
export default function USTaxSettings() {
  const [config, setConfig] = useState(EMPTY);

  const { data, isLoading: loading, isError: loadError, refetch } = useQuery({
    queryKey: ["us-tax-config"],
    queryFn: async () => (await endpoints.getUSTaxConfig()).data?.data || null,
  });
  useEffect(() => {
    if (data) setConfig((prev) => ({ ...prev, ...data }));
  }, [data]);

  const set = (patch) => setConfig((prev) => ({ ...prev, ...patch }));

  const saveMutation = useMutation({
    mutationFn: (cfg) => endpoints.updateUSTaxConfig(cfg),
    onSuccess: () => toast.success("US tax settings saved"),
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to save settings"),
  });
  const saving = saveMutation.isPending;

  const handleSave = (e) => {
    e.preventDefault();
    saveMutation.mutate(config);
  };

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="US tax identity (W-9)"
        description="The seller party shown on your US sales-tax invoices. Set your business country to United States in Settings to render US-format invoices."
      />

      {loading ? (
        <Skeleton className="h-72 w-full rounded-xl" />
      ) : loadError ? (
        <ErrorState
          title="Couldn't load US tax identity"
          message="We couldn't reach the settings service, so the form is hidden to avoid saving blanks over your saved identity."
          onRetry={() => refetch()}
        />
      ) : (
        <form onSubmit={handleSave}>
          <Card>
            <CardContent className="space-y-6 pt-6">
              <FormField
                label="Legal name"
                htmlFor="legal_name"
                description="Your registered business name, as it should appear on invoices."
              >
                <Input
                  id="legal_name"
                  value={config.legal_name}
                  onChange={(e) => set({ legal_name: e.target.value })}
                  placeholder="Acme Inc"
                />
              </FormField>

              <FormField
                label="EIN"
                htmlFor="ein"
                description="Employer Identification Number (your W-9 tax id), e.g. 12-3456789."
              >
                <Input
                  id="ein"
                  value={config.ein}
                  onChange={(e) => set({ ein: e.target.value })}
                  placeholder="12-3456789"
                  className="font-mono"
                />
              </FormField>

              <FormField label="Address" htmlFor="address">
                <Input
                  id="address"
                  value={config.address}
                  onChange={(e) => set({ address: e.target.value })}
                  placeholder="1 Market St, San Francisco, CA 94105"
                />
              </FormField>

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
