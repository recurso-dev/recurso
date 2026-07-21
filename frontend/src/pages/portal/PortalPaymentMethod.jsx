import { useEffect, useMemo, useState } from "react";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  PaymentElement,
  AddressElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";
import { Loader2, CheckCircle2 } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

// SetupForm confirms the SetupIntent (card entered in the Payment Element goes
// browser->Stripe directly), then finalizes server-side so the saved card
// becomes the customer's default. Must render inside <Elements>.
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
      const res = await fetch(`${apiBase}/portal/api/payment-method/confirm`, {
        method: "POST",
        credentials: "include",
        headers: { ...authHeaders, "Content-Type": "application/json" },
        body: JSON.stringify({ setup_intent_id: setupIntent.id }),
      });
      const body = await res.json().catch(() => ({}));
      if (res.ok && body.data?.status === "saved") {
        onSaved(body.data.card);
      } else {
        setError(
          body?.error?.message ||
            "Card was collected but couldn't be saved. Please try again."
        );
      }
    } else {
      setError("Card verification is still processing — please try again shortly.");
    }
    setSaving(false);
  };

  return (
    <form onSubmit={handleSubmit} className="mt-4 space-y-4">
      {/* Billing name + address, attached to the saved card via confirmSetup.
          Required so the stored payment method can later be charged off-session
          for India-export (foreign-currency) invoices. */}
      <AddressElement options={{ mode: "billing" }} />
      <PaymentElement />
      {error && (
        <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
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

// PortalPaymentMethod is the self-serve card-update dialog. On open it requests
// a SetupIntent, then mounts the Stripe Payment Element. PANs never touch
// Recurso.
export default function PortalPaymentMethod({
  open,
  onOpenChange,
  apiBase,
  authHeaders,
  onSaved,
}) {
  const [clientSecret, setClientSecret] = useState(null);
  const [publishableKey, setPublishableKey] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  // Deployment doesn't support self-serve card update (e.g. Razorpay-only —
  // no Stripe SetupIntent). Distinct from a transient error: nothing to retry.
  const [unavailable, setUnavailable] = useState(false);
  const [done, setDone] = useState(false);
  // UPI mandate re-authorization (ENG-5 Phase 3a): the alternative on
  // Razorpay deployments. On success we leave the page for Razorpay's hosted
  // authorization; activation lands via the token.confirmed webhook.
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
            "UPI re-authorization isn't available right now. Please contact the merchant."
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
    [publishableKey]
  );

  // When the dialog opens, reset and request a fresh SetupIntent.
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setClientSecret(null);
    setPublishableKey(null);
    setDone(false);
    setError(null);
    setUnavailable(false);
    setLoading(true);
    fetch(`${apiBase}/portal/api/payment-method/setup-intent`, {
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
        if (!res.ok) throw new Error(body?.error?.message || "Could not start card update");
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
  }, [open, apiBase]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Update payment method</DialogTitle>
          <DialogDescription>
            Your card is entered securely with Stripe — Recurso never sees your
            full card number.
          </DialogDescription>
        </DialogHeader>

        {done ? (
          <div className="flex flex-col items-center py-6 text-center">
            <CheckCircle2 className="mb-2 h-10 w-10 text-emerald-600" />
            <p className="text-sm text-stone-600">Your card has been updated.</p>
          </div>
        ) : unavailable ? (
          <div className="mt-4 space-y-3">
            <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-3 text-sm text-amber-800">
              Card update isn't available on this account. If you pay through
              UPI Autopay, you can re-authorize your mandate below — you'll be
              taken to a secure Razorpay page to approve it.
            </div>
            {mandateError && (
              <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                {mandateError}
              </div>
            )}
            <Button
              onClick={startMandateReauth}
              disabled={mandateLoading}
              className="w-full"
            >
              {mandateLoading && <Loader2 className="h-4 w-4 animate-spin" />}
              Re-authorize UPI Autopay
            </Button>
          </div>
        ) : error ? (
          <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {error}
          </div>
        ) : loading || !clientSecret || !stripePromise ? (
          <div className="flex justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-stone-400" />
          </div>
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
