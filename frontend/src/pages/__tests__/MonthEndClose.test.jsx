import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import MonthEndClose from "../MonthEndClose";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getClosePack: vi.fn(),
    exportGeneralLedger: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const basePack = {
  period: { month: 8, year: 2026 },
  trial_balance: { balanced: true, total_debits: 100000, total_credits: 100000, accounts: [] },
  reconciliation: { discrepancies: 0, invoices_checked: 10 },
  deferred_revenue: {
    rollforward: { opening: 686736_50, added: 33210_40, released: 29936_40, closing: 690010_50 },
    recognition: { deferred_balance: 4274_50 },
    awaiting_payment: 712590_00,
  },
};

const wrapper = ({ children }) => (
  <MemoryRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </MemoryRouter>
);

describe("MonthEndClose — deferred tie-out framing", () => {
  beforeEach(() => vi.clearAllMocks());

  // The audit's highest trust-ROI fix (#466): the word "unexplained" must never
  // reach a customer. A non-zero residual is presented as a reconciling item
  // with the schedules-build-at-payment explanation, not a scary label.
  it("never shows the word 'unexplained' and frames a residual as 'to reconcile'", async () => {
    endpoints.getClosePack.mockResolvedValue({
      data: { data: { ...basePack, deferred_revenue: { ...basePack.deferred_revenue, ties: false, unexplained_delta: -26854_00 } } },
    });
    render(<MonthEndClose />, { wrapper });

    await waitFor(() => expect(screen.getAllByText(/to reconcile/i).length).toBeGreaterThan(0));
    // The alarming word is gone from the rendered page.
    expect(screen.queryByText(/unexplained/i)).toBeNull();
    // The legitimate awaiting-payment story is surfaced with its explanation.
    expect(screen.getAllByText(/Awaiting payment/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/recognition schedule is built when the invoice is paid/i)).toBeInTheDocument();
  });

  it("shows a green tie-out when the books tie exactly", async () => {
    endpoints.getClosePack.mockResolvedValue({
      data: { data: { ...basePack, deferred_revenue: { ...basePack.deferred_revenue, ties: true, unexplained_delta: 0 } } },
    });
    render(<MonthEndClose />, { wrapper });

    await waitFor(() => expect(screen.getByText(/Deferred ties out/i)).toBeInTheDocument());
    expect(screen.queryByText(/to reconcile/i)).toBeNull();
    expect(screen.queryByText(/unexplained/i)).toBeNull();
  });
});
