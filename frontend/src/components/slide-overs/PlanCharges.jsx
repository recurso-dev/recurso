import { useEffect, useMemo, useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";

import { endpoints } from "../../lib/api";
import { useToast } from "../Toast";
import { formatCurrency } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

// Usage-charge editor for a plan (roadmap: plan-charges visual editor). Mirrors
// the backend charge models in internal/core/domain/metering.go and the
// PUT-replace semantics of PlanCharges (whole set replaced per save, like
// entitlements). Money entry follows the house convention: unit rates are
// decimal strings in MAJOR units (sub-paise rates are first-class), while
// package/flat amounts are entered in major units and stored as int64 minor
// units (×100), matching CreatePlan's price handling.

const MODELS = [
  { value: "per_unit", label: "Per unit" },
  { value: "graduated", label: "Graduated (tiered)" },
  { value: "volume", label: "Volume (whole-qty tier)" },
  { value: "package", label: "Package (bundles)" },
  { value: "percentage", label: "Percentage (of value)" },
  { value: "graduated_percentage", label: "Graduated percentage" },
  { value: "dynamic", label: "Dynamic (per-event price)" },
];

const MODEL_LABEL = Object.fromEntries(MODELS.map((m) => [m.value, m.label]));

// Tier shape covers both per-unit tiers (graduated/volume: unit_amount) and
// percentage tiers (graduated_percentage: rate). Only the relevant field is
// shown/sent per model.
const blankTier = () => ({ up_to: "", unit_amount: "", rate: "", flat_amount: "" });

const isTierModel = (m) => m === "graduated" || m === "volume" || m === "graduated_percentage";

const newRow = (metricId = "") => ({
  metric_id: metricId,
  charge_model: "per_unit",
  unit_amount: "",
  package_amount: "",
  package_size: "",
  tiers: [blankTier()],
  // percentage fields (money fields entered in major units, stored ×100)
  rate: "",
  fixed_amount: "",
  free_amount: "",
  min_amount: "",
  max_amount: "",
  // A4 dimensional pricing (per_unit/percentage only in the editor): a property
  // key + per-value rates. Each filter: { value, amount } (amount is the
  // per-value unit_amount for per_unit, or rate for percentage).
  filter_key: "",
  filters: [],
  pay_in_advance: false,
  hsn_code: "",
});

// filterEligible: the editor supports dimensional pricing for the single-field
// models. Tier/dynamic filters are API-only in v1.
const filterEligible = (m) => m === "per_unit" || m === "percentage";
const blankFilter = () => ({ value: "", amount: "" });

// payInAdvanceEligible mirrors the backend: only non-cumulative models can be
// billed per event (per_unit, percentage, dynamic).
const payInAdvanceEligible = (m) => m === "per_unit" || m === "percentage" || m === "dynamic";
>>>>>>> origin/main

// toEditorRows converts loaded charges (Amounts keyed by currency) into flat
// editor rows for the plan's currency. Amounts for other currencies are not
// shown in v1 — the plan has a single price currency.
const toEditorRows = (charges, currency) =>
  charges.map((ch) => {
    const a = (ch.amounts && (ch.amounts[currency] || ch.amounts[currency?.toUpperCase()])) || {};
    const row = newRow(ch.metric_id);
    row.charge_model = ch.charge_model;
    row.hsn_code = ch.hsn_code || "";
    row.pay_in_advance = !!ch.pay_in_advance;
    if (ch.filter_key && filterEligible(ch.charge_model)) {
      row.filter_key = ch.filter_key;
      row.filters = (ch.filters || []).map((f) => {
        const fa = (f.amounts && (f.amounts[currency] || f.amounts[currency?.toUpperCase()])) || {};
        return { value: f.value || "", amount: (ch.charge_model === "percentage" ? fa.rate : fa.unit_amount) || "" };
      });
    }
    if (ch.charge_model === "per_unit") {
      row.unit_amount = a.unit_amount || "";
    } else if (ch.charge_model === "package") {
      row.package_amount = a.package_amount != null ? String(a.package_amount / 100) : "";
      row.package_size = a.package_size != null ? String(a.package_size) : "";
    } else if (ch.charge_model === "percentage") {
      row.rate = a.rate || "";
      row.fixed_amount = a.fixed_amount ? String(a.fixed_amount / 100) : "";
      row.free_amount = a.free_units ? String(a.free_units / 100) : "";
      row.min_amount = a.min_amount ? String(a.min_amount / 100) : "";
      row.max_amount = a.max_amount ? String(a.max_amount / 100) : "";
    } else if (ch.charge_model === "dynamic") {
      // No pricing config: the price arrives per event.
    } else {
      // graduated / volume (unit_amount) or graduated_percentage (rate).
      // graduated_percentage bands the monetary base, so its up_to is money
      // (stored minor units); per-unit tiers band unit counts.
      const isPct = ch.charge_model === "graduated_percentage";
      row.tiers = (a.tiers && a.tiers.length ? a.tiers : [blankTier()]).map((t) => ({
        up_to: t.up_to != null ? String(isPct ? t.up_to / 100 : t.up_to) : "",
        unit_amount: t.unit_amount || "",
        rate: t.rate || "",
        flat_amount: t.flat_amount ? String(t.flat_amount / 100) : "",
      }));
    }
    return row;
  });

// isDecimal reports whether s is a non-negative decimal (rates may be sub-paise
// like "0.0035"); mirrors the backend's parseRate acceptance loosely.
const isDecimal = (s) => s !== "" && s != null && /^\d*\.?\d+$/.test(String(s).trim());
const isNonNegMoney = (s) => s !== "" && s != null && Number(s) >= 0 && !Number.isNaN(Number(s));

// validateRows returns an error string or null. Mirrors SetPlanCharges +
// validateTiers so a config that passes here won't be rejected server-side.
const validateRows = (rows) => {
  const seen = new Set();
  for (let i = 0; i < rows.length; i++) {
    const r = rows[i];
    const n = i + 1;
    if (!r.metric_id) return `Charge ${n}: pick a metric.`;
    if (seen.has(r.metric_id)) return `Charge ${n}: a metric can only be charged once per plan.`;
    seen.add(r.metric_id);

    if (r.charge_model === "per_unit") {
      if (!isDecimal(r.unit_amount)) return `Charge ${n}: enter a valid per-unit rate.`;
    } else if (r.charge_model === "package") {
      if (!isNonNegMoney(r.package_amount)) return `Charge ${n}: enter a valid package price.`;
      if (!(Number(r.package_size) >= 1 && Number.isInteger(Number(r.package_size))))
        return `Charge ${n}: package size must be a whole number ≥ 1.`;
    } else if (r.charge_model === "percentage") {
      if (!isDecimal(r.rate)) return `Charge ${n}: enter a valid percentage rate.`;
      for (const [field, label] of [
        ["fixed_amount", "fixed fee"],
        ["free_amount", "free amount"],
        ["min_amount", "minimum"],
        ["max_amount", "maximum"],
      ]) {
        if (r[field] !== "" && !isNonNegMoney(r[field]))
          return `Charge ${n}: ${label} must be ≥ 0.`;
      }
      if (r.min_amount !== "" && r.max_amount !== "" && Number(r.max_amount) < Number(r.min_amount))
        return `Charge ${n}: maximum must be ≥ minimum.`;
    } else if (r.charge_model === "dynamic") {
      // No pricing config to validate — the price is supplied per event.
    } else {
      // graduated / volume (unit_amount) or graduated_percentage (rate).
      const isPct = r.charge_model === "graduated_percentage";
      const tiers = r.tiers;
      if (!tiers.length) return `Charge ${n}: add at least one tier.`;
      let prev = 0;
      for (let t = 0; t < tiers.length; t++) {
        const tier = tiers[t];
        const isLast = t === tiers.length - 1;
        if (isLast) {
          if (tier.up_to !== "") return `Charge ${n}: the last tier must be unbounded (leave "up to" empty).`;
        } else {
          // per-unit tiers band whole unit counts; percentage tiers band a
          // money value (major units, decimal ok).
          const bad = isPct ? !isNonNegMoney(tier.up_to) : !Number.isInteger(Number(tier.up_to)) || tier.up_to === "";
          if (bad) return `Charge ${n}, tier ${t + 1}: "up to" must be a ${isPct ? "valid amount" : "whole number"}.`;
          if (Number(tier.up_to) <= prev)
            return `Charge ${n}, tier ${t + 1}: "up to" must increase down the tiers.`;
          prev = Number(tier.up_to);
        }
        if (!isDecimal(isPct ? tier.rate : tier.unit_amount))
          return `Charge ${n}, tier ${t + 1}: enter a valid ${isPct ? "percentage" : "rate"}.`;
        if (tier.flat_amount !== "" && !isNonNegMoney(tier.flat_amount))
          return `Charge ${n}, tier ${t + 1}: flat fee must be ≥ 0.`;
      }
    }

    // A4 dimensional pricing (per_unit/percentage): validate filter values + rates.
    if (filterEligible(r.charge_model) && r.filter_key.trim()) {
      if (!r.filters.length) return `Charge ${n}: add at least one filter value, or clear the filter property.`;
      const seenVals = new Set();
      for (let f = 0; f < r.filters.length; f++) {
        const fv = r.filters[f];
        if (!fv.value.trim()) return `Charge ${n}, filter ${f + 1}: enter a property value.`;
        if (seenVals.has(fv.value.trim())) return `Charge ${n}, filter ${f + 1}: duplicate value.`;
        seenVals.add(fv.value.trim());
        if (!isDecimal(fv.amount)) return `Charge ${n}, filter ${f + 1}: enter a valid rate.`;
      }
    }
  }
  return null;
};

// rowsToPayload builds the PUT body: one ChargeInput per row with an Amounts
// map keyed by the plan currency.
const rowsToPayload = (rows, currency) =>
  rows.map((r) => {
    const money = (s) => (s === "" ? 0 : Math.round(Number(s) * 100));
    let amounts;
    if (r.charge_model === "per_unit") {
      amounts = { unit_amount: String(r.unit_amount).trim() };
    } else if (r.charge_model === "package") {
      amounts = {
        package_amount: money(r.package_amount),
        package_size: Number(r.package_size),
      };
    } else if (r.charge_model === "percentage") {
      amounts = {
        rate: String(r.rate).trim(),
        fixed_amount: money(r.fixed_amount),
        free_units: money(r.free_amount),
        min_amount: money(r.min_amount),
        max_amount: money(r.max_amount),
      };
    } else if (r.charge_model === "dynamic") {
      // The price is supplied per event; the charge carries no pricing config.
      // Still keyed by currency so the plan's currency selects this entry.
      amounts = {};
    } else {
      const isPct = r.charge_model === "graduated_percentage";
      amounts = {
        tiers: r.tiers.map((t, idx) => {
          const last = idx === r.tiers.length - 1 || t.up_to === "";
          const tier = {
            // per-unit tiers band unit counts; percentage tiers band money (×100).
            up_to: last ? null : isPct ? money(t.up_to) : Number(t.up_to),
            flat_amount: money(t.flat_amount),
          };
          if (isPct) tier.rate = String(t.rate).trim();
          else tier.unit_amount = String(t.unit_amount).trim();
          return tier;
        }),
      };
    }
    const payload = {
      metric_id: r.metric_id,
      charge_model: r.charge_model,
      amounts: { [currency]: amounts },
      // only send pay_in_advance for eligible models (backend rejects the rest)
      pay_in_advance: payInAdvanceEligible(r.charge_model) ? !!r.pay_in_advance : false,
      hsn_code: r.hsn_code.trim(),
    };
    // A4: dimensional pricing — one amounts entry per filter value.
    if (filterEligible(r.charge_model) && r.filter_key.trim() && r.filters.length) {
      const field = r.charge_model === "percentage" ? "rate" : "unit_amount";
      payload.filter_key = r.filter_key.trim();
      payload.filters = r.filters.map((f) => ({
        value: f.value.trim(),
        amounts: { [currency]: { [field]: String(f.amount).trim() } },
      }));
    }
    return payload;
  });

// chargeSummary renders a compact read-only description of a charge's pricing.
function chargeSummary(ch, currency) {
  const a = (ch.amounts && (ch.amounts[currency] || ch.amounts[currency?.toUpperCase()])) || {};
  if (ch.charge_model === "per_unit") {
    return `${a.unit_amount ?? "—"} ${currency}/unit`;
  }
  if (ch.charge_model === "package") {
    return `${formatCurrency(a.package_amount, currency)} per ${a.package_size} units`;
  }
  if (ch.charge_model === "percentage") {
    const fee = a.fixed_amount ? ` + ${formatCurrency(a.fixed_amount, currency)} fee` : "";
    return `${a.rate ?? "—"}% of value${fee}`;
  }
  if (ch.charge_model === "dynamic") {
    return "priced per event";
  }
  const count = (a.tiers || []).length;
  return `${count} tier${count === 1 ? "" : "s"}`;
}

export default function PlanCharges({ planId, currency }) {
  const toast = useToast();
  const [charges, setCharges] = useState([]);
  const [metrics, setMetrics] = useState([]);
  const [loadError, setLoadError] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [rows, setRows] = useState([]);
  const [validationError, setValidationError] = useState(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!planId) return;
    let cancelled = false;
    setIsEditing(false);
    setLoadError(false);
    Promise.all([endpoints.getPlanCharges(planId), endpoints.getBillableMetrics()])
      .then(([chRes, mRes]) => {
        if (cancelled) return;
        setCharges(chRes.data?.data || []);
        setMetrics(mRes.data?.data || []);
      })
      .catch(() => {
        if (!cancelled) {
          setCharges([]);
          setLoadError(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [planId]);

  const metricName = useMemo(() => {
    const map = {};
    metrics.forEach((m) => {
      map[m.id] = m.name || m.code;
    });
    return map;
  }, [metrics]);

  const startEditing = () => {
    setRows(toEditorRows(charges, currency));
    setValidationError(null);
    setIsEditing(true);
  };

  const updateRow = (i, patch) =>
    setRows((prev) => prev.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  const addRow = () => setRows((prev) => [...prev, newRow()]);
  const removeRow = (i) => setRows((prev) => prev.filter((_, idx) => idx !== i));

  const updateTier = (ri, ti, patch) =>
    setRows((prev) =>
      prev.map((r, idx) =>
        idx === ri
          ? { ...r, tiers: r.tiers.map((t, tIdx) => (tIdx === ti ? { ...t, ...patch } : t)) }
          : r
      )
    );
  const addTier = (ri) =>
    setRows((prev) =>
      prev.map((r, idx) =>
        idx === ri
          ? // new unbounded tier goes last; the previously-last tier gains a bound of ""
            { ...r, tiers: [...r.tiers, blankTier()] }
          : r
      )
    );
  const removeTier = (ri, ti) =>
    setRows((prev) =>
      prev.map((r, idx) =>
        idx === ri && r.tiers.length > 1
          ? { ...r, tiers: r.tiers.filter((_, tIdx) => tIdx !== ti) }
          : r
      )
    );

  // A4 dimensional-pricing filter helpers.
  const updateFilter = (ri, fi, patch) =>
    setRows((prev) =>
      prev.map((r, idx) =>
        idx === ri ? { ...r, filters: r.filters.map((f, fIdx) => (fIdx === fi ? { ...f, ...patch } : f)) } : r
      )
    );
  const addFilter = (ri) =>
    setRows((prev) => prev.map((r, idx) => (idx === ri ? { ...r, filters: [...r.filters, blankFilter()] } : r)));
  const removeFilter = (ri, fi) =>
    setRows((prev) =>
      prev.map((r, idx) => (idx === ri ? { ...r, filters: r.filters.filter((_, fIdx) => fIdx !== fi) } : r))
    );

  const handleSave = async () => {
    const error = validateRows(rows);
    if (error) {
      setValidationError(error);
      return;
    }
    setValidationError(null);
    setSaving(true);
    try {
      const res = await endpoints.setPlanCharges(planId, rowsToPayload(rows, currency));
      setCharges(res.data?.data || []);
      setIsEditing(false);
      toast.success("Usage charges saved");
    } catch (err) {
      setValidationError(
        err?.response?.data?.error?.message || "Failed to save charges"
      );
    } finally {
      setSaving(false);
    }
  };

  const selectCls =
    "h-8 rounded-md border border-input bg-transparent px-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">Usage charges</h3>
        {!isEditing && (
          <Button variant="outline" size="sm" onClick={startEditing} disabled={loadError}>
            <Pencil className="h-3.5 w-3.5" />
            Edit charges
          </Button>
        )}
      </div>

      {isEditing ? (
        <div className="flex flex-col gap-4">
          {metrics.length === 0 && (
            <p className="rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-800">
              No billable metrics yet. Create one on the Metering page before adding usage
              charges.
            </p>
          )}
          {rows.length === 0 && metrics.length > 0 && (
            <p className="text-sm text-muted-foreground">
              No usage charges. Add one below — saving an empty list removes all usage charges
              from this plan.
            </p>
          )}

          {rows.map((row, i) => (
            <div
              key={i}
              data-testid={`charge-row-${i}`}
              className="flex flex-col gap-3 rounded-lg border border-border bg-muted/40 p-3"
            >
              <div className="flex items-center gap-2">
                <select
                  value={row.metric_id}
                  onChange={(e) => updateRow(i, { metric_id: e.target.value })}
                  aria-label={`Metric ${i + 1}`}
                  className={selectCls + " min-w-0 flex-1"}
                >
                  <option value="">Select metric…</option>
                  {metrics.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.name || m.code}
                    </option>
                  ))}
                </select>
                <select
                  value={row.charge_model}
                  onChange={(e) => updateRow(i, { charge_model: e.target.value })}
                  aria-label={`Charge model ${i + 1}`}
                  className={selectCls}
                >
                  {MODELS.map((m) => (
                    <option key={m.value} value={m.value}>
                      {m.label}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={() => removeRow(i)}
                  aria-label={`Remove charge ${i + 1}`}
                  className="flex-none text-stone-400 transition-colors hover:text-red-500"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>

              {row.charge_model === "per_unit" && (
                <div className="flex items-center gap-2">
                  <label className="text-xs text-muted-foreground">Rate per unit ({currency})</label>
                  <Input
                    value={row.unit_amount}
                    onChange={(e) => updateRow(i, { unit_amount: e.target.value })}
                    placeholder="0.0035"
                    inputMode="decimal"
                    aria-label={`Per-unit rate ${i + 1}`}
                    className="h-8 w-32 font-mono"
                  />
                </div>
              )}

              {row.charge_model === "package" && (
                <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  <span>Charge</span>
                  <Input
                    value={row.package_amount}
                    onChange={(e) => updateRow(i, { package_amount: e.target.value })}
                    placeholder="5.00"
                    inputMode="decimal"
                    aria-label={`Package price ${i + 1}`}
                    className="h-8 w-24 font-mono"
                  />
                  <span>{currency} per</span>
                  <Input
                    value={row.package_size}
                    onChange={(e) => updateRow(i, { package_size: e.target.value })}
                    placeholder="1000"
                    inputMode="numeric"
                    aria-label={`Package size ${i + 1}`}
                    className="h-8 w-24 font-mono"
                  />
                  <span>units</span>
                </div>
              )}

              {isTierModel(row.charge_model) && (() => {
                const isPct = row.charge_model === "graduated_percentage";
                return (
                <div className="flex flex-col gap-2">
                  <div className="grid grid-cols-[1fr_1fr_1fr_auto] items-center gap-2 text-xs uppercase tracking-wide text-muted-foreground">
                    <span>{isPct ? `Up to (${currency})` : "Up to (units)"}</span>
                    <span>{isPct ? "Rate (%)" : `Rate/unit (${currency})`}</span>
                    <span>Flat fee ({currency})</span>
                    <span className="sr-only">Remove</span>
                  </div>
                  {row.tiers.map((tier, ti) => {
                    const isLast = ti === row.tiers.length - 1;
                    return (
                      <div
                        key={ti}
                        className="grid grid-cols-[1fr_1fr_1fr_auto] items-center gap-2"
                      >
                        <Input
                          value={isLast ? "" : tier.up_to}
                          onChange={(e) => updateTier(i, ti, { up_to: e.target.value })}
                          placeholder={isLast ? "∞" : isPct ? "10000" : "100"}
                          disabled={isLast}
                          inputMode={isPct ? "decimal" : "numeric"}
                          aria-label={`Charge ${i + 1} tier ${ti + 1} up to`}
                          className="h-8 font-mono"
                        />
                        <Input
                          value={isPct ? tier.rate : tier.unit_amount}
                          onChange={(e) => updateTier(i, ti, isPct ? { rate: e.target.value } : { unit_amount: e.target.value })}
                          placeholder={isPct ? "2.5" : "1.00"}
                          inputMode="decimal"
                          aria-label={`Charge ${i + 1} tier ${ti + 1} rate`}
                          className="h-8 font-mono"
                        />
                        <Input
                          value={tier.flat_amount}
                          onChange={(e) => updateTier(i, ti, { flat_amount: e.target.value })}
                          placeholder="0.00"
                          inputMode="decimal"
                          aria-label={`Charge ${i + 1} tier ${ti + 1} flat fee`}
                          className="h-8 font-mono"
                        />
                        <button
                          type="button"
                          onClick={() => removeTier(i, ti)}
                          disabled={row.tiers.length === 1}
                          aria-label={`Remove tier ${ti + 1}`}
                          className="text-stone-400 transition-colors hover:text-red-500 disabled:opacity-30"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    );
                  })}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="self-start"
                    onClick={() => addTier(i)}
                  >
                    <Plus className="h-3.5 w-3.5" />
                    Add tier
                  </Button>
                </div>
                );
              })()}

              {row.charge_model === "percentage" && (
                <div className="flex flex-col gap-2">
                  <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <label htmlFor={`pct-rate-${i}`}>Rate</label>
                    <Input
                      id={`pct-rate-${i}`}
                      value={row.rate}
                      onChange={(e) => updateRow(i, { rate: e.target.value })}
                      placeholder="2.5"
                      inputMode="decimal"
                      aria-label={`Percentage rate ${i + 1}`}
                      className="h-8 w-20 font-mono"
                    />
                    <span>% of the metered value, in {currency}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                    {[
                      ["fixed_amount", "Fixed fee", "0.30"],
                      ["free_amount", "Free amount", "0.00"],
                      ["min_amount", "Minimum", "0.00"],
                      ["max_amount", "Maximum", "0.00"],
                    ].map(([field, label, ph]) => (
                      <label key={field} className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {label} ({currency})
                        <Input
                          value={row[field]}
                          onChange={(e) => updateRow(i, { [field]: e.target.value })}
                          placeholder={ph}
                          inputMode="decimal"
                          aria-label={`Charge ${i + 1} ${label}`}
                          className="h-8 font-mono"
                        />
                      </label>
                    ))}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Free amount is deducted before the rate; the line is clamped to
                    min/max (0 = none).
                  </p>
                </div>
              )}

              {row.charge_model === "dynamic" && (
                <p className="rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
                  No pricing to configure — send the exact price on each usage event
                  as <code className="font-mono">dynamic_amount</code> (minor units).
                  The charge bills the sum for the period.
                </p>
              )}

              {filterEligible(row.charge_model) && (
                <div className="flex flex-col gap-2 rounded-md border border-dashed border-border p-2.5">
                  <div className="flex items-center gap-2">
                    <label className="text-xs text-muted-foreground">Price by property</label>
                    <Input
                      value={row.filter_key}
                      onChange={(e) => updateRow(i, { filter_key: e.target.value })}
                      placeholder="e.g. region (optional)"
                      aria-label={`Charge ${i + 1} filter property`}
                      className="h-8 w-40 font-mono"
                    />
                  </div>
                  {row.filter_key.trim() && (
                    <>
                      {row.filters.map((f, fi) => (
                        <div key={fi} className="grid grid-cols-[1fr_1fr_auto] items-center gap-2">
                          <Input
                            value={f.value}
                            onChange={(e) => updateFilter(i, fi, { value: e.target.value })}
                            placeholder="value (e.g. us)"
                            aria-label={`Charge ${i + 1} filter ${fi + 1} value`}
                            className="h-8 font-mono"
                          />
                          <Input
                            value={f.amount}
                            onChange={(e) => updateFilter(i, fi, { amount: e.target.value })}
                            placeholder={row.charge_model === "percentage" ? "rate %" : `rate/unit`}
                            inputMode="decimal"
                            aria-label={`Charge ${i + 1} filter ${fi + 1} rate`}
                            className="h-8 font-mono"
                          />
                          <button
                            type="button"
                            onClick={() => removeFilter(i, fi)}
                            aria-label={`Remove filter ${fi + 1}`}
                            className="text-stone-400 transition-colors hover:text-red-500"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      ))}
                      <Button variant="ghost" size="sm" className="self-start" onClick={() => addFilter(i)}>
                        <Plus className="h-3.5 w-3.5" />
                        Add value
                      </Button>
                      <p className="text-xs text-muted-foreground">
                        Events matching no value use the rate above (the default).
                      </p>
                    </>
                  )}
                </div>
              )}
              {payInAdvanceEligible(row.charge_model) && (
                <label className="flex items-start gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={row.pay_in_advance}
                    onChange={(e) => updateRow(i, { pay_in_advance: e.target.checked })}
                    aria-label={`Charge ${i + 1} bill in advance`}
                    className="mt-0.5"
                  />
                  <span>
                    Bill in advance — rate each usage event as it arrives and add it to
                    the next invoice, instead of aggregating at period close.
                  </span>
                </label>
              )}

              <div className="flex items-center gap-2">
                <label className="text-xs text-muted-foreground">HSN/SAC</label>
                <Input
                  value={row.hsn_code}
                  onChange={(e) => updateRow(i, { hsn_code: e.target.value })}
                  placeholder="Empty = plan default"
                  aria-label={`Charge ${i + 1} HSN code`}
                  className="h-8 w-40 font-mono"
                />
              </div>
            </div>
          ))}

          {validationError && (
            <p role="alert" className="text-sm text-red-600">
              {validationError}
            </p>
          )}

          <div className="flex items-center justify-between">
            <Button variant="ghost" size="sm" onClick={addRow} disabled={metrics.length === 0}>
              <Plus className="h-4 w-4" />
              Add charge
            </Button>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsEditing(false)}
                disabled={saving}
              >
                Cancel
              </Button>
              <Button size="sm" onClick={handleSave} disabled={saving}>
                {saving ? "Saving…" : "Save charges"}
              </Button>
            </div>
          </div>
        </div>
      ) : loadError ? (
        <p className="text-sm text-red-600">Failed to load usage charges.</p>
      ) : charges.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No usage charges configured. Add metered pricing to bill this plan on usage.
        </p>
      ) : (
        <div className="flex flex-col gap-2">
          {charges.map((ch) => (
            <div
              key={ch.id}
              className="flex items-center justify-between gap-3 rounded-lg border border-border bg-muted/40 px-3 py-2"
            >
              <div className="min-w-0">
                <p className="truncate text-sm text-foreground">
                  {ch.metric?.name || metricName[ch.metric_id] || ch.metric_id}
                </p>
                <p className="text-xs text-muted-foreground">{chargeSummary(ch, currency)}</p>
              </div>
              <Badge variant="neutral">{MODEL_LABEL[ch.charge_model] || ch.charge_model}</Badge>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
