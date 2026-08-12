import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "@/components/ui/sonner";
import { useNavigate } from "react-router";
import { Sparkles } from "lucide-react";

import { endpoints } from "../lib/api";
import { cn, toMinorUnits, currencyDecimals } from "@/lib/utils";
import { useFormErrors } from "@/lib/useFormErrors";
import { usePlans } from "@/lib/useCustomers";
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

// Currency choices when the tenant has no plans yet (mirrors CreatePlan).
const FALLBACK_CURRENCIES = ["USD", "INR", "EUR", "GBP"];

const symbolFor = (cur) => {
  try {
    return (
      new Intl.NumberFormat("en-US", { style: "currency", currency: cur })
        .formatToParts(0)
        .find((p) => p.type === "currency")?.value || cur
    );
  } catch {
    return cur;
  }
};

const CreateCoupon = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [formData, setFormData] = useState({
    code: "",
    discount_type: "percent",
    discount_value: "",
    duration: "once",
    duration_months: "",
    max_redemptions: "",
    active: true,
  });

  const setField = (key, value) => setFormData((prev) => ({ ...prev, [key]: value }));
  const close = () => navigate("/coupons");
  const { errors, validate, clearError } = useFormErrors();

  // Amount-off coupons apply as raw minor units against the invoice subtotal,
  // so the major→minor conversion must use a real currency's exponent — a USD
  // assumption made a "¥500 off" coupon worth ¥50,000 (100×) and a KWD one 10×
  // small. Offer the currencies the catalog actually bills in, most common
  // first, and convert with the selected one.
  const { plans } = usePlans();
  const currencies = useMemo(() => {
    // Currency lives on each plan's PRICES (a plan has no top-level currency
    // field) — reading p.currency here silently produced an empty count map
    // and a USD-first fallback for everyone, defeating the dominant-currency
    // default for non-USD catalogs.
    const counts = new Map();
    for (const p of plans) {
      for (const price of p.prices || []) {
        if (price.currency) counts.set(price.currency, (counts.get(price.currency) || 0) + 1);
      }
    }
    const sorted = [...counts.entries()].sort((a, b) => b[1] - a[1]).map(([c]) => c);
    return sorted.length ? sorted : FALLBACK_CURRENCIES;
  }, [plans]);
  const [pickedCurrency, setPickedCurrency] = useState(null);
  const currency = pickedCurrency || currencies[0];
  const isPercent = formData.discount_type.toLowerCase().includes("percent");

  const generateCode = () => {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    let result = "";
    for (let i = 0; i < 10; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setField("code", result);
  };

  const createMutation = useMutation({
    mutationFn: (payload) => endpoints.createCoupon(payload),
    onSuccess: () => {
      // Pickers and the list share a 60s coupons cache — surface the new coupon
      // now, or landing on /coupons shows a stale list without it.
      queryClient.invalidateQueries({ queryKey: ["coupons"] });
      navigate("/coupons");
    },
    onError: (error) =>
      toast.error(error?.response?.data?.error?.message || "Failed to create coupon"),
  });
  const loading = createMutation.isPending;

  const handleSubmit = (e) => {
    e.preventDefault();
    const errs = {};
    if (!formData.code.trim()) errs.code = "Coupon code is required.";
    const value = Number(formData.discount_value);
    if (!formData.discount_value || !(value > 0)) {
      errs.discount_value = "Enter a discount greater than zero.";
    } else if (isPercent && value > 100) {
      errs.discount_value = "Percent off cannot exceed 100.";
    }
    if (formData.duration === "repeating" && !(Number(formData.duration_months) > 0)) {
      errs.duration_months = "Enter how many months the discount repeats.";
    }
    if (!validate(errs)) return;
    createMutation.mutate({
      code: formData.code,
      discount_type: isPercent ? "percent" : "amount",
      // Amount-off is typed in major units (e.g. 25 = $25) but the API expects
      // minor units, so "$25 off" must send 2500 — sending 25 created a $0.25
      // coupon (ENG-152). The exponent is the selected currency's (¥500 → 500,
      // KWD 25 → 25000). Percent stays a plain integer.
      discount_value: isPercent
        ? parseInt(formData.discount_value)
        : toMinorUnits(formData.discount_value, currency),
      duration: formData.duration.toLowerCase(),
      duration_months:
        formData.duration === "repeating" && formData.duration_months
          ? parseInt(formData.duration_months)
          : null,
    });
  };

  return (
    <Sheet open onOpenChange={(open) => !open && close()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Create new coupon</SheetTitle>
          <SheetDescription>
            Define a discount code customers can apply at checkout.
          </SheetDescription>
        </SheetHeader>

        <form
          id="create-coupon-form"
          onSubmit={handleSubmit}
          className="flex-1 space-y-8 overflow-y-auto px-6 py-6"
        >
          {/* Code */}
          <section className="space-y-4">
            <FormField
              label="Coupon code"
              htmlFor="code"
              required
              error={errors.code}
              description="Customers will enter this code at checkout."
            >
              <div className="flex items-center gap-2">
                <Input
                  id="code"
                  name="code"
                  placeholder="e.g. SUMMER25OFF"
                  value={formData.code}
                  onChange={(e) => {
                    setField("code", e.target.value);
                    clearError("code");
                  }}
                />
                <Button type="button" variant="outline" onClick={generateCode} className="shrink-0">
                  <Sparkles className="h-4 w-4" />
                  Generate
                </Button>
              </div>
            </FormField>
          </section>

          <Separator />

          {/* Discount */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Discount</h3>
            <div className="grid grid-cols-2 gap-4">
              <FormField label="Discount type" htmlFor="discount_type">
                <Select
                  value={formData.discount_type}
                  onValueChange={(v) => setField("discount_type", v)}
                >
                  <SelectTrigger id="discount_type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="percent">Percent off</SelectItem>
                    <SelectItem value="amount">Amount off</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>

              <FormField
                label="Discount value"
                htmlFor="discount_value"
                required
                error={errors.discount_value}
              >
                <div className="relative">
                  <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-sm text-muted-foreground">
                    {isPercent ? "%" : symbolFor(currency)}
                  </span>
                  <Input
                    id="discount_value"
                    name="discount_value"
                    type="number"
                    min={isPercent ? "1" : String(10 ** -currencyDecimals(currency))}
                    step={isPercent ? "1" : String(10 ** -currencyDecimals(currency))}
                    placeholder="25"
                    value={formData.discount_value}
                    onChange={(e) => {
                      setField("discount_value", e.target.value);
                      clearError("discount_value");
                    }}
                    className="pl-7"
                  />
                </div>
              </FormField>
            </div>

            {!isPercent && (
              <FormField
                label="Currency"
                htmlFor="coupon_currency"
                description="The fixed amount is denominated in this currency and deducted from the invoice subtotal."
              >
                <Select value={currency} onValueChange={setPickedCurrency}>
                  <SelectTrigger id="coupon_currency" className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {currencies.map((c) => (
                      <SelectItem key={c} value={c}>
                        {c}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            )}
          </section>

          <Separator />

          {/* Duration & limits */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Duration &amp; limits</h3>
            <div className="grid grid-cols-2 gap-4">
              <FormField label="Duration" htmlFor="duration">
                <Select value={formData.duration} onValueChange={(v) => setField("duration", v)}>
                  <SelectTrigger id="duration">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="forever">Forever</SelectItem>
                    <SelectItem value="once">Once</SelectItem>
                    <SelectItem value="repeating">Limited time (repeating)</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>

              {formData.duration === "repeating" && (
                <FormField
                  label="Duration in months"
                  htmlFor="duration_months"
                  required
                  error={errors.duration_months}
                >
                  <Input
                    id="duration_months"
                    name="duration_months"
                    type="number"
                    min="1"
                    placeholder="e.g. 12"
                    value={formData.duration_months}
                    onChange={(e) => {
                      setField("duration_months", e.target.value);
                      clearError("duration_months");
                    }}
                  />
                </FormField>
              )}
            </div>

            <FormField
              label="Max redemptions"
              htmlFor="max_redemptions"
              description="Optional — leave blank for unlimited."
            >
              <Input
                id="max_redemptions"
                name="max_redemptions"
                type="number"
                placeholder="Enter max redemptions"
                value={formData.max_redemptions}
                onChange={(e) => setField("max_redemptions", e.target.value)}
              />
            </FormField>

            <div className="flex items-center justify-between">
              <div className="flex flex-col">
                <p className="text-sm font-medium text-foreground">Status</p>
                <p className="text-xs text-muted-foreground">Set the coupon as active or inactive.</p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={formData.active}
                onClick={() => setField("active", !formData.active)}
                className={cn(
                  "relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                  formData.active ? "bg-primary" : "bg-border"
                )}
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out",
                    formData.active ? "translate-x-5" : "translate-x-0"
                  )}
                />
              </button>
            </div>
          </section>
        </form>

        <SheetFooter>
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" form="create-coupon-form" disabled={loading}>
            {loading ? "Creating..." : "Create coupon"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
};

export default CreateCoupon;
