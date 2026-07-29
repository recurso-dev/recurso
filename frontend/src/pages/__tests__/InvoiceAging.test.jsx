import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import InvoiceAging from "../InvoiceAging";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getInvoiceAging: vi.fn() },
}));
vi.mock("@/components/patterns/ReportScopeSelect", () => ({
  ReportScopeSelect: () => <div data-testid="scope-select" />,
}));

const report = {
  reporting_currency: "USD",
  total_outstanding: 150000,
  total_count: 3,
  buckets: [
    { label: "current", amount: 100000, count: 2 },
    { label: "1-30", amount: 50000, count: 1 },
  ],
};

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("InvoiceAging page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows the outstanding total and aging buckets", async () => {
    endpoints.getInvoiceAging.mockResolvedValue({ data: { data: report } });
    render(<InvoiceAging />, { wrapper });
    await waitFor(() => expect(screen.getByText("Total outstanding")).toBeInTheDocument());
    // $1,500.00 total outstanding
    expect(screen.getByText("$1,500.00")).toBeInTheDocument();
    expect(screen.getByText("1–30 days")).toBeInTheDocument();
  });

  it("shows an all-clear empty state when nothing is outstanding", async () => {
    endpoints.getInvoiceAging.mockResolvedValue({
      data: { data: { reporting_currency: "USD", total_outstanding: 0, total_count: 0, buckets: [] } },
    });
    render(<InvoiceAging />, { wrapper });
    await waitFor(() => expect(screen.getByText(/No open invoices/i)).toBeInTheDocument());
  });
});
