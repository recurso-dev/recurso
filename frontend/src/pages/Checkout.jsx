import { useState, useEffect, useMemo } from "react";
import { useParams } from "react-router-dom";
import { CheckCircle2, AlertCircle, Clock } from "lucide-react";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  PaymentElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";

import { API_ROOT as API_BASE } from "../lib/api";
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
      <PaymentElement />
      {error && (
        <div className="rounded-lg bg-red-50 px-3 py-3 text-sm text-red-700 ring-1 ring-inset ring-red-600/20">
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
          : `Pay ${invoice.currency} ${invoice.display_amount}`}
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
  useEffect(() => {
    const pi = new URLSearchParams(window.location.search).get("payment_intent");
    if (!pi) return;
    fetch(
      `${API_BASE}/checkout/${id}/success?payment_intent=${encodeURIComponent(pi)}`
    )
      .then((res) => res.json())
      .then((data) => {
        if (data.data?.status === "paid") setStatus("paid");
        else setStatus("processing");
      })
      .catch(() => {});
  }, [id]);

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
      <div className="flex min-h-screen items-center justify-center bg-zinc-50">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-emerald-500 border-t-transparent" />
      </div>
    );
  }

  if (error && !invoice) {
    return (
      <CheckoutShell>
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-50">
            <AlertCircle className="h-6 w-6 text-red-500" />
          </div>
          <h1 className="text-xl font-semibold tracking-tight text-zinc-900">
            Invoice not found
          </h1>
          <p className="mt-1 text-sm text-zinc-500">{error}</p>
        </div>
      </CheckoutShell>
    );
  }

  if (status === "paid") {
    return (
      <CheckoutShell>
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-emerald-50">
            <CheckCircle2 className="h-7 w-7 text-emerald-600" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">
            Payment successful
          </h1>
          <p className="mt-2 text-sm text-zinc-500">
            Invoice {invoice?.invoice_number} has been paid.
          </p>
          <p className="mt-4 text-xs text-zinc-400">You can close this page.</p>
        </div>
      </CheckoutShell>
    );
  }

  if (status === "processing") {
    return (
      <CheckoutShell>
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-amber-50">
            <Clock className="h-7 w-7 text-amber-500" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">
            Payment processing
          </h1>
          <p className="mt-2 text-sm text-zinc-500">
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
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500 text-2xl font-bold text-white">
          R
        </div>
        <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">
          Checkout
        </h1>
      </div>

      <div className="mb-6 space-y-3 rounded-xl border border-zinc-200 bg-zinc-50 p-4">
        <SummaryRow label="Invoice" value={invoice.invoice_number} strong />
        {invoice.subtotal !== invoice.total && (
          <>
            <SummaryRow
              label="Subtotal"
              value={`${invoice.currency} ${(invoice.subtotal / 100).toFixed(2)}`}
            />
            <SummaryRow
              label="Tax"
              value={`${invoice.currency} ${(invoice.tax_amount / 100).toFixed(2)}`}
            />
            <div className="border-t border-zinc-200 pt-2" />
          </>
        )}
        <div className="flex items-center justify-between">
          <span className="text-sm text-zinc-500">Total</span>
          <span className="text-lg font-bold tabular-nums text-zinc-900">
            {invoice.currency} {invoice.display_amount}
          </span>
        </div>
        <SummaryRow label="Due date" value={invoice.due_date} />
      </div>

      {error && (
        <div className="mb-4 rounded-lg bg-red-50 px-3 py-3 text-sm text-red-700 ring-1 ring-inset ring-red-600/20">
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
      ) : gateway === "razorpay" ? (
        <div className="space-y-3">
          <Button
            onClick={() => openRazorpay(rzpOrder)}
            size="lg"
            className="w-full"
          >
            {`Pay ${invoice.currency} ${invoice.display_amount}`}
          </Button>
          <p className="text-center text-xs text-zinc-400">
            A secure Razorpay window opens to complete your payment (UPI, cards,
            netbanking).
          </p>
        </div>
      ) : gateway && gateway !== "stripe" ? (
        <div className="rounded-lg bg-amber-50 px-3 py-3 text-sm text-amber-800 ring-1 ring-inset ring-amber-600/20">
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
            : `Pay ${invoice.currency} ${invoice.display_amount}`}
        </Button>
      )}

      <p className="mt-4 text-center text-xs text-zinc-400">Powered by Recurso</p>
    </CheckoutShell>
  );
}

function CheckoutShell({ children }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4">
      <div className="w-full max-w-md rounded-2xl border border-zinc-200 bg-white p-8 shadow-sm">
        {children}
      </div>
    </div>
  );
}

function SummaryRow({ label, value, strong }) {
  return (
    <div className="flex justify-between">
      <span className="text-sm text-zinc-500">{label}</span>
      <span
        className={
          strong
            ? "text-sm font-semibold text-zinc-900"
            : "text-sm tabular-nums text-zinc-900"
        }
      >
        {value}
      </span>
    </div>
  );
}
