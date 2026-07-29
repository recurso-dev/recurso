import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import TrialBalance from "../TrialBalance";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getTrialBalance: vi.fn(),
    exportGeneralLedger: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
// The scope selector fetches entities; stub it out — it's not under test here.
vi.mock("@/components/patterns/ReportScopeSelect", () => ({
  ReportScopeSelect: () => <div data-testid="scope-select" />,
}));

const balanced = {
  balanced: true,
  reporting_currency: "USD",
  total_debits: 1000000,
  total_credits: 1000000,
  accounts: [
    { code: 1000, name: "Cash", debit: 1000000, credit: 0, balance: 1000000 },
    { code: 4000, name: "Revenue", debit: 0, credit: 1000000, balance: -1000000 },
  ],
};

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("TrialBalance page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows the balanced invariant and totals when books balance", async () => {
    endpoints.getTrialBalance.mockResolvedValue({ data: { data: balanced } });
    render(<TrialBalance />, { wrapper });
    await waitFor(() => expect(screen.getByText(/Books balance/i)).toBeInTheDocument());
    expect(screen.getByText("Total debits")).toBeInTheDocument();
    expect(screen.getByText("Total credits")).toBeInTheDocument();
    expect(screen.getByText("Balanced")).toBeInTheDocument();
  });

  it("flags an out-of-balance trial balance", async () => {
    endpoints.getTrialBalance.mockResolvedValue({
      data: { data: { ...balanced, balanced: false, total_credits: 900000 } },
    });
    render(<TrialBalance />, { wrapper });
    await waitFor(() => expect(screen.getByText(/Out of balance/i)).toBeInTheDocument());
    expect(screen.getByText("Unbalanced")).toBeInTheDocument();
  });
});
