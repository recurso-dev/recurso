// Formats an amount given in the currency's lowest unit (e.g. paise) into a
// human-readable string like "₹2,000.00".
export function formatMoney(amountMinor: number, currency: string): string {
  try {
    return new Intl.NumberFormat("en-IN", {
      style: "currency",
      currency,
    }).format(amountMinor / 100);
  } catch {
    // Fall back if the currency code is unknown to Intl.
    return `${(amountMinor / 100).toFixed(2)} ${currency}`;
  }
}
