import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import { toMinorUnits } from "@/lib/utils";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

const CreateCreditNote = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  // Shared cached customer list (limit 1000 — the old raw fetch defaulted to
  // limit=10 and silently truncated the dropdown).
  const { customers } = useCustomers();
  const [error, setError] = useState(null);
  const [errors, setErrors] = useState({});
  const [formData, setFormData] = useState({
    customer_id: "",
    amount: "",
    currency: "USD",
    reason: "",
    invoice_id: "", // Optional
  });

  const close = () => navigate("/credit-notes");

  const setField = (name, value) =>
    setFormData((prev) => ({ ...prev, [name]: value }));

  const handleChange = (e) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const validate = () => {
    const next = {};
    if (!formData.customer_id) next.customer_id = "Select a customer.";
    if (!formData.amount) next.amount = "Enter a credit amount.";
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const createMutation = useMutation({
    mutationFn: (payload) => endpoints.createCreditNote(payload),
    onSuccess: () => {
      // Read-your-write: the list is cached for 60s, so invalidate before
      // navigating or the new credit note won't show until the cache expires.
      queryClient.invalidateQueries({ queryKey: ["credit-notes"] });
      navigate("/credit-notes");
    },
    onError: (err) => {
      console.error(err);
      setError(
        err?.response?.data?.error?.message || "Failed to create credit note"
      );
    },
  });
  const loading = createMutation.isPending;

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!validate()) return;
    setError(null);

    // Convert amount to cents
    const payload = {
      ...formData,
      amount: toMinorUnits(formData.amount, formData.currency),
      invoice_id: formData.invoice_id ? formData.invoice_id : null,
    };
    if (!payload.invoice_id) delete payload.invoice_id;

    createMutation.mutate(payload);
  };

  return (
    <Sheet open onOpenChange={(open) => !open && close()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Create credit note</SheetTitle>
          <SheetDescription>
            Issue a credit to a customer that can be applied to an invoice.
          </SheetDescription>
        </SheetHeader>

        <form
          id="create-credit-note-form"
          onSubmit={handleSubmit}
          className="flex-1 space-y-6 overflow-y-auto px-6 py-6"
        >
          {error && (
            <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 ring-1 ring-inset ring-red-600/20">
              {error}
            </div>
          )}

          <FormField
            label="Customer"
            htmlFor="customer_id"
            required
            error={errors.customer_id}
          >
            <Select
              value={formData.customer_id}
              onValueChange={(v) => setField("customer_id", v)}
            >
              <SelectTrigger id="customer_id">
                <SelectValue placeholder="Select a customer..." />
              </SelectTrigger>
              <SelectContent>
                {customers.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name} ({c.email})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>

          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
            <FormField
              label="Credit amount"
              htmlFor="amount"
              required
              error={errors.amount}
            >
              <div className="relative">
                <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-muted-foreground">
                  USD
                </span>
                <Input
                  id="amount"
                  type="number"
                  step="0.01"
                  name="amount"
                  value={formData.amount}
                  onChange={handleChange}
                  placeholder="0.00"
                  className="pl-12"
                />
              </div>
            </FormField>

            <FormField label="Linked invoice (optional)" htmlFor="invoice_id">
              <Input
                id="invoice_id"
                type="text"
                name="invoice_id"
                value={formData.invoice_id}
                onChange={handleChange}
                placeholder="Invoice ID (UUID)..."
              />
            </FormField>
          </div>

          <FormField label="Reason for credit" htmlFor="reason">
            <textarea
              id="reason"
              name="reason"
              rows={4}
              value={formData.reason}
              onChange={handleChange}
              placeholder="e.g. Service downtime compensation"
              className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
            />
          </FormField>
        </form>

        <SheetFooter>
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" form="create-credit-note-form" disabled={loading}>
            {loading ? "Issuing..." : "Issue credit note"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
};

export default CreateCreditNote;
