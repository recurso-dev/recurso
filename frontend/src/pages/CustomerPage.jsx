import { useCallback, useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link2, Pencil } from "lucide-react";

import { endpoints } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { usePlans } from "../lib/useCustomers";
import CustomerDetail from "../components/slide-overs/CustomerDetail";
import { formatDate } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedRow,
  RelatedEmpty,
} from "@/components/patterns/ObjectPage";
import { AuditTrail } from "@/components/patterns/AuditTrail";
import { ObjectTimeline } from "@/components/patterns/ObjectTimeline";
import { FinancialSummary } from "@/components/patterns/FinancialSummary";
import { AttentionBanner } from "@/components/patterns/AttentionBanner";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Money } from "@/components/ui/money";
import { StatusBadge } from "@/components/ui/status-badge";
import { CopyableId } from "@/components/ui/copyable-id";

function billingAddress(addr) {
  if (!addr) return null;
  const parts = [addr.line1, addr.city, addr.state, addr.zip, addr.country].filter(Boolean);
  return parts.length ? parts.join(", ") : null;
}

/**
 * CustomerPage — the customer's full object page at /customers/:id
 * (DASHBOARD_REDESIGN.md Phase 5): identity header, summary attributes,
 * related subscriptions/invoices/credit/wallets, metadata + audit rail.
 * Editing reuses the existing CustomerDetail sheet.
 */
export default function CustomerPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);

  const {
    data: customer,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["customer", id],
    queryFn: async () => (await endpoints.getCustomer(id)).data.data,
    enabled: Boolean(id),
  });

  const { data: subscriptions = [] } = useQuery({
    queryKey: ["subscriptions", { customer_id: id }],
    queryFn: async () =>
      (await endpoints.getSubscriptions({ customer_id: id, limit: 50 })).data.data || [],
    enabled: Boolean(id),
  });

  const { data: invoicesPage } = useQuery({
    queryKey: ["invoices", { customer_id: id }],
    queryFn: async () =>
      (await endpoints.getInvoices({ customer_id: id, per_page: 5 })).data,
    enabled: Boolean(id),
  });
  const invoices = invoicesPage?.data || [];
  const invoiceTotal = invoicesPage?.pagination?.total ?? invoices.length;

  const { data: creditNotes = [] } = useQuery({
    queryKey: ["credit-notes", { customer_id: id }],
    queryFn: async () =>
      (await endpoints.getCreditNotes({ customer_id: id })).data.data || [],
    enabled: Boolean(id),
  });

  const { data: wallets = [] } = useQuery({
    queryKey: ["customerWallets", id],
    queryFn: async () => (await endpoints.getCustomerWallets(id)).data.data || [],
    enabled: Boolean(id),
  });

  const {
    data: financials,
    isLoading: financialsLoading,
    error: financialsError,
  } = useQuery({
    queryKey: ["customerFinancialSummary", id],
    queryFn: async () => (await endpoints.getCustomerFinancialSummary(id)).data.data,
    enabled: Boolean(id),
  });
  const currencies = financials?.currencies || [];

  // Exceptions-first: past-due invoices are the customer's live financial
  // exception. One item per currency that has any, tone by amount.
  const attention = currencies
    .filter((c) => c.past_due_count > 0)
    .map((c) => ({
      tone: "danger",
      text: (
        <>
          {c.past_due_count} past-due {c.past_due_count === 1 ? "invoice" : "invoices"} —{" "}
          <Money amountMinor={c.past_due} currency={c.currency} /> outstanding
        </>
      ),
    }));

  const { plans: planList } = usePlans();
  const planName = (planId) =>
    planList.find((p) => p.id === planId)?.name || planId?.slice(0, 8);

  const copyPortalLink = useCallback(() => {
    const url = `${window.location.origin}/portal/login?email=${encodeURIComponent(customer?.email || "")}`;
    navigator.clipboard.writeText(url);
    toast.success("Portal link copied");
  }, [customer?.email]);

  const handleChanged = (updated) => {
    if (updated?.id) {
      queryClient.invalidateQueries({ queryKey: ["customer", updated.id] });
    }
    queryClient.invalidateQueries({ queryKey: ["customers"] });
  };

  if (isLoading) {
    return (
      <div aria-busy="true">
        <Skeleton className="mb-2 h-4 w-24" />
        <Skeleton className="mb-6 h-8 w-64" />
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <Skeleton className="h-64 lg:col-span-2" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (error || !customer) {
    const status = error?.response?.status;
    return (
      <ErrorState
        title={status === 404 ? "Customer not found" : "Couldn't load this customer"}
        message={
          status === 404
            ? "This customer doesn't exist or was removed."
            : error?.response?.data?.error?.message || error?.message
        }
        onRetry={status === 404 ? undefined : refetch}
      />
    );
  }

  const isArchived = customer.active === false;

  return (
    <div>
      <ObjectHeader
        backTo="/customers"
        backLabel="Customers"
        kicker="Customer"
        title={customer.name || customer.email}
        badge={<StatusBadge status={isArchived ? "archived" : "active"} />}
        meta={
          <>
            <span>{customer.email}</span>
            <CopyableId value={customer.id} />
          </>
        }
        actions={
          <>
            <Button variant="outline" onClick={copyPortalLink}>
              <Link2 className="h-4 w-4" />
              Portal link
            </Button>
            <Button onClick={() => setEditOpen(true)}>
              <Pencil className="h-4 w-4" />
              Edit
            </Button>
          </>
        }
      />

      <AttentionBanner items={attention} />

      <ObjectPageLayout
        rail={
          <>
            <ObjectSection title="Metadata">
              <AttributeList
                columns={1}
                items={[
                  { label: "Customer ID", value: <CopyableId value={customer.id} /> },
                  {
                    label: "Ledger account",
                    value: customer.ledger_account_id ? (
                      <Link
                        to={`/ledger/accounts/${customer.ledger_account_id}`}
                        className="font-mono text-xs text-primary underline-offset-2 hover:underline"
                      >
                        {customer.ledger_account_id.slice(0, 8)}…
                      </Link>
                    ) : null,
                  },
                  { label: "Created", value: formatDate(customer.created_at) },
                  {
                    label: "Risk score",
                    value:
                      customer.risk_score != null ? String(customer.risk_score) : null,
                  },
                ]}
              />
            </ObjectSection>
            <ObjectSection title="Timeline">
              <ObjectTimeline objectId={customer.id} />
            </ObjectSection>
            <ObjectSection title="Audit trail">
              <AuditTrail entityType="customers" entityId={customer.id} />
            </ObjectSection>
          </>
        }
      >
        <ObjectSection title="Financial summary">
          <FinancialSummary
            currencies={currencies}
            isLoading={financialsLoading}
            error={financialsError}
          />
        </ObjectSection>

        <ObjectSection title="Overview">
          <AttributeList
            items={[
              { label: "Email", value: customer.email },
              { label: "Phone", value: customer.phone },
              { label: "Tax ID", value: customer.gstin || customer.tax_id },
              {
                label: "Tax type",
                value: customer.tax_type ? (
                  <span className="capitalize">{customer.tax_type}</span>
                ) : null,
              },
              { label: "Billing address", value: billingAddress(customer.billing_address) },
              { label: "Place of supply", value: customer.place_of_supply },
            ]}
          />
        </ObjectSection>

        <ObjectSection
          title={`Subscriptions${subscriptions.length ? ` (${subscriptions.length})` : ""}`}
          flush
        >
          {subscriptions.length === 0 ? (
            <RelatedEmpty>No subscriptions for this customer.</RelatedEmpty>
          ) : (
            <div className="divide-y divide-border">
              {subscriptions.map((s) => (
                <RelatedRow key={s.id} to={`/subscriptions/${s.id}`}>
                  <span className="min-w-0 truncate font-medium text-foreground">
                    {planName(s.plan_id)}
                  </span>
                  <span className="flex shrink-0 items-center gap-3">
                    <span className="hidden text-muted-foreground sm:inline">
                      renews {formatDate(s.current_period_end)}
                    </span>
                    <StatusBadge status={s.status || "unknown"} />
                  </span>
                </RelatedRow>
              ))}
            </div>
          )}
        </ObjectSection>

        <ObjectSection
          title={`Invoices${invoiceTotal ? ` (${invoiceTotal})` : ""}`}
          flush
          action={
            invoiceTotal > invoices.length ? (
              <Link
                to="/invoices"
                className="text-xs font-medium text-primary underline-offset-2 hover:underline"
              >
                View all
              </Link>
            ) : null
          }
        >
          {invoices.length === 0 ? (
            <RelatedEmpty>No invoices for this customer.</RelatedEmpty>
          ) : (
            <div className="divide-y divide-border">
              {invoices.map((inv) => (
                <RelatedRow key={inv.id} to={`/invoices/${inv.id}`}>
                  <span className="min-w-0 truncate font-medium text-foreground">
                    {inv.invoice_number || inv.id.slice(0, 8)}
                  </span>
                  <span className="flex shrink-0 items-center gap-3">
                    <span className="hidden text-muted-foreground sm:inline">
                      {formatDate(inv.created_at)}
                    </span>
                    <Money amountMinor={inv.total} currency={inv.currency} />
                    <StatusBadge status={inv.status || "unknown"} />
                  </span>
                </RelatedRow>
              ))}
            </div>
          )}
        </ObjectSection>

        {(creditNotes.length > 0 || wallets.length > 0) && (
          <ObjectSection title="Credit & wallets" flush>
            <div className="divide-y divide-border">
              {creditNotes.slice(0, 5).map((cn) => (
                <RelatedRow key={cn.id} to={`/credit-notes/${cn.id}`}>
                  <span className="min-w-0 truncate font-medium text-foreground">
                    {cn.reference || `Credit note ${cn.id.slice(0, 8)}`}
                  </span>
                  <span className="flex shrink-0 items-center gap-3">
                    <Money amountMinor={cn.balance ?? cn.amount} currency={cn.currency} />
                    <StatusBadge status={cn.status || "unknown"} />
                  </span>
                </RelatedRow>
              ))}
              {wallets.map((w) => (
                <RelatedRow key={w.id} to="/wallets">
                  <span className="min-w-0 truncate font-medium text-foreground">
                    {w.currency} wallet
                  </span>
                  <span className="flex shrink-0 items-center gap-3">
                    <Money amountMinor={w.balance} currency={w.currency} />
                    {w.closed_at ? <StatusBadge status="closed" /> : null}
                  </span>
                </RelatedRow>
              ))}
            </div>
          </ObjectSection>
        )}
      </ObjectPageLayout>

      <CustomerDetail
        customer={customer}
        isOpen={editOpen}
        onClose={() => setEditOpen(false)}
        onChanged={handleChanged}
      />
    </div>
  );
}
