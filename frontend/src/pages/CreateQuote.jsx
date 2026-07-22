import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCustomers } from "@/lib/useCustomers";
import { useNavigate } from "react-router-dom";
import { Plus, Trash2 } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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

const CreateQuote = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { customers } = useCustomers();
  const [error, setError] = useState(null);
  const [errors, setErrors] = useState({});

  const [formData, setFormData] = useState({
    customer_id: "",
    currency: "USD",
    notes: "",
    terms: "Payment due within 30 days of acceptance.",
    tax_amount: 0,
    discount_amount: 0,
    valid_until: "",
    line_items: [{ description: "", quantity: 1, unit_price: 0 }],
  });

  const createMutation = useMutation({
    mutationFn: (payload) => endpoints.createQuote(payload),
    onSuccess: () => {
      // The quotes list caches for 60s — invalidate so the new quote shows.
      queryClient.invalidateQueries({ queryKey: ["quotes"] });
      navigate("/quotes");
    },
    onError: (err) => setError(err.response?.data?.error?.message || "Failed to create quote"),
  });
  const loading = createMutation.isPending;

  const close = () => navigate("/quotes");

  const setField = (name, value) =>
    setFormData((prev) => ({ ...prev, [name]: value }));

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const handleLineItemChange = (index, field, value) => {
    const newItems = [...formData.line_items];
    newItems[index] = { ...newItems[index], [field]: value };
    setFormData((prev) => ({ ...prev, line_items: newItems }));
  };

  const addLineItem = () => {
    setFormData((prev) => ({
      ...prev,
      line_items: [...prev.line_items, { description: "", quantity: 1, unit_price: 0 }],
    }));
  };

  const removeLineItem = (index) => {
    if (formData.line_items.length > 1) {
      setFormData((prev) => ({
        ...prev,
        line_items: prev.line_items.filter((_, i) => i !== index),
      }));
    }
  };

  const calculateSubtotal = () =>
    formData.line_items.reduce(
      (sum, item) => sum + item.quantity * item.unit_price,
      0
    );

  const calculateTotal = () =>
    calculateSubtotal() + Number(formData.tax_amount) - Number(formData.discount_amount);

  const validate = () => {
    const next = {};
    if (!formData.customer_id) next.customer_id = "Select a customer.";
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!validate()) return;
    setError(null);
    createMutation.mutate({
      customer_id: formData.customer_id,
      currency: formData.currency,
      notes: formData.notes,
      terms: formData.terms,
      tax_amount: parseInt(formData.tax_amount) || 0,
      discount_amount: parseInt(formData.discount_amount) || 0,
      valid_until: formData.valid_until
        ? new Date(formData.valid_until).toISOString()
        : null,
      line_items: formData.line_items.map((item) => ({
        description: item.description,
        quantity: parseInt(item.quantity) || 1,
        unit_price: parseInt(item.unit_price) || 0,
        amount: (parseInt(item.quantity) || 1) * (parseInt(item.unit_price) || 0),
      })),
    });
  };

  return (
    <Sheet open onOpenChange={(open) => !open && close()}>
      <SheetContent side="right" className="w-full sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle>Create quote</SheetTitle>
          <SheetDescription>Create a new quote for a customer.</SheetDescription>
        </SheetHeader>

        <form
          id="create-quote-form"
          onSubmit={handleSubmit}
          className="flex-1 space-y-8 overflow-y-auto px-6 py-6"
        >
          {error && (
            <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 ring-1 ring-inset ring-red-600/20">
              {error}
            </div>
          )}

          {/* Quote details */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Quote details</h3>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
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
                    <SelectValue placeholder="Select customer" />
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

              <FormField label="Currency" htmlFor="currency">
                <Select
                  value={formData.currency}
                  onValueChange={(v) => setField("currency", v)}
                >
                  <SelectTrigger id="currency">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="USD">USD</SelectItem>
                    <SelectItem value="EUR">EUR</SelectItem>
                    <SelectItem value="GBP">GBP</SelectItem>
                    <SelectItem value="INR">INR</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>

              <FormField label="Valid until" htmlFor="valid_until">
                <Input
                  id="valid_until"
                  type="date"
                  name="valid_until"
                  value={formData.valid_until}
                  onChange={handleChange}
                />
              </FormField>
            </div>
          </section>

          <Separator />

          {/* Line items */}
          <section className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-foreground">Line items</h3>
              <Button type="button" variant="outline" size="sm" onClick={addLineItem}>
                <Plus className="h-4 w-4" />
                Add item
              </Button>
            </div>

            <div className="space-y-3">
              {formData.line_items.map((item, index) => (
                <div
                  key={index}
                  className="grid grid-cols-12 items-end gap-3 rounded-lg border border-border bg-muted/40 p-4"
                >
                  <div className="col-span-12 md:col-span-5">
                    <Label className="mb-1 text-xs text-muted-foreground">
                      Description
                    </Label>
                    <Input
                      type="text"
                      value={item.description}
                      onChange={(e) =>
                        handleLineItemChange(index, "description", e.target.value)
                      }
                      placeholder="Item description"
                      required
                    />
                  </div>
                  <div className="col-span-4 md:col-span-2">
                    <Label className="mb-1 text-xs text-muted-foreground">Qty</Label>
                    <Input
                      type="number"
                      value={item.quantity}
                      onChange={(e) =>
                        handleLineItemChange(
                          index,
                          "quantity",
                          parseInt(e.target.value) || 1
                        )
                      }
                      min="1"
                      required
                    />
                  </div>
                  <div className="col-span-4 md:col-span-2">
                    <Label className="mb-1 text-xs text-muted-foreground">
                      Unit price (¢)
                    </Label>
                    <Input
                      type="number"
                      value={item.unit_price}
                      onChange={(e) =>
                        handleLineItemChange(
                          index,
                          "unit_price",
                          parseInt(e.target.value) || 0
                        )
                      }
                      min="0"
                      required
                    />
                  </div>
                  <div className="col-span-3 md:col-span-2 text-right">
                    <Label className="mb-1 block text-xs text-muted-foreground">
                      Amount
                    </Label>
                    <p className="py-2 text-sm font-medium tabular-nums text-foreground">
                      {formatCurrency(item.quantity * item.unit_price, formData.currency)}
                    </p>
                  </div>
                  <div className="col-span-1">
                    <button
                      type="button"
                      onClick={() => removeLineItem(index)}
                      disabled={formData.line_items.length === 1}
                      aria-label="Remove line item"
                      className="rounded-md p-2 text-red-500 transition-colors hover:bg-red-50 disabled:opacity-50"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>

            {/* Totals */}
            <div className="space-y-2 border-t border-border pt-4 text-right">
              <div className="flex items-center justify-end gap-8">
                <span className="text-sm text-muted-foreground">Subtotal</span>
                <span className="w-28 text-sm font-medium tabular-nums text-foreground">
                  {formatCurrency(calculateSubtotal(), formData.currency)}
                </span>
              </div>
              <div className="flex items-center justify-end gap-4">
                <span className="text-sm text-muted-foreground">Tax (¢)</span>
                <Input
                  type="number"
                  name="tax_amount"
                  value={formData.tax_amount}
                  onChange={handleChange}
                  className="w-28 text-right"
                />
              </div>
              <div className="flex items-center justify-end gap-4">
                <span className="text-sm text-muted-foreground">Discount (¢)</span>
                <Input
                  type="number"
                  name="discount_amount"
                  value={formData.discount_amount}
                  onChange={handleChange}
                  className="w-28 text-right"
                />
              </div>
              <div className="flex items-center justify-end gap-8 border-t border-border pt-2">
                <span className="text-base font-semibold text-foreground">Total</span>
                <span className="w-28 text-base font-bold tabular-nums text-primary">
                  {formatCurrency(calculateTotal(), formData.currency)}
                </span>
              </div>
            </div>
          </section>

          <Separator />

          {/* Notes & terms */}
          <section className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground">Notes &amp; terms</h3>
            <FormField label="Notes (visible to customer)" htmlFor="notes">
              <textarea
                id="notes"
                name="notes"
                value={formData.notes}
                onChange={handleChange}
                rows={3}
                placeholder="Additional notes for the customer..."
                className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
              />
            </FormField>
            <FormField label="Terms & conditions" htmlFor="terms">
              <textarea
                id="terms"
                name="terms"
                value={formData.terms}
                onChange={handleChange}
                rows={3}
                className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
              />
            </FormField>
          </section>
        </form>

        <SheetFooter>
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" form="create-quote-form" disabled={loading}>
            {loading ? "Creating..." : "Create quote"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
};

export default CreateQuote;
