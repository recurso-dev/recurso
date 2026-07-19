import { useState } from "react";
import { Pencil, Archive, ArchiveRestore } from "lucide-react";

import { endpoints } from "../../lib/api";
import { formatDate } from "@/lib/utils";
import { toast } from "@/components/ui/sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Input } from "@/components/ui/input";
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

// A titled group of fields, rendered only when it has at least one value.
const Section = ({ title, children }) => (
  <div className="space-y-4">
    <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70">
      {title}
    </p>
    <dl className="space-y-4">{children}</dl>
  </div>
);

// A labelled input inside the edit form.
const EditField = ({ label, value, onChange, mono, placeholder, type = "text" }) => (
  <div>
    <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
      {label}
    </p>
    <Input
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className={mono ? "font-mono" : undefined}
      aria-label={label}
    />
  </div>
);

// Join the non-empty parts of a billing address into a readable block.
const formatAddress = (addr) => {
  if (!addr) return null;
  const parts = [addr.line1, addr.line2, addr.city, addr.state, addr.zip, addr.country]
    .map((p) => (p || "").trim())
    .filter(Boolean);
  return parts.length ? parts.join(", ") : null;
};

const CustomerDetail = ({ customer, isOpen, onClose, onChanged }) => {
  const [isEditing, setIsEditing] = useState(false);
  const [form, setForm] = useState(null);
  const [saving, setSaving] = useState(false);
  const [archiveOpen, setArchiveOpen] = useState(false);
  const [archiving, setArchiving] = useState(false);

  if (!customer) return null;

  const risk = customer.churn_risk ?? customer.risk_score ?? null;
  const activeSubs = customer.activeSubs ?? customer.active_subscriptions ?? 0;
  const address = formatAddress(customer.billing_address);
  const hasBilling =
    address || customer.tax_type || customer.gstin || customer.place_of_supply;
  // active may be absent on stale list rows; treat missing as active.
  const isArchived = customer.active === false;

  const startEdit = () => {
    setForm({
      name: customer.name || "",
      email: customer.email || "",
      phone: customer.phone || "",
      gstin: customer.gstin || "",
      place_of_supply: customer.place_of_supply || "",
      line1: customer.billing_address?.line1 || "",
      city: customer.billing_address?.city || "",
      state: customer.billing_address?.state || "",
      zip: customer.billing_address?.zip || "",
      country: customer.billing_address?.country || "",
    });
    setIsEditing(true);
  };

  const save = async () => {
    setSaving(true);
    try {
      const res = await endpoints.updateCustomer(customer.id, form);
      setIsEditing(false);
      toast.success("Customer updated");
      onChanged?.(res.data?.data);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to update customer");
    } finally {
      setSaving(false);
    }
  };

  const toggleArchive = async () => {
    setArchiving(true);
    try {
      const res = await endpoints.updateCustomer(customer.id, { active: isArchived });
      setArchiveOpen(false);
      toast.success(isArchived ? "Customer restored" : "Customer archived");
      onChanged?.(res.data?.data);
    } catch (err) {
      // Most common: the backend's archive gate (active subscriptions).
      toast.error(err?.response?.data?.error?.message || "Failed to update customer");
    } finally {
      setArchiving(false);
    }
  };

  const set = (key) => (v) => setForm((f) => ({ ...f, [key]: v }));

  return (
    <>
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Customer details</SheetTitle>
        </SheetHeader>

        <div className="space-y-6 px-6 py-6">
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="truncate text-base font-semibold text-foreground">
                {customer.name || "Unnamed customer"}
              </p>
              <p className="truncate text-sm text-muted-foreground">
                {customer.email}
              </p>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              {isArchived && <Badge variant="neutral">Archived</Badge>}
              <Badge variant={activeSubs > 0 ? "success" : "neutral"}>
                {activeSubs} active
              </Badge>
            </div>
          </div>

          {!isEditing && (
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={startEdit}>
                <Pencil className="h-3.5 w-3.5" />
                Edit customer
              </Button>
              <Button variant="outline" size="sm" onClick={() => setArchiveOpen(true)}>
                {isArchived ? (
                  <ArchiveRestore className="h-3.5 w-3.5" />
                ) : (
                  <Archive className="h-3.5 w-3.5" />
                )}
                {isArchived ? "Restore" : "Archive"}
              </Button>
            </div>
          )}

          <Separator />

          {isEditing ? (
            <div className="flex flex-col gap-4 rounded-lg border border-border bg-muted/40 p-4">
              <EditField label="Name" value={form.name} onChange={set("name")} />
              <EditField label="Email" type="email" value={form.email} onChange={set("email")} />
              <EditField label="Phone" value={form.phone} onChange={set("phone")} />
              <EditField label="GSTIN" mono value={form.gstin} onChange={set("gstin")} placeholder="Optional" />
              <EditField
                label="Place of supply"
                value={form.place_of_supply}
                onChange={set("place_of_supply")}
                placeholder="State code"
              />
              <EditField label="Address line" value={form.line1} onChange={set("line1")} />
              <div className="grid grid-cols-2 gap-3">
                <EditField label="City" value={form.city} onChange={set("city")} />
                <EditField label="State" value={form.state} onChange={set("state")} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <EditField label="ZIP" value={form.zip} onChange={set("zip")} />
                <EditField
                  label="Country (ISO-2)"
                  value={form.country}
                  onChange={(v) => set("country")(v.toUpperCase())}
                  placeholder="IN"
                />
              </div>
              <div className="flex justify-end gap-2">
                <Button variant="outline" size="sm" onClick={() => setIsEditing(false)} disabled={saving}>
                  Cancel
                </Button>
                <Button size="sm" onClick={save} disabled={saving || !form.email.trim()}>
                  {saving ? "Saving…" : "Save customer"}
                </Button>
              </div>
            </div>
          ) : (
            <>
              {/* Contact */}
              <Section title="Contact">
                <Field label="Email">{customer.email || "—"}</Field>
                <Field label="Phone">{customer.phone?.trim() || "—"}</Field>
              </Section>

              {/* Billing & tax — only when there's something to show */}
              {hasBilling && (
                <>
                  <Separator />
                  <Section title="Billing & tax">
                    {address && <Field label="Billing address">{address}</Field>}
                    {customer.tax_type && (
                      <Field label="Tax type">
                        <span className="capitalize">{customer.tax_type}</span>
                      </Field>
                    )}
                    {customer.gstin && (
                      <Field label="GSTIN" mono>
                        {customer.gstin}
                      </Field>
                    )}
                    {customer.place_of_supply && (
                      <Field label="Place of supply">{customer.place_of_supply}</Field>
                    )}
                  </Section>
                </>
              )}

              <Separator />

              {/* Record */}
              <Section title="Record">
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
                {customer.referral_code && (
                  <Field label="Referral code" mono>
                    {customer.referral_code}
                  </Field>
                )}
              </Section>
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>

      <ConfirmDialog
        open={archiveOpen}
        onOpenChange={setArchiveOpen}
        title={isArchived ? "Restore this customer?" : "Archive this customer?"}
        description={
          isArchived
            ? "The customer becomes available for new subscriptions again."
            : "Archiving is blocked while the customer has active subscriptions. Billing history is kept and the customer can be restored at any time."
        }
        confirmLabel={isArchived ? "Restore customer" : "Archive customer"}
        destructive={!isArchived}
        busy={archiving}
        onConfirm={toggleArchive}
      />
    </>
  );
};

export default CustomerDetail;
