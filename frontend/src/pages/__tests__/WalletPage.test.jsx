import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import WalletPage from "../WalletPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getWallet: vi.fn(),
    getWalletTransactions: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn().mockResolvedValue({ data: { data: [] } }),
    topUpWallet: vi.fn(),
    setWalletAutoRecharge: vi.fn(),
    closeWallet: vi.fn(),
  },
}));

const wallet = {
  id: "w-1",
  customer_id: "cus-1",
  currency: "USD",
  balance: 7000,
  auto_recharge_threshold: 1000,
  auto_recharge_amount: 5000,
  created_at: "2026-08-01T10:00:00Z",
  closed_at: null,
};

// Two open top-ups (paid 5000 + promo 2000 = 7000 balance, so the split
// reconciles) plus a drain that settled an invoice.
const txs = [
  {
    id: "t-1",
    type: "top_up",
    source: "manual",
    amount: 5000,
    remaining: 5000,
    balance_after: 5000,
    created_at: "2026-08-01T10:00:00Z",
  },
  {
    id: "t-2",
    type: "top_up",
    source: "promotional",
    amount: 2000,
    remaining: 2000,
    balance_after: 7000,
    created_at: "2026-08-02T10:00:00Z",
  },
  {
    id: "t-3",
    type: "drain",
    amount: 1500,
    balance_after: 5500,
    invoice_id: "inv-9",
    created_at: "2026-08-03T10:00:00Z",
  },
];

function renderPage(id = "w-1") {
  return render(
    <MemoryRouter initialEntries={[`/wallets/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/wallets/:id" element={<WalletPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("WalletPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getWallet.mockResolvedValue({ data: { data: wallet } });
    endpoints.getWalletTransactions.mockResolvedValue({ data: { data: txs } });
  });

  it("shows the balance, the reconciling paid/promo split, and the movement ledger", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Drainable balance")).toBeInTheDocument());
    // The refundable/forfeitable split is shown because it reconciles to balance.
    expect(screen.getByText("Refundable (paid)")).toBeInTheDocument();
    expect(screen.getByText("Forfeitable (promo)")).toBeInTheDocument();
    // Movement ledger renders each type.
    expect(screen.getByText("Drain")).toBeInTheDocument();
    expect(screen.getAllByText("Top-up").length).toBeGreaterThanOrEqual(2);
  });

  it("links a drain to the invoice it settled", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("link", { name: /settled an invoice/i })).toBeInTheDocument()
    );
    expect(screen.getByRole("link", { name: /settled an invoice/i })).toHaveAttribute(
      "href",
      "/invoices/inv-9"
    );
  });

  it("renders a not-found state when the wallet is missing", async () => {
    endpoints.getWallet.mockRejectedValue({ response: { status: 404 } });
    renderPage("missing");
    await waitFor(() => expect(screen.getByText(/not found/i)).toBeInTheDocument());
  });
});
