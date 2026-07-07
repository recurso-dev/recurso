import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { KeyRound, Pencil } from "lucide-react";

import { endpoints } from "@/lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Read-only display field styled to match the light theme.
function ReadOnlyField({ label, value, mono, children }) {
  return (
    <div className="space-y-1.5">
      <Label className="text-foreground">{label}</Label>
      <div
        className={
          "flex items-center justify-between rounded-md border border-border bg-muted/40 px-3 py-2 text-sm text-foreground" +
          (mono ? " font-mono text-muted-foreground" : "")
        }
      >
        <span className="truncate">{value}</span>
        {children}
      </div>
    </div>
  );
}

export default function Profile() {
  const navigate = useNavigate();
  const [account, setAccount] = useState({ name: "", email: "", id: "" });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchAccount = async () => {
      try {
        const response = await endpoints.getAccount();
        if (response.data.data) {
          setAccount({
            name: response.data.data.name,
            email: response.data.data.email,
            id: response.data.data.id,
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

  return (
    <div>
      <PageHeader
        title="Account profile"
        description="View your account identity and security information."
        actions={
          <Button variant="outline" onClick={() => navigate("/settings")}>
            <Pencil className="h-4 w-4" />
            Edit profile
          </Button>
        }
      />

      <div className="max-w-2xl space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Account information</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col items-start gap-8 md:flex-row">
              <div className="flex h-20 w-20 shrink-0 items-center justify-center rounded-full border border-border bg-emerald-50 text-2xl font-semibold text-emerald-700">
                {account.name ? account.name.charAt(0).toUpperCase() : "A"}
              </div>

              <div className="grid w-full flex-1 gap-4">
                <ReadOnlyField
                  label="Account name"
                  value={loading ? "Loading..." : account.name}
                />
                <ReadOnlyField label="Email address" value={loading ? "Loading..." : account.email}>
                  {!loading && <Badge variant="success">Verified</Badge>}
                </ReadOnlyField>
                <ReadOnlyField
                  label="Tenant ID"
                  value={loading ? "Loading..." : account.id}
                  mono
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Security</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="mb-5 text-sm text-muted-foreground">
              Your account uses API keys for authentication. You can manage your keys in
              the Developers section.
            </p>
            <div className="flex items-center gap-4 rounded-lg border border-border bg-muted/40 p-4">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-white text-zinc-500">
                <KeyRound className="h-4 w-4" />
              </div>
              <div className="flex-1">
                <h3 className="text-sm font-medium text-foreground">
                  API key authentication
                </h3>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  You are currently authenticated via a tenant API key.
                </p>
              </div>
              <Button variant="link" className="h-auto p-0" onClick={() => navigate("/developers")}>
                Manage keys
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
