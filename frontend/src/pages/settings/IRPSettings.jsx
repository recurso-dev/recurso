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
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    try {
      const response = await endpoints.getIRPConfig();
      if (response.data?.data) {
        setConfig((prev) => ({ ...prev, ...response.data.data }));
      }
    } catch (err) {
      // Config may not exist yet, that's OK.
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      await endpoints.updateIRPConfig(config);
      toast.success("IRP configuration saved successfully");
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to save configuration");
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const response = await endpoints.testIRPConfig();
      setTestResult(response.data);
    } catch (err) {
      setTestResult({
        success: false,
        message: err?.response?.data?.error?.message || "Connection test failed",
      });
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="IRP settings"
        description="Configure NIC Invoice Registration Portal credentials for e-invoicing."
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
                  <h3 className="text-sm font-medium text-foreground">Enable e-invoicing</h3>
                  <p className="text-sm text-muted-foreground">
                    Generate IRN for B2B invoices via NIC IRP.
                  </p>
                </div>
                <button
                  type="button"
                  role="switch"
                  aria-checked={config.is_enabled}
                  onClick={() => setConfig((prev) => ({ ...prev, is_enabled: !prev.is_enabled }))}
                  className={cn(
                    "relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                    config.is_enabled ? "bg-primary" : "bg-stone-200"
                  )}
                >
                  <span
                    className={cn(
                      "pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out",
                      config.is_enabled ? "translate-x-5" : "translate-x-0"
                    )}
                  />
                </button>
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
