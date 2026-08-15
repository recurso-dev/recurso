// Ledger posting-code semantics (ADR-002), shared by the Ledger page and the
// Journal Entry object page so both name a posting the same way. "Code 3" means
// nothing to an operator; "Payment" does.

// What each movement IS, in words.
export const CODE_LABEL = {
  1: "Invoice raised",
  2: "Revenue recognized",
  3: "Payment",
  4: "Refund",
  5: "Refund — deferred reversal",
  6: "Output tax",
  7: "Credit applied",
  8: "Credit note",
  9: "Refund — tax reversal",
  10: "TDS receivable",
  11: "Wallet top-up",
  12: "Wallet drain",
  13: "Wallet refund",
  14: "Wallet forfeit",
  15: "Wallet expiry",
  16: "Downgrade credit",
  17: "Downgrade — tax reversal",
  18: "Credit expiry",
  19: "Payment reversal",
  20: "Credit void",
  21: "Downgrade — revenue reversal",
  22: "Write-off",
  23: "Write-off — tax reversal",
  24: "Write-off recovery",
  25: "Write-off recovery — tax",
  26: "Bad debt (write-off)",
  27: "Bad debt recovery",
};

export const codeLabel = (c) => CODE_LABEL[c] || `Code ${c}`;

// What a transaction's reference_id points at, derived from its code (each
// posting site in service/ledger.go stamps one reference kind per code).
// Invoice references drill through to the invoice; the rest are labeled
// honestly rather than an ambiguous "invoice / payment".
export const REF_KIND = {
  1: "invoice", 3: "invoice", 6: "invoice", 10: "invoice", 12: "invoice",
  19: "invoice", 22: "invoice", 23: "invoice", 24: "invoice", 25: "invoice",
  26: "invoice", 27: "invoice",
  4: "credit note", 5: "credit note", 9: "credit note", 16: "credit note",
  17: "credit note", 18: "credit note", 20: "credit note", 21: "credit note",
  2: "recognition entry",
  11: "wallet transaction", 13: "wallet transaction", 14: "wallet transaction",
  15: "wallet transaction",
};

export const refKind = (c) => REF_KIND[c] || "source record";

// The canonical route for a posting's reference object, or null when the
// reference isn't an addressable object today (recognition entries, wallet
// transactions, etc. have no object page). Only invoice references drill
// through — the honest, non-fabricated set.
export function refRoute(code, referenceId) {
  if (!referenceId) return null;
  return refKind(code) === "invoice" ? `/invoices/${referenceId}` : null;
}
