import { useState } from "react";
import { Mail, X, Loader2 } from "lucide-react";

import { endpoints } from "@/lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { toast } from "@/components/ui/sonner";

// VerifyEmailBanner nudges a signed-in user with an unverified email to confirm
// it. It renders only when the current user object is explicitly unverified
// (email_verified === false) — so legacy API-key sessions (no user) and
// verified users never see it. Dismissal is session-local; the banner returns
// on next load until the email is actually verified.
export default function VerifyEmailBanner() {
  const { user } = useAuth() || {};
  const [dismissed, setDismissed] = useState(false);
  const [sending, setSending] = useState(false);

  if (!user || user.email_verified !== false || dismissed) return null;

  const resend = async () => {
    if (sending) return;
    setSending(true);
    try {
      await endpoints.resendVerification();
      toast.success("Verification email sent. Check your inbox.");
    } catch {
      toast.error("Couldn't send the email. Please try again.");
    } finally {
      setSending(false);
    }
  };

  return (
    <div
      role="status"
      className="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b border-warning/20 bg-warning/5 px-4 py-2 text-xs text-warning sm:px-6"
    >
      <Mail className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      <span className="min-w-0">
        <span className="font-semibold">Verify your email.</span>{" "}
        <span className="hidden sm:inline">
          We sent a confirmation link to{" "}
          <span className="font-medium">{user.email}</span>. Verify it to secure
          your account.
        </span>
      </span>
      <div className="ml-auto flex items-center gap-1">
        <button
          type="button"
          onClick={resend}
          disabled={sending}
          className="inline-flex items-center gap-1 rounded-md border border-warning/40 bg-white/60 px-2.5 py-1 font-medium text-warning transition-colors hover:bg-white disabled:opacity-60"
        >
          {sending && <Loader2 className="h-3 w-3 animate-spin" aria-hidden="true" />}
          {sending ? "Sending…" : "Resend email"}
        </button>
        <button
          type="button"
          onClick={() => setDismissed(true)}
          aria-label="Dismiss"
          className="flex h-6 w-6 items-center justify-center rounded-md text-warning transition-colors hover:bg-warning/15 hover:text-warning"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  );
}
