import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import RevenueByGeography from "../RevenueByGeography";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getRevenueByGeography: vi.fn() },
}));

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("RevenueByGeography page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders total MRR and per-country segments", async () => {
    endpoints.getRevenueByGeography.mockResolvedValue({
      data: {
        data: {
          reporting_currency: "USD",
          total_mrr: 100000,
          segments: [
            { label: "United States", mrr: 70000, share_pct: 70 },
            { label: "India", mrr: 30000, share_pct: 30 },
          ],
        },
      },
    });
    render(<RevenueByGeography />, { wrapper });
    await waitFor(() => expect(screen.getByText("Total MRR")).toBeInTheDocument());
    expect(screen.getByText("$1,000.00")).toBeInTheDocument();
    expect(screen.getAllByText("United States").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("$700.00").length).toBeGreaterThanOrEqual(1);
  });

  it("shows the empty state with no revenue", async () => {
    endpoints.getRevenueByGeography.mockResolvedValue({
      data: { data: { reporting_currency: "USD", total_mrr: 0, segments: [] } },
    });
    render(<RevenueByGeography />, { wrapper });
    await waitFor(() => expect(screen.getByText("No revenue yet")).toBeInTheDocument());
  });
});
