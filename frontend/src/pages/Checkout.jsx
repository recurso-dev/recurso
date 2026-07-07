import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { CheckCircle2, AlertCircle } from "lucide-react";

import { API_ROOT as API_BASE } from "../lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

export default function Checkout() {
  const { id } = useParams();
  const [invoice, setInvoice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [paying, setPaying] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    fetch(`${API_BASE}/checkout/${id}`)
      .then((res) => {
        if (!res.ok) throw new Error("Invoice not found");
        return res.json();
      })
      .then((data) => {
        setInvoice(data.data);
        if (data.data.status === "paid") {
          setSuccess(true);
        }
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [id]);

  const handlePay = async () => {
    setPaying(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/checkout/${id}/pay`, { method: "POST" });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error?.message || "Payment failed");
      }
      await res.json();

      // Mark as success via the success endpoint
      const successRes = await fetch(`${API_BASE}/checkout/${id}/success`);
      if (successRes.ok) {
        setSuccess(true);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setPaying(false);
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
      <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4">
        <div className="w-full max-w-md rounded-2xl border border-zinc-200 bg-white p-8 text-center shadow-sm">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-50">
            <AlertCircle className="h-6 w-6 text-red-500" />
          </div>
          <h1 className="text-xl font-semibold tracking-tight text-zinc-900">
            Invoice not found
          </h1>
          <p className="mt-1 text-sm text-zinc-500">{error}</p>
        </div>
      </div>
    );
  }

  if (success) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4">
        <div className="w-full max-w-md rounded-2xl border border-zinc-200 bg-white p-8 text-center shadow-sm">
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
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4">
      <div className="w-full max-w-md rounded-2xl border border-zinc-200 bg-white p-8 shadow-sm">
        <div className="mb-6 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500 text-2xl font-bold text-white">
            R
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">Checkout</h1>
        </div>

        <div className="mb-6 space-y-3 rounded-xl border border-zinc-200 bg-zinc-50 p-4">
          <div className="flex justify-between">
            <span className="text-sm text-zinc-500">Invoice</span>
            <span className="text-sm font-semibold text-zinc-900">
              {invoice.invoice_number}
            </span>
          </div>
          {invoice.subtotal !== invoice.total && (
            <>
              <div className="flex justify-between">
                <span className="text-sm text-zinc-500">Subtotal</span>
                <span className="text-sm tabular-nums text-zinc-900">
                  {invoice.currency} {(invoice.subtotal / 100).toFixed(2)}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-zinc-500">Tax</span>
                <span className="text-sm tabular-nums text-zinc-900">
                  {invoice.currency} {(invoice.tax_amount / 100).toFixed(2)}
                </span>
              </div>
              <div className="border-t border-zinc-200 pt-2" />
            </>
          )}
          <div className="flex items-center justify-between">
            <span className="text-sm text-zinc-500">Total</span>
            <span className="text-lg font-bold tabular-nums text-zinc-900">
              {invoice.currency} {invoice.display_amount}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-sm text-zinc-500">Due date</span>
            <span className="text-sm text-zinc-900">{invoice.due_date}</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-zinc-500">Status</span>
            <Badge variant="warning" className="uppercase">
              {invoice.status}
            </Badge>
          </div>
        </div>

        {error && (
          <div className="mb-4 rounded-lg bg-red-50 px-3 py-3 text-sm text-red-700 ring-1 ring-inset ring-red-600/20">
            {error}
          </div>
        )}

        <Button onClick={handlePay} disabled={paying} size="lg" className="w-full">
          {paying ? "Processing..." : `Pay ${invoice.currency} ${invoice.display_amount}`}
        </Button>

        <p className="mt-4 text-center text-xs text-zinc-400">Powered by Recurso</p>
      </div>
    </div>
  );
}
