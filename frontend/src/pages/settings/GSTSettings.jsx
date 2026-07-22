import { useEffect, useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Save, Check, AlertCircle } from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { EntityScopeSelect } from "@/components/patterns/EntityScopeSelect";
import { FormField } from "@/components/patterns/FormField";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function GSTSettings() {
  const [config, setConfig] = useState({
    gstin: "",
    state_code: "",
    state_name: "",
    sac_code: "998314",
    gst_rate: 18,
    pan: "",
    legal_name: "",
    trade_name: "",
    address: "",
    has_lut: false,
  });
  const [validation, setValidation] = useState(null);
  const [entityId, setEntityId] = useState("");

  const { data, isLoading: loading, isError: loadError, refetch } = useQuery({
    queryKey: ["gst-config", entityId],
    queryFn: async () => (await endpoints.getGSTConfig(entityId)).data.data || null,
  });
  useEffect(() => {
    if (data) setConfig(data);
  }, [data]);

  const validateMutation = useMutation({
    mutationFn: (gstin) => endpoints.validateGSTIN(gstin),
    onSuccess: (response) => {
      setValidation(response.data);
      if (response.data.valid) {
        setConfig((prev) => ({
          ...prev,
          state_code: response.data.state_code,
          state_name: response.data.state_name,
          pan: response.data.pan,
        }));
      }
    },
    onError: () => setValidation({ valid: false, message: "Validation failed" }),
  });
  const validating = validateMutation.isPending;

  const saveMutation = useMutation({
    mutationFn: (cfg) => endpoints.updateGSTConfig(cfg, entityId),
    onSuccess: () => toast.success("GST configuration saved successfully"),
    onError: () => toast.error("Failed to save configuration"),
  });
  const saving = saveMutation.isPending;

  const validateGSTIN = () => {
    if (!config.gstin || config.gstin.length !== 15) {
      setValidation({ valid: false, message: "GSTIN must be 15 characters" });
      return;
    }
    validateMutation.mutate(config.gstin);
  };

  const saveConfig = () => saveMutation.mutate(config);

  const textareaClass =
    "flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1";

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="GST configuration"
        description="Configure your GST details for invoice generation."
        actions={
          <div className="flex items-center gap-3">
            <EntityScopeSelect value={entityId} onChange={setEntityId} />
            <Button onClick={saveConfig} disabled={saving || loading}>
              <Save className="h-4 w-4" />
              {saving ? "Saving..." : "Save configuration"}
            </Button>
          </div>
        }
      />

      {loading ? (
        <div className="space-y-4">
          <Skeleton className="h-40 w-full rounded-xl" />
          <Skeleton className="h-40 w-full rounded-xl" />
        </div>
      ) : loadError ? (
        <ErrorState
          title="Couldn't load GST configuration"
          message="We couldn't reach the settings service. Please try again."
          onRetry={refetch}
        />
      ) : (
        <div className="space-y-6">
          {/* GSTIN Details */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">GSTIN details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <FormField label="GSTIN" htmlFor="gstin">
                <div className="flex gap-2">
                  <Input
                    id="gstin"
                    value={config.gstin}
                    onChange={(e) =>
                      setConfig({ ...config, gstin: e.target.value.toUpperCase() })
                    }
                    placeholder="22AAAAA0000A1Z5"
                    maxLength={15}
                    className="flex-1"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={validateGSTIN}
                    disabled={validating || !config.gstin}
                  >
                    {validating ? "Validating..." : "Validate"}
                  </Button>
                </div>
                {validation && (
                  <p
                    className={
                      "mt-2 flex items-center gap-1.5 text-xs font-medium " +
                      (validation.valid ? "text-emerald-600" : "text-red-600")
                    }
                  >
                    {validation.valid ? (
                      <Check className="h-3.5 w-3.5" />
                    ) : (
                      <AlertCircle className="h-3.5 w-3.5" />
                    )}
                    {validation.message}
                  </p>
                )}
              </FormField>

              <div className="grid gap-5 sm:grid-cols-2">
                <FormField label="State code" htmlFor="state_code">
                  <Input
                    id="state_code"
                    value={config.state_code}
                    readOnly
                    placeholder="Auto-filled from GSTIN"
                    className="bg-muted/40"
                  />
                </FormField>
                <FormField label="State name" htmlFor="state_name">
                  <Input
                    id="state_name"
                    value={config.state_name}
                    readOnly
                    placeholder="Auto-filled from GSTIN"
                    className="bg-muted/40"
                  />
                </FormField>
              </div>

              <FormField label="PAN" htmlFor="pan" className="sm:max-w-xs">
                <Input
                  id="pan"
                  value={config.pan}
                  onChange={(e) => setConfig({ ...config, pan: e.target.value.toUpperCase() })}
                  placeholder="AAAAA0000A"
                  maxLength={10}
                />
              </FormField>
            </CardContent>
          </Card>

          {/* Business Details */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Business details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-5 sm:grid-cols-2">
                <FormField label="Legal name" htmlFor="legal_name">
                  <Input
                    id="legal_name"
                    value={config.legal_name}
                    onChange={(e) => setConfig({ ...config, legal_name: e.target.value })}
                    placeholder="As per registration"
                  />
                </FormField>
                <FormField label="Trade name" htmlFor="trade_name">
                  <Input
                    id="trade_name"
                    value={config.trade_name}
                    onChange={(e) => setConfig({ ...config, trade_name: e.target.value })}
                    placeholder="Brand name"
                  />
                </FormField>
              </div>

              <FormField label="Registered address" htmlFor="address">
                <textarea
                  id="address"
                  value={config.address}
                  onChange={(e) => setConfig({ ...config, address: e.target.value })}
                  placeholder="Full registered address"
                  rows={3}
                  className={textareaClass}
                />
              </FormField>
            </CardContent>
          </Card>

          {/* Tax Settings */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Tax settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-5 sm:grid-cols-2">
                <FormField
                  label="SAC code"
                  htmlFor="sac_code"
                  description="Default for SaaS: 998314"
                >
                  <Input
                    id="sac_code"
                    value={config.sac_code}
                    onChange={(e) => setConfig({ ...config, sac_code: e.target.value })}
                    placeholder="998314"
                  />
                </FormField>
                <FormField
                  label="GST rate (%)"
                  htmlFor="gst_rate"
                  description="Standard rate for software: 18%"
                >
                  <Input
                    id="gst_rate"
                    type="number"
                    value={config.gst_rate}
                    onChange={(e) =>
                      setConfig({ ...config, gst_rate: parseFloat(e.target.value) })
                    }
                    min={0}
                    max={28}
                  />
                </FormField>
              </div>

              <div className="flex items-start gap-3">
                <input
                  id="has_lut"
                  type="checkbox"
                  checked={config.has_lut}
                  onChange={(e) => setConfig({ ...config, has_lut: e.target.checked })}
                  className="mt-0.5 h-4 w-4 rounded border-input accent-emerald-600 focus:ring-ring"
                />
                <label htmlFor="has_lut" className="text-sm">
                  <span className="font-medium text-foreground">
                    LUT (Letter of Undertaking) for exports
                  </span>
                  <span className="mt-0.5 block text-xs text-muted-foreground">
                    Enable for 0% GST on export of services.
                  </span>
                </label>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
