import { useState } from "react";
import { toast } from "sonner";
import { useNavigate } from "react-router-dom";
import { Sparkles } from "lucide-react";

import { endpoints } from "../lib/api";
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

const CreateCoupon = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
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

  const generateCode = () => {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    let result = "";
    for (let i = 0; i < 10; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setField("code", result);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    const isPercent = formData.discount_type.toLowerCase().includes("percent");
    const payload = {
      code: formData.code,
      discount_type: isPercent ? "percent" : "amount",
      // Amount-off is typed in major units (e.g. 25 = $25) but the API expects
      // minor units, so "$25 off" must send 2500 — sending 25 created a $0.25
      // coupon (ENG-152). Percent stays a plain integer.
      discount_value: isPercent
        ? parseInt(formData.discount_value)
        : toMinorUnits(formData.discount_value),
      duration: formData.duration.toLowerCase(),
      duration_months:
        formData.duration === "repeating" && formData.duration_months
          ? parseInt(formData.duration_months)
          : null,
    };

    try {
      await endpoints.createCoupon(payload);
      navigate("/coupons");
    } catch (error) {
      toast.error(error?.response?.data?.error?.message || "Failed to create coupon");
    } finally {
      setLoading(false);
    }
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
              description="Customers will enter this code at checkout."
            >
              <div className="flex items-center gap-2">
                <Input
                  id="code"
                  name="code"
                  required
                  placeholder="e.g. SUMMER25OFF"
                  value={formData.code}
                  onChange={(e) => setField("code", e.target.value)}
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

              <FormField label="Discount value" htmlFor="discount_value" required>
                <div className="relative">
                  <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-sm text-muted-foreground">
                    {formData.discount_type === "percent" ? "%" : "$"}
                  </span>
                  <Input
                    id="discount_value"
                    name="discount_value"
                    type="number"
                    min="1"
                    required
                    placeholder="25"
                    value={formData.discount_value}
                    onChange={(e) => setField("discount_value", e.target.value)}
                    className="pl-7"
                  />
                </div>
              </FormField>
            </div>
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
                <FormField label="Duration in months" htmlFor="duration_months" required>
                  <Input
                    id="duration_months"
                    name="duration_months"
                    type="number"
                    min="1"
                    required
                    placeholder="e.g. 12"
                    value={formData.duration_months}
                    onChange={(e) => setField("duration_months", e.target.value)}
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
                  formData.active ? "bg-primary" : "bg-stone-200"
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
