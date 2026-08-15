import { Link } from "react-router";

import { cn } from "@/lib/utils";

const NIL_UUID = "00000000-0000-0000-0000-000000000000";

/**
 * LedgerAccountLink — a journal leg's account, linked to its account page so any
 * posting drills into the account it moved (audit §7: postings were navigational
 * dead-ends). Extracted from the Ledger entry-detail sheet and shared with
 * <JournalEntries> and any object page that lists postings.
 *
 *  - id present → a real <Link> to /ledger/accounts/:id (⌘-click / copy-address
 *    both work).
 *  - id absent  → the `label` as plain text. Invoice / credit-note journal rows
 *    (`GeneralLedgerRow`) carry the account name + code but NOT the account id,
 *    so their legs render as text today and light up as links the moment the
 *    backend adds `debit_account_id` / `credit_account_id` to that row — no
 *    frontend change needed. (BACKEND GAP, tracked in the dashboard audit.)
 *  - neither    → an em dash.
 *
 * `label` may be a string or a node; omit it and a short id is shown instead.
 * `className` styles both the link and the text fallback (font-size/weight are
 * inherited from the parent so the same component fits the Ledger sheet and the
 * mono JournalEntries grid).
 */
export function LedgerAccountLink({ id, label, className }) {
  const linkable = id && id !== NIL_UUID;
  const content = label || (linkable ? `${String(id).slice(0, 8)}…` : null);

  if (!linkable) {
    return content ? (
      <span className={className}>{content}</span>
    ) : (
      <span className="text-muted-foreground">—</span>
    );
  }

  return (
    <Link
      to={`/ledger/accounts/${id}`}
      title="Open this account"
      className={cn("text-primary underline-offset-2 hover:underline", className)}
    >
      {content}
    </Link>
  );
}

export default LedgerAccountLink;
