import { Link, useParams } from "react-router";
import { ArrowRight } from "lucide-react";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import { useObjectQuery } from "@/lib/useObjectQuery";
import { humanizeFailure } from "@/lib/failureLabels";
import { formatDateTime } from "@/lib/utils";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedRow,
  RelatedEmpty,
  ObjectPageSkeleton,
  ObjectNotFound,
  ObjectPageError,
} from "@/components/patterns/ObjectPage";
import { AttentionBanner } from "@/components/patterns/AttentionBanner";
import { SubscriptionRef } from "@/components/patterns/SubscriptionRef";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import { ObjectTimeline } from "@/components/patterns/ObjectTimeline";
import { Money } from "@/components/ui/money";
import { StatusBadge } from "@/components/ui/status-badge";
import { CopyableId } from "@/components/ui/copyable-id";
import { Overline } from "@/components/ui/overline";

// A payment attempt has no first-class ledger transaction of its own — its
// postings live on the invoice it settles (payment posts against the invoice's
// AR, ADR-002). So the accounting drill is "open the invoice's journal", stated
// honestly rather than implying a per-payment ledger entry.

// One-line, human answer to "what happened to this money?" for the summary.
function outcomeLine(payment) {
  switch (payment.status) {
    case "succeeded":
      return { tone: "text-success", text: "Settled successfully." };
    case "processing":
      return { tone: "text-warning", text: "In flight — awaiting settlement from the bank." };
    case "initiated":
      return { tone: "text-muted-foreground", text: "Initiated — not yet processing." };
    case "failed":
      return {
        tone: "text-destructive",
        text: payment.failure_code
          ? `Failed — ${humanizeFailure(payment.failure_code)}.`
          : "Failed at the gateway.",
      };
    case "returned":
      return {
        tone: "text-destructive",
        text: payment.failure_code
          ? `Returned by the bank after settling — ${humanizeFailure(payment.failure_code)}.`
          : "Returned by the bank after it had settled.",
      };
    default:
      return { tone: "text-muted-foreground", text: "" };
  }
}

export default function PaymentPage() {
  const { id } = useParams();
  const { names: customerNames } = useCustomers();

  const {
    object: payment,
    loading,
    notFound,
    isError,
    error,
    refetch,
  } = useObjectQuery(
    ["payment", id],
    async () => (await endpoints.getPayment(id)).data.data,
    { enabled: Boolean(id) }
  );

  if (loading) return <ObjectPageSkeleton />;
  if (notFound) {
    return (
      <ObjectNotFound
        objectLabel="payment"
        identifier={id ? String(id).slice(0, 8) : undefined}
        backTo="/payments"
        backLabel="Payments"
      />
    );
  }
  if (isError) {
    return (
      <ObjectPageError objectLabel="payment" error={error} onRetry={refetch} backTo="/payments" backLabel="Payments" />
    );
  }

  const outcome = outcomeLine(payment);
  const attention = [];
  if (payment.status === "failed") {
    attention.push({
      tone: "danger",
      text: payment.failure_code
        ? `Payment failed — ${humanizeFailure(payment.failure_code)}.`
        : "Payment failed — the gateway declined this attempt.",
    });
  } else if (payment.status === "returned") {
    attention.push({
      tone: "danger",
      text: "Payment returned — this settled and was later reversed by the bank.",
    });
  }

  return (
    <div>
      <ObjectHeader
        backTo="/payments"
        backLabel="Payments"
        kicker="Payment"
        title={payment.invoice_number ? `Payment · ${payment.invoice_number}` : `Payment ${String(payment.id).slice(0, 8)}`}
        badge={<StatusBadge status={payment.status} flashOnChange />}
        amount={<Money amountMinor={payment.amount} currency={payment.currency} size="hero" />}
        amountLabel={outcome.text ? <span className={outcome.tone}>{outcome.text}</span> : null}
        meta={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <CopyableId value={payment.id} />
            <span className="text-muted-foreground">{formatDateTime(payment.created_at)}</span>
          </span>
        }
      />

      <AttentionBanner items={attention} />

      <ObjectPageLayout
        rail={
          <>
            <ObjectSection title="Details">
              <AttributeList
                columns={1}
                items={[
                  { label: "Gateway", value: payment.gateway },
                  { label: "Method", value: payment.method },
                  {
                    label: "Gateway reference",
                    value: payment.gateway_payment_intent_id ? (
                      <span className="font-mono text-xs">{payment.gateway_payment_intent_id}</span>
                    ) : null,
                  },
                  { label: "Payment ID", value: <CopyableId value={payment.id} /> },
                  { label: "Started", value: formatDateTime(payment.created_at) },
                  { label: "Updated", value: formatDateTime(payment.updated_at) },
                  {
                    label: "Settled",
                    value: payment.settled_at ? formatDateTime(payment.settled_at) : null,
                  },
                  {
                    label: "Gateway failure code",
                    value: payment.failure_code ? (
                      <span className="font-mono text-xs">{payment.failure_code}</span>
                    ) : null,
                  },
                ]}
              />
            </ObjectSection>
            <ObjectSection title="Timeline">
              <ObjectTimeline objectId={payment.id} />
            </ObjectSection>
          </>
        }
      >
        {/* The financial graph: who paid, against what, for which subscription. */}
        <ObjectSection title="Related" flush>
          {payment.invoice_id ? (
            <RelatedRow to={`/invoices/${payment.invoice_id}`}>
              <Overline as="span">Invoice</Overline>
              <span className="font-medium">{payment.invoice_number || "View invoice"}</span>
            </RelatedRow>
          ) : (
            <RelatedEmpty>No invoice linked.</RelatedEmpty>
          )}
          {payment.customer_id ? (
            <RelatedRow to={`/customers/${payment.customer_id}`}>
              <Overline as="span">Customer</Overline>
              <CustomerName id={payment.customer_id} names={customerNames} link={false} />
            </RelatedRow>
          ) : null}
          {payment.subscription_id ? (
            <RelatedRow to={`/subscriptions/${payment.subscription_id}`}>
              <Overline as="span">Subscription</Overline>
              <SubscriptionRef subscriptionId={payment.subscription_id} />
            </RelatedRow>
          ) : null}
        </ObjectSection>

        {/* Accounting: a payment posts against its invoice's ledger, not a
            standalone entry — send the operator to the invoice's journal. */}
        {payment.invoice_id ? (
          <ObjectSection title="Accounting impact">
            <p className="text-sm text-muted-foreground">
              This payment posts to the ledger against its invoice (debit Cash / credit Accounts
              Receivable). View the balanced entries on the invoice.
            </p>
            <Link
              to={`/invoices/${payment.invoice_id}`}
              className="mt-2 inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
            >
              Invoice journal entries
              <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          </ObjectSection>
        ) : null}
      </ObjectPageLayout>
    </div>
  );
}
