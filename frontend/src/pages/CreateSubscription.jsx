import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { endpoints } from "../lib/api";
import { queryClient } from "@/lib/queryClient";
import { toast } from "@/components/ui/sonner";
import ConsentCheckbox from "../components/ui/ConsentCheckbox";
import { formatCurrency, formatDate } from "@/lib/utils";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export default function CreateSubscription() {
  const navigate = useNavigate();

  const [customers, setCustomers] = useState([]);
  const [plans, setPlans] = useState([]);
  const [loadingData, setLoadingData] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  const [formData, setFormData] = useState({
    customer_id: "",
    plan_id: "",
    start_date: new Date().toISOString().split("T")[0],
    billing_anchor_type: "acquisition",
    payment_terms: "due_on_receipt",
    consent_granted: false,
  });

  const setField = (key, value) => setFormData((f) => ({ ...f, [key]: value }));
  const close = () => navigate("/subscriptions");

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [custRes, plansRes] = await Promise.all([
          endpoints.getCustomers(),
          endpoints.getPlans(),
        ]);
        setCustomers(custRes.data.data || []);
        setPlans(plansRes.data.data || []);
      } catch (error) {
        console.error("Failed to fetch data:", error);
      } finally {
        setLoadingData(false);
      }
    };
    fetchData();
  }, []);

  const selectedCustomer = customers.find((c) => c.id === formData.customer_id);
  const selectedPlan = plans.find((p) => p.id === formData.plan_id);

  const priceObj = selectedPlan?.prices?.[0];
  const amountMinor = priceObj?.amount ?? 0;
  const currency = priceObj?.currency || "USD";

  const handleSubmit = async (e) => {
    e.preventDefault();

    if (!formData.customer_id || !formData.plan_id) {
      toast.warning("Please select a customer and a plan.");
      return;
    }

    if (!formData.consent_granted) {
      toast.warning("Please authorize recurring billing to continue.");
      return;
    }

    setSubmitting(true);
    try {
      // Payload byte-for-byte identical to the original create-subscription form.
      const payload = {
        customer_id: formData.customer_id,
        plan_id: formData.plan_id,
        start_date: new Date(formData.start_date).toISOString(),
        billing_anchor_type: formData.billing_anchor_type,
        payment_terms: formData.payment_terms,
      };
      const res = await endpoints.createSubscription(payload);
      const sub = res.data.data;

      if (sub && sub.razorpay_subscription_id) {
        const options = {
          key: import.meta.env.VITE_RAZORPAY_KEY_ID,
          subscription_id: sub.razorpay_subscription_id,
          name: "Billify Recurso",
          description: `Subscription for ${selectedPlan?.name || "Plan"}`,
          handler: function () {
            queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
            navigate("/subscriptions");
          },
          prefill: {
            name: selectedCustomer?.name,
            email: selectedCustomer?.email,
            contact: selectedCustomer?.phone,
          },
          theme: {
            color: "#10B981",
          },
          modal: {
            ondismiss: function () {
              queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
              navigate("/subscriptions");
            },
          },
        };

        const rzp = new window.Razorpay(options);
        rzp.open();
      } else {
        queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
        navigate("/subscriptions");
      }
    } catch (error) {
      console.error("Failed to create subscription:", error);
      toast.error(
        error?.response?.data?.error?.message || "Failed to create subscription"
      );
      setSubmitting(false);
      return;
    }
    if (!loadingData) setSubmitting(false);
  };

  return (
    <Sheet open onOpenChange={(open) => !open && close()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Create new subscription</SheetTitle>
          <SheetDescription>
            Create a new subscription for an existing customer.
          </SheetDescription>
        </SheetHeader>

        <form
          id="create-subscription-form"
          onSubmit={handleSubmit}
          className="flex-1 space-y-8 overflow-y-auto px-6 py-6"
        >
          {/* Customer & Plan */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Customer &amp; plan</h3>
            <FormField label="Customer" htmlFor="customer">
              <Select
                value={formData.customer_id}
                onValueChange={(v) => setField("customer_id", v)}
              >
                <SelectTrigger id="customer">
                  <SelectValue placeholder="Select a customer" />
                </SelectTrigger>
                <SelectContent>
                  {customers.map((customer) => (
                    <SelectItem key={customer.id} value={customer.id}>
                      {customer.name} ({customer.email})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
            <FormField label="Plan" htmlFor="plan">
              <Select
                value={formData.plan_id}
                onValueChange={(v) => setField("plan_id", v)}
              >
                <SelectTrigger id="plan">
                  <SelectValue placeholder="Select a plan" />
                </SelectTrigger>
                <SelectContent>
                  {plans.map((plan) => (
                    <SelectItem key={plan.id} value={plan.id}>
                      {plan.name} - {formatCurrency(plan.prices?.[0]?.amount, plan.prices?.[0]?.currency)}/
                      {plan.interval_unit}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
          </section>

          <Separator />

          {/* Scheduling & Billing */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">
              Scheduling &amp; billing
            </h3>
            <FormField label="Start date" htmlFor="start_date">
              <Input
                id="start_date"
                type="date"
                value={formData.start_date}
                onChange={(e) => setField("start_date", e.target.value)}
              />
            </FormField>
            <FormField
              label="Billing anchor"
              htmlFor="billing_anchor_type"
              description={
                formData.billing_anchor_type === "first_of_month"
                  ? "First period will be prorated. All renewals align to the 1st."
                  : "Billing repeats from the subscription start date."
              }
            >
              <Select
                value={formData.billing_anchor_type}
                onValueChange={(v) => setField("billing_anchor_type", v)}
              >
                <SelectTrigger id="billing_anchor_type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="acquisition">Acquisition date (default)</SelectItem>
                  <SelectItem value="first_of_month">
                    Calendar billing (1st of month)
                  </SelectItem>
                </SelectContent>
              </Select>
            </FormField>
            <FormField
              label="Payment terms"
              htmlFor="payment_terms"
              description="Number of days after invoice date before payment is due."
            >
              <Select
                value={formData.payment_terms}
                onValueChange={(v) => setField("payment_terms", v)}
              >
                <SelectTrigger id="payment_terms">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="due_on_receipt">Due on receipt (default)</SelectItem>
                  <SelectItem value="net15">Net 15</SelectItem>
                  <SelectItem value="net30">Net 30</SelectItem>
                  <SelectItem value="net45">Net 45</SelectItem>
                  <SelectItem value="net60">Net 60</SelectItem>
                </SelectContent>
              </Select>
            </FormField>
          </section>

          <Separator />

          {/* Billing authorization */}
          <section className="space-y-3">
            <h3 className="text-sm font-semibold text-foreground">
              Billing authorization
            </h3>
            <ConsentCheckbox
              type="recurring_billing"
              onConsentChange={(c) => setField("consent_granted", c.granted)}
              planName={selectedPlan?.name || "the selected plan"}
              amount={
                selectedPlan?.prices?.[0]?.amount
                  ? formatCurrency(selectedPlan.prices[0].amount, currency)
                  : ""
              }
              billingInterval={selectedPlan?.interval_unit || "month"}
            />
          </section>

          <Separator />

          {/* Summary */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Summary</h3>
            {selectedCustomer && selectedPlan ? (
              <div className="rounded-lg border border-border bg-muted/30 p-4">
                <dl className="space-y-2.5 text-sm">
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">Customer</dt>
                    <dd className="truncate font-medium text-foreground">
                      {selectedCustomer.name}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">Plan</dt>
                    <dd className="truncate font-medium text-foreground">
                      {selectedPlan.name}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">Starts on</dt>
                    <dd className="font-medium text-foreground">
                      {formatDate(formData.start_date)}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">Billing</dt>
                    <dd className="font-medium text-foreground">
                      {formData.billing_anchor_type === "first_of_month"
                        ? "Calendar (1st)"
                        : "Acquisition"}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">Terms</dt>
                    <dd className="font-medium text-foreground">
                      {formData.payment_terms === "due_on_receipt"
                        ? "Due on Receipt"
                        : formData.payment_terms.replace("net", "Net ")}
                    </dd>
                  </div>
                </dl>
                <Separator className="my-3" />
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-foreground">Total</span>
                  <span className="text-sm font-semibold tabular-nums text-foreground">
                    {formatCurrency(amountMinor, currency)}
                  </span>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Select a customer and plan to see the summary.
              </p>
            )}
          </section>
        </form>

        <SheetFooter>
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" form="create-subscription-form" disabled={submitting}>
            {submitting ? "Creating..." : "Create subscription"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
