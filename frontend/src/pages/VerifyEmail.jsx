import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { Layers, MailCheck, AlertTriangle, ArrowLeft, Loader2 } from "lucide-react";

import { endpoints } from "@/lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

// VerifyEmail consumes the ?token= from the emailed link, confirms the address,
// and (if the user is logged in) refreshes auth state so the verify banner
// clears. States: verifying → success | invalid.
export default function VerifyEmail() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token") || "";
  const navigate = useNavigate();
  const { refreshUser } = useAuth() || {};

  const [status, setStatus] = useState(token ? "verifying" : "invalid");
  // Guard against React 18 StrictMode double-invoking the effect (which would
  // fire the single-use token twice and flip a valid verification to "invalid").
  const ran = useRef(false);

  useEffect(() => {
    if (!token || ran.current) return;
    ran.current = true;
    let active = true;
    (async () => {
      try {
        await endpoints.verifyEmail(token);
        if (!active) return;
        setStatus("success");
        // Clear the banner for a logged-in user; harmless if unauthenticated.
        if (refreshUser) await refreshUser();
      } catch {
        if (active) setStatus("invalid");
      }
    })();
    return () => {
      active = false;
    };
  }, [token, refreshUser]);

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-stone-50 px-4 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
            <Layers className="h-6 w-6" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Email verification
          </h1>
        </div>

        <Card>
          <CardContent className="p-6">
            {status === "verifying" && (
              <div
                className="flex flex-col items-center text-center"
                role="status"
                aria-live="polite"
              >
                <Loader2 className="mb-4 h-6 w-6 animate-spin text-muted-foreground" />
                <p className="text-sm text-foreground">Confirming your email…</p>
              </div>
            )}

            {status === "success" && (
              <div className="flex flex-col items-center text-center" role="status">
                <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
                  <MailCheck className="h-6 w-6" />
                </div>
                <p className="text-sm font-medium text-foreground">
                  Your email is verified.
                </p>
                <p className="mt-2 text-xs text-muted-foreground">
                  Thanks — your account is fully set up.
                </p>
                <Button className="mt-5 w-full" onClick={() => navigate("/")}>
                  Go to dashboard
                </Button>
              </div>
            )}

            {status === "invalid" && (
              <div className="flex flex-col items-center text-center" role="alert">
                <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-red-50 text-red-600">
                  <AlertTriangle className="h-6 w-6" />
                </div>
                <p className="text-sm text-foreground">
                  This verification link is invalid or has expired.
                </p>
                <p className="mt-2 text-xs text-muted-foreground">
                  Sign in and use “Resend verification” to get a fresh link.
                </p>
                <Button
                  variant="outline"
                  className="mt-5 w-full"
                  onClick={() => navigate("/login")}
                >
                  Go to login
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        <div className="mt-6 text-center">
          <Link
            to="/login"
            className="inline-flex items-center gap-1 text-sm font-medium text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to login
          </Link>
        </div>
      </div>
    </div>
  );
}
