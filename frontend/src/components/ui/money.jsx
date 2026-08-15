import { cn, fromMinorUnits } from "@/lib/utils";

// The deliberately-small money size vocabulary. `md` is the default and matches
// the historical behavior (inherits the ambient text size), so unsized callers
// are unchanged. Use these instead of ad-hoc per-page text-* classes so every
// amount speaks the same financial language:
//   sm   — inline / secondary metadata (a small amount beside other text)
//   md   — default: table cells and standard amounts
//   lg   — object metric strips (MRR, financial-summary figures)
//   hero — the object's single dominant amount (the object-page hero)
const MONEY_SIZE = {
  sm: "text-xs",
  md: "",
  lg: "text-lg font-semibold",
  hero: "text-2xl font-semibold",
};

// Money renders a monetary amount as data: tabular mono numerals with the
// currency symbol set in a muted tone (the `.money` signature). amountMinor is
// in the currency's smallest unit (cents/paise/etc.) — decimals shown are the
// currency's own (exponent-aware: JPY 0, KWD 3). null/undefined render as the
// currency's zero. The full amount is the element's text content, so screen
// readers announce it verbatim; `size` only affects visual scale.
export function Money({ amountMinor, currency = "USD", size = "md", className }) {
  const value = fromMinorUnits(amountMinor, currency);
  const parts = new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency || "USD",
  }).formatToParts(value);

  return (
    <span className={cn("money", MONEY_SIZE[size] ?? MONEY_SIZE.md, className)}>
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
