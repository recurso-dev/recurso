import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * cn — merge Tailwind class names with conflict resolution.
 * Usage: cn("px-2 py-1", condition && "bg-success/50", className)
 */
export function cn(...inputs) {
  return twMerge(clsx(inputs));
}

/**
 * currencyDecimals — how many minor-unit digits a currency has (2 for USD/EUR,
 * 0 for JPY/KRW, 3 for KWD/BHD). Derived from Intl so we don't hardcode /100.
 * Falls back to 2 for unknown/invalid codes.
 */
export function currencyDecimals(currency = "USD") {
  try {
    return (
      new Intl.NumberFormat("en-US", { style: "currency", currency: currency || "USD" })
        .resolvedOptions().maximumFractionDigits ?? 2
    );
  } catch {
    return 2;
  }
}

/**
 * fromMinorUnits — minor-unit integer (e.g. cents) → major-unit number, using
 * the currency's real exponent. 4200 USD → 42, 4200 JPY → 4200, 4200 KWD → 4.2.
 */
export function fromMinorUnits(amountMinor, currency = "USD") {
  return (Number(amountMinor) || 0) / 10 ** currencyDecimals(currency);
}

/**
 * toMinorUnits — major-unit input (e.g. a form field) → minor-unit integer for
 * the API. Uses the currency's real exponent so JPY/KWD amounts aren't mangled.
 */
export function toMinorUnits(amount, currency = "USD") {
  const factor = 10 ** currencyDecimals(currency);
  return Math.round((Number(amount) || 0) * factor);
}

/**
 * formatCurrency — format minor-unit integer amounts as currency. The API
 * returns money in the smallest currency unit; the decimals shown are the
 * currency's own (Intl decides — 2 for USD, 0 for JPY, 3 for KWD).
 */
export function formatCurrency(amountMinor, currency = "USD") {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency || "USD",
  }).format(fromMinorUnits(amountMinor, currency));
}

/**
 * formatCurrencyHeadline — formatCurrency for KPI tiles and other headline
 * numerals: a whole amount drops its ".00" tail ($18,675 not $18,675.00),
 * while non-zero cents are kept so small amounts (−$0.82) stay visible.
 * Table cells should keep formatCurrency/Money — mixed precision there breaks
 * column alignment.
 */
export function formatCurrencyHeadline(amountMinor, currency = "USD") {
  const major = fromMinorUnits(amountMinor, currency);
  const isWhole = Number.isInteger(major);
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency || "USD",
    ...(isWhole ? { maximumFractionDigits: 0 } : {}),
  }).format(major);
}

/**
 * formatNumber — compact/grouped number formatting for metrics.
 */
export function formatNumber(value, options = {}) {
  return new Intl.NumberFormat("en-US", options).format(Number(value) || 0);
}

/**
 * formatDate — short human date from an ISO string or Date.
 */
export function formatDate(input, options = { month: "short", day: "numeric", year: "numeric" }) {
  if (!input) return "—";
  const d = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString("en-US", options);
}

/**
 * formatDateTime — the one timestamp format ("Aug 12, 2026, 9:41 AM").
 * The audit found six competing date treatments, four of them raw
 * `toLocaleString()` (locale-dependent widths that break column alignment).
 * Fixed en-US locale to match formatDate; use this for every timestamp cell.
 */
export function formatDateTime(input) {
  if (!input) return "—";
  const d = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

// shortId renders the first 8 characters of a UUID for tables where the full
// id is noise — one canonical form ("abcd1234…", em-dash when absent) so every
// page truncates ids the same way.
export function shortId(id) {
  return id ? `${String(id).slice(0, 8)}…` : "—";
}
