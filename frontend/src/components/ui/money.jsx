import { cn } from "@/lib/utils";

// Money renders a monetary amount as data: tabular mono numerals with the
// currency symbol set in a muted tone. amountMinor is in the currency's
// smallest unit (cents/paise).
export function Money({ amountMinor, currency = "USD", className }) {
  const value = (Number(amountMinor) || 0) / 100;
  const parts = new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency || "USD",
    maximumFractionDigits: 2,
  }).formatToParts(value);

  return (
    <span className={cn("money", className)}>
      {parts.map((p, i) =>
        p.type === "currency" ? (
          <span key={i} className="money-symbol">
            {p.value}
          </span>
        ) : (
          <span key={i}>{p.value}</span>
        )
      )}
    </span>
  );
}

export default Money;
