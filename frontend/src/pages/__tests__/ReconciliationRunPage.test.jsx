import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import ReconciliationRunPage from "../ReconciliationRunPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getReconciliationRun: vi.fn(),
    getUsers: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));

const run = (over = {}) => ({
  id: "run_1",
  run_by: null,
  run_at: "2026-08-14T10:00:00Z",
  invoices_checked: 305,
  paid_invoices_checked: 264,
  total_discrepancies: 2,
  tb_compared: false,
  tb_accounts_checked: 0,
  tb_transfers_checked: 0,
  created_at: "2026-08-14T10:00:00Z",
  discrepancies_truncated: false,
  discrepancies: [
    { type: "invoice_amount_mismatch", invoice_id: "inv_1", transaction_id: "tx_1", expected_amount: 10000, found_amount: 9000 },
    { type: "ledger_unbalanced", expected_amount: 500, found_amount: 0 },
  ],
  ...over,
});

function renderPage(id = "run_1") {
  return render(
    <MemoryRouter initialEntries={[`/finance/reconciliation/runs/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/finance/reconciliation/runs/:id" element={<ReconciliationRunPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("ReconciliationRunPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("explains each discrepancy (what/why) and links the transaction + invoice", async () => {
    endpoints.getReconciliationRun.mockResolvedValue({ data: { data: run() } });
    renderPage();
    await waitFor(() => expect(screen.getByText("Invoice amount mismatch")).toBeInTheDocument());
    // The one-line "why" is present.
    expect(screen.getByText(/equal the invoice total/)).toBeInTheDocument();
    const hrefs = Array.from(document.querySelectorAll("a")).map((a) => a.getAttribute("href"));
    expect(hrefs).toContain("/invoices/inv_1");
    expect(hrefs).toContain("/ledger/transactions/tx_1");
  });

  it("shows a clean 'tied out' result for a balanced run with no discrepancies", async () => {
    endpoints.getReconciliationRun.mockResolvedValue({
      data: { data: run({ total_discrepancies: 0, discrepancies: [] }) },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText(/tied out/)).toBeInTheDocument());
    expect(screen.getByText("Reconciled")).toBeInTheDocument();
  });

  it("banners the exception and badges the count when a run has discrepancies (Batch E parity)", async () => {
    endpoints.getReconciliationRun.mockResolvedValue({ data: { data: run() } }); // total 2
    renderPage();
    // Canonical StatusBadge in the header carries the count.
    await waitFor(() => expect(screen.getByText("2 discrepancies")).toBeInTheDocument());
    // The exception surfaces as an AttentionBanner (like Invoice/Subscription) —
    // "Review each below" is unique to the banner (vs the Result section copy).
    expect(screen.getByText(/Review each below/i)).toBeInTheDocument();
  });

  it("is honest when a run counted discrepancies but stored no detail (pre-persistence run)", async () => {
    endpoints.getReconciliationRun.mockResolvedValue({
      data: { data: run({ total_discrepancies: 5, discrepancies: [] }) },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText(/predates per-run discrepancy storage/)).toBeInTheDocument());
  });

  it("flags a truncated discrepancy set", async () => {
    endpoints.getReconciliationRun.mockResolvedValue({
      data: { data: run({ total_discrepancies: 120, discrepancies_truncated: true }) },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText(/Showing 2 of 120/)).toBeInTheDocument());
  });

  it("shows a 404 state for a missing run", async () => {
    endpoints.getReconciliationRun.mockRejectedValue({ response: { status: 404 } });
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Reconciliation run not found")).toBeInTheDocument()
    );
  });
});
