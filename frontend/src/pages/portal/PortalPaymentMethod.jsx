import { useEffect, useMemo, useState } from "react";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  PaymentElement,
  AddressElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";
import { Loader2, CheckCircle2, CreditCard, Landmark } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

// finalize posts the confirmed SetupIntent id to the server, which saves the
// method as the customer's default. Shared by the card and bank flows.
async function finalize(apiBase, authHeaders, setupIntentID) {
  const res = await fetch(`${apiBase}/portal/api/payment-method/confirm`, {
    method: "POST",
    credentials: "include",
    headers: { ...authHeaders, "Content-Type": "application/json" },
    body: JSON.stringify({ setup_intent_id: setupIntentID }),
  });
  const body = await res.json().catch(() => ({}));
  if (res.ok && body.data?.status === "saved") return body.data.card;
  throw new Error(
    body?.error?.message || "The payment method was collected but couldn't be saved.",
  );
}

// SetupForm confirms a card SetupIntent (Payment Element → Stripe directly),
// then finalizes server-side. Must render inside <Elements>.
function SetupForm({ apiBase, authHeaders, onSaved }) {
  const stripe = useStripe();
  const elements = useElements();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!stripe || !elements) return;
    setSaving(true);
    setError(null);

    const { error: confirmError, setupIntent } = await stripe.confirmSetup({
      elements,
      confirmParams: { return_url: window.location.href },
      redirect: "if_required",
    });

    if (confirmError) {
      setError(confirmError.message || "Could not save the card.");
      setSaving(false);
      return;
    }

    if (setupIntent?.status === "succeeded") {
      try {
        onSaved(await finalize(apiBase, authHeaders, setupIntent.id));
      } catch (err) {
        setError(err.message);
      }
    } else {
      setError("Card verification is still processing — please try again shortly.");
    }
    setSaving(false);
  };

  return (
    <form onSubmit={handleSubmit} className="mt-4 space-y-4">
      {/* Billing name + address, attached to the saved card via confirmSetup, so
          it can be charged off-session later (e.g. India-export invoices). */}
      <AddressElement options={{ mode: "billing" }} />
      <PaymentElement />
      {error && (
        <div className="rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}
      <DialogFooter>
        <Button type="submit" disabled={!stripe || saving} className="w-full">
          {saving && <Loader2 className="h-4 w-4 animate-spin" />}
          Save card
        </Button>
      </DialogFooter>
    </form>
  );
}

// BankSetupForm runs the ACH flow: Stripe's Financial Connections modal collects
// + verifies the bank account (collectBankAccountForSetup), we confirm the
// mandate (confirmUsBankAccountSetup), then finalize server-side. Bank details
// go browser→Stripe directly — none reaches Recurso. (US Market Readiness, Inc 3a.)
function BankSetupForm({ stripePromise, clientSecret, apiBase, authHeaders, onSaved }) {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);

  const connect = async () => {
    if (!name.trim() || !email.trim()) {
      setError("Enter the account holder's name and email.");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const stripe = await stripePromise;
      if (!stripe) throw new Error("Payment library failed to load.");

      // Opens the Financial Connections bank-login modal (instant verification).
      const collected = await stripe.collectBankAccountForSetup({
        clientSecret,
        params: {
          payment_method_type: "us_bank_account",
          payment_method_data: {
            billing_details: { name: name.trim(), email: email.trim() },
          },
        },
        expand: ["payment_method"],
      });
      if (collected.error) throw new Error(collected.error.message || "Bank connection was cancelled.");

      let setupIntent = collected.setupIntent;
      // After collection Stripe requires an explicit mandate confirmation.
      if (setupIntent?.status === "requires_confirmation") {
        const confirmed = await stripe.confirmUsBankAccountSetup(clientSecret);
        if (confirmed.error) throw new Error(confirmed.error.message || "Could not authorize the bank debit.");
        setupIntent = confirmed.setupIntent;
      }

      if (setupIntent?.status === "succeeded") {
        onSaved(await finalize(apiBase, authHeaders, setupIntent.id));
      } else if (setupIntent?.status === "processing") {
        // Micro-deposit verification (not the instant path) — not supported yet.
        setError("This bank needs manual verification, which isn't supported yet. Please use a card.");
      } else {
        setError("Bank verification didn't complete — please try again.");
      }
    } catch (err) {
      setError(err.message);
    }
    setSaving(false);
  };

  return (
    <div className="mt-4 space-y-4">
      <div className="space-y-2">
        <Label htmlFor="ach-name">Account holder name</Label>
        <Input id="ach-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Jane Doe" />
      </div>
      <div className="space-y-2">
        <Label htmlFor="ach-email">Email</Label>
        <Input id="ach-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="jane@example.com" />
      </div>
      <p className="text-xs text-muted-foreground">
        You'll be taken to your bank to connect it securely — Recurso never sees your
        bank credentials. By connecting, you authorize ACH debits for your invoices.
      </p>
      {error && (
        <div className="rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}
      <DialogFooter>
        <Button onClick={connect} disabled={saving} className="w-full">
          {saving && <Loader2 className="h-4 w-4 animate-spin" />}
          Connect bank account
        </Button>
      </DialogFooter>
    </div>
  );
}

// PortalPaymentMethod is the self-serve payment-method dialog: card (Payment
// Element) or US bank / ACH (Financial Connections). PANs and bank credentials
// never touch Recurso.
export default function PortalPaymentMethod({
  open,
  onOpenChange,
  apiBase,
  authHeaders,
  onSaved,
}) {
  const [mode, setMode] = useState("card"); // 'card' | 'bank'
  const [clientSecret, setClientSecret] = useState(null);
  const [publishableKey, setPublishableKey] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [unavailable, setUnavailable] = useState(false);
  const [done, setDone] = useState(false);
  const [mandateLoading, setMandateLoading] = useState(false);
  const [mandateError, setMandateError] = useState(null);

  const startMandateReauth = async () => {
    setMandateLoading(true);
    setMandateError(null);
    try {
      const res = await fetch(`${apiBase}/portal/api/payment-method/mandate`, {
        method: "POST",
        credentials: "include",
        headers: authHeaders,
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok || !body.data?.auth_url) {
        throw new Error(
          body?.error?.message ||
            "UPI re-authorization isn't available right now. Please contact the merchant.",
        );
      }
      window.location.href = body.data.auth_url;
    } catch (err) {
      setMandateError(err.message);
      setMandateLoading(false);
    }
  };

  const stripePromise = useMemo(
    () => (publishableKey ? loadStripe(publishableKey) : null),
    [publishableKey],
  );

  // Request a fresh SetupIntent whenever the dialog opens or the method changes.
  // Card → /setup-intent (Payment Element); bank → /bank-setup-intent (ACH).
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setClientSecret(null);
    setPublishableKey(null);
    setDone(false);
    setError(null);
    setUnavailable(false);
    setLoading(true);
    const path = mode === "bank" ? "bank-setup-intent" : "setup-intent";
    fetch(`${apiBase}/portal/api/payment-method/${path}`, {
      method: "POST",
      credentials: "include",
      headers: authHeaders,
    })
      .then(async (res) => {
        const body = await res.json().catch(() => ({}));
        if (res.status === 503) {
          setUnavailable(true);
          return null;
        }
        if (!res.ok) throw new Error(body?.error?.message || "Could not start setup");
        return body;
      })
      .then((body) => {
        if (cancelled || !body) return;
        setClientSecret(body.data.client_secret);
        setPublishableKey(body.data.publishable_key);
      })
      .catch((err) => !cancelled && setError(err.message))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [open, mode, apiBase]); // eslint-disable-line react-hooks/exhaustive-deps

  const savedMessage = mode === "bank" ? "Your bank account has been saved." : "Your card has been updated.";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Update payment method</DialogTitle>
          <DialogDescription>
            Entered securely with Stripe — Recurso never sees your full card
            number or bank credentials.
          </DialogDescription>
        </DialogHeader>

        {/* Method selector (hidden once done / on the mandate-only fallback). */}
        {!done && !unavailable && (
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={() => setMode("card")}
              className={`flex items-center justify-center gap-2 rounded-md border px-3 py-2 text-sm font-medium transition-colors ${
                mode === "card"
                  ? "border-primary bg-success/5 text-success"
                  : "border-border text-muted-foreground hover:bg-muted"
              }`}
            >
              <CreditCard className="h-4 w-4" /> Card
            </button>
            <button
              type="button"
              onClick={() => setMode("bank")}
              className={`flex items-center justify-center gap-2 rounded-md border px-3 py-2 text-sm font-medium transition-colors ${
                mode === "bank"
                  ? "border-primary bg-success/5 text-success"
                  : "border-border text-muted-foreground hover:bg-muted"
              }`}
            >
              <Landmark className="h-4 w-4" /> US bank (ACH)
            </button>
          </div>
        )}

        {done ? (
          <div className="flex flex-col items-center py-6 text-center">
            <CheckCircle2 className="mb-2 h-10 w-10 text-success" />
            <p className="text-sm text-muted-foreground">{savedMessage}</p>
          </div>
        ) : unavailable ? (
          <div className="mt-4 space-y-3">
            <div className="rounded-md border border-warning/20 bg-warning/5 px-3 py-3 text-sm text-warning">
              Self-serve setup isn't available on this account. If you pay through
              UPI Autopay, you can re-authorize your mandate below — you'll be
              taken to a secure Razorpay page to approve it.
            </div>
            {mandateError && (
              <div className="rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {mandateError}
              </div>
            )}
            <Button onClick={startMandateReauth} disabled={mandateLoading} className="w-full">
              {mandateLoading && <Loader2 className="h-4 w-4 animate-spin" />}
              Re-authorize UPI Autopay
            </Button>
          </div>
        ) : error ? (
          <div className="mt-4 rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive">
            {error}
          </div>
        ) : loading || !clientSecret || !stripePromise ? (
          <div className="flex justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-subtle" />
          </div>
        ) : mode === "bank" ? (
          <BankSetupForm
            stripePromise={stripePromise}
            clientSecret={clientSecret}
            apiBase={apiBase}
            authHeaders={authHeaders}
            onSaved={(card) => {
              setDone(true);
              onSaved?.(card);
            }}
          />
        ) : (
          <Elements stripe={stripePromise} options={{ clientSecret }}>
            <SetupForm
              apiBase={apiBase}
              authHeaders={authHeaders}
              onSaved={(card) => {
                setDone(true);
                onSaved?.(card);
              }}
            />
          </Elements>
        )}
      </DialogContent>
    </Dialog>
  );
}
