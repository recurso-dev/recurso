# Spec: Phase 5 - Revenue Recognition (ASC 606) 💹

## Objective
To automate compliance with ASC 606 / IFRS 15 accounting standards. The system must track when revenue is earned (recognized) versus when it is collected (deferred liability). This is critical for SaaS companies to accurately report monthly performance regardless of billing cycles (e.g., recognizing 1/12th of an annual plan each month).

### User Stories
- As a CFO, I want to see a "Recognized Revenue" report for any given month.
- As an accountant, I want the system to automatically move funds from "Deferred Revenue" to "Recognized Revenue" accounts in the ledger.
- As a system, I need to handle proration and cancellations by adjusting future recognition schedules.

## Tech Stack
- **Language**: Go 1.20+
- **Database**: PostgreSQL (for schedules) + TigerBeetle (for double-entry ledger updates)
- **Reporting**: JSON API for frontend visualization

## Commands
- Build: `make build`
- Test: `go test ./internal/service/revrec_test.go`
- Run Scheduler: `go run cmd/api/main.go` (RevRec worker)

## Project Structure
- `internal/core/domain/revrec.go`      → Revenue schedules and recognition events
- `internal/core/service/revrec.go`    → Logic for ratable recognition calculation
- `internal/adapter/db/revrec_repo.go` → Persistence for schedules
- `internal/adapter/worker/revrec_worker.go` → Monthly job to "close the books" and move ledger entries

## Architecture: Ratable Recognition
1. **Schedule Creation**: When an invoice is marked as `PAID`, a `RevenueSchedule` is generated.
2. **Allocation**: For a $1200 annual subscription, the system creates 12 monthly "Recognition Events" of $100 each.
3. **Ledger Integration**: On the 1st of each month (or daily), the worker executes TigerBeetle transfers:
   - `Debit: Deferred Revenue (Liability)`
   - `Credit: Recognized Revenue (Income)`

## Security & Boundaries
- **Always**: Ensure double-entry integrity (Total Recognized + Total Deferred = Total Invoiced).
- **Always**: Log all manual adjustments to schedules for audit trails.
- **Ask first**: Changing the base currency of the ledger.
- **Never**: Allow recognition of revenue for unpaid invoices (unless using accrual-based triggers explicitly).

## Success Criteria
- [ ] `RevenueSchedule` generated automatically upon invoice payment.
- [ ] API endpoint `GET /v1/finance/revrec/report` returns monthly breakdown.
- [ ] Integration with TigerBeetle Ledger for automated liability-to-income transfers.
- [ ] Correct handling of cancellations (stopping future recognition).

## Open Questions
1. Should we support **Daily Ratable** (exact days in month) or **Monthly Linear** (equal parts regardless of month length)? (Initial: Monthly Linear for simplicity, with option for daily).
2. How to handle multi-currency? (Initial: Recognize in the transaction currency, report in the Ledger's base currency).
