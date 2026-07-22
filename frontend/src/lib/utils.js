import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * cn — merge Tailwind class names with conflict resolution.
 * Usage: cn("px-2 py-1", condition && "bg-emerald-500", className)
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
