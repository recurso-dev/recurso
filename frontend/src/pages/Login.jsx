import { useState, useEffect } from "react";
import { useNavigate, Link, useSearchParams } from "react-router-dom";
import axios from "axios";
import { Layers, LogIn, KeyRound, ShieldCheck, ArrowLeft } from "lucide-react";

import { API_BASE, API_ROOT, endpoints } from "@/lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(null);
  const [checking, setChecking] = useState(false);

  const [apiKeyMode, setApiKeyMode] = useState(false);
  const [key, setKey] = useState("");

  // Two-step (MFA) login state. When the server responds with mfa_required we
  // stash the short-lived token and swap the form to a code-entry step.
  const [mfaToken, setMfaToken] = useState(null);
  const [code, setCode] = useState("");

  // Enabled OAuth providers (Google/GitHub) — only shown if the server has
  // them configured. The buttons full-page-redirect to the start endpoint.
  const [providers, setProviders] = useState([]);
  const [searchParams] = useSearchParams();

  const { login, loginMfa, loginWithApiKey } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    endpoints
      .getOAuthProviders()
      .then((res) => setProviders((res.data?.providers || []).filter((p) => p.enabled)))
      .catch(() => setProviders([]));
    if (searchParams.get("error") === "oauth") {
      setError("Social sign-in failed. Please try again or use your email.");
    }
  }, [searchParams]);

  const handleLogin = async (e) => {
    e.preventDefault();
    if (checking) return;
    setChecking(true);
    setError(null);
    try {
      const res = await login(email, password);
      if (res?.mfa_required) {
        setMfaToken(res.mfa_token);
        setCode("");
      } else {
        navigate("/");
      }
    } catch (err) {
      setError(
        err?.response?.data?.error?.message === "invalid credentials" ||
          err?.response?.status === 401
          ? "Incorrect email or password."
          : "Could not reach the API. Is the server running?"
      );
    } finally {
      setChecking(false);
    }
  };

  const handleMfa = async (e) => {
    e.preventDefault();
    if (checking) return;
    setChecking(true);
    setError(null);
    try {
      await loginMfa(mfaToken, code);
      navigate("/");
    } catch (err) {
      setError(
        err?.response?.status === 401
          ? "That code is incorrect or expired."
          : "Could not verify the code. Please try again."
      );
    } finally {
      setChecking(false);
    }
  };

  const handleApiKeyLogin = async (e) => {
    e.preventDefault();
    if (!key || checking) return;
    setChecking(true);
    setError(null);
    try {
      await axios.get(`${API_BASE}/account`, {
        headers: { Authorization: `Bearer ${key}` },
      });
      loginWithApiKey(key);
      navigate("/");
    } catch (err) {
      setError(
        err?.response?.status === 401 || err?.response?.status === 403
          ? "That API key was rejected."
          : "Could not reach the API to verify the key."
      );
    } finally {
      setChecking(false);
    }
  };

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-zinc-50 px-4 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
            <Layers className="h-6 w-6" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            {mfaToken ? "Two-factor authentication" : "Log in to Recurso"}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {mfaToken
              ? "Enter the 6-digit code from your authenticator app."
              : apiKeyMode
              ? "Enter your tenant API secret key."
              : "Welcome back. Sign in to your workspace."}
          </p>
        </div>

        <Card>
          <CardContent className="p-6">
            {mfaToken ? (
              <form onSubmit={handleMfa} className="space-y-5">
                <FormField label="Authentication code" htmlFor="mfa-code" required>
                  <Input
                    id="mfa-code"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    maxLength={8}
                    autoFocus
                    required
                    value={code}
                    onChange={(e) =>
                      setCode(e.target.value.replace(/[^0-9]/g, ""))
                    }
                    placeholder="123456"
                    className="text-center font-mono text-lg tracking-[0.4em]"
                  />
                </FormField>

                {error && (
                  <p className="text-sm font-medium text-red-600" role="alert">
                    {error}
                  </p>
                )}

                <Button
                  type="submit"
                  className="w-full"
                  disabled={checking || code.length < 6}
                >
                  <ShieldCheck className="h-4 w-4" />
                  {checking ? "Verifying…" : "Verify & continue"}
                </Button>

                <button
                  type="button"
                  onClick={() => {
                    setMfaToken(null);
                    setCode("");
                    setError(null);
                  }}
                  className="flex w-full items-center justify-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                >
                  <ArrowLeft className="h-3 w-3" />
                  Back to login
                </button>
              </form>
            ) : !apiKeyMode ? (
              <>
              <form onSubmit={handleLogin} className="space-y-5">
                <FormField label="Email" htmlFor="email" required>
                  <Input
                    id="email"
                    type="email"
                    autoComplete="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="you@company.com"
                  />
                </FormField>
                <FormField label="Password" htmlFor="password" required>
                  <Input
                    id="password"
                    type="password"
                    autoComplete="current-password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••"
                  />
                </FormField>

                <div className="-mt-2 text-right">
                  <Link
                    to="/forgot-password"
                    className="text-xs font-medium text-primary hover:text-primary/80"
                  >
                    Forgot password?
                  </Link>
                </div>

                {error && (
                  <p className="text-sm font-medium text-red-600" role="alert">
                    {error}
                  </p>
                )}

                <Button type="submit" className="w-full" disabled={checking}>
                  <LogIn className="h-4 w-4" />
                  {checking ? "Signing in…" : "Sign in"}
                </Button>
              </form>

              {providers.length > 0 && (
                <div className="mt-5">
                  <div className="relative my-4">
                    <div className="absolute inset-0 flex items-center">
                      <span className="w-full border-t border-border" />
                    </div>
                    <div className="relative flex justify-center text-xs">
                      <span className="bg-card px-2 text-muted-foreground">
                        or continue with
                      </span>
                    </div>
                  </div>
                  <div className="space-y-2">
                    {providers.map((p) => (
                      <a
                        key={p.name}
                        href={`${API_ROOT}/auth/oauth/${p.name}/start`}
                        className="flex w-full items-center justify-center gap-2 rounded-lg border border-input bg-white px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
                      >
                        Continue with{" "}
                        {p.name.charAt(0).toUpperCase() + p.name.slice(1)}
                      </a>
                    ))}
                  </div>
                </div>
              )}
              </>
            ) : (
              <form onSubmit={handleApiKeyLogin} className="space-y-5">
                <FormField label="API secret key" htmlFor="apiKey" required>
                  <Input
                    id="apiKey"
                    type="password"
                    required
                    value={key}
                    onChange={(e) => setKey(e.target.value)}
                    placeholder="sk_..."
                    className="font-mono"
                  />
                </FormField>

                {error && (
                  <p className="text-sm font-medium text-red-600" role="alert">
                    {error}
                  </p>
                )}

                <Button type="submit" className="w-full" disabled={checking}>
                  <KeyRound className="h-4 w-4" />
                  {checking ? "Verifying…" : "Log in with API key"}
                </Button>
              </form>
            )}

            {!mfaToken && (
            <button
              type="button"
              onClick={() => {
                setApiKeyMode((m) => !m);
                setError(null);
              }}
              className="mt-4 w-full text-center text-xs text-muted-foreground hover:text-foreground"
            >
              {apiKeyMode ? "← Back to email login" : "Log in with an API key instead"}
            </button>
            )}
          </CardContent>
        </Card>

        <div className="mt-6 text-center">
          <p className="text-sm text-muted-foreground">
            Don't have a workspace?{" "}
            <Link to="/register" className="font-semibold text-primary hover:text-primary/80">
              Create one
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
