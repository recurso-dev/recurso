import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  Check,
  Copy,
  Download,
  FileText,
  Gift,
  Loader2,
  LogOut,
  Receipt,
  Wallet,
} from "lucide-react";

import { API_ROOT as API_BASE } from "../../lib/api";
import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatCard } from "@/components/patterns/StatCard";
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [copied, setCopied] = useState(false);
  const navigate = useNavigate();

  const sessionToken = localStorage.getItem("portal_session");

  useEffect(() => {
    if (!sessionToken) {
      navigate("/portal/login");
      return;
    }
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionToken, navigate]);

  const fetchData = async () => {
    try {
      const headers = {
        "X-Portal-Session": sessionToken,
        "Content-Type": "application/json",
      };

      // Fetch profile
      const profileRes = await fetch(`${API_BASE}/portal/api/profile`, {
        headers,
      });
      if (!profileRes.ok) {
        if (profileRes.status === 401) {
          localStorage.removeItem("portal_session");
          navigate("/portal/login");
          return;
        }
        throw new Error("Failed to fetch profile");
      }
      const profileData = await profileRes.json();
      setProfile(profileData);

      // Fetch invoices
      const invoicesRes = await fetch(`${API_BASE}/portal/api/invoices`, {
        headers,
      });
      if (invoicesRes.ok) {
        const invoicesData = await invoicesRes.json();
        setInvoices(invoicesData.data || []);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      await fetch(`${API_BASE}/portal/api/logout`, {
        method: "POST",
        headers: { "X-Portal-Session": sessionToken },
      });
    } catch (err) {
      // Ignore errors
    }
    localStorage.removeItem("portal_session");
    navigate("/portal/login");
  };

  const copyReferral = () => {
    if (!profile?.referral_code) return;
    navigator.clipboard.writeText(profile.referral_code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const paidTotal = invoices
    .filter((inv) => inv.status === "paid")
    .reduce((acc, inv) => acc + (inv.amount_due || 0), 0);
  const outstandingTotal = invoices
    .filter((inv) => inv.status !== "paid")
    .reduce((acc, inv) => acc + (inv.amount_due || 0), 0);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-50">
        <div className="text-center">
          <Loader2 className="mx-auto mb-4 h-8 w-8 animate-spin text-primary" />
          <p className="text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-zinc-50">
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
                  {invoices.map((invoice) => (
                    <TableRow key={invoice.id} className="hover:bg-muted/40">
                      <TableCell className="pl-6 font-medium text-foreground">
                        {invoice.id ? `${invoice.id.substring(0, 8)}…` : "—"}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(invoice.created_at)}
                      </TableCell>
                      <TableCell className="text-right tabular-nums text-foreground">
                        {formatCurrency(invoice.amount_due, invoice.currency)}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={invoiceStatusVariant(invoice.status)}
                          className="capitalize"
                        >
                          {(invoice.status || "unknown").replace("_", " ")}
                        </Badge>
                      </TableCell>
                      <TableCell className="pr-6 text-right">
                        <button
                          type="button"
                          className="inline-flex items-center gap-1 text-sm font-medium text-primary transition-colors hover:text-primary/80"
                        >
                          <Download className="h-3.5 w-3.5" />
                          PDF
                        </button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </main>
    </div>
  );
};

export default PortalDashboard;
