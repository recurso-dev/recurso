import { useEffect, useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Save } from "lucide-react";

import { endpoints } from "@/lib/api";
import { cn } from "@/lib/utils";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { EntityScopeSelect } from "@/components/patterns/EntityScopeSelect";
import { FormField } from "@/components/patterns/FormField";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export default function IRPSettings() {
  const [config, setConfig] = useState({
    environment: "sandbox",
    client_id: "",
    client_secret: "",
    username: "",
    password: "",
    gstin: "",
    is_enabled: false,
  });
  const [testResult, setTestResult] = useState(null);
  const [entityId, setEntityId] = useState("");

  // Load the saved config; a missing config just leaves the defaults.
  const { data, isLoading: loading } = useQuery({
    queryKey: ["irp-config", entityId],
    queryFn: async () => (await endpoints.getIRPConfig(entityId)).data?.data || null,
  });
  useEffect(() => {
    if (data) setConfig((prev) => ({ ...prev, ...data }));
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: (cfg) => endpoints.updateIRPConfig(cfg, entityId),
    onSuccess: () => toast.success("IRP configuration saved successfully"),
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to save configuration"),
  });
  const saving = saveMutation.isPending;

  const testMutation = useMutation({
    mutationFn: () => endpoints.testIRPConfig(entityId),
    onSuccess: (response) => setTestResult(response.data),
    onError: (err) =>
      setTestResult({
        success: false,
        message: err?.response?.data?.error?.message || "Connection test failed",
      }),
  });
  const testing = testMutation.isPending;

  const handleSave = (e) => {
    e.preventDefault();
    saveMutation.mutate(config);
  };

  const handleTest = () => {
    setTestResult(null);
    testMutation.mutate();
  };

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="IRP settings"
        description="Configure NIC Invoice Registration Portal credentials for e-invoicing."
        actions={<EntityScopeSelect value={entityId} onChange={setEntityId} />}
      />

      {loading ? (
        <Skeleton className="h-96 w-full rounded-xl" />
      ) : (
        <form onSubmit={handleSave}>
          <Card>
            <CardContent className="space-y-6 pt-6">
              {/* Enable Toggle */}
              <div className="flex items-center justify-between rounded-lg border border-border p-4">
                <div>
                  <h3 id="irp-enable-label" className="text-sm font-medium text-foreground">
                    Enable e-invoicing
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    Generate IRN for B2B invoices via NIC IRP.
                  </p>
                </div>
                <Switch
                  aria-labelledby="irp-enable-label"
                  checked={config.is_enabled}
                  onCheckedChange={(next) =>
                    setConfig((prev) => ({ ...prev, is_enabled: next }))
                  }
                />
              </div>

              <FormField label="Environment" htmlFor="environment">
                <Select
                  value={config.environment}
                  onValueChange={(value) =>
                    setConfig((prev) => ({ ...prev, environment: value }))
                  }
                >
                  <SelectTrigger id="environment">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="sandbox">Sandbox (Testing)</SelectItem>
                    <SelectItem value="production">Production</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>

              <FormField label="GSTIN" htmlFor="gstin">
                <Input
                  id="gstin"
                  value={config.gstin}
                  onChange={(e) =>
                    setConfig((prev) => ({ ...prev, gstin: e.target.value.toUpperCase() }))
                  }
                  placeholder="e.g., 33ABCDE1234F1Z5"
                  maxLength={15}
                  className="font-mono"
                />
              </FormField>

              <FormField label="Client ID" htmlFor="client_id">
                <Input
                  id="client_id"
                  value={config.client_id}
                  onChange={(e) =>
                    setConfig((prev) => ({ ...prev, client_id: e.target.value }))
                  }
                  placeholder="NIC API Client ID"
                />
              </FormField>

              <FormField label="Client secret" htmlFor="client_secret">
                <Input
                  id="client_secret"
                  type="password"
                  value={config.client_secret}
                  onChange={(e) =>
                    setConfig((prev) => ({ ...prev, client_secret: e.target.value }))
                  }
                  placeholder="NIC API Client Secret"
                />
              </FormField>

              <FormField label="Username" htmlFor="username">
                <Input
                  id="username"
                  value={config.username}
                  onChange={(e) =>
                    setConfig((prev) => ({ ...prev, username: e.target.value }))
                  }
                  placeholder="NIC API Username"
                />
              </FormField>

              <FormField label="Password" htmlFor="password">
                <Input
                  id="password"
                  type="password"
                  value={config.password}
                  onChange={(e) =>
                    setConfig((prev) => ({ ...prev, password: e.target.value }))
                  }
                  placeholder="NIC API Password"
                />
              </FormField>

              {testResult && (
                <div
                  className={cn(
                    "rounded-lg border px-4 py-3 text-sm",
                    testResult.success
                      ? "border-emerald-200 bg-emerald-50 text-emerald-800"
                      : "border-red-200 bg-red-50 text-red-800"
                  )}
                >
                  {testResult.message}
                </div>
              )}

              <div className="flex gap-3 border-t border-border pt-5">
                <Button type="submit" disabled={saving}>
                  <Save className="h-4 w-4" />
                  {saving ? "Saving..." : "Save configuration"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleTest}
                  disabled={testing}
                >
                  {testing ? "Testing..." : "Test connection"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </form>
      )}
    </div>
  );
}
