import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { endpoints } from "../lib/api";
import { queryClient } from "@/lib/queryClient";
import { useToast } from "../components/Toast";
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
  const toast = useToast();
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState({});
  const [form, setForm] = useState({
    name: "",
    email: "",
    phone: "",
    address: "",
    country: "United States",
    state: "California",
    tax_id: "",
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

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!validate()) return;
    setLoading(true);
    try {
      const isoCountry = COUNTRY_ISO[form.country] || "US";
      // Payload is byte-for-byte identical to the original create-customer form.
      const payload = {
        name: form.name,
        email: form.email,
        phone: form.phone,
        tax_id: form.tax_id,
        gstin: isIndia ? form.tax_id : "",
        place_of_supply: isIndia ? form.state : "",
        line1: form.address,
        country: isoCountry,
        state: form.state,
      };
      await endpoints.createCustomer(payload);
      toast.success("Customer created");
      // The list caches for 60s — without this the new customer is invisible.
      queryClient.invalidateQueries({ queryKey: ["customers"] });
      navigate("/customers");
    } catch (error) {
      toast.error(
        error?.response?.data?.error?.message || "Failed to create customer"
      );
    } finally {
      setLoading(false);
    }
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
