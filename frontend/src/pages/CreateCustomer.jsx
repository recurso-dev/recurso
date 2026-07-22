import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { endpoints } from "../lib/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "@/components/ui/sonner";
import { cn } from "@/lib/utils";
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

// ISO country map — identical mapping to the previous implementation.
const COUNTRY_ISO = {
  "United States": "US",
  India: "IN",
  Canada: "CA",
  "United Kingdom": "GB",
};

const INDIA_STATES = [
  { code: "TN", name: "Tamil Nadu" },
  { code: "KA", name: "Karnataka" },
  { code: "MH", name: "Maharashtra" },
  { code: "DL", name: "Delhi" },
  { code: "UP", name: "Uttar Pradesh" },
  { code: "GJ", name: "Gujarat" },
  { code: "KL", name: "Kerala" },
];

export default function CreateCustomer() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [errors, setErrors] = useState({});
  const [form, setForm] = useState({
    name: "",
    email: "",
    phone: "",
    address: "",
    country: "United States",
    state: "California",
    tax_id: "",
    tax_exempt: false,
    tax_exemption_number: "",
    tax_exemption_code: "",
  });

  const isIndia = form.country === "India";
  const setField = (key, value) => setForm((f) => ({ ...f, [key]: value }));
  const close = () => navigate("/customers");

  const validate = () => {
    const next = {};
    if (!form.name.trim()) next.name = "Customer name is required.";
    if (!form.email.trim()) next.email = "Email is required.";
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
      next.email = "Enter a valid email address.";
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const createMutation = useMutation({
    mutationFn: (payload) => endpoints.createCustomer(payload),
    onSuccess: () => {
      toast.success("Customer created");
      // The list caches for 60s — without this the new customer is invisible.
      queryClient.invalidateQueries({ queryKey: ["customers"] });
      navigate("/customers");
    },
    onError: (error) =>
      toast.error(error?.response?.data?.error?.message || "Failed to create customer"),
  });
  const loading = createMutation.isPending;

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!validate()) return;
    const isoCountry = COUNTRY_ISO[form.country] || "US";
    // Payload is byte-for-byte identical to the original create-customer form.
    createMutation.mutate({
      name: form.name,
      email: form.email,
      phone: form.phone,
      tax_id: form.tax_id,
      gstin: isIndia ? form.tax_id : "",
      place_of_supply: isIndia ? form.state : "",
      line1: form.address,
      country: isoCountry,
      state: form.state,
      // US sales-tax exemption (D2) — only meaningful outside India.
      tax_exempt: isIndia ? false : form.tax_exempt,
      tax_exemption_number: isIndia ? "" : form.tax_exemption_number,
      tax_exemption_code: isIndia ? "" : form.tax_exemption_code,
    });
  };

  return (
    <Sheet open onOpenChange={(open) => !open && close()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Add new customer</SheetTitle>
          <SheetDescription>
            Enter the details for your new customer.
          </SheetDescription>
        </SheetHeader>

        <form
          id="create-customer-form"
          onSubmit={handleSubmit}
          className="flex-1 space-y-8 overflow-y-auto px-6 py-6"
        >
          {/* Contact information */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">
              Contact information
            </h3>
            <FormField label="Customer name" htmlFor="name" required error={errors.name}>
              <Input
                id="name"
                placeholder="e.g., Acme Corporation"
                value={form.name}
                onChange={(e) => setField("name", e.target.value)}
                className={cn(errors.name && "border-red-400 focus-visible:ring-red-400")}
              />
            </FormField>
            <FormField label="Email address" htmlFor="email" required error={errors.email}>
              <Input
                id="email"
                type="email"
                placeholder="e.g., billing@acme.com"
                value={form.email}
                onChange={(e) => setField("email", e.target.value)}
                className={cn(errors.email && "border-red-400 focus-visible:ring-red-400")}
              />
            </FormField>
            <FormField label="Phone number" htmlFor="phone">
              <Input
                id="phone"
                placeholder="e.g., +1 (555) 123-4567"
                value={form.phone}
                onChange={(e) => setField("phone", e.target.value)}
              />
            </FormField>
          </section>

          <Separator />

          {/* Billing details */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Billing details</h3>
            <FormField label="Billing address" htmlFor="address">
              <textarea
                id="address"
                rows={3}
                placeholder="123 Main Street, Anytown, USA 12345"
                value={form.address}
                onChange={(e) => setField("address", e.target.value)}
                className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
              />
            </FormField>

            <div className="grid grid-cols-2 gap-4">
              <FormField label="Country" htmlFor="country">
                <Select
                  value={form.country}
                  onValueChange={(value) =>
                    setForm((f) => ({ ...f, country: value, state: "" }))
                  }
                >
                  <SelectTrigger id="country">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.keys(COUNTRY_ISO).map((c) => (
                      <SelectItem key={c} value={c}>
                        {c}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField
                label={isIndia ? "Place of supply (State)" : "State / Province"}
                htmlFor="state"
              >
                {isIndia ? (
                  <Select value={form.state} onValueChange={(v) => setField("state", v)}>
                    <SelectTrigger id="state">
                      <SelectValue placeholder="Select state" />
                    </SelectTrigger>
                    <SelectContent>
                      {INDIA_STATES.map((s) => (
                        <SelectItem key={s.code} value={s.code}>
                          {s.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Input
                    id="state"
                    placeholder="e.g. California"
                    value={form.state}
                    onChange={(e) => setField("state", e.target.value)}
                  />
                )}
              </FormField>
            </div>

            <FormField
              label={isIndia ? "GSTIN (Goods and Services Tax ID)" : "Tax ID / VAT Number"}
              htmlFor="tax_id"
            >
              <Input
                id="tax_id"
                placeholder={isIndia ? "e.g., 29ABCDE1234F1Z5" : "e.g., EU123456789"}
                value={form.tax_id}
                onChange={(e) => setField("tax_id", e.target.value)}
              />
            </FormField>

            {!isIndia && (
              <div className="rounded-lg border border-border p-4">
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <p className="text-sm font-medium">Tax exempt (US)</p>
                    <p className="text-sm text-muted-foreground">
                      Pass this customer's exemption to the tax provider so no US
                      sales tax is collected and the sale is recorded exempt.
                    </p>
                  </div>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={form.tax_exempt}
                    onClick={() => setField("tax_exempt", !form.tax_exempt)}
                    className={cn(
                      "relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                      form.tax_exempt ? "bg-primary" : "bg-stone-200",
                    )}
                  >
                    <span
                      className={cn(
                        "pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out",
                        form.tax_exempt ? "translate-x-5" : "translate-x-0",
                      )}
                    />
                  </button>
                </div>
                {form.tax_exempt && (
                  <div className="mt-4 grid gap-4 sm:grid-cols-2">
                    <FormField label="Exemption / certificate number" htmlFor="tax_exemption_number">
                      <Input
                        id="tax_exemption_number"
                        placeholder="e.g. RESALE-0001"
                        value={form.tax_exemption_number}
                        onChange={(e) => setField("tax_exemption_number", e.target.value)}
                      />
                    </FormField>
                    <FormField label="Entity-use code" htmlFor="tax_exemption_code">
                      <Input
                        id="tax_exemption_code"
                        placeholder="e.g. A (federal govt)"
                        value={form.tax_exemption_code}
                        onChange={(e) => setField("tax_exemption_code", e.target.value.toUpperCase())}
                      />
                    </FormField>
                  </div>
                )}
              </div>
            )}
          </section>
        </form>

        <SheetFooter>
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" form="create-customer-form" disabled={loading}>
            {loading ? "Creating..." : "Create customer"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
