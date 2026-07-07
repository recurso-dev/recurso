import { useEffect, useState } from "react";
import { Save } from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function Settings() {
  const [account, setAccount] = useState({ name: "", email: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const fetchAccount = async () => {
      try {
        const response = await endpoints.getAccount();
        if (response.data.data) {
          setAccount({
            name: response.data.data.name,
            email: response.data.data.email,
          });
        }
      } catch (error) {
        console.error("Failed to fetch account:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchAccount();
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      await endpoints.updateAccount(account);
      toast.success("Settings saved successfully.");
    } catch (error) {
      console.error("Failed to update account:", error);
      toast.error("Failed to save settings.");
    } finally {
      setSaving(false);
    }
  };

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
      </div>
    </div>
  );
}
