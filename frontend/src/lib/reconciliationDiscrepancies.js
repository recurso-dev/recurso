// Reconciliation discrepancy semantics, shared by the live Reconciliation page
// and the recorded-run object page so a discrepancy reads identically in both.
// A raw enum is useless to an operator — each type says WHAT disagrees and WHY.
// All 20 backend types are covered so a real discrepancy is never shown raw.
import { formatCurrency } from "@/lib/utils";

export const DISCREPANCIES = {
  missing_invoice_transaction: { label: "Missing invoice transaction", reason: "An issued invoice has no Code-1 issuance posting — the ledger never recorded it being raised." },
  invoice_amount_mismatch: { label: "Invoice amount mismatch", reason: "The invoice's issuance posting doesn't equal the invoice total." },
  missing_payment_transaction: { label: "Missing payment transaction", reason: "A paid invoice has no Code-3 payment posting — cash was recorded received but never posted." },
  payment_amount_mismatch: { label: "Payment amount mismatch", reason: "The payment posting doesn't equal what the invoice recorded as paid." },
  missing_credit_note_transaction: { label: "Missing credit-note transaction", reason: "A credit note exists with no ledger posting behind it." },
  missing_credit_application_transaction: { label: "Missing credit-application transaction", reason: "Credit was applied to an invoice but no Code-7 posting drew it down." },
  credit_application_amount_mismatch: { label: "Credit-application amount mismatch", reason: "The credit-applied posting doesn't equal the credit recorded against the invoice." },
  missing_write_off_transaction: { label: "Missing write-off transaction", reason: "An uncollectible invoice has no write-off reversal posting." },
  write_off_amount_mismatch: { label: "Write-off amount mismatch", reason: "The write-off posting doesn't equal the amount written off." },
  missing_tax_transaction: { label: "Missing tax transaction", reason: "A taxed invoice has no Code-6 tax reclass posting." },
  tax_amount_mismatch: { label: "Tax amount mismatch", reason: "The tax posting doesn't equal the invoice's tax." },
  orphaned_transaction: { label: "Orphaned transaction", reason: "A posting references a source object that no longer exists." },
  missing_in_tigerbeetle: { label: "Missing in TigerBeetle", reason: "A Postgres posting has no matching TigerBeetle transfer." },
  missing_in_postgres: { label: "Missing in Postgres", reason: "A TigerBeetle transfer has no matching Postgres posting." },
  tb_amount_mismatch: { label: "TigerBeetle amount mismatch", reason: "A posting's amount differs between Postgres and TigerBeetle." },
  ledger_unbalanced: { label: "Ledger unbalanced (debits ≠ credits)", reason: "Total debits don't equal total credits — the accounting identity itself is broken." },
  abnormal_account_balance: { label: "Wrong-sign account balance", reason: "An account holds a balance on the wrong side (e.g. a negative asset)." },
  deferred_below_scheduled_revenue: { label: "Deferred below scheduled revenue", reason: "Deferred Revenue is less than the recognition schedule still expects to release." },
  recognized_exceeds_invoice: { label: "Recognized exceeds invoice", reason: "More revenue has been recognized than the invoice can support." },
  customer_credit_liability_mismatch: { label: "Customer-credit liability mismatch", reason: "The Customer Credit liability doesn't equal the sum of outstanding credit balances." },
};

// Discrepancy amounts are minor units of the report's reporting currency; format
// them as real money (currency-exponent aware) so an operator can't misread the
// magnitude by 100x. Falls back to USD if the currency is somehow absent.
export const formatMinorUnits = (n, currency) =>
  typeof n === "number" ? formatCurrency(n, currency || "USD") : "—";

// Difference = found − expected, shown with an explicit + so an operator sees
// the direction and size of the gap (formatCurrency already carries the minus).
export const formatDifference = (d, currency) => {
  if (typeof d.found_amount !== "number" || typeof d.expected_amount !== "number") return "—";
  const diff = d.found_amount - d.expected_amount;
  return `${diff > 0 ? "+" : ""}${formatCurrency(diff, currency || "USD")}`;
};
