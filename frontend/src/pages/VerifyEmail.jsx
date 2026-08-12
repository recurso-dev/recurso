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
  // Hold refreshUser in a ref. Its identity changes whenever AuthProvider
  // re-renders (e.g. the moment its /auth/me bootstrap resolves) — if the verify
  // effect depended on it, that re-render would re-run the effect mid-request,
  // its cleanup would flip `active` to false, and the in-flight verification
  // would resolve into a dead closure that never sets status → spinner forever.
  const refreshRef = useRef(refreshUser);
  refreshRef.current = refreshUser;

  const [status, setStatus] = useState(token ? "verifying" : "invalid");
  // Fire the single-use token exactly once (started), and gate state updates on
  // whether the component is still mounted (mounted) — NOT on a per-run flag, so
  // an effect re-run can never discard a result that's already in flight. This
  // also covers React 18 StrictMode's double-invoke of mount effects.
  const started = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  useEffect(() => {
    if (!token || started.current) return;
    started.current = true;
    (async () => {
      try {
        await endpoints.verifyEmail(token);
        if (mounted.current) setStatus("success");
        // Clear the banner for a logged-in user; harmless if unauthenticated.
        if (refreshRef.current) await refreshRef.current();
      } catch {
        if (mounted.current) setStatus("invalid");
      }
    })();
  }, [token]);

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-muted px-4 py-12">
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
                <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-success/5 text-success">
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
                <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-destructive/5 text-destructive">
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
