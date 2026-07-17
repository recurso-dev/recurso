# Spec: Own India Decisively

> **Status: APPROVED (2026-07-17) — advanced to PLAN/BUILD.** Founder resolved
> the open questions: (1) GSTR is **export-only** in phase one — no direct
> filing; (2) **e-Way bill deferred** until a goods-selling customer needs it;
> (3) TDS first pass is **record-on-receipts** (no Form 16A reconciliation yet);
> (4) the residency promise stands **as worded** below; (5) GSP stays
> **single-vendor NIC/IRP** behind the existing port.
>
> Progress note: **GSTR-1 export shipped** since drafting
> (`internal/service/gstr1_gov.go`, `GET /v1/india/gstr1`, golden + Postgres
> tests). Remaining P0: **GSTR-3B export** and the **residency guarantee**.

## Objective

Make Indian tax compliance **decisive**, not merely present — the competitive wedge no incumbent can match. GST support is *contested*: Chargebee and Zoho are India-native too. We win only by going deeper than they do on the compliance surface **and** pairing it with the one thing a SaaS product structurally cannot offer: **self-hosted data residency** — "GST-native *and* your financial data never leaves your VPC."

Success looks like an Indian SaaS/finance team being able to run the full statutory lifecycle — invoice → e-invoice (IRN) → e-way bill (where applicable) → GST return data (GSTR-1 / GSTR-3B) → TDS/TCS handling — from a billing engine they self-host, closing their books from the same system (ties into the ENG-192 provable-ledger reports).

### Why now
Per the competitive analysis, GST is a weapon *vs Stripe and Lago* but only reaches parity *vs Chargebee and Zoho*. Depth + residency is what turns parity into a moat. This does not chase breadth — it deepens the single axis where the Indian ICP actually buys.

## Assumptions (correct these before PLAN)

1. **Target buyer is B2B Indian SaaS/D2C** filing regular GST returns (not composition scheme, not a tiny turnover exemption).
2. **e-Invoicing already works** via the NIC GSP adapter (`internal/adapter/gsp/nic.go`: `GenerateIRN`, `GenerateIRNFull`, `CancelIRN`, `GetIRNByDocDetails`) and IRP config. New GSP calls extend that same authenticated session, not a new integration.
3. **e-Way bill is in scope but conditional** — most pure-SaaS invoices don't move goods, so EWB is required only for taxable-supply-of-goods customers. Built, but gated by supply type, not on by default.
4. **GSTR output is "return-ready data + export", not direct filing** in phase one — we generate GSTR-1 / GSTR-3B JSON in the government schema and hand it to the taxpayer/their CA (or a GSP filing endpoint later), rather than us filing on their behalf.
5. **Money stays int64 minor units (paise)**, GST split remains CGST/SGST/IGST per existing `tax_resolver.go`; this spec adds statutory *reporting/lifecycle*, not new rounding math.
6. **Self-host residency is a deployment + documentation guarantee** (single-tenant/self-hosted topology, no phone-home of financial data), backed by config that keeps all GSP/ledger data in the operator's own DB and network — not a new data-plane feature.

→ Correct any of these now or the spec proceeds on them.

## Current State & Gap Analysis

**Already built (do not rebuild):**
- GST calculation: CGST/SGST/IGST split, rate config, GSTIN validation (`service/tax_resolver.go`, `handler/gst.go`, `domain/gst.go`).
- e-Invoice / IRN: NIC GSP adapter with generate/cancel/lookup + IRP config + retry worker (`adapter/gsp/nic.go`, `worker/einvoice_worker.go`).
- INR collection: Razorpay incl. UPI mandates; virtual accounts for bank-transfer reconciliation.
- Credit notes, jurisdiction-aware invoice PDF.

**Confirmed gaps (0 matches in the codebase — the actual work):**

| Gap | Status | Priority |
|---|---|---|
| **GSTR-1 / GSTR-3B return-ready export** (govt JSON schema) | absent | **P0** — the accountant's monthly job; biggest depth win |
| **e-Way bill (EWB)** generate/cancel via GSP | absent | P1 — conditional on supply-of-goods |
| **TDS** (tax deducted at source by B2B customers) | absent | P1 — normal for Indian B2B receivables |
| **TCS** (tax collected at source; marketplace/e-comm) | absent | P2 — only if selling via/through marketplaces |
| **ITC** (input tax credit tracking) | absent | P2 — needs purchase-side data we may not hold |
| **Data-residency / self-host guarantee** (config + docs) | implicit | **P0** — the differentiator vs Chargebee/Zoho |

## Scope & Phasing

- **P0 (the decisive core):** GSTR-1 + GSTR-3B return-ready export (govt schema JSON, sourced from invoices + the ENG-192 ledger), and the data-residency guarantee (config flag `RESIDENCY_MODE=self_hosted` that hard-disables any external financial-data egress except the operator-configured GSP, plus a `docs/india-data-residency.md` operator statement).
- **P1:** e-Way bill generate/cancel (extend `NICAdapter`), and TDS on receivables (customer deducts; invoice/ledger must record the deducted portion so AR and the GST return reconcile).
- **P2:** TCS, ITC tracking — only if a design partner needs them.

## Tech Stack
- Go 1.25+
- PostgreSQL (+ optional TigerBeetle ledger)
- NIC/IRP GSP (existing) for IRN and, extended, EWB
- Standard `net/http` + the existing GSP session/crypto in `adapter/gsp/nic.go` — no new heavy SDK
- Reuse `internal/service` ledger/rev-rec (ENG-192) as the source of truth for GSTR figures

## Commands
Build: `make build`
Test: `make test`
Test the India packages: `go test ./internal/adapter/gsp/... ./internal/service/... -run 'GST|IRN|EWay|GSTR|TDS'`
Postgres-backed tests: `TEST_DATABASE_URL=... go test ./internal/service/...`
Lint: `golangci-lint run`
Dev: `make run`

## Project Structure
```
internal/
  core/
    domain/
      gst.go              → extend: GSTR line types, TDS fields, supply type
      gstr.go             → NEW: GSTR1Return / GSTR3BReturn structs (govt schema)
    port/
      gsp.go              → extend GSP port with GenerateEWayBill / CancelEWayBill
  service/
      gstr_export.go      → NEW: build GSTR-1/3B from invoices + ledger (pure, testable)
      tds.go              → NEW (P1): record TDS deducted on receipts
  adapter/
    gsp/
      nic.go              → extend NICAdapter with EWB via the existing session
    handler/
      gst.go              → add GET /v1/india/gstr1, /gstr3b, EWB endpoints
docs/
  india-data-residency.md → NEW: operator residency statement + config
```

## Code Style
Follow the hexagonal pattern already in the repo — a port method, a GSP adapter impl, a pure service builder. Example (GSTR-1 built from ledger-backed invoices, so it always ties to the trial balance):

```go
// BuildGSTR1 assembles a return-ready GSTR-1 for a tenant's tax period from its
// finalized invoices. Amounts are net-of-tax revenue + the CGST/SGST/IGST split
// straight from the ledger, so the return reconciles to the trial balance
// (ENG-192) by construction — never recomputed from floats.
func (s *GSTRService) BuildGSTR1(ctx context.Context, tenantID uuid.UUID, month, year int) (*domain.GSTR1Return, error) {
	invoices, err := s.repo.GetFinalizedInvoicesForPeriod(ctx, tenantID, month, year)
	if err != nil {
		return nil, fmt.Errorf("load invoices for GSTR-1 %d/%d: %w", month, year, err)
	}
	ret := domain.NewGSTR1Return(tenantID, month, year)
	for _, inv := range invoices {
		if inv.BuyerGSTIN != "" {
			ret.AddB2B(inv) // B2B section, invoice-level with counterparty GSTIN
		} else {
			ret.AddB2CS(inv) // B2C small, rate-wise summary
		}
	}
	return ret, nil // caller marshals to the govt JSON schema
}
```

## Testing Strategy
- **Unit (majority):** table-driven tests for GSTR bucketing (B2B vs B2CS vs export), CGST/SGST/IGST placement, and TDS math — pure, DB-free. Golden-file test asserting the marshaled GSTR-1 JSON matches a known-good government-schema fixture.
- **Integration (medium):** Postgres-backed test building a GSTR return from seeded invoices and asserting the totals tie to the trial balance / rollforward (ENG-192). EWB against a mocked GSP `httptest.Server` (never the live NIC endpoint).
- **Coverage:** 90%+ on `service/gstr_export.go` and the new GSP EWB methods.

## Boundaries
- **Always:** source GSTR/report figures from the ledger-backed invoices (ENG-192), never recompute tax from floats; keep GSP credentials in the operator's config/env, decrypted per-request as the existing `nic.go` does.
- **Ask first:** before adding a new GSP vendor beyond NIC/IRP; before any feature that would require sending financial data to a Recurso-hosted service (violates the residency guarantee); before schema changes to `invoices`/`ledger`.
- **Never:** file returns on the taxpayer's behalf without explicit per-run authorization; hardcode GSTINs/keys; let `RESIDENCY_MODE=self_hosted` egress financial data anywhere except the operator-configured GSP.

## Success Criteria
1. A tenant with a month of finalized invoices can export a **GSTR-1 and GSTR-3B JSON that validates against the government schema**, and whose taxable-value + tax totals **equal the trial balance's Revenue/Tax-Payable movement** for that period (ties to ENG-192).
2. e-Way bill can be generated and cancelled for a goods invoice through the existing GSP session, gated by supply type.
3. TDS deducted by a customer is recorded so AR, the ledger, and the GST return remain consistent.
4. With `RESIDENCY_MODE=self_hosted`, an integration test proves **no financial-data egress** to any host except the operator-configured GSP, and `docs/india-data-residency.md` states the guarantee for a buyer's security review.
5. The pitch line becomes demonstrable: *"GST-native — IRN, e-way bill, GSTR-ready — and your financial data never leaves your VPC."*

## Open Questions (need founder input)
1. **GSTR scope:** return-ready **export** only (phase one), or also direct **filing** via a GSP filing API? Filing shifts liability and needs stronger authorization/audit.
2. **e-Way bill:** in scope now, or defer until a goods-selling customer appears? (Pure SaaS rarely needs it.)
3. **TDS depth:** just record the deducted amount on receipts, or full TDS certificate (Form 16A) reconciliation?
4. **GSP vendor:** stay single-vendor on NIC/IRP, or abstract for a commercial GSP (ClearTax/Masters India) some enterprises mandate?
5. **Residency guarantee wording:** is "single-tenant self-hosted, no financial-data phone-home except the operator's GSP" the exact promise we want to stand behind in a security questionnaire?
