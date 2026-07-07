import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import axios from "axios";
import { Layers, LogIn, KeyRound } from "lucide-react";

import { API_BASE } from "@/lib/api";
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

  const { login, loginWithApiKey } = useAuth();
  const navigate = useNavigate();

  const handleLogin = async (e) => {
    e.preventDefault();
    if (checking) return;
    setChecking(true);
    setError(null);
    try {
      await login(email, password);
      navigate("/");
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
            Log in to Recurso
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {apiKeyMode
              ? "Enter your tenant API secret key."
              : "Welcome back. Sign in to your workspace."}
          </p>
        </div>

        <Card>
          <CardContent className="p-6">
            {!apiKeyMode ? (
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
