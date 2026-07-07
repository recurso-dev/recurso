import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Save, ShieldCheck, ChevronRight } from "lucide-react";

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

        <Card className="mt-6">
          <CardContent className="p-0">
            <Link
              to="/security"
              className="flex items-center justify-between gap-4 px-6 py-4 transition-colors hover:bg-muted/50"
            >
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-md bg-emerald-50 text-emerald-600">
                  <ShieldCheck className="h-4 w-4" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">Security</p>
                  <p className="text-xs text-muted-foreground">
                    Two-factor authentication and active sessions.
                  </p>
                </div>
              </div>
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            </Link>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
