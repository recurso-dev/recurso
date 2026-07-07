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
 * formatCurrency — format minor-unit integer amounts (cents) as currency.
 * The API returns money in the smallest currency unit.
 */
export function formatCurrency(amountMinor, currency = "USD") {
  const value = (Number(amountMinor) || 0) / 100;
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency || "USD",
    maximumFractionDigits: 2,
  }).format(value);
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
