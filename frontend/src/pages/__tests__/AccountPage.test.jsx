import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import AccountPage from "../AccountPage";
import { money } from "@/test/money";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getLedgerAccounts: vi.fn(),
    getLedgerEntries: vi.fn(),
  },
}));

const cashId = "acct-cash-1000";

function renderPage(id = cashId) {
  return render(
    <MemoryRouter initialEntries={[`/ledger/accounts/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/ledger/accounts/:id" element={<AccountPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("AccountPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getLedgerAccounts.mockResolvedValue({
      data: {
        data: [
          { id: cashId, name: "Cash", type: 1, code: 1000, currency: "usd", debits_posted: 500000, credits_posted: 100000, balance: 400000 },
        ],
      },
    });
    endpoints.getLedgerEntries.mockResolvedValue({
      data: {
        data: [
          {
            transaction_id: "tx1",
            timestamp: "2026-01-02T00:00:00Z",
            code: 3,
            amount: 118000,
            debit_account_id: cashId,
            debit_account_name: "Cash",
            debit_account_code: 1000,
            credit_account_id: "acct-ar-1100",
            credit_account_name: "Accounts Receivable",
            credit_account_code: 1100,
          },
        ],
      },
    });
  });

  it("shows the account identity, its position, and journal activity", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByRole("heading", { name: "Cash" })).toBeInTheDocument());
    expect(endpoints.getLedgerEntries).toHaveBeenCalledWith(
      expect.objectContaining({ account_id: cashId })
    );
    // Position: debits, credits, net balance.
    expect(screen.getByText("Position")).toBeInTheDocument();
    expect(screen.getByText(money("$5,000.00"))).toBeInTheDocument(); // debits posted
    expect(screen.getByText(money("$4,000.00"))).toBeInTheDocument(); // net balance
    // Journal activity: the posting hit Cash as a debit, against Accounts Receivable.
    expect(screen.getByText("Journal activity (1)")).toBeInTheDocument();
    expect(screen.getByText(/Accounts Receivable/)).toBeInTheDocument();
    expect(screen.getByText(money("$1,180.00"))).toBeInTheDocument();
  });

  it("names a per-customer sub-account from its postings when it's not in the chart", async () => {
    // The chart list doesn't include per-customer AR sub-accounts.
    endpoints.getLedgerAccounts.mockResolvedValue({ data: { data: [] } });
    endpoints.getLedgerEntries.mockResolvedValue({
      data: {
        data: [
          {
            transaction_id: "tx2",
            timestamp: "2026-01-02T00:00:00Z",
            code: 1,
            amount: 90000,
            debit_account_id: "acct-ar-cust",
            debit_account_name: "Accounts Receivable — Acme",
            debit_account_code: 1100,
            credit_account_id: "acct-deferred",
            credit_account_name: "Deferred Revenue",
            credit_account_code: 2100,
          },
        ],
      },
    });
    renderPage("acct-ar-cust");
    await waitFor(() =>
      expect(screen.getByRole("heading", { name: /Accounts Receivable — Acme/ })).toBeInTheDocument()
    );
    // No chart entry → running balance is not shown, but postings are.
    expect(screen.getByText(/per-customer sub-account/)).toBeInTheDocument();
    expect(screen.getByText(/Deferred Revenue/)).toBeInTheDocument();
  });

  it("shows a not-found state for an account with no chart entry and no postings", async () => {
    endpoints.getLedgerAccounts.mockResolvedValue({ data: { data: [] } });
    endpoints.getLedgerEntries.mockResolvedValue({ data: { data: [] } });
    renderPage("acct-ghost");
    await waitFor(() => expect(screen.getByText("Account not found")).toBeInTheDocument());
  });
});
