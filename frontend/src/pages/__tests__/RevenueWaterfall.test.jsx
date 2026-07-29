import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import RevenueWaterfall from "../RevenueWaterfall";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getRevenueWaterfall: vi.fn(),
    getDeferredRollforward: vi.fn(),
  },
}));

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("RevenueWaterfall page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getDeferredRollforward.mockResolvedValue({
      data: { data: { opening: 0, additions: 0, recognized: 0, closing: 0 } },
    });
  });

  it("renders the total recognized from the recognition curve", async () => {
    endpoints.getRevenueWaterfall.mockResolvedValue({
      data: { data: { reporting_currency: "USD", total_recognized: 500000, total_deferred: 200000 } },
    });
    render(<RevenueWaterfall />, { wrapper });
    await waitFor(() => expect(screen.getByText("Total recognized")).toBeInTheDocument());
    // $5,000.00 recognized
    expect(screen.getByText("$5,000.00")).toBeInTheDocument();
  });
});
