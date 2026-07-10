import { useEffect, useState } from "react";
import { Pencil, Plus, Trash2, Check, X } from "lucide-react";

import { endpoints } from "../../lib/api";
import { useToast } from "../Toast";
import { cn, formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";

// Mirrors the backend's feature-key rule (internal/service/entitlement.go).
const FEATURE_KEY_RE = /^[A-Za-z0-9][A-Za-z0-9._:-]*$/;
const MAX_FEATURE_KEY_LEN = 128;

// validateRows returns an error string, or null when the set is valid.
const validateEntitlementRows = (rows) => {
  const seen = new Set();
  for (const row of rows) {
    const key = row.feature_key.trim();
    if (!key) return "Every entitlement needs a feature key.";
    if (key.length > MAX_FEATURE_KEY_LEN)
      return `Feature key "${key}" exceeds ${MAX_FEATURE_KEY_LEN} characters.`;
    if (!FEATURE_KEY_RE.test(key))
      return `Feature key "${key}" may only contain letters, numbers, and . _ : - (must start with a letter or number).`;
    if (seen.has(key)) return `Duplicate feature key "${key}".`;
    seen.add(key);
    if (row.kind === "limit") {
      if (row.limit_value === "" || row.limit_value === null)
        return `"${key}" needs a limit value.`;
      const n = Number(row.limit_value);
      if (!Number.isInteger(n) || n < 0)
        return `"${key}" limit must be a whole number ≥ 0.`;
    }
  }
  return null;
};

// toApiPayload converts editor rows into the PUT body the backend expects:
// booleans carry bool_value only, limits carry limit_value only.
const entitlementRowsToPayload = (rows) =>
  rows.map((row) =>
    row.kind === "boolean"
      ? { feature_key: row.feature_key.trim(), kind: "boolean", bool_value: row.bool_value }
      : {
          feature_key: row.feature_key.trim(),
          kind: "limit",
          limit_value: Number(row.limit_value),
        }
  );

const toEditorRows = (ents) =>
  ents.map((ent) => ({
    feature_key: ent.feature_key,
    kind: ent.kind,
    bool_value: ent.kind === "boolean" ? !!ent.bool_value : true,
    limit_value:
      ent.kind === "limit" && ent.limit_value != null ? String(ent.limit_value) : "",
  }));

export default function PlanDetail({ plan, isOpen, onClose }) {
  const toast = useToast();
  const [entitlements, setEntitlements] = useState([]);
  const [entLoadError, setEntLoadError] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [rows, setRows] = useState([]);
  const [validationError, setValidationError] = useState(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!isOpen || !plan?.id) return;
    let cancelled = false;
    setIsEditing(false);
    setValidationError(null);
    setEntLoadError(false);
    endpoints
      .getPlanEntitlements(plan.id)
      .then((res) => {
        if (!cancelled) setEntitlements(res.data?.data || []);
      })
      .catch(() => {
        if (!cancelled) {
          setEntitlements([]);
          setEntLoadError(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [isOpen, plan?.id]);

  if (!plan) return null;

  const price = plan.prices && plan.prices[0];
  const currency = price ? price.currency.toUpperCase() : "USD";

  const startEditing = () => {
    setRows(toEditorRows(entitlements));
    setValidationError(null);
    setIsEditing(true);
  };

  const cancelEditing = () => {
    setIsEditing(false);
    setValidationError(null);
  };

  const addRow = () => {
    setRows((prev) => [
      ...prev,
      { feature_key: "", kind: "boolean", bool_value: true, limit_value: "" },
    ]);
  };

  const removeRow = (index) => {
    setRows((prev) => prev.filter((_, i) => i !== index));
  };

  const updateRow = (index, patch) => {
    setRows((prev) => prev.map((row, i) => (i === index ? { ...row, ...patch } : row)));
  };

  const handleSave = async () => {
    const error = validateEntitlementRows(rows);
    if (error) {
      setValidationError(error);
      return;
    }
    setValidationError(null);
    setSaving(true);
    try {
      const res = await endpoints.setPlanEntitlements(
        plan.id,
        entitlementRowsToPayload(rows)
      );
      setEntitlements(res.data?.data || []);
      setIsEditing(false);
      toast.success("Entitlements saved");
    } catch (err) {
      toast.error(
        err?.response?.data?.error?.message || "Failed to save entitlements"
      );
    } finally {
      setSaving(false);
    }
  };

  const detail = (label, value) => (
    <div className="flex flex-col gap-1">
      <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
      <div className="text-sm text-foreground">{value}</div>
    </div>
  );

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-3">
            {plan.name}
            <Badge variant={plan.active ? "success" : "neutral"}>
              {plan.active ? "Active" : "Inactive"}
            </Badge>
          </SheetTitle>
          <SheetDescription className="font-mono text-xs">{plan.id}</SheetDescription>
        </SheetHeader>

        <div className="flex-1 space-y-8 overflow-y-auto px-6 py-6">
          {/* Details */}
          <div className="grid grid-cols-2 gap-x-4 gap-y-5">
            {detail(
              "Price",
              price ? formatCurrency(price.amount, price.currency) : "—"
            )}
            {detail(
              "Billing interval",
              <span className="capitalize">
                {plan.interval_count > 1 ? `${plan.interval_count} ` : ""}
                {plan.interval_unit}
              </span>
            )}
            {detail("Created", formatDate(plan.created_at))}
            {detail("Currency", currency)}
            {detail(
              "Code",
              <span className="font-mono text-foreground">{plan.code}</span>
            )}
          </div>

          {/* Entitlements */}
          <div>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-foreground">Entitlements</h3>
              {!isEditing && (
                <Button variant="outline" size="sm" onClick={startEditing}>
                  <Pencil className="h-3.5 w-3.5" />
                  Edit
                </Button>
              )}
            </div>

            {isEditing ? (
              <div className="flex flex-col gap-3">
                {rows.length === 0 && (
                  <p className="text-sm text-muted-foreground">
                    No entitlements. Add one below — saving an empty list removes all
                    entitlements from this plan.
                  </p>
                )}
                {rows.map((row, index) => (
                  <div
                    key={index}
                    data-testid={`entitlement-row-${index}`}
                    className="flex items-center gap-2 rounded-lg border border-border bg-muted/40 p-3"
                  >
                    <div className="min-w-0 flex-1">
                      <Input
                        type="text"
                        value={row.feature_key}
                        onChange={(e) => updateRow(index, { feature_key: e.target.value })}
                        placeholder="feature_key (e.g. api.calls)"
                        aria-label={`Feature key ${index + 1}`}
                        className="h-8 font-mono"
                      />
                    </div>
                    <select
                      value={row.kind}
                      onChange={(e) => updateRow(index, { kind: e.target.value })}
                      aria-label={`Kind ${index + 1}`}
                      className="h-8 rounded-md border border-input bg-transparent px-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      <option value="boolean">boolean</option>
                      <option value="limit">limit</option>
                    </select>
                    {row.kind === "boolean" ? (
                      <label className="flex cursor-pointer items-center gap-1.5 whitespace-nowrap text-sm text-foreground">
                        <input
                          type="checkbox"
                          checked={row.bool_value}
                          onChange={(e) => updateRow(index, { bool_value: e.target.checked })}
                          aria-label={`Enabled ${index + 1}`}
                          className="rounded border-input text-primary focus:ring-ring"
                        />
                        Enabled
                      </label>
                    ) : (
                      <Input
                        type="number"
                        min="0"
                        step="1"
                        value={row.limit_value}
                        onChange={(e) => updateRow(index, { limit_value: e.target.value })}
                        placeholder="Limit"
                        aria-label={`Limit value ${index + 1}`}
                        className="h-8 w-24 flex-none"
                      />
                    )}
                    <button
                      type="button"
                      onClick={() => removeRow(index)}
                      aria-label={`Remove entitlement ${index + 1}`}
                      className="flex-none text-stone-400 transition-colors hover:text-red-500"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}

                {validationError && (
                  <p role="alert" className="text-sm text-red-600">
                    {validationError}
                  </p>
                )}

                <div className="flex items-center justify-between">
                  <Button variant="ghost" size="sm" onClick={addRow}>
                    <Plus className="h-4 w-4" />
                    Add entitlement
                  </Button>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={cancelEditing}
                      disabled={saving}
                    >
                      Cancel
                    </Button>
                    <Button size="sm" onClick={handleSave} disabled={saving}>
                      {saving ? "Saving…" : "Save entitlements"}
                    </Button>
                  </div>
                </div>
              </div>
            ) : entLoadError ? (
              <p className="text-sm text-red-600">Failed to load entitlements.</p>
            ) : entitlements.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No entitlements configured for this plan.
              </p>
            ) : (
              <div className="flex flex-col gap-2">
                {entitlements.map((ent) => {
                  const disabled = ent.kind === "boolean" && !ent.bool_value;
                  return (
                    <div key={ent.feature_key} className="flex items-center gap-3">
                      {disabled ? (
                        <X className="h-4 w-4 text-stone-400" />
                      ) : (
                        <Check className="h-4 w-4 text-emerald-500" />
                      )}
                      <p className="font-mono text-sm text-foreground">
                        {ent.feature_key}
                      </p>
                      {ent.kind === "limit" && (
                        <span className="text-xs text-muted-foreground">
                          limit: {ent.limit_value?.toLocaleString()}
                        </span>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Metadata */}
          <div>
            <h3 className="mb-4 text-sm font-semibold text-foreground">Metadata</h3>
            <div className="overflow-x-auto rounded-lg bg-muted p-4">
              <pre className={cn("font-mono text-xs text-foreground")}>
                {JSON.stringify(plan.metadata || {}, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
