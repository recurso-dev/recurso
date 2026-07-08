import { formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

const riskVariant = (score) => {
  if (score == null) return "neutral";
  if (score >= 50) return "destructive";
  if (score >= 25) return "warning";
  return "success";
};

const riskLabel = (score) => {
  if (score == null) return null;
  if (score >= 50) return "High";
  if (score >= 25) return "Medium";
  return "Low";
};

const Field = ({ label, children, mono }) => (
  <div>
    <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
      {label}
    </dt>
    <dd className={`mt-1 text-sm text-foreground ${mono ? "font-mono" : ""}`}>
      {children}
    </dd>
  </div>
);

// A titled group of fields, rendered only when it has at least one value.
const Section = ({ title, children }) => (
  <div className="space-y-4">
    <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70">
      {title}
    </p>
    <dl className="space-y-4">{children}</dl>
  </div>
);

// Join the non-empty parts of a billing address into a readable block.
const formatAddress = (addr) => {
  if (!addr) return null;
  const parts = [addr.line1, addr.line2, addr.city, addr.state, addr.zip, addr.country]
    .map((p) => (p || "").trim())
    .filter(Boolean);
  return parts.length ? parts.join(", ") : null;
};

const CustomerDetail = ({ customer, isOpen, onClose }) => {
  if (!customer) return null;

  const risk = customer.churn_risk ?? customer.risk_score ?? null;
  const activeSubs = customer.activeSubs ?? customer.active_subscriptions ?? 0;
  const address = formatAddress(customer.billing_address);
  const hasBilling =
    address || customer.tax_type || customer.gstin || customer.place_of_supply;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Customer details</SheetTitle>
        </SheetHeader>

        <div className="mt-6 space-y-6">
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="truncate text-base font-semibold text-foreground">
                {customer.name || "Unnamed customer"}
              </p>
              <p className="truncate text-sm text-muted-foreground">
                {customer.email}
              </p>
            </div>
            <Badge variant={activeSubs > 0 ? "success" : "neutral"}>
              {activeSubs} active
            </Badge>
          </div>

          <Separator />

          {/* Contact */}
          <Section title="Contact">
            <Field label="Email">{customer.email || "—"}</Field>
            <Field label="Phone">{customer.phone?.trim() || "—"}</Field>
          </Section>

          {/* Billing & tax — only when there's something to show */}
          {hasBilling && (
            <>
              <Separator />
              <Section title="Billing & tax">
                {address && <Field label="Billing address">{address}</Field>}
                {customer.tax_type && (
                  <Field label="Tax type">
                    <span className="capitalize">{customer.tax_type}</span>
                  </Field>
                )}
                {customer.gstin && (
                  <Field label="GSTIN" mono>
                    {customer.gstin}
                  </Field>
                )}
                {customer.place_of_supply && (
                  <Field label="Place of supply">{customer.place_of_supply}</Field>
                )}
              </Section>
            </>
          )}

          <Separator />

          {/* Record */}
          <Section title="Record">
            <Field label="Customer ID" mono>
              {customer.id}
            </Field>
            <Field label="Joined">
              {customer.created_at ? formatDate(customer.created_at) : "—"}
            </Field>
            {risk != null && (
              <Field label="Churn risk">
                <Badge variant={riskVariant(risk)}>
                  {risk} · {riskLabel(risk)}
                </Badge>
              </Field>
            )}
            <Field label="Active subscriptions">{activeSubs}</Field>
            {customer.referral_code && (
              <Field label="Referral code" mono>
                {customer.referral_code}
              </Field>
            )}
          </Section>
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default CustomerDetail;
