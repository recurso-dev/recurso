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

const CustomerDetail = ({ customer, isOpen, onClose }) => {
  if (!customer) return null;

  const risk = customer.churn_risk ?? customer.risk_score ?? null;
  const activeSubs = customer.activeSubs ?? customer.active_subscriptions ?? 0;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Customer details</SheetTitle>
        </SheetHeader>

        <div className="mt-6 space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-base font-semibold text-foreground">
                {customer.name}
              </p>
              <p className="text-sm text-muted-foreground">{customer.email}</p>
            </div>
            <Badge variant={activeSubs > 0 ? "success" : "neutral"}>
              {activeSubs} active
            </Badge>
          </div>

          <Separator />

          <dl className="space-y-5">
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
          </dl>
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default CustomerDetail;
