import React, { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import axios from "axios";
import {
  CreditCard,
  Download,
  FileText,
  ShieldAlert,
  Loader2,
} from "lucide-react";

import { API_BASE as API_BASE_URL } from "../../lib/api";
import { cn, formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Map an invoice status to a Badge variant (emerald-accented light palette).
const invoiceStatusVariant = (status) =>
  ({
    paid: "success",
    open: "info",
    past_due: "destructive",
    void: "neutral",
    draft: "neutral",
  })[status] || "neutral";

export default function CustomerPortal() {
  const { tenantId, customerId } = useParams();
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchPortalData = async () => {
      try {
        // Unauthenticated call for MVP
        const response = await axios.get(
          `${API_BASE_URL}/portal/${tenantId}/${customerId}`
        );
        setData(response.data);
      } catch (err) {
        setError(
          "Failed to load portal data. The link might be invalid or expired."
        );
      } finally {
        setLoading(false);
      }
    };

    fetchPortalData();
  }, [tenantId, customerId]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-50">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4">
        <Card className="w-full max-w-md text-center">
          <CardContent className="p-8">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-50 ring-1 ring-inset ring-red-600/20">
              <ShieldAlert className="h-6 w-6 text-red-600" />
            </div>
            <h2 className="text-lg font-semibold text-foreground">
              Access denied
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">{error}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const { customer, subscriptions, invoices } = data;

  return (
    <div className="min-h-screen bg-zinc-50">
      <div className="mx-auto max-w-3xl space-y-6 px-4 py-10 sm:px-6 lg:px-8">
        {/* Header */}
        <Card>
          <CardContent className="flex flex-col gap-4 p-6 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
                R
              </div>
              <div>
                <h1 className="text-lg font-semibold tracking-tight text-foreground">
                  Billing Portal
                </h1>
                <p className="text-sm text-muted-foreground">
                  Manage your subscriptions and invoices.
                </p>
              </div>
            </div>
            <div className="sm:text-right">
              <p className="text-sm font-medium text-foreground">
                {customer?.name || "—"}
              </p>
              <p className="text-sm text-muted-foreground">
                {customer?.email || "—"}
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Subscriptions */}
        <Card>
          <CardHeader className="flex-row items-center gap-2 space-y-0 border-b border-border">
            <CreditCard className="h-4 w-4 text-primary" />
            <CardTitle className="text-base">Active subscriptions</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {!subscriptions || subscriptions.length === 0 ? (
              <p className="px-6 py-8 text-center text-sm text-muted-foreground">
                No active subscriptions found.
              </p>
            ) : (
              <div className="divide-y divide-border">
                {subscriptions.map((sub) => (
                  <div
                    key={sub.id}
                    className="flex items-center justify-between px-6 py-5"
                  >
                    <div>
                      <h3 className="text-sm font-semibold text-foreground">
                        {sub.plan_name || "Plan"}
                      </h3>
                      <div className="mt-1 flex items-center gap-2">
                        <Badge
                          variant={
                            sub.status === "active" ? "success" : "neutral"
                          }
                          className="capitalize"
                        >
                          {sub.status || "unknown"}
                        </Badge>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-lg font-semibold tabular-nums text-foreground">
                        {formatCurrency(sub.price, sub.currency)}
                      </p>
                      <p className="mt-0.5 text-xs capitalize text-muted-foreground">
                        per {sub.billing_interval || "month"}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Invoices */}
        <Card>
          <CardHeader className="flex-row items-center gap-2 space-y-0 border-b border-border">
            <FileText className="h-4 w-4 text-primary" />
            <CardTitle className="text-base">Invoice history</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {!invoices || invoices.length === 0 ? (
              <p className="px-6 py-8 text-center text-sm text-muted-foreground">
                No invoices found.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="pl-6">Invoice</TableHead>
                    <TableHead>Issue date</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="pr-6 text-right">Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {invoices.map((inv) => (
                    <TableRow key={inv.id} className="hover:bg-muted/40">
                      <TableCell className="pl-6 font-medium text-foreground">
                        {inv.invoice_number || "—"}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(inv.issue_date)}
                      </TableCell>
                      <TableCell className="text-right tabular-nums text-foreground">
                        {formatCurrency(inv.total, inv.currency)}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={invoiceStatusVariant(inv.status)}
                          className={cn("capitalize")}
                        >
                          {(inv.status || "unknown").replace("_", " ")}
                        </Badge>
                      </TableCell>
                      <TableCell className="pr-6 text-right">
                        <a
                          href={`${API_BASE_URL}/invoices/${inv.id}/pdf`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1 text-sm font-medium text-primary transition-colors hover:text-primary/80"
                        >
                          <Download className="h-3.5 w-3.5" />
                          PDF
                        </a>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
