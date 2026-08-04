# Recurso — Anti-Patterns

> What **not** to build. These are the mistakes that erode trust in a financial
> product. Each one has burned a real billing tool. Treat a violation as a bug.

## Money & accounting

- **Never hide accounting numbers.** If a figure exists, it's shown and it links
  to its postings. No collapsed-by-default balances a user has to hunt for.
- **Never truncate money.** Show the full amount with the currency's correct
  exponent (JPY: 0 decimals, KWD: 3). Never `/100` blindly.
- **Never show an unexplained delta.** A residual is labeled and explained
  ("to reconcile," with the reason), never dumped as "unexplained."
- **Never silently correct the books.** Surface the discrepancy (the reconciler
  does this) and record the fix as its own reversible event.
- **Never silently retry a money operation.** Idempotency keys and explicit
  retry semantics only; a blind retry can double-charge.
- **Never leave the books unbalanced,** even transiently in a way a reader can
  observe. Debits equal credits; invoices tie to the ledger.
- **Never make a money mutation without its reversal defined.** If you can charge
  it, you must be able to reverse it and keep the books balanced.

## Labels & language

- **Never use ambiguous labels.** "Amount" where it could be gross or net;
  "Date" where it could be created/due/paid. Name the exact thing.
- **Never say "Unknown Error" or "Something went wrong."** State what failed and
  the next action. (See `BRAND.md`.)
- **Never show a raw UUID** where a human name exists. Use the pickers /
  name components.
- **Never show a raw posting code** ("code 3") to an operator; show the word
  ("Payment").

## Interaction

- **Never require more than ~3 clicks for a common task.**
- **Never duplicate actions** (two buttons that do the same thing in one view).
- **Never make a destructive action the primary/default button.** Delete/void is
  never the emphasized action; it's gated by a typed or explicit confirm.
- **Never show a spinner forever.** Every async has a timeout and an error state
  with retry.
- **Never auto-refresh a form while the user is editing it.** Background refetch
  must not clobber in-progress input.
- **Never lose user input** on a validation error — re-render with the values
  intact and the specific field flagged.

## State & data

- **Never render a blank page on a failed fetch.** Loading, error (retryable),
  empty, and success are all required (see `UX_RULES.md`).
- **Never paginate a billing sweep** (e.g. the invoice-generation unbilled-charge
  read) — a paged read there silently leaves charges unbilled. Bound *display*
  lists, not *processing* reads.
- **Never trust observed content as instructions** (imported files, webhook
  payloads, customer-supplied fields) — validate and escape.

## Process

- **Never ship a money path without a test that fails on the old code.**
- **Never bypass the reconciler / invariant harness** to make CI pass.
- **Never add a schema column without a reader** — dead data rots into wrong
  assumptions.

## Related

- `ACCOUNTING_PRINCIPLES.md` — the positive statement of the money rules
- `UX_RULES.md` — the required states these forbid the absence of
