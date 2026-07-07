import { useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { Layers, KeyRound, ArrowLeft, AlertTriangle } from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";

export default function ResetPassword() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token") || "";
  const navigate = useNavigate();

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [invalidToken, setInvalidToken] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (submitting) return;
    setError(null);

    if (password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    if (password !== confirm) {
      setError("Passwords don't match.");
      return;
    }

    setSubmitting(true);
    try {
      await endpoints.resetPassword(token, password);
      toast.success("Password reset. You can now sign in.");
      navigate("/login");
    } catch (err) {
      if (err?.response?.status === 400) {
        setInvalidToken(true);
      } else {
        setError("Could not reset your password. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  const missingOrInvalid = !token || invalidToken;

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-zinc-50 px-4 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
            <Layers className="h-6 w-6" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Choose a new password
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Pick something secure you haven't used before.
          </p>
        </div>

        <Card>
          <CardContent className="p-6">
            {missingOrInvalid ? (
              <div className="flex flex-col items-center text-center">
                <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-red-50 text-red-600">
                  <AlertTriangle className="h-6 w-6" />
                </div>
                <p className="text-sm text-foreground">
                  This reset link is invalid or has expired.
                </p>
                <p className="mt-2 text-xs text-muted-foreground">
                  Request a new one from the{" "}
                  <Link
                    to="/forgot-password"
                    className="font-medium text-primary hover:text-primary/80"
                  >
                    forgot password
                  </Link>{" "}
                  page.
                </p>
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="space-y-5">
                <FormField
                  label="New password"
                  htmlFor="password"
                  required
                  description="At least 8 characters."
                >
                  <Input
                    id="password"
                    type="password"
                    autoComplete="new-password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••"
                  />
                </FormField>
                <FormField label="Confirm password" htmlFor="confirm" required>
                  <Input
                    id="confirm"
                    type="password"
                    autoComplete="new-password"
                    required
                    value={confirm}
                    onChange={(e) => setConfirm(e.target.value)}
                    placeholder="••••••••"
                  />
                </FormField>

                {error && (
                  <p className="text-sm font-medium text-red-600" role="alert">
                    {error}
                  </p>
                )}

                <Button type="submit" className="w-full" disabled={submitting}>
                  <KeyRound className="h-4 w-4" />
                  {submitting ? "Saving…" : "Reset password"}
                </Button>
              </form>
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
