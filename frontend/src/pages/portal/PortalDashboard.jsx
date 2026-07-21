import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import {
  Check,
  Copy,
  CreditCard,
  Download,
  FileText,
  Gift,
  Loader2,
  LogOut,
  MessageSquare,
  Receipt,
  Wallet,
} from "lucide-react";

import { API_ROOT as API_BASE } from "../../lib/api";
import PortalPaymentMethod from "./PortalPaymentMethod";
import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { StatCard } from "@/components/patterns/StatCard";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const invoiceStatusVariant = (status) =>
  ({
    paid: "success",
    open: "warning",
    past_due: "destructive",
    void: "neutral",
    draft: "neutral",
  })[status] || "neutral";

const PortalDashboard = () => {
  const [profile, setProfile] = useState(null);
  const [invoices, setInvoices] = useState([]);
  const [disputesByInvoice, setDisputesByInvoice] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [copied, setCopied] = useState(false);
  const navigate = useNavigate();

  // Payment-method dialog state (the SetupIntent flow lives in PortalPaymentMethod)
  const [pmOpen, setPmOpen] = useState(false);

  // Dispute dialog state
  const [disputeInvoice, setDisputeInvoice] = useState(null);
  const [disputeReason, setDisputeReason] = useState("");
  const [disputeSaving, setDisputeSaving] = useState(false);
  const [disputeError, setDisputeError] = useState(null);

  // The portal session lives in an httpOnly cookie the server set at login, so
  // it is invisible to JS (immune to XSS) — every request authenticates by
  // sending that cookie (credentials: "include"), never a token read from JS
  // storage. authHeaders now carries only the content type.
  const authHeaders = { "Content-Type": "application/json" };

  // Open the invoice PDF (ENG-152). Fetches the cookie-authed portal endpoint and
  // opens the rendered invoice in a new tab.
  const handleDownloadPdf = async (invoice) => {
    try {
      const res = await fetch(
        `${API_BASE}/portal/api/invoices/${invoice.id}/pdf`,
        { credentials: "include" }
      );
      if (!res.ok) throw new Error("Couldn't open the invoice. Please try again.");
      const html = await res.text();
      const url = URL.createObjectURL(new Blob([html], { type: "text/html" }));
      window.open(url, "_blank", "noopener");
      setTimeout(() => URL.revokeObjectURL(url), 60000);
    } catch (err) {
      setError(err.message);
    }
  };

  useEffect(() => {
    // The httpOnly session cookie can't be read from JS to pre-check login, so
    // we just load data; fetchData() redirects to /portal/login on a 401.
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navigate]);

  // Returning from a 3DS/bank redirect during card setup: Stripe appends
  // ?setup_intent=... to the return_url. Finalize server-side so the saved
  // card actually becomes the customer's default — without this the card
  // exists on Stripe but is never persisted, and dunning keeps retrying the
  // old one.
  useEffect(() => {
    const setupIntentId = new URLSearchParams(window.location.search).get(
      "setup_intent",
    );
    if (!setupIntentId) return;
    window.history.replaceState(null, "", window.location.pathname);
    fetch(`${API_BASE}/portal/api/payment-method/confirm`, {
      method: "POST",
      credentials: "include",
      headers: authHeaders,
      body: JSON.stringify({ setup_intent_id: setupIntentId }),
    })
      .then(async (res) => {
        const body = await res.json().catch(() => ({}));
        if (res.ok && body.data?.status === "saved") {
          toast.success("Your payment method has been updated.");
          fetchData();
        } else {
          toast.error(
            body?.error?.message ||
              "We couldn't confirm your new payment method. Please try again.",
          );
        }
      })
      .catch(() =>
        toast.error(
          "We couldn't confirm your new payment method. Please try again.",
        ),
      );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const fetchDisputes = async () => {
    const res = await fetch(`${API_BASE}/portal/api/disputes`, {
      credentials: "include",
      headers: authHeaders,
    });
    if (!res.ok) return;
    const data = await res.json();
    const map = {};
    // Backend returns newest first; keep the first (latest) per invoice.
    (data.data || []).forEach((d) => {
      if (!map[d.invoice_id]) map[d.invoice_id] = d;
    });
    setDisputesByInvoice(map);
  };

  const fetchData = async () => {
    try {
      // Fetch profile
      const profileRes = await fetch(`${API_BASE}/portal/api/profile`, {
        credentials: "include",
        headers: authHeaders,
      });
      if (!profileRes.ok) {
        if (profileRes.status === 401) {
          navigate("/portal/login");
          return;
        }
        throw new Error("Failed to fetch profile");
      }
      const profileData = await profileRes.json();
      setProfile(profileData);

      // Fetch invoices
      const invoicesRes = await fetch(`${API_BASE}/portal/api/invoices`, {
        credentials: "include",
        headers: authHeaders,
      });
      if (invoicesRes.ok) {
        const invoicesData = await invoicesRes.json();
        setInvoices(invoicesData.data || []);
      }

      await fetchDisputes();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      // The server clears the httpOnly session cookie on this call.
      await fetch(`${API_BASE}/portal/api/logout`, {
        method: "POST",
        credentials: "include",
      });
    } catch (err) {
      // Ignore errors
    }
    navigate("/portal/login");
  };

  const copyReferral = () => {
    if (!profile?.referral_code) return;
    navigator.clipboard.writeText(profile.referral_code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const submitDispute = async (e) => {
    e.preventDefault();
    if (!disputeInvoice) return;
    setDisputeSaving(true);
    setDisputeError(null);
    try {
      const res = await fetch(
        `${API_BASE}/portal/api/invoices/${disputeInvoice.id}/dispute`,
        {
          method: "POST",
          credentials: "include",
          headers: authHeaders,
          body: JSON.stringify({ reason: disputeReason.trim() }),
        },
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body?.error?.message || "Failed to raise dispute");
      }
      const body = await res.json();
      if (body?.data) {
        setDisputesByInvoice((prev) => ({
          ...prev,
          [disputeInvoice.id]: body.data,
        }));
      }
      setDisputeInvoice(null);
      setDisputeReason("");
    } catch (err) {
      setDisputeError(err.message);
    } finally {
      setDisputeSaving(false);
    }
  };

  const openDisputeDialog = (invoice) => {
    const existing = disputesByInvoice[invoice.id];
    setDisputeReason(existing?.reason || "");
    setDisputeError(null);
    setDisputeInvoice(invoice);
  };

  // Invoices carry total/amount_paid (minor units); there is no amount_due
  // field in the payload.
  const paidTotal = invoices
    .filter((inv) => inv.status === "paid")
    .reduce((acc, inv) => acc + (inv.total || 0), 0);
  const outstandingTotal = invoices
    .filter((inv) => inv.status !== "paid" && inv.status !== "void")
    .reduce((acc, inv) => acc + ((inv.total || 0) - (inv.amount_paid || 0)), 0);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-stone-50">
        <div className="text-center">
          <Loader2 className="mx-auto mb-4 h-8 w-8 animate-spin text-primary" />
          <p className="text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-stone-50">
      {/* Header */}
      <header className="border-b border-border bg-background">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-4 py-4">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
              R
            </div>
            <span className="text-lg font-semibold tracking-tight text-foreground">
              Recurso
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPmOpen(true)}
            >
              <CreditCard className="h-4 w-4" />
              Update payment method
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigate("/portal/redeem")}
            >
              <Gift className="h-4 w-4" />
              Redeem gift
            </Button>
            <Button variant="ghost" size="sm" onClick={handleLogout}>
              <LogOut className="h-4 w-4" />
              Sign out
            </Button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="mx-auto max-w-4xl px-4 py-8">
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">
          Billing Portal
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          View your invoices and manage your subscription.
        </p>

        {error && (
          <div className="mt-6 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}

        {/* Quick Stats */}
        <div className="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="Total Invoices"
            value={invoices.length.toLocaleString()}
            icon={Receipt}
          />
          <StatCard
            label="Total Paid"
            value={formatCurrency(paidTotal)}
            icon={FileText}
          />
          <StatCard
            label="Outstanding"
            value={formatCurrency(outstandingTotal)}
            icon={Wallet}
          />

          {/* Referral card */}
          <Card className="flex flex-col justify-between bg-primary/5 p-5 ring-1 ring-inset ring-primary/10">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wide text-primary">
                Refer &amp; earn
              </p>
              <Gift className="h-4 w-4 text-primary" />
            </div>
            {profile?.referral_code ? (
              <div className="mt-3">
                <div className="flex items-center gap-2 rounded-md border border-primary/20 bg-background px-2.5 py-1.5">
                  <code className="flex-1 truncate font-mono text-sm font-semibold text-foreground">
                    {profile.referral_code}
                  </code>
                  <button
                    type="button"
                    onClick={copyReferral}
                    className="rounded p-1 text-muted-foreground transition-colors hover:text-primary"
                    title="Copy code"
                    aria-label="Copy referral code"
                  >
                    {copied ? (
                      <Check className="h-4 w-4 text-primary" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                  </button>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  Share this code to earn credits.
                </p>
              </div>
            ) : (
              <p className="mt-3 text-sm text-muted-foreground">
                Generating code...
              </p>
            )}
          </Card>
        </div>

        {/* Invoices Table */}
        <Card className="mt-8">
          <CardHeader className="border-b border-border">
            <CardTitle className="text-base">Invoices</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {invoices.length === 0 ? (
              <p className="px-6 py-8 text-center text-sm text-muted-foreground">
                No invoices found.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="pl-6">Invoice</TableHead>
                    <TableHead>Date</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="pr-6 text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {invoices.map((invoice) => {
                    const dispute = disputesByInvoice[invoice.id];
                    return (
                      <TableRow key={invoice.id} className="hover:bg-muted/40">
                        <TableCell className="pl-6 font-medium text-foreground">
                          {invoice.id ? `${invoice.id.substring(0, 8)}…` : "—"}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(invoice.created_at)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums text-foreground">
                          {formatCurrency(invoice.total, invoice.currency)}
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-col gap-1">
                            <Badge
                              variant={invoiceStatusVariant(invoice.status)}
                              className="w-fit capitalize"
                            >
                              {(invoice.status || "unknown").replace("_", " ")}
                            </Badge>
                            {dispute && (
                              <Badge
                                variant={
                                  dispute.status === "resolved"
                                    ? "success"
                                    : "warning"
                                }
                                className="w-fit capitalize"
                                title={dispute.reason}
                              >
                                {dispute.status === "resolved"
                                  ? "Query resolved"
                                  : "Query open"}
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="pr-6 text-right">
                          <div className="flex items-center justify-end gap-3">
                            <button
                              type="button"
                              onClick={() => openDisputeDialog(invoice)}
                              disabled={dispute?.status === "open"}
                              className="inline-flex items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
                            >
                              <MessageSquare className="h-3.5 w-3.5" />
                              {dispute?.status === "open"
                                ? "Query raised"
                                : "Query invoice"}
                            </button>
                            <button
                              type="button"
                              onClick={() => handleDownloadPdf(invoice)}
                              className="inline-flex items-center gap-1 text-sm font-medium text-primary transition-colors hover:text-primary/80"
                            >
                              <Download className="h-3.5 w-3.5" />
                              PDF
                            </button>
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </main>

      {/* Update payment method — real Stripe SetupIntent flow (PANs never touch Recurso) */}
      <PortalPaymentMethod
        open={pmOpen}
        onOpenChange={setPmOpen}
        apiBase={API_BASE}
        authHeaders={authHeaders}
        onSaved={() => window.location.reload()}
      />

      {/* Raise dispute dialog */}
      <Dialog
        open={!!disputeInvoice}
        onOpenChange={(open) => {
          if (!open) setDisputeInvoice(null);
        }}
      >
        <DialogContent>
          <form onSubmit={submitDispute}>
            <DialogHeader>
              <DialogTitle>Query this invoice</DialogTitle>
              <DialogDescription>
                {disputeInvoice
                  ? `Raise a query or dispute on invoice ${disputeInvoice.id?.substring(
                      0,
                      8,
                    )}…. Our team will review and get back to you.`
                  : ""}
              </DialogDescription>
            </DialogHeader>
            <div className="mt-4 space-y-4">
              {disputeError && (
                <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                  {disputeError}
                </div>
              )}
              <div className="space-y-1.5">
                <Label htmlFor="dispute_reason">Reason</Label>
                <textarea
                  id="dispute_reason"
                  rows={4}
                  className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  placeholder="Tell us what looks wrong with this invoice…"
                  value={disputeReason}
                  onChange={(e) => setDisputeReason(e.target.value)}
                  required
                />
              </div>
            </div>
            <DialogFooter className="mt-6">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setDisputeInvoice(null)}
                disabled={disputeSaving}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={disputeSaving}>
                {disputeSaving && <Loader2 className="h-4 w-4 animate-spin" />}
                Submit query
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default PortalDashboard;
