import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { endpoints } from "../lib/api";
import { queryClient } from "@/lib/queryClient";
import { useToast } from "../components/Toast";
import { cn, toMinorUnits } from "@/lib/utils";
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

const CURRENCIES = ["USD", "INR", "EUR", "GBP"];

export default function CreatePlan() {
  const navigate = useNavigate();
  const toast = useToast();
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState({});
  const [formData, setFormData] = useState({
    name: "",
    code: "",
    description: "",
    price: 99,
    currency: "USD",
    interval: "month",
  });

  const setField = (key, value) => setFormData((f) => ({ ...f, [key]: value }));
  const close = () => navigate("/plans");

  const validate = () => {
    const next = {};
    if (!String(formData.name).trim()) next.name = "Plan name is required.";
    if (!String(formData.code).trim()) next.code = "Plan code is required.";
    if (formData.price === "" || formData.price == null || isNaN(parseFloat(formData.price)))
      next.price = "Enter a valid price.";
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!validate()) return;
    setLoading(true);
    try {
      // Map form fields to the API contract: amount (minor units),
      // interval_unit + interval_count.
      const payload = {
        name: formData.name,
        code: formData.code,
        currency: formData.currency,
        amount: toMinorUnits(formData.price, formData.currency),
        interval_unit: formData.interval,
        interval_count: 1,
      };
      await endpoints.createPlan(payload);
      toast.success("Plan created");
      // Pickers and lists share a 60s plans cache — surface the new plan now.
      queryClient.invalidateQueries({ queryKey: ["plans"] });
      navigate("/plans");
    } catch (error) {
      toast.error(error?.response?.data?.error?.message || "Failed to create plan");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Sheet open onOpenChange={(open) => !open && close()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Create a new plan</SheetTitle>
          <SheetDescription>
            Configure the details for your new subscription plan.
          </SheetDescription>
        </SheetHeader>

        <form
          id="create-plan-form"
          onSubmit={handleSubmit}
          className="flex-1 space-y-8 overflow-y-auto px-6 py-6"
        >
          {/* Plan details */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Plan details</h3>
            <FormField label="Plan name" htmlFor="name" required error={errors.name}>
              <Input
                id="name"
                placeholder="e.g. Pro Tier"
                value={formData.name}
                onChange={(e) => setField("name", e.target.value)}
                className={cn(errors.name && "border-red-400 focus-visible:ring-red-400")}
              />
            </FormField>
            <FormField
              label="Plan code (slug)"
              htmlFor="code"
              required
              error={errors.code}
            >
              <Input
                id="code"
                placeholder="e.g. pro-monthly"
                value={formData.code}
                onChange={(e) => setField("code", e.target.value)}
                className={cn(errors.code && "border-red-400 focus-visible:ring-red-400")}
              />
            </FormField>
            <FormField label="Description" htmlFor="description">
              <Input
                id="description"
                placeholder="Briefly describe this plan"
                value={formData.description}
                onChange={(e) => setField("description", e.target.value)}
              />
            </FormField>
          </section>

          <Separator />

          {/* Pricing */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Pricing</h3>
            <div className="grid grid-cols-2 gap-4">
              <FormField label="Price" htmlFor="price" required error={errors.price}>
                <Input
                  id="price"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  value={formData.price}
                  onChange={(e) => setField("price", e.target.value)}
                  className={cn(
                    errors.price && "border-red-400 focus-visible:ring-red-400"
                  )}
                />
              </FormField>
              <FormField label="Currency" htmlFor="currency">
                <Select
                  value={formData.currency}
                  onValueChange={(v) => setField("currency", v)}
                >
                  <SelectTrigger id="currency">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CURRENCIES.map((c) => (
                      <SelectItem key={c} value={c}>
                        {c}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <FormField label="Billing interval" htmlFor="interval">
              <div className="flex w-full rounded-lg border border-input bg-muted/40 p-1">
                {[
                  { value: "month", label: "Monthly" },
                  { value: "year", label: "Yearly" },
                ].map((opt) => (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => setField("interval", opt.value)}
                    className={cn(
                      "flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-all",
                      formData.interval === opt.value
                        ? "bg-white text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </FormField>
          </section>
        </form>

        <SheetFooter>
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" form="create-plan-form" disabled={loading}>
            {loading ? "Creating..." : "Create plan"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
