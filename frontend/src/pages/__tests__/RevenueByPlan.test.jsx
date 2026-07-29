import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import RevenueByPlan from "../RevenueByPlan";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getRevenueByPlan: vi.fn() },
}));

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("RevenueByPlan page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders total MRR and per-plan segments", async () => {
    endpoints.getRevenueByPlan.mockResolvedValue({
      data: {
        data: {
          reporting_currency: "USD",
          total_mrr: 100000,
          segments: [
            { label: "Pro", mrr: 80000, share_pct: 80 },
            { label: "Starter", mrr: 20000, share_pct: 20 },
          ],
        },
      },
    });
    render(<RevenueByPlan />, { wrapper });
    await waitFor(() => expect(screen.getByText("Total MRR")).toBeInTheDocument());
    expect(screen.getByText("$1,000.00")).toBeInTheDocument(); // total
    expect(screen.getByText("$800.00")).toBeInTheDocument(); // Pro segment
    expect(screen.getByText("Starter")).toBeInTheDocument();
  });

  it("shows the empty state with no plan revenue", async () => {
    endpoints.getRevenueByPlan.mockResolvedValue({
      data: { data: { reporting_currency: "USD", total_mrr: 0, segments: [] } },
    });
    render(<RevenueByPlan />, { wrapper });
    await waitFor(() => expect(screen.getByText("No revenue yet")).toBeInTheDocument());
  });
});
