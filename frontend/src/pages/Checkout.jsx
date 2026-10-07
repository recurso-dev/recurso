import { useState, useEffect, useMemo } from "react";
import { useParams } from "react-router";
import { CheckCircle2, AlertCircle, Clock } from "lucide-react";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  PaymentElement,
  AddressElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";

import { API_ROOT as API_BASE } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { Button } from "@/components/ui/button";

// PaymentForm renders the Stripe Payment Element and confirms the intent. It
// must live inside <Elements>. On a card success it settles inline; methods
// that need a bank redirect come back to this page's return_url, where the
// parent verifies via ?payment_intent.
function PaymentForm({ invoice, onPaid, onProcessing }) {
  const stripe = useStripe();
  const elements = useElements();
  const { id } = useParams();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  // Verify + settle server-side (the endpoint checks the intent succeeded and
  // that its metadata invoice_id matches this invoice before marking paid).
  const settle = async (paymentIntentId) => {
    const res = await fetch(
      `${API_BASE}/checkout/${id}/success?payment_intent=${encodeURIComponent(paymentIntentId)}`
    );
    const data = await res.json().catch(() => ({}));
    if (res.ok && data.data?.status === "paid") {
      onPaid();
    } else if (data.data?.status === "failed") {
      setError("Payment could not be completed. Please try another method.");
    } else {
      onProcessing();
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!stripe || !elements) return;
    setSubmitting(true);
    setError(null);

    // redirect: "if_required" keeps card payments on-page and only redirects
    // for methods that require it (some banks). return_url brings those back
    // here with ?payment_intent for verification.
    const { error: confirmError, paymentIntent } = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: `${window.location.origin}${window.location.pathname}`,
      },
      redirect: "if_required",
    });

    if (confirmError) {
      setError(confirmError.message || "Payment could not be completed.");
      setSubmitting(false);
      return;
    }

    if (paymentIntent?.status === "succeeded") {
      await settle(paymentIntent.id);
    } else if (paymentIntent?.status === "processing") {
      // ACH and other delayed-settlement methods: authorized, settles later
      // (the webhook marks the invoice paid once funds clear).
      onProcessing();
    } else {
      setError("Payment is not complete. Please try another method.");
    }
    setSubmitting(false);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Billing name + address. Required for India-export (foreign-currency)
          Stripe charges; confirmPayment automatically attaches this element's
          value to the payment method's billing_details. */}
      <AddressElement options={{ mode: "billing" }} />
      <PaymentElement />
      {error && (
        <div className="rounded-lg bg-destructive/5 px-3 py-3 text-sm text-destructive ring-1 ring-inset ring-destructive/20">
          {error}
        </div>
      )}
      <Button
        type="submit"
        disabled={!stripe || submitting}
        size="lg"
        className="w-full"
      >
        {submitting
          ? "Processing..."
          : `Pay ${formatCurrency(invoice.total, invoice.currency)}`}
      </Button>
    </form>
  );
}

export default function Checkout() {
  const { id } = useParams();
  const [invoice, setInvoice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [status, setStatus] = useState("open"); // open | paid | processing
  const [initiating, setInitiating] = useState(false);

  // Payment session (from POST /pay)
  const [clientSecret, setClientSecret] = useState(null);
  const [publishableKey, setPublishableKey] = useState(null);
  const [gateway, setGateway] = useState(null);
  const [rzpOrder, setRzpOrder] = useState(null); // Razorpay order details

  // Wompi direct card form state
  const [wompiCard, setWompiCard] = useState({
    number: "",
    cvc: "",
    exp_month: "",
    exp_year: "",
    card_holder: "",
    installments: 1,
    save_card: true,
  });
  const [wompiSubmitting, setWompiSubmitting] = useState(false);
  const [wompiError, setWompiError] = useState(null);
  const [wompiToken, setWompiToken] = useState(null);
  // ...
  // (router will also store publicKey etc. above — these are scoped here)
  // ...
  const wompiPromise = useMemo(() => {
    if (!publishableKey) return null;
    // We don't need to load Wompi.js — tokenization goes straight to Wompi's
    // /v1/tokens/cards endpoint from the browser using the publishable key.
    return Promise.resolve({ sandbox: publishableKey.includes("test") });
  }, [publishableKey]);

  // Tokenize the card with Wompi API (browser-side, using the public key so the
  // PAN never reaches Recurso). Mirrors how the Wompi hosted widget works
  // internally — same endpoint, same shape.
  const wompiTokenize = async (card) => {
    const base = publishableKey.includes("test")
      ? "https://sandbox.wompi.co/v1"
      : "https://production.wompi.co/v1";
    const res = await fetch(`${base}/tokens/cards`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${publishableKey}`,
      },
      body: JSON.stringify({
        number: card.number,
        cvc: card.cvc,
        exp_month: card.exp_month,
        exp_year: card.exp_year,
        card_holder: card.card_holder,
      }),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      const errs = body?.error?.messages || body?.error?.reason || {};
      const msg = Object.values(errs).flat().join(" ") || body?.error?.reason || "Tokenization failed";
      throw new Error(msg);
    }
    const data = await res.json();
    return data.data.id; // "tok_..."
  };

  const submitWompi = async (e) => {
    e.preventDefault();
    setWompiSubmitting(true);
    setWompiError(null);
    try {
      // 1. Tokenize the card client-side via Wompi API
      const tokenId = await wompiTokenize(wompiCard);
      setWompiToken(tokenId);

      // 2. POST to our backend — it talks to Wompi's /transactions
      const res = await fetch(`${API_BASE}/checkout/${id}/wompi/pay`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          card_token: tokenId,
          installments: Number(wompiCard.installments) || 1,
          save_payment_method: wompiCard.save_card,
        }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error?.message || "Payment failed");

      // Recurso marks the invoice paid via /v1/transactions response — when
      // Wompi returns APPROVED synchronously, the server returns the invoice
      // already in `paid` state and we jump straight to the success screen.
      if (data.data?.status === "paid") {
        setStatus("paid");
        return;
      }
      // PENDING: wait for the webhook. We poll once after a short delay.
      await new Promise((r) => setTimeout(r, 2500));
      const verify = await fetch(`${API_BASE}/checkout/${id}`);
      const verifyData = await verify.json();
      if (verifyData.data?.status === "paid") setStatus("paid");
      else if (verifyData.data?.status === "failed") {
        setError("Payment was declined by Wompi. Please try another card.");
      } else {
        setStatus("processing");
      }
    } catch (err) {
      setWompiError(err.message);
    } finally {
      setWompiSubmitting(false);
    }
  };

  // Load the invoice for display + paid check.
  useEffect(() => {
    fetch(`${API_BASE}/checkout/${id}`)
      .then((res) => {
        if (!res.ok) throw new Error("Invoice not found");
        return res.json();
      })
      .then((data) => {
        setInvoice(data.data);
        if (data.data.status === "paid") setStatus("paid");
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [id]);

  // Returning from a bank redirect: Stripe appends ?payment_intent — verify it.
  // A declined/abandoned intent comes back "failed" and must never show the
  // "we've received your payment" screen.
  useEffect(() => {
    const pi = new URLSearchParams(window.location.search).get("payment_intent");
    if (!pi) return;
    fetch(
      `${API_BASE}/checkout/${id}/success?payment_intent=${encodeURIComponent(pi)}`
    )
      .then((res) => res.json())
      .then((data) => {
        if (data.data?.status === "paid") setStatus("paid");
        else if (data.data?.status === "failed") setStatus("failed");
        else setStatus("processing");
      })
      .catch(() => {});
  }, [id]);

  // Clear the redirect params and payment session for a fresh attempt.
  const handleRetryPayment = () => {
    window.history.replaceState(null, "", window.location.pathname);
    setGateway(null);
    setClientSecret(null);
    setPublishableKey(null);
    setRzpOrder(null);
    setError(null);
    setStatus("open");
  };

  // Load Stripe.js only once we have the publishable key from /pay.
  const stripePromise = useMemo(
    () => (publishableKey ? loadStripe(publishableKey) : null),
    [publishableKey]
  );

  // Razorpay Checkout.js is loaded on demand (only for INR / Razorpay orders).
  const loadRazorpayScript = () =>
    new Promise((resolve) => {
      if (window.Razorpay) return resolve(true);
      const s = document.createElement("script");
      s.src = "https://checkout.razorpay.com/v1/checkout.js";
      s.onload = () => resolve(true);
      s.onerror = () => resolve(false);
      document.body.appendChild(s);
    });

  const openRazorpay = async (order) => {
    if (!order) return;
    const ok = await loadRazorpayScript();
    if (!ok || !window.Razorpay) {
      setError("Could not load the payment gateway. Please try again.");
      return;
    }
    const rzp = new window.Razorpay({
      key: order.razorpay_key_id,
      order_id: order.order_id,
      amount: order.amount,
      currency: order.currency,
      name: "Recurso",
      description: `Invoice ${invoice?.invoice_number || ""}`,
      theme: { color: "#10b981" },
      handler: async (resp) => {
        // Verify server-side (signature + order↔invoice bind) then settle.
        const vres = await fetch(`${API_BASE}/checkout/${id}/razorpay/verify`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            razorpay_order_id: resp.razorpay_order_id,
            razorpay_payment_id: resp.razorpay_payment_id,
            razorpay_signature: resp.razorpay_signature,
          }),
        });
        const body = await vres.json().catch(() => ({}));
        if (vres.ok && body.data?.status === "paid") {
          setStatus("paid");
        } else {
          setError(
            body?.error?.message ||
              "We couldn't confirm your payment. If you were charged, it will be reconciled shortly."
          );
        }
      },
    });
    rzp.on("payment.failed", (r) =>
      setError(r?.error?.description || "Payment failed. Please try again.")
    );
    rzp.open();
  };

  const handleInitiate = async () => {
    setInitiating(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/checkout/${id}/pay`, {
        method: "POST",
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error?.message || "Could not start payment");
      setGateway(data.data.gateway);
      setClientSecret(data.data.client_secret || null);
      setPublishableKey(data.data.publishable_key || null);
      if (data.data.gateway === "razorpay") {
        setRzpOrder(data.data);
        openRazorpay(data.data);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setInitiating(false);
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  if (error && !invoice) {
    return (
      <CheckoutShell>
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/5">
            <AlertCircle className="h-6 w-6 text-destructive" />
          </div>
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            Invoice not found
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">{error}</p>
        </div>
      </CheckoutShell>
    );
  }

  if (status === "paid") {
    return (
      <CheckoutShell>
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-success/5">
            <CheckCircle2 className="h-7 w-7 text-success" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Payment successful
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Invoice {invoice?.invoice_number} has been paid.
          </p>
          <p className="mt-4 text-xs text-subtle">You can close this page.</p>
        </div>
      </CheckoutShell>
    );
  }

  if (status === "failed") {
    return (
      <CheckoutShell>
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-destructive/5">
            <AlertCircle className="h-7 w-7 text-destructive" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Payment not completed
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Your payment for invoice {invoice?.invoice_number} was declined or
            cancelled. You have not been charged.
          </p>
          <Button onClick={handleRetryPayment} size="lg" className="mt-6 w-full">
            Try again
          </Button>
        </div>
      </CheckoutShell>
    );
  }

  if (status === "processing") {
    return (
      <CheckoutShell>
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-warning/5">
            <Clock className="h-7 w-7 text-warning" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Payment processing
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            We've received your payment for invoice {invoice?.invoice_number}.
            Bank payments (like ACH) take a few business days to clear — you'll
            get a receipt once it settles. No further action is needed.
          </p>
        </div>
      </CheckoutShell>
    );
  }

  const showStripeForm = gateway === "stripe" && clientSecret && stripePromise;

  return (
    <CheckoutShell>
      <div className="mb-6 text-center">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-success/50 text-2xl font-bold text-white">
          R
        </div>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">
          Checkout
        </h1>
      </div>

      <div className="mb-6 space-y-3 rounded-xl border border-border bg-muted p-4">
        <SummaryRow label="Invoice" value={invoice.invoice_number} strong />
        {invoice.subtotal !== invoice.total && (
          <>
            <SummaryRow
              label="Subtotal"
              value={formatCurrency(invoice.subtotal, invoice.currency)}
            />
            <SummaryRow
              label="Tax"
              value={formatCurrency(invoice.tax_amount, invoice.currency)}
            />
            <div className="border-t border-border pt-2" />
          </>
        )}
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Total</span>
          <span className="text-lg font-bold tabular-nums text-foreground">
            {formatCurrency(invoice.total, invoice.currency)}
          </span>
        </div>
        <SummaryRow label="Due date" value={invoice.due_date} />
      </div>

      {error && (
        <div className="mb-4 rounded-lg bg-destructive/5 px-3 py-3 text-sm text-destructive ring-1 ring-inset ring-destructive/20">
          {error}
        </div>
      )}

      {showStripeForm ? (
        <Elements stripe={stripePromise} options={{ clientSecret }}>
          <PaymentForm
            invoice={invoice}
            onPaid={() => setStatus("paid")}
            onProcessing={() => setStatus("processing")}
          />
        </Elements>
      ) : gateway === "stripe" ? (
        // gateway said "stripe" but the key or client_secret is missing —
        // showing the Pay button again would mint a new PaymentIntent per
        // click with no form ever appearing.
        <div className="rounded-lg bg-warning/5 px-3 py-3 text-sm text-warning ring-1 ring-inset ring-warning/20">
          This checkout isn't fully configured. Please contact the sender to
          arrange payment.
        </div>
      ) : gateway === "razorpay" ? (
        <div className="space-y-3">
          <Button
            onClick={() => openRazorpay(rzpOrder)}
            size="lg"
            className="w-full"
          >
            {`Pay ${formatCurrency(invoice.total, invoice.currency)}`}
          </Button>
          <p className="text-center text-xs text-subtle">
            A secure Razorpay window opens to complete your payment (UPI, cards,
            netbanking).
          </p>
        </div>
      ) : gateway === "wompi" ? (
        // Wompi direct card form. Tokenize via the Wompi public API from
        // the browser, then POST the token to Recurso's /wompi/pay endpoint
        // which finalises the charge through the private Wompi API. Card
        // details never leave the browser for the merchant.
        <form onSubmit={submitWompi} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="wompi-holder">Cardholder name</Label>
            <Input
              id="wompi-holder"
              value={wompiCard.card_holder}
              onChange={(e) =>
                setWompiCard((p) => ({ ...p, card_holder: e.target.value }))
              }
              placeholder="As appears on card"
              required
              className="font-mono"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="wompi-number">Card number</Label>
            <Input
              id="wompi-number"
              value={wompiCard.number}
              onChange={(e) => {
                const digits = e.target.value.replace(/\D/g, "").slice(0, 19);
                setWompiCard((p) => ({ ...p, number: digits }));
              }}
              placeholder="4242 4242 4242 4242"
              inputMode="numeric"
              autoComplete="cc-number"
              required
              className="font-mono"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="wompi-exp">Expiry (MM/YY)</Label>
              <Input
                id="wompi-exp"
                value={wompiCard.exp_month && wompiCard.exp_year
                  ? `${wompiCard.exp_month}/${wompiCard.exp_year}`
                  : ""}
                onChange={(e) => {
                  const v = e.target.value.replace(/\D/g, "").slice(0, 4);
                  const mm = v.slice(0, 2);
                  const yy = v.slice(2, 4);
                  setWompiCard((p) => ({
                    ...p,
                    exp_month: mm,
                    exp_year: yy,
                  }));
                }}
                placeholder="12/28"
                inputMode="numeric"
                autoComplete="cc-exp"
                required
                className="font-mono"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="wompi-cvc">CVC</Label>
              <Input
                id="wompi-cvc"
                type="password"
                value={wompiCard.cvc}
                onChange={(e) =>
                  setWompiCard((p) => ({ ...p, cvc: e.target.value.slice(0, 4) }))
                }
                placeholder="123"
                inputMode="numeric"
                autoComplete="cc-csc"
                required
                className="font-mono"
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="wompi-inst">Cuotas (1-36)</Label>
            <Input
              id="wompi-inst"
              type="number"
              min="1"
              max="36"
              value={wompiCard.installments}
              onChange={(e) =>
                setWompiCard((p) => ({ ...p, installments: Number(e.target.value) || 1 }))
              }
              className="font-mono"
            />
          </div>
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <input
              type="checkbox"
              checked={wompiCard.save_card}
              onChange={(e) =>
                setWompiCard((p) => ({ ...p, save_card: e.target.checked }))
              }
              className="h-4 w-4 rounded border-input text-primary focus-visible:ring-2 focus-visible:ring-ring"
            />
            Save this card for future invoices
          </label>
          {wompiError && (
            <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {wompiError}
            </div>
          )}
          <Button
            type="submit"
            disabled={wompiSubmitting}
            size="lg"
            className="w-full"
          >
            {wompiSubmitting
              ? "Processing…"
              : `Pay ${formatCurrency(invoice.total, invoice.currency)}`}
          </Button>
          <p className="text-center text-xs text-subtle">
            Card data is sent directly to Wompi for tokenization. Recurso never
            sees your card number or CVC.
          </p>
        </form>
      ) : gateway && gateway !== "stripe" ? (
        <div className="rounded-lg bg-warning/5 px-3 py-3 text-sm text-warning ring-1 ring-inset ring-warning/20">
          Self-serve checkout for {invoice.currency} isn't available here yet.
          Please contact the sender to arrange payment.
        </div>
      ) : (
        <Button
          onClick={handleInitiate}
          disabled={initiating}
          size="lg"
          className="w-full"
        >
          {initiating
            ? "Starting..."
            : `Pay ${formatCurrency(invoice.total, invoice.currency)}`}
        </Button>
      )}

      <p className="mt-4 text-center text-xs text-subtle">Powered by Recurso</p>
    </CheckoutShell>
  );
}

function CheckoutShell({ children }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted p-4">
      <div className="w-full max-w-md rounded-2xl border border-border bg-white p-8 shadow-sm">
        {children}
      </div>
    </div>
  );
}

function SummaryRow({ label, value, strong }) {
  return (
    <div className="flex justify-between">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span
        className={
          strong
            ? "text-sm font-semibold text-foreground"
            : "text-sm tabular-nums text-foreground"
        }
      >
        {value}
      </span>
    </div>
  );
}
