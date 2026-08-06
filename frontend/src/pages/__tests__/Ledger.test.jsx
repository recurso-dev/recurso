import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Ledger from "../Ledger";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getLedgerAccounts: vi.fn(),
    getLedgerEntries: vi.fn(),
    getCustomers: vi.fn(),
  },
}));

// Only the tenant-level AR account is in the accounts list; the per-customer
// AR sub-account (id === customer id) is NOT — it used to fall back to a
// truncated raw UUID.
const ACCOUNTS = [
  { id: "acc-ar", name: "Accounts Receivable", code: 1100, currency: "USD" },
];

const ENTRIES = [
  {
    id: "txn-1",
    timestamp: "2026-08-01T00:00:00Z",
    amount: 12500,
    // Debit: a per-customer AR sub-account, named via the entry's own fields.
    debit_account_id: "cust-1",
    debit_account_name: "Accounts Receivable",
    debit_account_code: 1100,
    // Credit: a Revenue account not present in the accounts list.
    credit_account_id: "acc-rev",
    credit_account_name: "Revenue",
    credit_account_code: 4000,
    reference_id: "inv-1",
  },
];

const wrap = (ui) => (
  <MemoryRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {ui}
    </QueryClientProvider>
  </MemoryRouter>
);

describe("Ledger — account naming", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getLedgerAccounts.mockResolvedValue({ data: { data: ACCOUNTS } });
    endpoints.getLedgerEntries.mockResolvedValue({ data: { data: ENTRIES } });
    endpoints.getCustomers.mockResolvedValue({
      data: { data: [{ id: "cust-1", name: "Acme Inc" }] },
    });
  });

  it("names ledger accounts from the entry (never a truncated UUID), tagging AR with the customer", async () => {
    render(wrap(<Ledger />));

    // Per-customer AR sub-account: named from the entry + tagged with the
    // customer, not shown as "cust-1…".
    await waitFor(() =>
      expect(screen.getByText("Accounts Receivable — Acme Inc (1100)")).toBeInTheDocument()
    );
    // Revenue account (absent from the accounts list) is named from the entry.
    expect(screen.getByText("Revenue (4000)")).toBeInTheDocument();
    // No raw truncated account UUID leaks into the row.
    expect(screen.queryByText(/^cust-1…$/)).toBeNull();
    expect(screen.queryByText(/^acc-rev…$/)).toBeNull();
  });
});
