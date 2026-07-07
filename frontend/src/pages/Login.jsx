import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import axios from "axios";
import { Layers, KeyRound } from "lucide-react";

import { API_BASE } from "@/lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";

export default function Login() {
  const [key, setKey] = useState("");
  const [error, setError] = useState(null);
  const [checking, setChecking] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleLogin = async (e) => {
    e.preventDefault();
    if (!key || checking) return;
    setChecking(true);
    setError(null);
    try {
      // Validate the key before storing it so a typo fails here,
      // not on the first dashboard request.
      await axios.get(`${API_BASE}/account`, {
        headers: { Authorization: `Bearer ${key}` },
      });
      login(key);
      navigate("/");
    } catch (err) {
      if (err?.response?.status === 401 || err?.response?.status === 403) {
        setError("That API key was rejected. Check it and try again.");
      } else {
        setError("Could not reach the API to verify the key. Is the server running?");
      }
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
            Enter your API secret key to continue.
          </p>
        </div>

        <Card>
          <CardContent className="p-6">
            <form onSubmit={handleLogin} className="space-y-5">
              <FormField label="API secret key" htmlFor="apiKey" required>
                <Input
                  id="apiKey"
                  name="apiKey"
                  type="password"
                  required
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  placeholder="recurso_sk_..."
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
                {checking ? "Verifying…" : "Log in"}
              </Button>
            </form>
          </CardContent>
        </Card>

        <div className="mt-6 text-center">
          <p className="text-sm text-muted-foreground">
            Don't have a workspace?{" "}
            <Link to="/register" className="font-semibold text-primary hover:text-primary/80">
              Create new tenant
            </Link>
          </p>
          <p className="mt-3 text-xs text-zinc-400">
            Use the API key from your tenant registration.
          </p>
        </div>
      </div>
    </div>
  );
}
