import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";

import { endpoints } from "../lib/api";
import { useCustomers } from "@/lib/useCustomers";
import { toMinorUnits, formatCurrency, shortId } from "@/lib/utils";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
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
    expires_at: "", // Optional — blank means the credit never expires
  });

  // Invoices for the invoice picker. Fetched once and filtered client-side to
  // the selected customer — a credit note may only link one of that customer's
  // own invoices, never a UUID typed by hand (which risked cross-customer links
  // and mismatched currencies).
  const { data: allInvoices = [] } = useQuery({
    queryKey: ["invoices", "for-credit-note"],
    queryFn: async () => (await endpoints.getInvoices({ per_page: 250 })).data?.data || [],
  });
  const customerInvoices = useMemo(
    () =>
      formData.customer_id
        ? allInvoices.filter((i) => i.customer_id === formData.customer_id)
        : [],
    [allInvoices, formData.customer_id]
  );

  const close = () => navigate("/credit-notes");

  // Selecting a customer resets any linked invoice (the list changes) and
  // returns the currency to the USD default for a standalone credit.
  const selectCustomer = (id) =>
    setFormData((prev) => ({ ...prev, customer_id: id, invoice_id: "", currency: "USD" }));

  // Linking an invoice locks the credit to that invoice's currency — a credit
  // against an invoice must be issued in the same currency.
  const NONE = "__none__";
  const selectInvoice = (value) => {
    if (value === NONE) {
      setFormData((prev) => ({ ...prev, invoice_id: "", currency: "USD" }));
      return;
    }
    const inv = customerInvoices.find((i) => i.id === value);
    setFormData((prev) => ({
      ...prev,
      invoice_id: value,
      currency: inv?.currency || prev.currency,
    }));
  };

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
    // Expiry is optional; send an end-of-day RFC3339 timestamp so the credit is
    // usable through the selected date, or omit it entirely (never expires).
    if (formData.expires_at) {
      payload.expires_at = new Date(`${formData.expires_at}T23:59:59Z`).toISOString();
    } else {
      delete payload.expires_at;
    }

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
              onValueChange={selectCustomer}
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
                  {formData.currency}
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

            <FormField
              label="Linked invoice (optional)"
              htmlFor="invoice_id"
              description={
                formData.invoice_id
                  ? "The credit is issued in this invoice's currency."
                  : undefined
              }
            >
              <Select
                value={
                  formData.customer_id ? formData.invoice_id || NONE : undefined
                }
                onValueChange={selectInvoice}
                disabled={!formData.customer_id}
              >
                <SelectTrigger id="invoice_id">
                  <SelectValue
                    placeholder={
                      formData.customer_id
                        ? "No linked invoice"
                        : "Select a customer first"
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>No linked invoice</SelectItem>
                  {customerInvoices.map((inv) => (
                    <SelectItem key={inv.id} value={inv.id}>
                      {(inv.invoice_number || shortId(inv.id))} —{" "}
                      {formatCurrency(inv.total, inv.currency)} · {inv.status}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
          </div>

          <FormField
            label="Expires on (optional)"
            htmlFor="expires_at"
            description="Leave blank for a credit that never expires. On this date, any unused balance is written off."
          >
            <Input
              id="expires_at"
              type="date"
              name="expires_at"
              value={formData.expires_at}
              onChange={handleChange}
              className="sm:max-w-[12rem]"
            />
          </FormField>

          <FormField label="Reason for credit" htmlFor="reason">
            <Textarea
              id="reason"
              name="reason"
              rows={4}
              value={formData.reason}
              onChange={handleChange}
              placeholder="e.g. Service downtime compensation"
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
