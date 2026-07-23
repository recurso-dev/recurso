import { useEffect, useState } from "react";
import { Pencil, Archive, ArchiveRestore } from "lucide-react";

import { endpoints } from "../../lib/api";
import { formatDate, cn, fromMinorUnits, currencyDecimals } from "@/lib/utils";
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

const fmtMoney = (minor, currency) => {
  const d = currencyDecimals(currency);
  return `${fromMinorUnits(minor, currency).toLocaleString(undefined, {
    minimumFractionDigits: d,
    maximumFractionDigits: d,
  })} ${currency}`;
};

// Map a credit-note status to a Badge variant.
const creditStatusVariant = (status) =>
  ({
    issued: "success",
    used: "neutral",
    void: "neutral",
    pending_approval: "warning",
    rejected: "destructive",
    expired: "warning",
  })[status] || "neutral";

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
  const [consents, setConsents] = useState([]);
  const [consentsLoading, setConsentsLoading] = useState(false);
  const [revokingId, setRevokingId] = useState(null);
  const [credit, setCredit] = useState(null);

  const custId = customer?.id;
  useEffect(() => {
    if (!isOpen || !custId) return;
    let cancelled = false;
    setConsentsLoading(true);
    endpoints
      .getCustomerConsents(custId)
      .then((res) => !cancelled && setConsents(res.data?.data || []))
      .catch(() => !cancelled && setConsents([]))
      .finally(() => !cancelled && setConsentsLoading(false));
    // Account-credit statement (ledger-backed credits).
    setCredit(null);
    endpoints
      .getCreditStatement(custId)
      .then((res) => !cancelled && setCredit(res.data?.data || null))
      .catch(() => !cancelled && setCredit(null));
    return () => {
      cancelled = true;
    };
  }, [isOpen, custId]);

  // Export the statement's grants + applications as a CSV the customer's finance
  // team can reconcile against their own books.
  const exportCreditCsv = () => {
    if (!credit) return;
    const esc = (v) => `"${String(v ?? "").replace(/"/g, '""')}"`;
    const rows = [["kind", "date", "reason_or_invoice", "currency", "type_or_status", "amount", "balance"]];
    (credit.grants || []).forEach((g) =>
      rows.push(["grant", g.created_at, g.reason || "", g.currency, `${g.type}/${g.status}`, g.amount, g.balance]),
    );
    (credit.applications || []).forEach((a) =>
      rows.push(["application", a.created_at, a.invoice_number || a.invoice_id, a.currency, "applied", -a.amount, ""]),
    );
    const csv = rows.map((r) => r.map(esc).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `credit-statement-${customer?.name || custId}.csv`;
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
  };

  const revokeConsent = async (id) => {
    setRevokingId(id);
    try {
      await endpoints.revokeConsent(id);
      toast.success("Consent revoked.");
      const res = await endpoints.getCustomerConsents(custId);
      setConsents(res.data?.data || []);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to revoke consent");
    } finally {
      setRevokingId(null);
    }
  };

  if (!customer) return null;

  const risk = customer.churn_risk ?? customer.risk_score ?? null;
  // The customers list serves the count as active_subs; older callers used
  // activeSubs / active_subscriptions. Read all three or the badge shows 0.
  const activeSubs =
    customer.active_subs ?? customer.activeSubs ?? customer.active_subscriptions ?? 0;
  const address = formatAddress(customer.billing_address);
  const hasBilling =
    address ||
    customer.tax_type ||
    customer.gstin ||
    customer.place_of_supply ||
    customer.tax_exempt;
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
      tax_exempt: customer.tax_exempt || false,
      tax_exemption_number: customer.tax_exemption_number || "",
      tax_exemption_code: customer.tax_exemption_code || "",
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
              <div className="flex items-center justify-between rounded-md border border-border bg-background px-3 py-2">
                <span className="text-sm">Tax exempt (US)</span>
                <button
                  type="button"
                  role="switch"
                  aria-checked={form.tax_exempt}
                  onClick={() => set("tax_exempt")(!form.tax_exempt)}
                  className={cn(
                    "relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                    form.tax_exempt ? "bg-primary" : "bg-stone-200",
                  )}
                >
                  <span
                    className={cn(
                      "pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out",
                      form.tax_exempt ? "translate-x-4" : "translate-x-0",
                    )}
                  />
                </button>
              </div>
              {form.tax_exempt && (
                <div className="grid grid-cols-2 gap-3">
                  <EditField
                    label="Exemption number"
                    value={form.tax_exemption_number}
                    onChange={set("tax_exemption_number")}
                    placeholder="RESALE-0001"
                  />
                  <EditField
                    label="Entity-use code"
                    mono
                    value={form.tax_exemption_code}
                    onChange={(v) => set("tax_exemption_code")(v.toUpperCase())}
                    placeholder="A"
                  />
                </div>
              )}
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
                    {customer.tax_exempt && (
                      <Field label="Tax exempt">
                        Yes
                        {customer.tax_exemption_number
                          ? ` · ${customer.tax_exemption_number}`
                          : ""}
                        {customer.tax_exemption_code
                          ? ` (${customer.tax_exemption_code})`
                          : ""}
                      </Field>
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

              {/* Account credit (ledger-backed credits) */}
              {credit && ((credit.balances?.length || 0) > 0 || (credit.grants?.length || 0) > 0) && (
                <>
                  <Separator />
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70">
                        Account credit
                      </p>
                      <Button variant="outline" size="sm" onClick={exportCreditCsv}>
                        Export CSV
                      </Button>
                    </div>

                    {/* Spendable balance per currency */}
                    {(credit.balances?.length || 0) > 0 ? (
                      <div className="flex flex-wrap gap-2">
                        {credit.balances.map((b, i) => (
                          <Badge key={i} variant="success">
                            {fmtMoney(b.balance, b.currency)} available
                          </Badge>
                        ))}
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">No spendable balance.</p>
                    )}

                    {/* Grants */}
                    {(credit.grants?.length || 0) > 0 && (
                      <div className="space-y-1.5">
                        <p className="text-xs font-medium text-muted-foreground">Grants</p>
                        {credit.grants.map((g) => (
                          <div
                            key={g.id}
                            className="flex items-center justify-between gap-3 border-b border-border py-1.5 last:border-0"
                          >
                            <div className="min-w-0">
                              <p className="truncate text-sm text-foreground">{g.reason || "Credit"}</p>
                              <div className="flex items-center gap-2">
                                <Badge variant={creditStatusVariant(g.status)}>{g.status}</Badge>
                                <span className="text-xs text-muted-foreground capitalize">{g.type}</span>
                              </div>
                            </div>
                            <div className="shrink-0 text-right tabular-nums">
                              <p className="text-sm text-foreground">{fmtMoney(g.amount, g.currency)}</p>
                              <p className="text-xs text-muted-foreground">
                                {fmtMoney(g.balance, g.currency)} left
                              </p>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}

                    {/* Draw-down history */}
                    {(credit.applications?.length || 0) > 0 && (
                      <div className="space-y-1.5">
                        <p className="text-xs font-medium text-muted-foreground">Applied to invoices</p>
                        {credit.applications.map((a, i) => (
                          <div key={i} className="flex items-center justify-between gap-3 text-sm">
                            <span className="truncate font-mono text-xs text-muted-foreground">
                              {a.invoice_number || a.invoice_id}
                            </span>
                            <span className="tabular-nums text-foreground">
                              −{fmtMoney(a.amount, a.currency)}
                            </span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </>
              )}

              {/* Consent audit trail (GDPR) — view and revoke */}
              {(consentsLoading || consents.length > 0) && (
                <>
                  <Separator />
                  <Section title="Consent">
                    {consentsLoading ? (
                      <p className="text-sm text-muted-foreground">Loading…</p>
                    ) : (
                      <div className="space-y-2">
                        {consents.map((c) => {
                          const active = c.granted && !c.revoked_at;
                          return (
                            <div
                              key={c.id}
                              className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
                            >
                              <div className="min-w-0">
                                <div className="flex items-center gap-2">
                                  <span className="truncate text-sm font-medium text-foreground">
                                    {String(c.consent_type || "consent").replace(/_/g, " ")}
                                  </span>
                                  <Badge variant={active ? "success" : "neutral"}>
                                    {active ? "granted" : "revoked"}
                                  </Badge>
                                </div>
                                <p className="text-xs text-muted-foreground">
                                  {active
                                    ? `Granted ${c.granted_at ? formatDate(c.granted_at) : "—"}`
                                    : `Revoked ${c.revoked_at ? formatDate(c.revoked_at) : "—"}`}
                                  {c.version ? ` · v${c.version}` : ""}
                                </p>
                              </div>
                              {active && (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  disabled={revokingId === c.id}
                                  onClick={() => revokeConsent(c.id)}
                                >
                                  {revokingId === c.id ? "Revoking…" : "Revoke"}
                                </Button>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </Section>
                </>
              )}
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
