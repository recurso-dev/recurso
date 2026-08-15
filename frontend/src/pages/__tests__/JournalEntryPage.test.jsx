import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import JournalEntryPage from "../JournalEntryPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getLedgerTransaction: vi.fn(),
  },
}));

const entry = (over = {}) => ({
  transaction_id: "tx_1",
  timestamp: "2026-08-14T10:00:00Z",
  code: 3, // Payment
  debit_account_id: "acc_cash",
  debit_account_code: 1000,
  debit_account_name: "Cash",
  credit_account_id: "acc_ar",
  credit_account_code: 1100,
  credit_account_name: "Accounts Receivable",
  amount: 9900,
  reference_id: "inv_5",
  description: "Payment received",
  accounting_version: 2,
  ...over,
});

function renderPage(id = "tx_1") {
  return render(
    <MemoryRouter initialEntries={[`/ledger/transactions/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/ledger/transactions/:id" element={<JournalEntryPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("JournalEntryPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("names the posting in words and shows labeled DR/CR legs that link to accounts", async () => {
    endpoints.getLedgerTransaction.mockResolvedValue({ data: { data: entry() } });
    renderPage();
    // Posting named in words (code 3 → "Payment"), not a raw code.
    await waitFor(() => expect(screen.getAllByText("Payment").length).toBeGreaterThan(0));
    // Labeled debit/credit (not color-only).
    expect(screen.getByText("DR")).toBeInTheDocument();
    expect(screen.getByText("CR")).toBeInTheDocument();
    // Each leg links to its ledger account page.
    const hrefs = Array.from(document.querySelectorAll("a")).map((a) => a.getAttribute("href"));
    expect(hrefs).toContain("/ledger/accounts/acc_cash");
    expect(hrefs).toContain("/ledger/accounts/acc_ar");
  });

  it("links an invoice-referenced posting to its source invoice", async () => {
    endpoints.getLedgerTransaction.mockResolvedValue({ data: { data: entry() } });
    renderPage();
    await waitFor(() => expect(screen.getByText("Source")).toBeInTheDocument());
    const hrefs = Array.from(document.querySelectorAll("a")).map((a) => a.getAttribute("href"));
    expect(hrefs).toContain("/invoices/inv_5");
  });

  it("labels a non-addressable reference honestly instead of a dead link", async () => {
    // code 2 = revenue recognition → "recognition entry", not an object page.
    endpoints.getLedgerTransaction.mockResolvedValue({
      data: { data: entry({ code: 2, reference_id: "sch_1" }) },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText("Source")).toBeInTheDocument());
    expect(screen.getByText(/not a separately addressable object/i)).toBeInTheDocument();
    const hrefs = Array.from(document.querySelectorAll("a")).map((a) => a.getAttribute("href"));
    expect(hrefs).not.toContain("/invoices/sch_1");
  });

  it("shows a 404 state for a missing transaction", async () => {
    endpoints.getLedgerTransaction.mockRejectedValue({ response: { status: 404 } });
    renderPage();
    await waitFor(() => expect(screen.getByText("Journal entry not found")).toBeInTheDocument());
  });
});
