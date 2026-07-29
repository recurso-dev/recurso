import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import MRRWaterfall from "../MRRWaterfall";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getMRRWaterfall: vi.fn() },
}));
vi.mock("@/components/patterns/ReportScopeSelect", () => ({
  ReportScopeSelect: () => <div data-testid="scope-select" />,
}));

const wf = {
  reporting_currency: "USD",
  starting_mrr: 100000,
  new: 20000,
  expansion: 10000,
  reactivation: 5000,
  contraction: 3000,
  churned: 12000,
  ending_mrr: 120000,
  net_dollar_retention: 105.5,
  gross_dollar_retention: 92.0,
};

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("MRRWaterfall page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the starting/ending MRR and the net movement", async () => {
    endpoints.getMRRWaterfall.mockResolvedValue({ data: { data: wf } });
    render(<MRRWaterfall />, { wrapper });
    await waitFor(() =>
      expect(screen.getAllByText("Starting MRR").length).toBeGreaterThanOrEqual(1)
    );
    expect(screen.getAllByText("Ending MRR").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("$1,000.00").length).toBeGreaterThanOrEqual(1); // starting
    expect(screen.getAllByText("$1,200.00").length).toBeGreaterThanOrEqual(1); // ending
    // net change = ending - starting = +$200.00
    expect(screen.getAllByText(/\+\$200\.00/).length).toBeGreaterThanOrEqual(1);
  });

  it("requests the waterfall for the default (trailing-month) range", async () => {
    endpoints.getMRRWaterfall.mockResolvedValue({ data: { data: wf } });
    render(<MRRWaterfall />, { wrapper });
    await waitFor(() => expect(endpoints.getMRRWaterfall).toHaveBeenCalled());
    // Called with (start, end, params) — two date strings and a params object.
    const call = endpoints.getMRRWaterfall.mock.calls[0];
    expect(typeof call[0]).toBe("string");
    expect(typeof call[1]).toBe("string");
  });
});
