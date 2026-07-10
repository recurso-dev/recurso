import { useState } from "react";
import { Link } from "react-router-dom";
import { Layers, Mail, ArrowLeft, CheckCircle2 } from "lucide-react";

import { endpoints } from "@/lib/api";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";

export default function ForgotPassword() {
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (submitting) return;
    setSubmitting(true);
    try {
      // Always resolves 200 regardless of whether the account exists — the
      // server intentionally doesn't reveal account existence.
      await endpoints.forgotPassword(email);
    } catch {
      // Even on network error we show the generic success state so we never
      // leak whether an account exists. (A retry is available via the link.)
    } finally {
      setSubmitting(false);
      setSent(true);
    }
  };

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-stone-50 px-4 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
            <Layers className="h-6 w-6" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Reset your password
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {sent
              ? "Check your inbox for the next step."
              : "Enter your email and we'll send you a reset link."}
          </p>
        </div>

        <Card>
          <CardContent className="p-6">
            {sent ? (
              <div className="flex flex-col items-center text-center">
                <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
                  <CheckCircle2 className="h-6 w-6" />
                </div>
                <p className="text-sm text-foreground">
                  If that account exists, we've emailed a reset link. Follow it
                  to choose a new password.
                </p>
                <p className="mt-2 text-xs text-muted-foreground">
                  Didn't get anything? Check your spam folder or{" "}
                  <button
                    type="button"
                    onClick={() => setSent(false)}
                    className="font-medium text-primary hover:text-primary/80"
                  >
                    try again
                  </button>
                  .
                </p>
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="space-y-5">
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

                <Button type="submit" className="w-full" disabled={submitting}>
                  <Mail className="h-4 w-4" />
                  {submitting ? "Sending…" : "Send reset link"}
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
