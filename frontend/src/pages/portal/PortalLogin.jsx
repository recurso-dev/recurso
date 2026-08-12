import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { CheckCircle2, Loader2, Mail } from "lucide-react";

import { API_ROOT as API_BASE } from "../../lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";

const PortalLogin = () => {
  // Support prefilled links from the admin dashboard: /portal/login?email=…
  const [searchParams] = useSearchParams();
  const [email, setEmail] = useState(searchParams.get("email") || "");
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [devLink, setDevLink] = useState(null);
  const [error, setError] = useState(null);
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/portal/auth/request`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });

      const data = await response.json();

      if (response.ok) {
        setSuccess(true);
        // For development, show the magic link
        if (data._dev_link) {
          setDevLink(data._dev_link);
        }
      } else {
        setError(data.error?.message || "Failed to send login link");
      }
    } catch (err) {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleDevLogin = async () => {
    if (!devLink) return;

    try {
      // credentials: "include" stores the httpOnly session cookie the server
      // sets on success; the token is never held in JS storage.
      const response = await fetch(`${API_BASE}${devLink}`, {
        credentials: "include",
      });

      if (response.ok) {
        navigate("/portal/dashboard");
      } else {
        setError("Failed to verify link");
      }
    } catch (err) {
      setError("Failed to verify link");
    }
  };

  const Logo = () => (
    <div className="mb-8 flex items-center justify-center gap-2">
      <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary text-lg font-bold text-primary-foreground">
        R
      </div>
      <span className="text-xl font-semibold tracking-tight text-foreground">
        Recurso
      </span>
    </div>
  );

  if (success) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted px-4">
        <div className="w-full max-w-md">
          <Card>
            <CardContent className="p-8 text-center">
              <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-success/5 ring-1 ring-inset ring-success/20">
                <CheckCircle2 className="h-7 w-7 text-primary" />
              </div>
              <h2 className="text-xl font-semibold text-foreground">
                Check your email
              </h2>
              <p className="mt-2 text-sm text-muted-foreground">
                We&apos;ve sent a login link to{" "}
                <strong className="font-medium text-foreground">{email}</strong>
              </p>

              {/* Development only */}
              {devLink && (
                <div className="mt-6 rounded-lg border border-warning/20 bg-warning/5 p-4 text-left">
                  <p className="mb-2 text-xs font-medium text-warning">
                    Development mode
                  </p>
                  <Button
                    onClick={handleDevLogin}
                    variant="outline"
                    className="w-full border-warning/40 bg-warning/15 text-warning hover:bg-warning/25"
                  >
                    Click here to login (dev only)
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted px-4">
      <div className="w-full max-w-md">
        <Card>
          <CardContent className="p-8">
            <Logo />

            <h1 className="text-center text-xl font-semibold text-foreground">
              Customer Portal
            </h1>
            <p className="mt-2 text-center text-sm text-muted-foreground">
              Enter your email to access your billing portal.
            </p>

            {error && (
              <div className="mt-6 rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                {error}
              </div>
            )}

            <form onSubmit={handleSubmit} className="mt-6 space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">Email address</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@company.com"
                  required
                />
              </div>

              <Button type="submit" disabled={loading} className="w-full">
                {loading ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Sending...
                  </>
                ) : (
                  <>
                    <Mail className="h-4 w-4" />
                    Send login link
                  </>
                )}
              </Button>
            </form>

            <p className="mt-6 text-center text-sm text-muted-foreground">
              We&apos;ll email you a magic link for password-free sign in.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default PortalLogin;
