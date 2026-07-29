import { render, screen, waitFor } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Metering from "../Metering";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getBillableMetrics: vi.fn(),
    getUsageAlerts: vi.fn().mockResolvedValue({ data: { data: [] } }),
    createBillableMetric: vi.fn(),
    updateBillableMetric: vi.fn(),
    deleteBillableMetric: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Metering page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders billable metrics", async () => {
    endpoints.getBillableMetrics.mockResolvedValue({
      data: { data: [{ id: "m1", name: "API Calls", code: "api_calls", aggregation_type: "count" }] },
    });
    render(<Metering />, { wrapper });
    await waitFor(() => expect(screen.getByText("API Calls")).toBeInTheDocument());
  });

  it("shows the empty state with no metrics", async () => {
    endpoints.getBillableMetrics.mockResolvedValue({ data: { data: [] } });
    render(<Metering />, { wrapper });
    await waitFor(() => expect(screen.getByText("No billable metrics yet")).toBeInTheDocument());
  });
});
